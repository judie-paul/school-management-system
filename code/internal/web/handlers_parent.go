package web

import (
	"net/http"
)

// Every handler here uses identityFrom(r).student, set from the session.
// No student ID is ever read from the URL or form, so a parent cannot reach
// another student's records by editing the request.

func (s *Server) parentHome(w http.ResponseWriter, r *http.Request) {
	stu := identityFrom(r).student
	s.render(w, r, http.StatusOK, "parent_home.html", pageData{Title: stu.FullName})
}

func (s *Server) parentResults(w http.ResponseWriter, r *http.Request) {
	stu := identityFrom(r).student
	exams, err := s.store.StudentExams(stu.ID, true)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	examID := formID(r, "exam_id")
	if examID == 0 && len(exams) > 0 {
		examID = exams[0].ID
	}
	data := map[string]any{"Exams": exams, "ExamID": examID}
	// Only show an exam from the parent's own published list.
	for _, e := range exams {
		if e.ID == examID {
			res, err := s.store.StudentResults(stu.ID, e.ID, true)
			if err != nil {
				s.serverError(w, r, err)
				return
			}
			data["Selected"] = examResults{Exam: e, Results: res}
		}
	}
	s.render(w, r, http.StatusOK, "parent_results.html", pageData{Title: "Exam results", Data: data})
}

func (s *Server) parentFees(w http.ResponseWriter, r *http.Request) {
	stu := identityFrom(r).student
	summaries, err := s.store.FeeSummaries(stu.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	termID := formID(r, "term_id")
	if termID == 0 && len(summaries) > 0 {
		termID = summaries[0].Term.ID
	}
	data := map[string]any{"Summaries": summaries, "TermID": termID}
	for _, f := range summaries {
		if f.Term.ID == termID {
			payments, err := s.store.Payments(stu.ID, termID)
			if err != nil {
				s.serverError(w, r, err)
				return
			}
			data["Selected"], data["Payments"] = f, payments
		}
	}
	s.render(w, r, http.StatusOK, "parent_fees.html", pageData{Title: "School fees", Data: data})
}
