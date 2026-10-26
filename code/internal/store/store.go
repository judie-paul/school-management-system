// Package store contains all database access and the business rules that
// go with it (validation, fee balances, result checks).
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store { return &Store{db: db} }

var (
	ErrNotFound       = errors.New("not found")
	ErrBadCredentials = errors.New("invalid credentials")
)

// ValidationError is a user-facing message that is safe to show in the UI.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// cleanText collapses whitespace and enforces a required, length-limited field.
func cleanText(s, field string, max int) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "", invalid("%s is required.", field)
	}
	if utf8.RuneCountInString(s) > max {
		return "", invalid("%s must be at most %d characters.", field, max)
	}
	return s, nil
}

func (s *Store) exists(table string, id int64) (bool, error) {
	var n int
	// table is always a constant supplied by this package, never user input.
	err := s.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

func (s *Store) mustExist(table string, id int64, msg string) error {
	ok, err := s.exists(table, id)
	if err != nil {
		return err
	}
	if !ok {
		return invalid("%s", msg)
	}
	return nil
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
