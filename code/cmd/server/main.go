// Command server runs the school management web application.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"school"
	"school/internal/auth"
	"school/internal/db"
	"school/internal/store"
	"school/internal/web"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := flag.String("addr", envOr("SCHOOL_ADDR", "127.0.0.1:8080"), "address to listen on")
	dbPath := flag.String("db", envOr("SCHOOL_DB", "school.db"), "path to the SQLite database file")
	secure := flag.Bool("secure-cookies", os.Getenv("SCHOOL_SECURE_COOKIES") == "1",
		"mark cookies Secure (enable when serving over HTTPS)")
	resetUser := flag.String("reset-password", "", "set a new random password for this staff username, print it and exit")
	testMode := flag.Bool("test", false, "use a separate test database (school-test.db) filled with sample data")
	flag.Parse()

	if *testMode {
		dbSet := false
		flag.Visit(func(f *flag.Flag) { dbSet = dbSet || f.Name == "db" })
		if !dbSet {
			*dbPath = "school-test.db"
		}
	}

	conn, err := db.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	migrations, _ := fs.Sub(school.Assets, "migrations")
	if err := db.Migrate(conn, migrations); err != nil {
		log.Fatal(err)
	}
	st := store.New(conn)

	if *resetUser != "" {
		if err := resetPassword(st, *resetUser); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *testMode {
		if err := loadTestData(conn, *dbPath); err != nil {
			log.Fatal(err)
		}
	} else if err := ensureAdmin(st); err != nil {
		log.Fatal(err)
	}

	srv, err := web.New(st, school.Assets, *secure)
	if err != nil {
		log.Fatal(err)
	}
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("School Manager listening on http://%s (database: %s)", *addr, *dbPath)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// ensureAdmin creates the first administrator on an empty database. The
// password comes from SCHOOL_ADMIN_PASSWORD or is generated and printed once.
func ensureAdmin(st *store.Store) error {
	n, err := st.CountStaff()
	if err != nil || n > 0 {
		return err
	}
	password := os.Getenv("SCHOOL_ADMIN_PASSWORD")
	generated := password == ""
	if generated {
		password = auth.RandomToken()[:16]
	}
	if _, err := st.CreateStaff("admin", "Administrator", store.RoleAdmin, password); err != nil {
		return fmt.Errorf("create initial admin: %w", err)
	}
	if generated {
		fmt.Printf("\n  Initial administrator account created.\n  Username: admin\n  Password: %s\n  Save this password now; it will not be shown again.\n\n", password)
	} else {
		fmt.Println("Initial administrator account 'admin' created with the password from SCHOOL_ADMIN_PASSWORD.")
	}
	return nil
}

// loadTestData fills an empty database with the sample data in test.sql and
// prints the logins to try.
func loadTestData(conn *sql.DB, dbPath string) error {
	script, err := fs.ReadFile(school.Assets, "test.sql")
	if err != nil {
		return err
	}
	loaded, err := db.SeedIfEmpty(conn, string(script))
	if err != nil {
		return err
	}
	if loaded {
		fmt.Printf("\n  TEST MODE: loaded sample data into %s\n", dbPath)
	} else {
		fmt.Printf("\n  TEST MODE: %s already has data; sample data not reloaded.\n"+
			"  Delete %s (and its -wal/-shm files) to start fresh.\n", dbPath, dbPath)
	}
	fmt.Print(`
  Staff login  (/login):         admin / admin-test
                                 teacher / teacher-test
                                 finance / finance-test
  Parent login (/parent/login):  ADM-001 / TEST-0001  (up to ADM-007 / TEST-0007)

`)
	return nil
}

func resetPassword(st *store.Store, username string) error {
	id, err := st.StaffIDByUsername(username)
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no staff account named %q", username)
	}
	if err != nil {
		return err
	}
	password := auth.RandomToken()[:16]
	if err := st.SetStaffPassword(id, password); err != nil {
		return err
	}
	if err := st.SetStaffActive(id, true); err != nil {
		return err
	}
	fmt.Printf("New password for %s: %s\n", username, password)
	return nil
}
