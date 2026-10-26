package web

import (
	"fmt"
	"net/http"
	"time"

	"school/internal/store"
)

func (s *Server) feesIndex(w http.ResponseWriter, r *http.Request) {
	termID, classID := formID(r, "term_id"), formID(r, "class_id")
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
	var rows []store.ClassFeeRow
	if termID != 0 && classID != 0 {
		if rows, err = s.store.ClassFeeSummaries(classID, termID); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	s.render(w, r, http.StatusOK, "fees.html", pageData{Title: "Fees", Data: map[string]any{
		"Terms": terms, "Classes": classes, "TermID": termID, "ClassID": classID, "Rows": rows,
	}})
}

func (s *Server) feesClassAssessment(w http.ResponseWriter, r *http.Request) {
	termID, classID := formID(r, "term_id"), formID(r, "class_id")
	back := fmt.Sprintf("/fees?term_id=%d&class_id=%d", termID, classID)
	amount, err := store.ParseMoney(r.PostFormValue("amount"))
	if err != nil {
		s.fail(w, r, err, back)
		return
	}
	n, err := s.store.SetClassFeeAssessment(classID, termID, amount)
	if err != nil {
		s.fail(w, r, err, back)
		return
	}
	s.redirect(w, r, back, fmt.Sprintf("Required fees of %s set for %d active student(s).", store.FormatMoney(amount), n))
}

func (s *Server) studentFees(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	stu, err := s.store.StudentByID(id)
	if err != nil {
		s.fail(w, r, err, "/students")
		return
	}
	terms, err := s.store.ListTerms()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	summaries, err := s.store.FeeSummaries(id)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	termID := formID(r, "term_id")
	if termID == 0 && len(terms) > 0 {
		termID = terms[0].ID
	}
	data := map[string]any{
		"Student": stu, "Terms": terms, "Summaries": summaries, "TermID": termID,
		"Today": time.Now().Format("2006-01-02"),
	}
	if termID != 0 {
		sel, err := s.store.FeeSummary(id, termID)
		if err != nil {
			s.fail(w, r, err, fmt.Sprintf("/students/%d/fees", id))
			return
		}
		payments, err := s.store.Payments(id, termID)
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		data["Selected"], data["Payments"] = sel, payments
	}
	s.render(w, r, http.StatusOK, "student_fees.html", pageData{Title: "Fees: " + stu.FullName, Data: data})
}

func (s *Server) studentFeeAssessment(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	termID := formID(r, "term_id")
	back := fmt.Sprintf("/students/%d/fees?term_id=%d", id, termID)
	amount, err := store.ParseMoney(r.PostFormValue("amount"))
	if err == nil {
		err = s.store.SetFeeAssessment(id, termID, amount)
	}
	if err != nil {
		s.fail(w, r, err, back)
		return
	}
	s.redirect(w, r, back, "Required fees saved.")
}

func (s *Server) studentPayment(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	termID := formID(r, "term_id")
	back := fmt.Sprintf("/students/%d/fees?term_id=%d", id, termID)
	amount, err := store.ParseMoney(r.PostFormValue("amount"))
	if err == nil {
		err = s.store.RecordPayment(id, termID, amount, r.PostFormValue("paid_on"),
			r.PostFormValue("reference"), identityFrom(r).staff.ID)
	}
	if err != nil {
		s.fail(w, r, err, back)
		return
	}
	s.redirect(w, r, back, fmt.Sprintf("Payment of %s recorded.", store.FormatMoney(amount)))
}
