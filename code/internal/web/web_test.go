package web_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"school"
	"school/internal/store"
	"school/internal/testutil"
	"school/internal/web"
)

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func newServer(t *testing.T, st *store.Store) *httptest.Server {
	t.Helper()
	srv, err := web.New(st, school.Assets, false)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func newClient(t *testing.T, ts *httptest.Server) *client {
	jar, _ := cookiejar.New(nil)
	c := &client{t: t, base: ts.URL, http: &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // inspect redirects instead of following
		},
	}}
	c.get("/login") // obtain the CSRF cookie
	return c
}

func (c *client) csrf() string {
	u, _ := url.Parse(c.base)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "csrf_token" {
			return ck.Value
		}
	}
	return ""
}

func (c *client) get(path string) (int, string) {
	c.t.Helper()
	resp, err := c.http.Get(c.base + path)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func (c *client) post(path string, form url.Values) (int, string) {
	c.t.Helper()
	if form.Get("csrf_token") == "" {
		form.Set("csrf_token", c.csrf())
	}
	resp, err := c.http.PostForm(c.base+path, form)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func (c *client) staffLogin(user, pass string) {
	c.t.Helper()
	if code, _ := c.post("/login", url.Values{"username": {user}, "password": {pass}}); code != http.StatusSeeOther {
		c.t.Fatalf("staff login failed: status %d", code)
	}
}

func (c *client) parentLogin(admission, code string) int {
	c.t.Helper()
	status, _ := c.post("/parent/login", url.Values{"admission_no": {admission}, "access_code": {code}})
	return status
}

// twoStudents creates two students in the same class with distinct published
// marks and fee records.
func twoStudents(t *testing.T, st *store.Store, f testutil.Fixture) (aID int64, aCode string, bID int64) {
	t.Helper()
	aID, aCode, err := st.CreateStudent("ADM-A", "Alice Akinyi", f.ClassID)
	if err != nil {
		t.Fatal(err)
	}
	bID, _, err = st.CreateStudent("ADM-B", "Brian Barasa", f.ClassID)
	if err != nil {
		t.Fatal(err)
	}
	exam, err := st.CreateExam("Midterm", f.Term1, f.ClassID, 100)
	if err != nil {
		t.Fatal(err)
	}
	err = st.SaveExamMarks(exam, f.MathID, f.AdminID, []store.MarkEntry{{StudentID: aID, Raw: "77"}, {StudentID: bID, Raw: "33"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetExamPublished(exam, true); err != nil {
		t.Fatal(err)
	}
	if err := st.SetFeeAssessment(aID, f.Term1, 1111100); err != nil {
		t.Fatal(err)
	}
	if err := st.SetFeeAssessment(bID, f.Term1, 2222200); err != nil {
		t.Fatal(err)
	}
	return aID, aCode, bID
}

func TestParentLogin(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	_, code, _ := twoStudents(t, st, f)
	ts := newServer(t, st)

	c := newClient(t, ts)
	if status := c.parentLogin("ADM-A", "WRONG-CODE"); status != http.StatusUnauthorized {
		t.Fatalf("wrong code: status %d", status)
	}
	if status, _ := c.get("/parent"); status != http.StatusSeeOther {
		t.Fatalf("portal reachable without login: status %d", status)
	}
	if status := c.parentLogin("ADM-A", code); status != http.StatusSeeOther {
		t.Fatalf("correct code: status %d", status)
	}
	status, body := c.get("/parent")
	if status != http.StatusOK || !strings.Contains(body, "Alice Akinyi") {
		t.Fatalf("portal home: status %d", status)
	}

	// After logout the portal is closed again.
	c.post("/logout", url.Values{})
	if status, _ := c.get("/parent"); status != http.StatusSeeOther {
		t.Fatalf("portal reachable after logout: status %d", status)
	}
}

func TestParentCannotSeeOtherStudent(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	_, aCode, bID := twoStudents(t, st, f)
	ts := newServer(t, st)
	c := newClient(t, ts)
	if c.parentLogin("ADM-A", aCode) != http.StatusSeeOther {
		t.Fatal("login failed")
	}

	// Tampered query parameters must not switch the student.
	for _, path := range []string{
		"/parent?student_id=" + fmt.Sprint(bID),
		"/parent/results?student_id=" + fmt.Sprint(bID),
		"/parent/fees?student_id=" + fmt.Sprint(bID) + "&term_id=" + fmt.Sprint(f.Term1),
	} {
		status, body := c.get(path)
		if status != http.StatusOK {
			t.Fatalf("%s: status %d", path, status)
		}
		if strings.Contains(body, "Brian") || strings.Contains(body, "33 / 100") || strings.Contains(body, "22,222.00") {
			t.Errorf("%s leaked the other student's data", path)
		}
	}
	_, body := c.get("/parent/results")
	if !strings.Contains(body, "77 / 100") {
		t.Error("own results missing")
	}
	_, body = c.get("/parent/fees")
	if !strings.Contains(body, "11,111.00") {
		t.Error("own fees missing")
	}

	// Staff pages are closed to parents.
	if status, _ := c.get(fmt.Sprintf("/students/%d", bID)); status != http.StatusSeeOther {
		t.Errorf("parent reached staff student page: status %d", status)
	}
	if status, _ := c.get(fmt.Sprintf("/students/%d/fees", bID)); status != http.StatusSeeOther {
		t.Errorf("parent reached staff fees page: status %d", status)
	}
}

func TestDraftResultsHiddenFromParents(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	id, code, err := st.CreateStudent("ADM-D", "Dana", f.ClassID)
	if err != nil {
		t.Fatal(err)
	}
	exam, _ := st.CreateExam("Draft Exam", f.Term1, f.ClassID, 100)
	if err := st.SaveExamMarks(exam, f.MathID, f.AdminID, []store.MarkEntry{{StudentID: id, Raw: "64"}}); err != nil {
		t.Fatal(err)
	}
	ts := newServer(t, st)
	c := newClient(t, ts)
	c.parentLogin("ADM-D", code)

	_, body := c.get(fmt.Sprintf("/parent/results?exam_id=%d", exam))
	if strings.Contains(body, "Draft Exam") || strings.Contains(body, "64 / 100") {
		t.Fatal("draft exam visible to parent")
	}
	st.SetExamPublished(exam, true)
	_, body = c.get(fmt.Sprintf("/parent/results?exam_id=%d", exam))
	if !strings.Contains(body, "64 / 100") {
		t.Fatal("published exam not visible to parent")
	}
}

func TestTeacherCannotModifyFees(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	aID, _, _ := twoStudents(t, st, f)
	if _, err := st.CreateStaff("teacher1", "Tom Teacher", store.RoleTeacher, "teacher-pass"); err != nil {
		t.Fatal(err)
	}
	ts := newServer(t, st)
	c := newClient(t, ts)
	c.staffLogin("teacher1", "teacher-pass")

	term := fmt.Sprint(f.Term1)
	attempts := map[string]url.Values{
		fmt.Sprintf("/students/%d/fees/payments", aID):   {"term_id": {term}, "amount": {"500"}, "paid_on": {"2026-01-10"}},
		fmt.Sprintf("/students/%d/fees/assessment", aID): {"term_id": {term}, "amount": {"0"}},
		"/fees/class-assessment":                         {"term_id": {term}, "class_id": {fmt.Sprint(f.ClassID)}, "amount": {"0"}},
	}
	for path, form := range attempts {
		if status, _ := c.post(path, form); status != http.StatusForbidden {
			t.Errorf("POST %s as teacher: status %d, want 403", path, status)
		}
	}
	if status, _ := c.get(fmt.Sprintf("/students/%d/fees", aID)); status != http.StatusForbidden {
		t.Errorf("teacher viewed fee page: status %d", status)
	}

	sum, _ := st.FeeSummary(aID, f.Term1)
	if sum.Paid != 0 || sum.Required != 1111100 {
		t.Fatalf("fee records changed by teacher: %+v", sum)
	}

	// Teachers can still enter marks.
	exams, _ := st.ListExams()
	status, _ := c.post(fmt.Sprintf("/exams/%d/marks", exams[0].ID), url.Values{
		"subject_id": {fmt.Sprint(f.EnglishID)}, fmt.Sprintf("m_%d", aID): {"50"},
	})
	if status != http.StatusSeeOther {
		t.Fatalf("teacher could not save marks: status %d", status)
	}

	// Finance staff can record payments.
	st.CreateStaff("finance1", "Fiona Finance", store.RoleFinance, "finance-pass")
	fc := newClient(t, ts)
	fc.staffLogin("finance1", "finance-pass")
	status, _ = fc.post(fmt.Sprintf("/students/%d/fees/payments", aID), url.Values{"term_id": {term}, "amount": {"500"}, "paid_on": {"2026-01-10"}})
	if status != http.StatusSeeOther {
		t.Fatalf("finance could not record payment: status %d", status)
	}
	sum, _ = st.FeeSummary(aID, f.Term1)
	if sum.Paid != 50000 {
		t.Fatalf("payment not recorded: %+v", sum)
	}
}

func TestCSRFRequired(t *testing.T) {
	st := testutil.NewStore(t)
	testutil.Seed(t, st)
	ts := newServer(t, st)
	c := newClient(t, ts)
	c.staffLogin("admin", "admin-password")
	status, _ := c.post("/setup/classes", url.Values{"name": {"Hacked"}, "csrf_token": {"wrong-token-wrong-token-wrong-token"}})
	if status != http.StatusForbidden {
		t.Fatalf("POST with bad CSRF token: status %d, want 403", status)
	}
	classes, _ := st.ListClasses()
	for _, cl := range classes {
		if cl.Name == "Hacked" {
			t.Fatal("class created despite bad CSRF token")
		}
	}
}

// TestPagesRender smoke-tests that every staff page renders for an admin.
func TestPagesRender(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	aID, _, _ := twoStudents(t, st, f)
	ts := newServer(t, st)
	c := newClient(t, ts)
	c.staffLogin("admin", "admin-password")
	exams, _ := st.ListExams()
	for _, p := range []string{
		"/dashboard", "/students", "/students?q=ali", "/students/new", fmt.Sprintf("/students/%d", aID),
		fmt.Sprintf("/students/%d/edit", aID), fmt.Sprintf("/students/%d/fees", aID),
		"/exams", fmt.Sprintf("/exams/%d", exams[0].ID), "/fees",
		fmt.Sprintf("/fees?term_id=%d&class_id=%d", f.Term1, f.ClassID), "/staff", "/setup",
	} {
		if status, body := c.get(p); status != http.StatusOK {
			t.Errorf("GET %s: status %d: %.200s", p, status, body)
		}
	}
}
