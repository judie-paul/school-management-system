package web

import (
	"net/http"

	"school/internal/store"
)

func (s *Server) staffList(w http.ResponseWriter, r *http.Request) {
	staff, err := s.store.ListStaff()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "staff.html", pageData{Title: "Staff accounts", Data: map[string]any{
		"StaffList": staff, "Roles": store.Roles,
	}})
}

func (s *Server) staffCreate(w http.ResponseWriter, r *http.Request) {
	_, err := s.store.CreateStaff(r.PostFormValue("username"), r.PostFormValue("full_name"),
		r.PostFormValue("role"), r.PostFormValue("password"))
	if err != nil {
		s.fail(w, r, err, "/staff")
		return
	}
	s.redirect(w, r, "/staff", "Staff account created.")
}

func (s *Server) staffSetActive(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	active := r.PostFormValue("active") == "1"
	if !active && id == identityFrom(r).staff.ID {
		s.fail(w, r, &store.ValidationError{Msg: "You cannot deactivate your own account."}, "/staff")
		return
	}
	if err := s.store.SetStaffActive(id, active); err != nil {
		s.fail(w, r, err, "/staff")
		return
	}
	s.redirect(w, r, "/staff", "Account updated.")
}

func (s *Server) staffSetPassword(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r)
	if err := s.store.SetStaffPassword(id, r.PostFormValue("password")); err != nil {
		s.fail(w, r, err, "/staff")
		return
	}
	if id == identityFrom(r).staff.ID {
		// Changing your own password ends your sessions, including this one.
		s.redirect(w, r, "/login", "Password changed. Please log in again.")
		return
	}
	s.redirect(w, r, "/staff", "Password changed. That user has been logged out.")
}

func (s *Server) setupPage(w http.ResponseWriter, r *http.Request) {
	classes, err := s.store.ListClasses()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	subjects, err := s.store.ListSubjects()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	terms, err := s.store.ListTerms()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "setup.html", pageData{Title: "School setup", Data: map[string]any{
		"Classes": classes, "Subjects": subjects, "Terms": terms,
	}})
}

func (s *Server) setupClass(w http.ResponseWriter, r *http.Request) {
	if _, err := s.store.CreateClass(r.PostFormValue("name")); err != nil {
		s.fail(w, r, err, "/setup")
		return
	}
	s.redirect(w, r, "/setup", "Class added.")
}

func (s *Server) setupSubject(w http.ResponseWriter, r *http.Request) {
	if _, err := s.store.CreateSubject(r.PostFormValue("name")); err != nil {
		s.fail(w, r, err, "/setup")
		return
	}
	s.redirect(w, r, "/setup", "Subject added.")
}

func (s *Server) setupTerm(w http.ResponseWriter, r *http.Request) {
	if _, err := s.store.CreateTerm(r.PostFormValue("year"), r.PostFormValue("name")); err != nil {
		s.fail(w, r, err, "/setup")
		return
	}
	s.redirect(w, r, "/setup", "Term added.")
}
