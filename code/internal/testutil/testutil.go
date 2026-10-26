// Package testutil creates throwaway databases for tests.
package testutil

import (
	"io/fs"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"school"
	"school/internal/auth"
	"school/internal/db"
	"school/internal/store"
)

// NewStore returns a store backed by a fresh, migrated SQLite file in a
// temporary directory that is removed when the test ends.
func NewStore(t *testing.T) *store.Store {
	t.Helper()
	auth.Cost = bcrypt.MinCost
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	migrations, _ := fs.Sub(school.Assets, "migrations")
	if err := db.Migrate(conn, migrations); err != nil {
		t.Fatal(err)
	}
	return store.New(conn)
}

// Fixture holds IDs of commonly needed records.
type Fixture struct {
	ClassID, OtherClassID int64
	Term1, Term2          int64
	MathID, EnglishID     int64
	AdminID               int64
}

// Seed creates two classes, two terms, two subjects and an admin.
func Seed(t *testing.T, st *store.Store) Fixture {
	t.Helper()
	var f Fixture
	var err error
	must := func(id int64, e error) int64 {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	f.ClassID = must(st.CreateClass("Grade 7"))
	f.OtherClassID = must(st.CreateClass("Grade 8"))
	f.Term1 = must(st.CreateTerm("2026", "Term 1"))
	f.Term2 = must(st.CreateTerm("2026", "Term 2"))
	f.MathID = must(st.CreateSubject("Mathematics"))
	f.EnglishID = must(st.CreateSubject("English"))
	f.AdminID, err = st.CreateStaff("admin", "Admin User", store.RoleAdmin, "admin-password")
	must(f.AdminID, err)
	return f
}
