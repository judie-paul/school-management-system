package store_test

import (
	"errors"
	"testing"
	"time"

	"school/internal/store"
	"school/internal/testutil"
)

func isValidation(err error) bool {
	var v *store.ValidationError
	return errors.As(err, &v)
}

func TestDuplicateAdmissionNumberRejected(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)

	if _, _, err := st.CreateStudent("ADM-001", "Jane Doe", f.ClassID); err != nil {
		t.Fatal(err)
	}
	// Same number in different case and with whitespace is still a duplicate.
	_, _, err := st.CreateStudent("  adm-001 ", "John Doe", f.ClassID)
	if !isValidation(err) {
		t.Fatalf("expected validation error for duplicate admission number, got %v", err)
	}

	// Editing another student to take the number is also rejected.
	id2, _, err := st.CreateStudent("ADM-002", "John Doe", f.ClassID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateStudent(id2, "ADM-001", "John Doe", f.ClassID, true); !isValidation(err) {
		t.Fatalf("expected validation error on update, got %v", err)
	}
}

func TestParentAuthentication(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	id, code, err := st.CreateStudent("ADM-100", "Mary Wanjiku", f.ClassID)
	if err != nil {
		t.Fatal(err)
	}

	got, err := st.AuthenticateParent("adm-100", code)
	if err != nil || got.ID != id {
		t.Fatalf("valid credentials rejected: %v", err)
	}
	// Codes are accepted without the dash and in lower case.
	if _, err := st.AuthenticateParent("ADM-100", " "+string(append([]byte(code[:4]), code[5:]...))+" "); err != nil {
		t.Fatalf("normalized code rejected: %v", err)
	}
	if _, err := st.AuthenticateParent("ADM-100", "WRNG-CODE"); !errors.Is(err, store.ErrBadCredentials) {
		t.Fatalf("wrong code: got %v", err)
	}
	if _, err := st.AuthenticateParent("NOPE", code); !errors.Is(err, store.ErrBadCredentials) {
		t.Fatalf("unknown admission number: got %v", err)
	}

	newCode, err := st.ResetAccessCode(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AuthenticateParent("ADM-100", code); !errors.Is(err, store.ErrBadCredentials) {
		t.Fatal("old code still works after reset")
	}
	if _, err := st.AuthenticateParent("ADM-100", newCode); err != nil {
		t.Fatalf("new code rejected: %v", err)
	}
}

func TestInvalidMarksRejected(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	stu, _, _ := st.CreateStudent("ADM-1", "A Student", f.ClassID)
	other, _, _ := st.CreateStudent("ADM-2", "Other Class", f.OtherClassID)
	exam, err := st.CreateExam("Midterm", f.Term1, f.ClassID, 50)
	if err != nil {
		t.Fatal(err)
	}

	for _, raw := range []string{"-1", "50.5", "abc", "NaN", "Inf"} {
		err := st.SaveExamMarks(exam, f.MathID, f.AdminID, []store.MarkEntry{{StudentID: stu, Raw: raw}})
		if !isValidation(err) {
			t.Errorf("marks %q: expected validation error, got %v", raw, err)
		}
	}
	// Student from another class cannot get results in this exam.
	err = st.SaveExamMarks(exam, f.MathID, f.AdminID, []store.MarkEntry{{StudentID: other, Raw: "10"}})
	if !isValidation(err) {
		t.Errorf("student from other class: expected validation error, got %v", err)
	}

	// One bad entry rejects the whole batch.
	err = st.SaveExamMarks(exam, f.MathID, f.AdminID, []store.MarkEntry{{StudentID: stu, Raw: "40"}, {StudentID: stu, Raw: "99"}})
	if !isValidation(err) {
		t.Fatalf("expected batch rejection, got %v", err)
	}
	marks, _ := st.ExamMarks(exam, f.MathID)
	if len(marks) != 0 {
		t.Fatalf("no marks should be saved after a rejected batch, got %v", marks)
	}

	// Valid marks save, and saving again updates rather than duplicating.
	for _, raw := range []string{"42.5", "45"} {
		if err := st.SaveExamMarks(exam, f.MathID, f.AdminID, []store.MarkEntry{{StudentID: stu, Raw: raw}}); err != nil {
			t.Fatal(err)
		}
	}
	res, _ := st.StudentResults(stu, exam, false)
	if len(res) != 1 || res[0].Marks != 45 || res[0].Percent() != 90 {
		t.Fatalf("unexpected results: %+v", res)
	}
}

func TestFeeBalances(t *testing.T) {
	st := testutil.NewStore(t)
	f := testutil.Seed(t, st)
	stu, _, _ := st.CreateStudent("ADM-1", "A Student", f.ClassID)
	today := time.Now().Format("2006-01-02")

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(st.SetFeeAssessment(stu, f.Term1, 1500000)) // 15,000.00
	must(st.SetFeeAssessment(stu, f.Term2, 1200000)) // 12,000.00

	sum, _ := st.FeeSummary(stu, f.Term1)
	if sum.Status() != "Unpaid" || sum.Outstanding() != 1500000 {
		t.Fatalf("before payment: %s %d", sum.Status(), sum.Outstanding())
	}

	// Partial payment.
	must(st.RecordPayment(stu, f.Term1, 500000, today, "MPESA-1", f.AdminID))
	sum, _ = st.FeeSummary(stu, f.Term1)
	if sum.Status() != "Partially paid" || sum.Outstanding() != 1000000 || sum.Paid != 500000 {
		t.Fatalf("partial: %s outstanding=%d paid=%d", sum.Status(), sum.Outstanding(), sum.Paid)
	}

	// Fully paid gives zero balance.
	must(st.RecordPayment(stu, f.Term1, 1000000, today, "", f.AdminID))
	sum, _ = st.FeeSummary(stu, f.Term1)
	if sum.Status() != "Fully paid" || sum.Balance() != 0 || sum.Credit() != 0 {
		t.Fatalf("full: %s balance=%d", sum.Status(), sum.Balance())
	}

	// Term 2 is independent of term 1 payments.
	sum2, _ := st.FeeSummary(stu, f.Term2)
	if sum2.Status() != "Unpaid" || sum2.Outstanding() != 1200000 || sum2.Paid != 0 {
		t.Fatalf("term 2 affected by term 1 payments: %+v", sum2)
	}

	// Overpayment shows as credit, not unpaid.
	must(st.RecordPayment(stu, f.Term2, 1300000, today, "", f.AdminID))
	sum2, _ = st.FeeSummary(stu, f.Term2)
	if sum2.Status() != "Fully paid" || sum2.Credit() != 100000 || sum2.Outstanding() != 0 {
		t.Fatalf("overpayment: %s credit=%d", sum2.Status(), sum2.Credit())
	}

	all, _ := st.FeeSummaries(stu)
	if len(all) != 2 {
		t.Fatalf("expected 2 term summaries, got %d", len(all))
	}

	// Invalid payments are rejected.
	if err := st.RecordPayment(stu, f.Term1, 0, today, "", f.AdminID); !isValidation(err) {
		t.Errorf("zero amount accepted: %v", err)
	}
	if err := st.RecordPayment(stu, f.Term1, 100, "2999-01-01", "", f.AdminID); !isValidation(err) {
		t.Errorf("future date accepted: %v", err)
	}
}

func TestParseAndFormatMoney(t *testing.T) {
	for in, want := range map[string]int64{"1500": 150000, "1,500.5": 150050, "0.05": 5, "12000.00": 1200000} {
		got, err := store.ParseMoney(in)
		if err != nil || got != want {
			t.Errorf("ParseMoney(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "-5", "1.234", "abc", "1e5"} {
		if _, err := store.ParseMoney(bad); err == nil {
			t.Errorf("ParseMoney(%q) accepted", bad)
		}
	}
	if got := store.FormatMoney(123456789); got != "1,234,567.89" {
		t.Errorf("FormatMoney = %q", got)
	}
	if got := store.FormatMoney(-5); got != "-0.05" {
		t.Errorf("FormatMoney negative = %q", got)
	}
}
