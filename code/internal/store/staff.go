package store

import (
	"errors"
	"regexp"
	"slices"
	"strings"

	"school/internal/auth"
)

const (
	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
	RoleFinance = "finance"
)

var Roles = []string{RoleAdmin, RoleTeacher, RoleFinance}

type Staff struct {
	ID           int64
	Username     string
	FullName     string
	Role         string
	Active       bool
	passwordHash string
}

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9._-]{3,40}$`)

func validatePassword(p string) error {
	if len(p) < 8 {
		return invalid("Password must be at least 8 characters.")
	}
	if len(p) > 72 { // bcrypt limit
		return invalid("Password must be at most 72 characters.")
	}
	return nil
}

func (s *Store) CreateStaff(username, fullName, role, password string) (int64, error) {
	username = strings.TrimSpace(username)
	if !usernameRe.MatchString(username) {
		return 0, invalid("Username must be 3-40 letters, digits, dots, dashes or underscores.")
	}
	fullName, err := cleanText(fullName, "Full name", 100)
	if err != nil {
		return 0, err
	}
	if !slices.Contains(Roles, role) {
		return 0, invalid("Choose a valid role.")
	}
	if err := validatePassword(password); err != nil {
		return 0, err
	}
	hash, err := auth.HashSecret(password)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO staff (username, full_name, role, password_hash) VALUES (?, ?, ?, ?)`,
		username, fullName, role, hash)
	if isUniqueViolation(err) {
		return 0, invalid("The username %q is already taken.", username)
	}
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const staffCols = `id, username, full_name, role, active, password_hash`

type scanner interface{ Scan(dest ...any) error }

func scanStaff(row scanner) (*Staff, error) {
	var st Staff
	if err := row.Scan(&st.ID, &st.Username, &st.FullName, &st.Role, &st.Active, &st.passwordHash); err != nil {
		return nil, notFound(err)
	}
	return &st, nil
}

func (s *Store) StaffByID(id int64) (*Staff, error) {
	return scanStaff(s.db.QueryRow(`SELECT `+staffCols+` FROM staff WHERE id = ?`, id))
}

// AuthenticateStaff returns the active staff member matching the credentials,
// or ErrBadCredentials.
func (s *Store) AuthenticateStaff(username, password string) (*Staff, error) {
	st, err := scanStaff(s.db.QueryRow(`SELECT `+staffCols+` FROM staff WHERE username = ?`, strings.TrimSpace(username)))
	if errors.Is(err, ErrNotFound) {
		auth.SpendCheckTime(password)
		return nil, ErrBadCredentials
	}
	if err != nil {
		return nil, err
	}
	if !auth.CheckSecret(st.passwordHash, password) || !st.Active {
		return nil, ErrBadCredentials
	}
	return st, nil
}

func (s *Store) ListStaff() ([]Staff, error) {
	rows, err := s.db.Query(`SELECT ` + staffCols + ` FROM staff ORDER BY active DESC, full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Staff
	for rows.Next() {
		st, err := scanStaff(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *st)
	}
	return out, rows.Err()
}

func (s *Store) CountStaff() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM staff`).Scan(&n)
	return n, err
}

// SetStaffActive enables or disables an account. Disabling also ends the
// account's sessions.
func (s *Store) SetStaffActive(id int64, active bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE staff SET active = ? WHERE id = ?`, active, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if !active {
		if _, err := tx.Exec(`DELETE FROM sessions WHERE staff_id = ?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetStaffPassword sets a new password and ends the account's sessions.
func (s *Store) SetStaffPassword(id int64, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := auth.HashSecret(password)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE staff SET password_hash = ? WHERE id = ?`, hash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE staff_id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// StaffIDByUsername is used by the command-line password reset.
func (s *Store) StaffIDByUsername(username string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM staff WHERE username = ?`, username).Scan(&id)
	return id, notFound(err)
}
