package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"school/internal/store"
)

func (s *Server) examList(w http.ResponseWriter, r *http.Request) {
	exams, err := s.store.ListExams()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	terms, err := s.store.ListTerms()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	classes, err := s.store.ListClasses()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "exams.html", pageData{Title: "Examinations", Data: map[string]any{
		"Exams": exams, "Terms": terms, "Classes": classes,
	}})
}

func (s *Server) examCreate(w http.ResponseWriter, r *http.Request) {
	maxMarks, err := store.ParseMaxMarks(r.PostFormValue("max_marks"))
	if err != nil {
		s.fail(w, r, err, "/exams")
		return
	}
	id, err := s.store.CreateExam(r.PostFormValue("name"), formID(r, "term_id"), formID(r, "class_id"), maxMarks)
	if err != nil {
		s.fail(w, r, err, "/exams")
		return
	}
	s.redirect(w, r, fmt.Sprintf("/exams/%d", id), "Exam created. It is a draft until you publish it.")
}

type markRow struct {
	Student store.Student
	Value   string
}

// renderExam shows the marks entry grid for one subject. entered holds the
// values the user just submitted, so they are kept when validation fails.
func (s *Server) renderExam(w http.ResponseWriter, r *http.Request, examID, subjectID int64, entered map[int64]string, errMsg string) {
	exam, err := s.store.ExamByID(examID)
	if err != nil {
		s.fail(w, r, err, "/exams")
		return
	}
	subjects, err := s.store.ListSubjects()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if subjectID == 0 && len(subjects) > 0 {
		subjectID = subjects[0].ID
	}
	students, err := s.store.SearchStudents("", exam.ClassID, false)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	marks, err := s.store.ExamMarks(examID, subjectID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	rows := make([]markRow, len(students))
	for i, st := range students {
		rows[i].Student = st
		if v, ok := entered[st.ID]; ok {
			rows[i].Value = v
		} else if m, ok := marks[st.ID]; ok {
			rows[i].Value = strconv.FormatFloat(m, 'f', -1, 64)
		}
	}
	status := http.StatusOK
	if errMsg != "" {
		status = http.StatusUnprocessableEntity
	}
	s.render(w, r, status, "exam.html", pageData{Title: exam.Name, Error: errMsg, Data: map[string]any{
		"Exam": exam, "Subjects": subjects, "SubjectID": subjectID, "Rows": rows,
	}})
}

func (s *Server) examShow(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	s.renderExam(w, r, id, formID(r, "subject_id"), nil, "")
}

// examSaveMarks reads fields named m_<studentID> for the chosen subject.
func (s *Server) examSaveMarks(w http.ResponseWriter, r *http.Request) {
	examID, _ := pathID(r)
	subjectID := formID(r, "subject_id")
	var entries []store.MarkEntry
	entered := map[int64]string{}
	for key, vals := range r.PostForm {
		idStr, ok := strings.CutPrefix(key, "m_")
		if !ok || len(vals) != 1 {
			continue
		}
		sid, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		entries = append(entries, store.MarkEntry{StudentID: sid, Raw: vals[0]})
		entered[sid] = vals[0]
	}
	err := s.store.SaveExamMarks(examID, subjectID, identityFrom(r).staff.ID, entries)
	var ve *store.ValidationError
	if errors.As(err, &ve) {
		s.renderExam(w, r, examID, subjectID, entered, ve.Msg+" No marks were saved.")
		return
	}
	if err != nil {
		s.fail(w, r, err, "/exams")
		return
	}
	s.redirect(w, r, fmt.Sprintf("/exams/%d?subject_id=%d", examID, subjectID), "Marks saved.")
}

func (s *Server) examPublish(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	publish := r.PostFormValue("published") == "1"
	if err := s.store.SetExamPublished(id, publish); err != nil {
		s.fail(w, r, err, "/exams")
		return
	}
	msg := "Exam published. Parents can now see its results."
	if !publish {
		msg = "Exam moved back to draft. Parents can no longer see its results."
	}
	s.redirect(w, r, fmt.Sprintf("/exams/%d", id), msg)
}
