# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

School management web app: a Go monolith with server-rendered `html/template` pages, plain CSS, no JS, and a single SQLite file. `PROMPT.MD` is the original spec and defines the scope. Don't add features beyond it unless security requires them. The user is on a limited AI plan, so work in small, verified steps.

## Hard constraints

- **Windows, no CGO/gcc.** Use only the pure-Go `modernc.org/sqlite` driver (registered as `"sqlite"`). Never use `mattn/go-sqlite3` or any other dependency that needs CGO.
- **Layout.**
  - `code/` holds all source; it is the Go module named `school`.
  - `build/school.exe` is the build output.
  - `README.MD` sits at the repo root. Keep it current when commands or behaviour change.

## Commands (run from `code/`)

- Build: `go build -o ..\build\school.exe .\cmd\server`, or `build.ps1 [-Test]` from the root.
- Run: `..\build\school.exe` (flags: `-addr`, `-db`, `-secure-cookies`, `-reset-password <user>`).
- Run with sample data: `..\build\school.exe --test`. This uses `school-test.db` and seeds it from `code/test.sql` via `db.SeedIfEmpty`, but only when the staff table is empty.
- Run all tests: `go test ./...`
- Run a single test: `go test ./internal/web -run TestName -v`
- Vet: `go vet ./...`

## Architecture

- `assets.go` (package `school`) uses `go:embed` to embed `templates/`, `static/` and `migrations/`, so the exe is self-contained. If you add an asset directory, add it there too.
- `test.sql` holds the sample data. It uses explicit IDs and precomputed bcrypt hashes, and access codes are hashed in normalized form (`TEST0001`). If the schema changes, update this file too; `TestSampleData` checks that it still loads and that the documented logins and fee statuses still hold.
- `internal/db` opens SQLite and applies migrations.
  - Pragmas are set per connection in the DSN: foreign keys, `busy_timeout`, WAL, and `_txlock=immediate`.
  - `Migrate` runs `migrations/*.sql` in name order and records each in `schema_migrations`. Change the schema only by adding a new numbered file; never edit `001_init.sql`.
- `internal/store` holds every SQL query and every business rule: validation, fee balances, result checks.
  - It returns `*store.ValidationError` for user-safe messages and `store.ErrNotFound` / `ErrBadCredentials` otherwise.
  - Money is stored as integer cents. Use `ParseMoney` / `FormatMoney`.
- `internal/web` contains the handlers.
  - `routes.go` is the single place where role checks live: `requireStaff(h, roles...)` and `requireParent`.
  - Middleware order: security headers → CSRF (double-submit cookie; every POST form must include `{{template "csrf" .CSRF}}`) → session loading.
  - `s.fail(...)` turns a ValidationError into a flash message plus a redirect, and logs other errors behind a generic 500.
- Templates: each page defines `content` and is parsed together with `layout.html`. `layout.html` also holds shared partials (`csrf`, `resultsTable`, `feeSummaryTable`, `paymentsTable`, `feeStatus`). Pages receive `pageData`; handler data is under `.Data`.
- The CSP disallows inline styles and scripts. Use CSS classes in `static/css/style.css`.

## Invariants that must hold

- **Parent isolation.** Parent handlers take the student only from `identityFrom(r).student`, which comes from the session. Never read a student ID from the request.
- **Teachers never touch fees.** Fee routes are limited to admin and finance.
- **Draft results stay hidden.** Parents see only exams with `published = 1`; store queries take `publishedOnly`.
- **Access codes are never stored in plaintext.** A code is shown once, on `access_code.html`.

These invariants are covered by `internal/web/web_test.go`. Store rules are covered by `internal/store/store_test.go`. Tests use `testutil.NewStore(t)`, which creates a temporary DB and lowers the bcrypt cost.
