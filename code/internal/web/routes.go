package web

import (
	"net/http"

	"school/internal/store"
)

const (
	admin   = store.RoleAdmin
	teacher = store.RoleTeacher
	finance = store.RoleFinance
)

func (s *Server) routes() {
	m := http.NewServeMux()
	staff, parent := s.requireStaff, s.requireParent

	m.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(s.static)))
	m.HandleFunc("GET /{$}", s.home)

	m.HandleFunc("GET /login", s.loginPage)
	m.HandleFunc("POST /login", s.login)
	m.HandleFunc("GET /parent/login", s.parentLoginPage)
	m.HandleFunc("POST /parent/login", s.parentLogin)
	m.HandleFunc("POST /logout", s.logout)

	m.HandleFunc("GET /dashboard", staff(s.dashboard))

	// Students: every staff role can view; only admins change them.
	m.HandleFunc("GET /students", staff(s.studentList))
	m.HandleFunc("GET /students/new", staff(s.studentNew, admin))
	m.HandleFunc("POST /students", staff(s.studentCreate, admin))
	m.HandleFunc("GET /students/{id}", staff(s.studentShow))
	m.HandleFunc("GET /students/{id}/edit", staff(s.studentEdit, admin))
	m.HandleFunc("POST /students/{id}", staff(s.studentUpdate, admin))
	m.HandleFunc("POST /students/{id}/reset-code", staff(s.studentResetCode, admin))

	// Exams and results: admins and teachers; only admins create/publish.
	m.HandleFunc("GET /exams", staff(s.examList, admin, teacher))
	m.HandleFunc("POST /exams", staff(s.examCreate, admin))
	m.HandleFunc("GET /exams/{id}", staff(s.examShow, admin, teacher))
	m.HandleFunc("POST /exams/{id}/marks", staff(s.examSaveMarks, admin, teacher))
	m.HandleFunc("POST /exams/{id}/publish", staff(s.examPublish, admin))

	// Fees: admins and finance staff only. Teachers are never allowed here.
	m.HandleFunc("GET /fees", staff(s.feesIndex, admin, finance))
	m.HandleFunc("POST /fees/class-assessment", staff(s.feesClassAssessment, admin, finance))
	m.HandleFunc("GET /students/{id}/fees", staff(s.studentFees, admin, finance))
	m.HandleFunc("POST /students/{id}/fees/assessment", staff(s.studentFeeAssessment, admin, finance))
	m.HandleFunc("POST /students/{id}/fees/payments", staff(s.studentPayment, admin, finance))

	// Administration.
	m.HandleFunc("GET /staff", staff(s.staffList, admin))
	m.HandleFunc("POST /staff", staff(s.staffCreate, admin))
	m.HandleFunc("POST /staff/{id}/active", staff(s.staffSetActive, admin))
	m.HandleFunc("POST /staff/{id}/password", staff(s.staffSetPassword, admin))
	m.HandleFunc("GET /setup", staff(s.setupPage, admin))
	m.HandleFunc("POST /setup/classes", staff(s.setupClass, admin))
	m.HandleFunc("POST /setup/subjects", staff(s.setupSubject, admin))
	m.HandleFunc("POST /setup/terms", staff(s.setupTerm, admin))

	// Parent portal.
	m.HandleFunc("GET /parent", parent(s.parentHome))
	m.HandleFunc("GET /parent/results", parent(s.parentResults))
	m.HandleFunc("GET /parent/fees", parent(s.parentFees))

	s.mux = m
}
