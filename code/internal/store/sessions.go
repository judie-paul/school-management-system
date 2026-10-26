package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"school/internal/auth"
)

// Session belongs to either a staff member or a parent (via StudentID).
type Session struct {
	StaffID   int64
	StudentID int64
}

// Only a hash of the session token is stored, so a leaked database cannot be
// used to hijack sessions.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func nullID(id int64) sql.NullInt64 { return sql.NullInt64{Int64: id, Valid: id != 0} }

// CreateSession starts a session for a staff member (studentID 0) or a parent
// (staffID 0) and returns the token for the cookie.
func (s *Store) CreateSession(staffID, studentID int64, ttl time.Duration) (string, error) {
	now := time.Now()
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now.Unix()); err != nil {
		return "", err
	}
	token := auth.RandomToken()
	_, err := s.db.Exec(`INSERT INTO sessions (token_hash, staff_id, student_id, expires_at) VALUES (?, ?, ?, ?)`,
		hashToken(token), nullID(staffID), nullID(studentID), now.Add(ttl).Unix())
	return token, err
}

func (s *Store) SessionByToken(token string) (*Session, error) {
	var staffID, studentID sql.NullInt64
	err := s.db.QueryRow(`SELECT staff_id, student_id FROM sessions WHERE token_hash = ? AND expires_at > ?`,
		hashToken(token), time.Now().Unix()).Scan(&staffID, &studentID)
	if err != nil {
		return nil, notFound(err)
	}
	return &Session{StaffID: staffID.Int64, StudentID: studentID.Int64}, nil
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}
