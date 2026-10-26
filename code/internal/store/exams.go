package store

import (
	"math"
	"strconv"
	"strings"
)

type Exam struct {
	ID        int64
	Name      string
	TermID    int64
	Year      string
	TermName  string
	ClassID   int64
	ClassName string
	MaxMarks  float64
	Published bool
}

func (e Exam) TermLabel() string { return e.Year + " - " + e.TermName }

type Result struct {
	SubjectID   int64
	SubjectName string
	Marks       float64
	MaxMarks    float64
}

func (r Result) Percent() float64 { return r.Marks / r.MaxMarks * 100 }

const examSelect = `SELECT e.id, e.name, e.term_id, y.name, t.name, e.class_id, c.name, e.max_marks, e.published
	FROM exams e
	JOIN terms t ON t.id = e.term_id
	JOIN academic_years y ON y.id = t.academic_year_id
	JOIN classes c ON c.id = e.class_id`

const examOrder = ` ORDER BY y.name DESC, t.id DESC, c.name, e.name`

func scanExam(row scanner) (*Exam, error) {
	var e Exam
	err := row.Scan(&e.ID, &e.Name, &e.TermID, &e.Year, &e.TermName, &e.ClassID, &e.ClassName, &e.MaxMarks, &e.Published)
	if err != nil {
		return nil, notFound(err)
	}
	return &e, nil
}

func (s *Store) queryExams(query string, args ...any) ([]Exam, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Exam
	for rows.Next() {
		e, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// ParseMaxMarks parses the maximum marks field; blank means 100.
func ParseMaxMarks(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 100, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > 1000 {
		return 0, invalid("Maximum marks must be a number greater than 0 and at most 1000.")
	}
	return v, nil
}

func (s *Store) CreateExam(name string, termID, classID int64, maxMarks float64) (int64, error) {
	name, err := cleanText(name, "Exam name", 80)
	if err != nil {
		return 0, err
	}
	if maxMarks <= 0 || maxMarks > 1000 || math.IsNaN(maxMarks) {
		return 0, invalid("Maximum marks must be greater than 0 and at most 1000.")
	}
	if err := s.mustExist("terms", termID, "Choose a valid term."); err != nil {
		return 0, err
	}
	if err := s.mustExist("classes", classID, "Choose a valid class."); err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO exams (name, term_id, class_id, max_marks) VALUES (?, ?, ?, ?)`,
		name, termID, classID, maxMarks)
	if isUniqueViolation(err) {
		return 0, invalid("An exam called %q already exists for that class and term.", name)
	}
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListExams() ([]Exam, error) { return s.queryExams(examSelect + examOrder) }

func (s *Store) ExamByID(id int64) (*Exam, error) {
	return scanExam(s.db.QueryRow(examSelect+` WHERE e.id = ?`, id))
}

func (s *Store) SetExamPublished(id int64, published bool) error {
	res, err := s.db.Exec(`UPDATE exams SET published = ? WHERE id = ?`, published, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ParseMarks validates a marks value against the exam's maximum.
func ParseMarks(raw string, max float64) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > max {
		return 0, invalid("Marks must be a number between 0 and %s.", strconv.FormatFloat(max, 'f', -1, 64))
	}
	return v, nil
}

// MarkEntry is one student's raw marks from the entry form. A blank value
// removes any existing result for that student and subject.
type MarkEntry struct {
	StudentID int64
	Raw       string
}

// SaveExamMarks validates every entry first, then writes them all in one
// transaction, so either all marks are saved or none are.
func (s *Store) SaveExamMarks(examID, subjectID, staffID int64, entries []MarkEntry) error {
	exam, err := s.ExamByID(examID)
	if err != nil {
		return err
	}
	if err := s.mustExist("subjects", subjectID, "Choose a valid subject."); err != nil {
		return err
	}
	students, err := s.SearchStudents("", exam.ClassID, true)
	if err != nil {
		return err
	}
	inClass := make(map[int64]string, len(students))
	for _, st := range students {
		inClass[st.ID] = st.FullName
	}

	type op struct {
		studentID int64
		marks     float64
		remove    bool
	}
	ops := make([]op, 0, len(entries))
	for _, e := range entries {
		name, ok := inClass[e.StudentID]
		if !ok {
			return invalid("A student in the form is not in class %s.", exam.ClassName)
		}
		if strings.TrimSpace(e.Raw) == "" {
			ops = append(ops, op{studentID: e.StudentID, remove: true})
			continue
		}
		m, err := ParseMarks(e.Raw, exam.MaxMarks)
		if err != nil {
			return invalid("%s: %s", name, err.Error())
		}
		ops = append(ops, op{studentID: e.StudentID, marks: m})
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, o := range ops {
		if o.remove {
			_, err = tx.Exec(`DELETE FROM exam_results WHERE student_id = ? AND exam_id = ? AND subject_id = ?`,
				o.studentID, examID, subjectID)
		} else {
			_, err = tx.Exec(`INSERT INTO exam_results (student_id, exam_id, subject_id, marks, max_marks, updated_by)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT (student_id, exam_id, subject_id) DO UPDATE SET
					marks = excluded.marks, max_marks = excluded.max_marks,
					updated_by = excluded.updated_by, updated_at = CURRENT_TIMESTAMP`,
				o.studentID, examID, subjectID, o.marks, exam.MaxMarks, staffID)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ExamMarks returns studentID -> marks for one exam subject.
func (s *Store) ExamMarks(examID, subjectID int64) (map[int64]float64, error) {
	rows, err := s.db.Query(`SELECT student_id, marks FROM exam_results WHERE exam_id = ? AND subject_id = ?`, examID, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]float64{}
	for rows.Next() {
		var id int64
		var m float64
		if err := rows.Scan(&id, &m); err != nil {
			return nil, err
		}
		out[id] = m
	}
	return out, rows.Err()
}

// StudentExams lists exams the student has results in. With publishedOnly,
// draft exams are excluded (used by the parent portal).
func (s *Store) StudentExams(studentID int64, publishedOnly bool) ([]Exam, error) {
	return s.queryExams(examSelect+`
		WHERE (e.published = 1 OR ? = 0)
		AND EXISTS (SELECT 1 FROM exam_results r WHERE r.exam_id = e.id AND r.student_id = ?)`+examOrder,
		publishedOnly, studentID)
}

// StudentResults returns one student's subject results for an exam.
func (s *Store) StudentResults(studentID, examID int64, publishedOnly bool) ([]Result, error) {
	rows, err := s.db.Query(`SELECT r.subject_id, sub.name, r.marks, r.max_marks
		FROM exam_results r
		JOIN subjects sub ON sub.id = r.subject_id
		JOIN exams e ON e.id = r.exam_id
		WHERE r.student_id = ? AND r.exam_id = ? AND (e.published = 1 OR ? = 0)
		ORDER BY sub.name`, studentID, examID, publishedOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.SubjectID, &r.SubjectName, &r.Marks, &r.MaxMarks); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
