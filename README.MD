# School Manager

A simple school management web app. It is a single Go program that uses server-rendered HTML pages and one SQLite database file. It runs on Windows with no C compiler: it uses the pure-Go `modernc.org/sqlite` driver, so CGO is not needed.

Staff can do the following:
- Register students
- Enter exam results
- Record school fees

Parents log in with their child's admission number and an access code to see results and fee balances.

## Folders

| Folder | Contents |
|---|---|
| `code/` | All source code (the Go module) |
| `build/` | The compiled program, `school.exe` |

The templates, CSS and database migrations are embedded in `school.exe`, so you only need the one file to run the app.

## Requirements

- Go 1.22 or newer. It was developed with Go 1.26 on Windows 11.
- No gcc and no CGO.

## Build

From the repository root in PowerShell, run:

```powershell
powershell -ExecutionPolicy Bypass -File build.ps1          # build only
powershell -ExecutionPolicy Bypass -File build.ps1 -Test    # run tests, then build
```

Or build manually:

```powershell
cd code
go build -o ..\build\school.exe .\cmd\server
```

## Run

```powershell
.\build\school.exe
```

Then open http://127.0.0.1:8080 in a browser.

### Try it out with test data

To explore the app without setting anything up, start it in test mode:

```powershell
.\build\school.exe --test
```

This uses a separate database, `school-test.db`, so your real `school.db` is never touched. On the first start the app fills it with the sample data from `code/test.sql`: two classes, five subjects, four terms across 2025–2026, seven students, exams with results, and fee records. It then prints these logins:

| Login page | Username / admission no. | Password / access code | Role |
|---|---|---|---|
| http://127.0.0.1:8080/login | `admin` | `admin-test` | Administrator |
| http://127.0.0.1:8080/login | `teacher` | `teacher-test` | Teacher |
| http://127.0.0.1:8080/login | `finance` | `finance-test` | Finance |
| http://127.0.0.1:8080/parent/login | `ADM-001` … `ADM-007` | `TEST-0001` … `TEST-0007` | Parent |

Things to look at:
- In the current term (2026 Term 3), ADM-001 has fully paid, ADM-002 and ADM-004 have partly paid, ADM-003 and ADM-006 haven't paid, and ADM-005 has overpaid, so a credit is shown.
- ADM-002 also overpaid in 2026 Term 1. That credit does not change any other term's balance.
- The Grade 7 "Midterm" in 2026 Term 2 is a **draft**. Staff can see it, but parents can't until an admin publishes it.
- ADM-007 is an inactive student whose 2025 records are kept.
- Log in as `teacher` and try to open a fee page. You'll get "Not allowed".

The changes you make in test mode are kept between runs. To start over with fresh sample data, stop the app and delete `school-test.db`, plus `school-test.db-wal` and `school-test.db-shm` if they exist. The sample data is only loaded into an empty database, never into one that already has accounts.

**Never use the test logins for real data.** Their passwords are published in this README.

**First run:** if the database has no staff accounts, the app creates an administrator named `admin` and prints its password in the console once. Save it. To choose the password yourself, set it before the first run:

```powershell
$env:SCHOOL_ADMIN_PASSWORD = "choose-a-strong-password"; .\build\school.exe
```

**Getting started after logging in as admin:**
1. Open **Setup** and add classes (for example, "Grade 7"), subjects, and academic terms (year "2026", term "Term 1").
2. Open **Staff** and create teacher and finance accounts.
3. Open **Students → Register student**. The parent access code is shown once. Give it to the parent.
4. Open **Exams** and create an exam. Enter marks per subject, then click **Publish** so parents can see the results.
5. Open **Fees** to set the required fees for a class or student per term, then record payments from the student's fee page.

Parents log in at http://127.0.0.1:8080/parent/login.

### Options

| Flag | Environment variable | Default | Meaning |
|---|---|---|---|
| `-addr` | `SCHOOL_ADDR` | `127.0.0.1:8080` | Listen address. Use `0.0.0.0:8080` to allow other machines. |
| `-db` | `SCHOOL_DB` | `school.db` | SQLite database file, relative to the current folder |
| `-secure-cookies` | `SCHOOL_SECURE_COOKIES=1` | off | Turn this on when the site is served over HTTPS |
| `--test` | | off | Uses `school-test.db` (unless `-db` is given) and loads the sample data if it's empty |
| `-reset-password <user>` | | | Sets a new random password for a staff account, prints it and exits. Use this if the admin password is lost. |

**Back up your data** by copying the database file while the server is stopped. The app uses SQLite WAL mode, so also copy the `school.db-wal` and `school.db-shm` files if they exist.

## Tests

```powershell
cd code
go test ./...                                         # all tests
go test ./internal/web -run TestTeacherCannotModifyFees -v   # a single test
```

Tests use a temporary database and never touch `school.db`.

## Roles

| | Admin | Teacher | Finance | Parent |
|---|---|---|---|---|
| View students | ✓ | ✓ | ✓ | own child only |
| Register/edit students, reset access codes | ✓ | | | |
| Create/publish exams | ✓ | | | |
| Enter/edit marks | ✓ | ✓ | | |
| Set fees, record payments | ✓ | | ✓ | |
| View fees | ✓ | | ✓ | own child only |
| Staff accounts, setup | ✓ | | | |

## Design decisions and defaults

- **Fees** are set per student per term. Each term's balance is independent: balance = required fees − payments. The status is Fully paid, Partially paid or Unpaid. An overpayment is shown as a credit. On the Fees page, "Apply to class" sets the same amount for every active student in the class.
- **Marks** must be between 0 and the exam's maximum marks, which defaults to 100. Each student gets one result per exam and subject; saving again updates it. If any mark in a batch is invalid, nothing in the batch is saved. Clearing a box removes that result. Grades are not computed in this version; only marks and percentages are shown.
- **Exams** start as drafts. Parents only see exams that have been published.
- **Inactive students** keep all their records, and their parents can still log in to view history.
- **Access codes** look like `K7PQ-M2XD`. Case, dashes and spaces are ignored when parents type them. Only a bcrypt hash is stored. Resetting a code logs out any parent using the old one.
- **Security:**
  - Staff passwords and access codes are hashed with bcrypt.
  - Sessions are kept server-side for 12 hours, and only a hash of each token is stored.
  - Cookies are HttpOnly and SameSite=Lax.
  - Every form is protected against CSRF.
  - Every SQL query is parameterized.
  - Permissions are checked on the server for every route.
  - There is no login rate limiting yet. If the app is exposed to the internet, put it behind HTTPS (for example, a reverse proxy) and turn on `-secure-cookies`.
