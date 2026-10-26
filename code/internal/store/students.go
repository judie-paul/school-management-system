package store

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"school/internal/auth"
)

type Student struct {
	ID          int64
	AdmissionNo string
	FullName    string
	ClassID     int64
	ClassName   string
	Active      bool
}

var admissionRe = regexp.MustCompile(`^[A-Z0-9/_-]{1,30}$`)

func cleanAdmissionNo(no string) (string, error) {
	no = strings.ToUpper(strings.TrimSpace(no))
	if !admissionRe.MatchString(no) {
		return "", invalid("Admission number must be 1-30 letters, digits, '/', '-' or '_'.")
	}
	return no, nil
}

func (s *Store) validateStudent(admissionNo, fullName string, classID int64) (string, string, error) {
	no, err := cleanAdmissionNo(admissionNo)
	if err != nil {
		return "", "", err
	}
	name, err := cleanText(fullName, "Full name", 100)
	if err != nil {
		return "", "", err
	}
	if err := s.mustExist("classes", classID, "Choose a valid class."); err != nil {
		return "", "", err
	}
	return no, name, nil
}

// CreateStudent registers a student and returns the plaintext access code.
// Only its hash is stored, so the caller must show it to staff now.
func (s *Store) CreateStudent(admissionNo, fullName string, classID int64) (int64, string, error) {
	no, name, err := s.validateStudent(admissionNo, fullName, classID)
	if err != nil {
		return 0, "", err
	}
	code := auth.NewAccessCode()
	hash, err := auth.HashSecret(auth.NormalizeAccessCode(code))
	if err != nil {
		return 0, "", err
	}
	res, err := s.db.Exec(`INSERT INTO students (admission_no, full_name, class_id, access_code_hash) VALUES (?, ?, ?, ?)`,
		no, name, classID, hash)
	if isUniqueViolation(err) {
		return 0, "", invalid("Admission number %s is already in use.", no)
	}
	if err != nil {
		return 0, "", err
	}
	id, err := res.LastInsertId()
	return id, code, err
}

func (s *Store) UpdateStudent(id int64, admissionNo, fullName string, classID int64, active bool) error {
	no, name, err := s.validateStudent(admissionNo, fullName, classID)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE students SET admission_no = ?, full_name = ?, class_id = ?, active = ? WHERE id = ?`,
		no, name, classID, active, id)
	if isUniqueViolation(err) {
		return invalid("Admission number %s is already in use.", no)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

const studentSelect = `SELECT s.id, s.admission_no, s.full_name, s.class_id, c.name, s.active
	FROM students s JOIN classes c ON c.id = s.class_id`

func scanStudent(row scanner) (*Student, error) {
	var st Student
	if err := row.Scan(&st.ID, &st.AdmissionNo, &st.FullName, &st.ClassID, &st.ClassName, &st.Active); err != nil {
		return nil, notFound(err)
	}
	return &st, nil
}

func (s *Store) StudentByID(id int64) (*Student, error) {
	return scanStudent(s.db.QueryRow(studentSelect+` WHERE s.id = ?`, id))
}

func (s *Store) queryStudents(query string, args ...any) ([]Student, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Student
	for rows.Next() {
		st, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *st)
	}
	return out, rows.Err()
}

// SearchStudents matches q against name or admission number. classID 0 means
// any class. At most 200 rows are returned.
func (s *Store) SearchStudents(q string, classID int64, includeInactive bool) ([]Student, error) {
	where := []string{"1=1"}
	var args []any
	if q = strings.TrimSpace(q); q != "" {
		like := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
		where = append(where, `(s.full_name LIKE ? ESCAPE '\' OR s.admission_no LIKE ? ESCAPE '\')`)
		args = append(args, like, like)
	}
	if classID != 0 {
		where = append(where, "s.class_id = ?")
		args = append(args, classID)
	}
	if !includeInactive {
		where = append(where, "s.active = 1")
	}
	return s.queryStudents(studentSelect+` WHERE `+strings.Join(where, " AND ")+
		` ORDER BY c.name, s.full_name LIMIT 200`, args...)
}

// ResetAccessCode issues a new access code and logs out any parent sessions
// that used the old one.
func (s *Store) ResetAccessCode(id int64) (string, error) {
	code := auth.NewAccessCode()
	hash, err := auth.HashSecret(auth.NormalizeAccessCode(code))
	if err != nil {
		return "", err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE students SET access_code_hash = ? WHERE id = ?`, hash, id)
	if err != nil {
		return "", err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "", ErrNotFound
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE student_id = ?`, id); err != nil {
		return "", err
	}
	return code, tx.Commit()
}

// AuthenticateParent checks an admission number and access code. Inactive
// students can still log in so parents keep access to historical records.
func (s *Store) AuthenticateParent(admissionNo, code string) (*Student, error) {
	var id int64
	var hash string
	err := s.db.QueryRow(`SELECT id, access_code_hash FROM students WHERE admission_no = ?`,
		strings.ToUpper(strings.TrimSpace(admissionNo))).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		auth.SpendCheckTime(code)
		return nil, ErrBadCredentials
	}
	if err != nil {
		return nil, err
	}
	if !auth.CheckSecret(hash, auth.NormalizeAccessCode(code)) {
		return nil, ErrBadCredentials
	}
	return s.StudentByID(id)
}
