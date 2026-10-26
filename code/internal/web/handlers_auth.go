package web

import (
	"errors"
	"net/http"

	"school/internal/store"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	switch {
	case id.staff != nil:
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	case id.student != nil:
		http.Redirect(w, r, "/parent", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/parent/login", http.StatusSeeOther)
	}
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "login.html", pageData{Title: "Staff login"})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	st, err := s.store.AuthenticateStaff(username, r.PostFormValue("password"))
	if errors.Is(err, store.ErrBadCredentials) {
		s.render(w, r, http.StatusUnauthorized, "login.html", pageData{
			Title: "Staff login", Error: "Invalid username or password.", Data: username,
		})
		return
	}
	if err == nil {
		err = s.startSession(w, r, st.ID, 0)
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) parentLoginPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "parent_login.html", pageData{Title: "Parent login"})
}

func (s *Server) parentLogin(w http.ResponseWriter, r *http.Request) {
	admission := r.PostFormValue("admission_no")
	stu, err := s.store.AuthenticateParent(admission, r.PostFormValue("access_code"))
	if errors.Is(err, store.ErrBadCredentials) {
		s.render(w, r, http.StatusUnauthorized, "parent_login.html", pageData{
			Title: "Parent login", Error: "Admission number or access code is incorrect.", Data: admission,
		})
		return
	}
	if err == nil {
		err = s.startSession(w, r, 0, stu.ID)
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/parent", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if id.token != "" {
		if err := s.store.DeleteSession(id.token); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1})
	target := "/parent/login"
	if id.staff != nil {
		target = "/login"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "dashboard.html", pageData{Title: "Dashboard"})
}
