// Package web contains the HTTP handlers, middleware and template rendering.
package web

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"school/internal/auth"
	"school/internal/store"
)

const (
	sessionCookie = "sid"
	csrfCookie    = "csrf_token"
	flashCookie   = "flash"
	sessionTTL    = 12 * time.Hour
)

type Server struct {
	store         *store.Store
	pages         map[string]*template.Template
	static        fs.FS
	secureCookies bool
	mux           *http.ServeMux
}

// New builds the server. assets must contain templates/ and static/.
// secureCookies should be true when the site is served over HTTPS.
func New(st *store.Store, assets fs.FS, secureCookies bool) (*Server, error) {
	pages, err := loadTemplates(assets)
	if err != nil {
		return nil, err
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	s := &Server{store: st, pages: pages, static: static, secureCookies: secureCookies}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(s.csrf(s.loadSession(s.mux)))
}

// ---- templates ----

var funcs = template.FuncMap{
	"money": store.FormatMoney,
	"marks": func(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) },
	"pct":   func(f float64) string { return fmt.Sprintf("%.1f%%", f) },
	"isRole": func(st *store.Staff, roles ...string) bool {
		return st != nil && slices.Contains(roles, st.Role)
	},
}

func loadTemplates(assets fs.FS) (map[string]*template.Template, error) {
	files, err := fs.Glob(assets, "templates/*.html")
	if err != nil {
		return nil, err
	}
	pages := map[string]*template.Template{}
	for _, f := range files {
		name := path.Base(f)
		if name == "layout.html" {
			continue
		}
		t, err := template.New(name).Funcs(funcs).ParseFS(assets, "templates/layout.html", f)
		if err != nil {
			return nil, err
		}
		pages[name] = t
	}
	return pages, nil
}

// pageData is passed to every template. Handlers set Title, Error and Data;
// render fills in the rest.
type pageData struct {
	Title   string
	Staff   *store.Staff
	Student *store.Student // set in the parent portal
	CSRF    string
	Flash   string
	Error   string
	Data    any
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, page string, pd pageData) {
	id := identityFrom(r)
	pd.Staff, pd.Student = id.staff, id.student
	pd.CSRF = csrfFrom(r)
	if c, err := r.Cookie(flashCookie); err == nil {
		http.SetCookie(w, &http.Cookie{Name: flashCookie, Path: "/", MaxAge: -1})
		if v, err := url.QueryUnescape(c.Value); err == nil {
			kind, msg, _ := strings.Cut(v, "|")
			if kind == "err" && pd.Error == "" {
				pd.Error = msg
			} else if kind == "ok" {
				pd.Flash = msg
			}
		}
	}
	t, ok := s.pages[page]
	if !ok {
		s.serverError(w, r, fmt.Errorf("unknown template %s", page))
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", pd); err != nil {
		s.serverError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (s *Server) setFlash(w http.ResponseWriter, kind, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name: flashCookie, Value: url.QueryEscape(kind + "|" + msg), Path: "/",
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	})
}

// redirect sends the browser to target with an optional success message.
func (s *Server) redirect(w http.ResponseWriter, r *http.Request, target, okMsg string) {
	if okMsg != "" {
		s.setFlash(w, "ok", okMsg)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// fail handles an error from the store: validation errors go back to the
// form page as a message, anything else is logged and hidden from the user.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error, back string) {
	var ve *store.ValidationError
	switch {
	case errors.As(err, &ve):
		s.setFlash(w, "err", ve.Msg)
		http.Redirect(w, r, back, http.StatusSeeOther)
	case errors.Is(err, store.ErrNotFound):
		s.notFound(w, r)
	default:
		s.serverError(w, r, err)
	}
}

func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("error: %s %s: %v", r.Method, r.URL.Path, err)
	http.Error(w, "Something went wrong. Please try again.", http.StatusInternalServerError)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusNotFound, "error.html", pageData{Title: "Not found", Data: "The page you requested was not found."})
}

func (s *Server) forbidden(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusForbidden, "error.html", pageData{Title: "Not allowed", Data: "You do not have permission to do that."})
}

// ---- request helpers ----

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

// formID reads a positive integer from the query string or form; 0 if absent.
func formID(r *http.Request, key string) int64 {
	id, err := strconv.ParseInt(r.FormValue(key), 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
}

// ---- middleware ----

type ctxKey int

const (
	keyIdentity ctxKey = iota
	keyCSRF
)

// identity is who is making the request: a staff member, a parent (bound to
// exactly one student) or nobody.
type identity struct {
	token   string
	staff   *store.Staff
	student *store.Student
}

func identityFrom(r *http.Request) *identity {
	if id, ok := r.Context().Value(keyIdentity).(*identity); ok {
		return id
	}
	return &identity{}
}

func csrfFrom(r *http.Request) string {
	v, _ := r.Context().Value(keyCSRF).(string)
	return v
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

// csrf implements the double-submit cookie pattern: every non-GET request
// must carry a form field matching the csrf_token cookie. Combined with
// SameSite cookies this blocks cross-site form posts, including login CSRF.
func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, err := r.Cookie(csrfCookie); err == nil && len(c.Value) >= 32 {
			token = c.Value
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			sent := r.PostFormValue("csrf_token")
			if token == "" || subtle.ConstantTimeCompare([]byte(sent), []byte(token)) != 1 {
				http.Error(w, "Your form expired. Go back, reload the page and try again.", http.StatusForbidden)
				return
			}
		}
		if token == "" {
			token = auth.RandomToken()
			http.SetCookie(w, &http.Cookie{
				Name: csrfCookie, Value: token, Path: "/",
				HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
			})
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), keyCSRF, token)))
	})
}

func (s *Server) loadSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := &identity{}
		if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
			sess, err := s.store.SessionByToken(c.Value)
			switch {
			case err == nil:
				id.token = c.Value
				if sess.StaffID != 0 {
					if st, err := s.store.StaffByID(sess.StaffID); err == nil && st.Active {
						id.staff = st
					}
				} else if stu, err := s.store.StudentByID(sess.StudentID); err == nil {
					id.student = stu
				}
			case !errors.Is(err, store.ErrNotFound):
				log.Printf("session lookup: %v", err)
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), keyIdentity, id)))
	})
}

// requireStaff allows only logged-in staff, optionally limited to roles.
func (s *Server) requireStaff(h http.HandlerFunc, roles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st := identityFrom(r).staff
		if st == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if len(roles) > 0 && !slices.Contains(roles, st.Role) {
			s.forbidden(w, r)
			return
		}
		h(w, r)
	}
}

// requireParent allows only a logged-in parent. Parent handlers must take the
// student from identityFrom(r).student, never from the request.
func (s *Server) requireParent(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if identityFrom(r).student == nil {
			http.Redirect(w, r, "/parent/login", http.StatusSeeOther)
			return
		}
		h(w, r)
	}
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, staffID, studentID int64) error {
	if old := identityFrom(r).token; old != "" {
		s.store.DeleteSession(old)
	}
	token, err := s.store.CreateSession(staffID, studentID, sessionTTL)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", MaxAge: int(sessionTTL.Seconds()),
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	})
	return nil
}
