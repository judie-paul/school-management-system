// Package db opens the SQLite database and applies migrations.
package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"

	_ "modernc.org/sqlite" // pure-Go SQLite driver: no CGO, builds on Windows
)

// Open opens (creating if needed) the SQLite database at path. Every pooled
// connection gets foreign keys, a busy timeout and WAL mode. Transactions
// start with BEGIN IMMEDIATE so concurrent writers wait on the busy timeout
// instead of failing on lock upgrade.
func Open(path string) (*sql.DB, error) {
	dsn := path + "?_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_txlock=immediate"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	return conn, nil
}

// Migrate applies every *.sql file in fsys, in name order, that has not been
// applied yet. Each file runs in its own transaction.
func Migrate(conn *sql.DB, fsys fs.FS) error {
	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, name := range files {
		var n int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		script, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		tx, err := conn.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(script)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
