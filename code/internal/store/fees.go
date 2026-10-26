package store

import (
	"database/sql"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FeeSummary is the fee position of one student for one term. Each term is
// independent: payments for one term never affect another term's balance.
type FeeSummary struct {
	Term     Term
	Assessed bool  // whether required fees have been set
	Required int64 // cents
	Paid     int64 // cents
}

// Balance is Required - Paid. Negative means the student is in credit.
func (f FeeSummary) Balance() int64 { return f.Required - f.Paid }

func (f FeeSummary) Outstanding() int64 { return max(f.Balance(), 0) }

func (f FeeSummary) Credit() int64 { return max(-f.Balance(), 0) }

func (f FeeSummary) Status() string {
	switch {
	case !f.Assessed && f.Paid == 0:
		return "Not set"
	case f.Paid >= f.Required:
		return "Fully paid"
	case f.Paid > 0:
		return "Partially paid"
	default:
		return "Unpaid"
	}
}

type Payment struct {
	ID         int64
	Amount     int64 // cents
	PaidOn     string
	Reference  string
	RecordedBy string
}

var moneyRe = regexp.MustCompile(`^\d{1,10}(\.\d{1,2})?$`)

// ParseMoney parses "1500", "1,500.5" or "1500.50" into cents.
func ParseMoney(raw string) (int64, error) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if !moneyRe.MatchString(raw) {
		return 0, invalid("Enter an amount like 1500 or 1500.50.")
	}
	whole, frac, _ := strings.Cut(raw, ".")
	for len(frac) < 2 {
		frac += "0"
	}
	w, _ := strconv.ParseInt(whole, 10, 64)
	f, _ := strconv.ParseInt(frac, 10, 64)
	return w*100 + f, nil
}

// FormatMoney formats cents as "1,500.00" (with a leading "-" if negative).
func FormatMoney(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := strconv.FormatInt(cents/100, 10)
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return sign + b.String() + "." + strconv.FormatInt(cents%100+100, 10)[1:]
}

func (s *Store) checkStudentTerm(studentID, termID int64) error {
	if err := s.mustExist("students", studentID, "Student not found."); err != nil {
		return err
	}
	return s.mustExist("terms", termID, "Choose a valid term.")
}

// SetFeeAssessment sets (or replaces) the required fees for a student's term.
func (s *Store) SetFeeAssessment(studentID, termID, amount int64) error {
	if amount < 0 {
		return invalid("Required fees cannot be negative.")
	}
	if err := s.checkStudentTerm(studentID, termID); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO fee_assessments (student_id, term_id, amount_cents) VALUES (?, ?, ?)
		ON CONFLICT (student_id, term_id) DO UPDATE SET amount_cents = excluded.amount_cents, updated_at = CURRENT_TIMESTAMP`,
		studentID, termID, amount)
	return err
}

// SetClassFeeAssessment sets the same required fees for every active student
// in a class and returns how many students were updated.
func (s *Store) SetClassFeeAssessment(classID, termID, amount int64) (int64, error) {
	if amount < 0 {
		return 0, invalid("Required fees cannot be negative.")
	}
	if err := s.mustExist("classes", classID, "Choose a valid class."); err != nil {
		return 0, err
	}
	if err := s.mustExist("terms", termID, "Choose a valid term."); err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO fee_assessments (student_id, term_id, amount_cents)
		SELECT id, ?, ? FROM students WHERE class_id = ? AND active = 1
		ON CONFLICT (student_id, term_id) DO UPDATE SET amount_cents = excluded.amount_cents, updated_at = CURRENT_TIMESTAMP`,
		termID, amount, classID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RecordPayment stores a payment. paidOn must be YYYY-MM-DD and not in the future.
func (s *Store) RecordPayment(studentID, termID, amount int64, paidOn, reference string, staffID int64) error {
	if amount <= 0 {
		return invalid("Payment amount must be greater than zero.")
	}
	paidOn = strings.TrimSpace(paidOn)
	d, err := time.Parse("2006-01-02", paidOn)
	if err != nil {
		return invalid("Payment date must be a valid date (YYYY-MM-DD).")
	}
	if d.After(time.Now().AddDate(0, 0, 1)) {
		return invalid("Payment date cannot be in the future.")
	}
	reference = strings.Join(strings.Fields(reference), " ")
	if len(reference) > 100 {
		return invalid("Reference must be at most 100 characters.")
	}
	if err := s.checkStudentTerm(studentID, termID); err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO fee_payments (student_id, term_id, amount_cents, paid_on, reference, recorded_by)
		VALUES (?, ?, ?, ?, ?, ?)`, studentID, termID, amount, paidOn, reference, staffID)
	return err
}

func (s *Store) Payments(studentID, termID int64) ([]Payment, error) {
	rows, err := s.db.Query(`SELECT p.id, p.amount_cents, p.paid_on, p.reference, st.full_name
		FROM fee_payments p JOIN staff st ON st.id = p.recorded_by
		WHERE p.student_id = ? AND p.term_id = ?
		ORDER BY p.paid_on DESC, p.id DESC`, studentID, termID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Payment
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.Amount, &p.PaidOn, &p.Reference, &p.RecordedBy); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// feeCols selects required fees (NULL if not assessed) and total paid for
// student ? and term t, in that order of placeholders.
const feeCols = `
	(SELECT fa.amount_cents FROM fee_assessments fa WHERE fa.student_id = ?1 AND fa.term_id = t.id),
	(SELECT COALESCE(SUM(p.amount_cents), 0) FROM fee_payments p WHERE p.student_id = ?1 AND p.term_id = t.id)`

func scanFee(row scanner, f *FeeSummary, extra ...any) error {
	var req sql.NullInt64
	dest := append([]any{&f.Term.ID, &f.Term.Year, &f.Term.Name, &req, &f.Paid}, extra...)
	if err := row.Scan(dest...); err != nil {
		return notFound(err)
	}
	f.Assessed, f.Required = req.Valid, req.Int64
	return nil
}

func (s *Store) FeeSummary(studentID, termID int64) (FeeSummary, error) {
	var f FeeSummary
	err := scanFee(s.db.QueryRow(`SELECT t.id, y.name, t.name,`+feeCols+`
		FROM terms t JOIN academic_years y ON y.id = t.academic_year_id
		WHERE t.id = ?2`, studentID, termID), &f)
	return f, err
}

// FeeSummaries lists every term for which the student has fees set or
// payments recorded, newest first.
func (s *Store) FeeSummaries(studentID int64) ([]FeeSummary, error) {
	rows, err := s.db.Query(`SELECT t.id, y.name, t.name,`+feeCols+`
		FROM terms t JOIN academic_years y ON y.id = t.academic_year_id
		WHERE EXISTS (SELECT 1 FROM fee_assessments fa WHERE fa.student_id = ?1 AND fa.term_id = t.id)
		   OR EXISTS (SELECT 1 FROM fee_payments p WHERE p.student_id = ?1 AND p.term_id = t.id)
		ORDER BY y.name DESC, t.id DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FeeSummary
	for rows.Next() {
		var f FeeSummary
		if err := scanFee(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

type ClassFeeRow struct {
	Student Student
	FeeSummary
}

// ClassFeeSummaries returns the fee position of every active student in a
// class for one term.
func (s *Store) ClassFeeSummaries(classID, termID int64) ([]ClassFeeRow, error) {
	students, err := s.SearchStudents("", classID, false)
	if err != nil {
		return nil, err
	}
	out := make([]ClassFeeRow, 0, len(students))
	for _, st := range students {
		f, err := s.FeeSummary(st.ID, termID)
		if err != nil {
			return nil, err
		}
		out = append(out, ClassFeeRow{Student: st, FeeSummary: f})
	}
	return out, nil
}
