package web

import (
	"errors"
	"fmt"
	"net/http"

	"school/internal/store"
)

func (s *Server) studentList(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("q")
	classID := formID(r, "class_id")
	inactive := r.FormValue("inactive") == "1"
	students, err := s.store.SearchStudents(q, classID, inactive)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	classes, err := s.store.ListClasses()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "students.html", pageData{Title: "Students", Data: map[string]any{
		"Students": students, "Classes": classes, "Q": q, "ClassID": classID, "Inactive": inactive,
	}})
}

type studentForm struct {
	IsNew   bool
	Student store.Student
	Classes []store.Class
}

func (s *Server) renderStudentForm(w http.ResponseWriter, r *http.Request, status int, f studentForm, errMsg string) {
	classes, err := s.store.ListClasses()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	f.Classes = classes
	title := "Edit student"
	if f.IsNew {
		title = "Register student"
	}
	s.render(w, r, status, "student_form.html", pageData{Title: title, Error: errMsg, Data: f})
}

func studentFromForm(r *http.Request) store.Student {
	return store.Student{
		AdmissionNo: r.PostFormValue("admission_no"),
		FullName:    r.PostFormValue("full_name"),
		ClassID:     formID(r, "class_id"),
		Active:      r.PostFormValue("active") == "1",
	}
}

func (s *Server) studentNew(w http.ResponseWriter, r *http.Request) {
	s.renderStudentForm(w, r, http.StatusOK, studentForm{IsNew: true, Student: store.Student{Active: true}}, "")
}

func (s *Server) studentCreate(w http.ResponseWriter, r *http.Request) {
	in := studentFromForm(r)
	id, code, err := s.store.CreateStudent(in.AdmissionNo, in.FullName, in.ClassID)
	var ve *store.ValidationError
	if errors.As(err, &ve) {
		in.Active = true
		s.renderStudentForm(w, r, http.StatusUnprocessableEntity, studentForm{IsNew: true, Student: in}, ve.Msg)
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.showAccessCode(w, r, id, code, "Student registered.")
}

// showAccessCode renders the one-time page displaying a parent access code.
// The code is never stored in plaintext, so this is the only time it is shown.
func (s *Server) showAccessCode(w http.ResponseWriter, r *http.Request, studentID int64, code, msg string) {
	stu, err := s.store.StudentByID(studentID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	s.render(w, r, http.StatusOK, "access_code.html", pageData{Title: "Parent access code", Flash: msg, Data: map[string]any{
		"Student": stu, "Code": code,
	}})
}

type examResults struct {
	Exam    store.Exam
	Results []store.Result
}

func (e examResults) Totals() (total, max float64) {
	for _, r := range e.Results {
		total += r.Marks
		max += r.MaxMarks
	}
	return
}

func (e examResults) Total() float64 { t, _ := e.Totals(); return t }
func (e examResults) Max() float64   { _, m := e.Totals(); return m }
func (e examResults) Percent() float64 {
	t, m := e.Totals()
	if m == 0 {
		return 0
	}
	return t / m * 100
}

func (s *Server) loadExamResults(studentID int64, publishedOnly bool) ([]examResults, error) {
	exams, err := s.store.StudentExams(studentID, publishedOnly)
	if err != nil {
		return nil, err
	}
	out := make([]examResults, 0, len(exams))
	for _, e := range exams {
		res, err := s.store.StudentResults(studentID, e.ID, publishedOnly)
		if err != nil {
			return nil, err
		}
		out = append(out, examResults{Exam: e, Results: res})
	}
	return out, nil
}

func (s *Server) studentShow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.notFound(w, r)
		return
	}
	stu, err := s.store.StudentByID(id)
	if err != nil {
		s.fail(w, r, err, "/students")
		return
	}
	data := map[string]any{"Student": stu}
	role := identityFrom(r).staff.Role
	if role == admin || role == teacher {
		if data["Exams"], err = s.loadExamResults(id, false); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	if role == admin || role == finance {
		if data["Fees"], err = s.store.FeeSummaries(id); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	s.render(w, r, http.StatusOK, "student.html", pageData{Title: stu.FullName, Data: data})
}

func (s *Server) studentEdit(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	stu, err := s.store.StudentByID(id)
	if err != nil {
		s.fail(w, r, err, "/students")
		return
	}
	s.renderStudentForm(w, r, http.StatusOK, studentForm{Student: *stu}, "")
}

func (s *Server) studentUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	in := studentFromForm(r)
	in.ID = id
	err := s.store.UpdateStudent(id, in.AdmissionNo, in.FullName, in.ClassID, in.Active)
	var ve *store.ValidationError
	if errors.As(err, &ve) {
		s.renderStudentForm(w, r, http.StatusUnprocessableEntity, studentForm{Student: in}, ve.Msg)
		return
	}
	if err != nil {
		s.fail(w, r, err, "/students")
		return
	}
	s.redirect(w, r, fmt.Sprintf("/students/%d", id), "Student updated.")
}

func (s *Server) studentResetCode(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	code, err := s.store.ResetAccessCode(id)
	if err != nil {
		s.fail(w, r, err, "/students")
		return
	}
	s.showAccessCode(w, r, id, code, "A new access code was generated. The old code no longer works.")
}
