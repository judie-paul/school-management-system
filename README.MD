# School Manager

School Manager is a simple program that helps a school keep track of:

- **Students**: names, admission numbers and classes
- **Exam results**: marks for each subject in each exam
- **School fees**: what each student must pay each term, what they have paid, and what they still owe

Parents get their own page where they can check their child's results and fee balance from a phone or computer.

You use School Manager in a web browser such as Chrome, Edge or Firefox, but it runs on your own computer. You don't need an internet connection to use it on that computer.

This guide has two parts:

- **[Part 1: Using School Manager](#part-1-using-school-manager)** is for everyone. No technical knowledge is needed.
- **[Part 2: Technical information](#part-2-technical-information)** is for the person who installs or maintains the program.

---

# Part 1: Using School Manager

## Who can do what

There are four kinds of users. Each one sees only what they need.

| User | What they can do |
|---|---|
| **Administrator** | Everything: register students, create exams, publish results, manage fees, and create accounts for teachers and finance staff |
| **Teacher** | View students and enter or correct exam marks. Teachers **cannot** see or change fee records. |
| **Finance staff** | View students, set the fees each student must pay, and record payments |
| **Parent** | See **only their own child's** results and fee records. Parents don't need to sign up; the school gives them a login code. |

Here is the same information as a checklist:

| | Admin | Teacher | Finance | Parent |
|---|---|---|---|---|
| View students | ✓ | ✓ | ✓ | own child only |
| Register/edit students, reset access codes | ✓ | | | |
| Create/publish exams | ✓ | | | |
| Enter/edit marks | ✓ | ✓ | | |
| Set fees, record payments | ✓ | | ✓ | |
| View fees | ✓ | | ✓ | own child only |
| Staff accounts, setup | ✓ | | | |

## Try it out first (with practice data)

Before entering real information, you can try School Manager with made-up students, results and fees. Nothing you do in practice mode affects real school records.

### Step 1: Start School Manager in practice mode

1. Open the **`build`** folder (the folder that contains `school.exe`).
2. Click once on the **address bar** at the top of the folder window (where the folder's location is shown).
3. Type `cmd` and press **Enter**. A black window opens.
4. In the black window, type `school.exe --test` and press **Enter**.
5. The window shows a message starting with **"TEST MODE"** and lists the practice logins. **Leave this window open.** School Manager runs only while it is open.

If there is no `school.exe` in the `build` folder, ask the person who set up the program to create it. The steps are in Part 2.

### Step 2: Open it in your web browser

Open your web browser and go to: **http://127.0.0.1:8080**

### Step 3: Log in with a practice account

| Who you want to try | Login page | Username | Password |
|---|---|---|---|
| Administrator | http://127.0.0.1:8080/login | `admin` | `admin-test` |
| Teacher | http://127.0.0.1:8080/login | `teacher` | `teacher-test` |
| Finance staff | http://127.0.0.1:8080/login | `finance` | `finance-test` |
| Parent | http://127.0.0.1:8080/parent/login | Admission number `ADM-001` to `ADM-007` | Access code `TEST-0001` to `TEST-0007` (matching the number) |

For example, a parent of student ADM-003 logs in with admission number `ADM-003` and access code `TEST-0003`.

### Things worth trying

- **Fees in the current term (2026 Term 3):**

  | Student | What you'll see |
  |---|---|
  | ADM-001 | Fully paid |
  | ADM-002 and ADM-004 | Partly paid |
  | ADM-003 and ADM-006 | Nothing paid yet |
  | ADM-005 | Paid too much, so a **credit** is shown |

- **Each term is separate:** ADM-002 also paid too much in 2026 Term 1. That extra money doesn't change any other term's balance.
- **Draft results stay hidden:** the Grade 7 "Midterm" in 2026 Term 2 is still a **draft**. Staff can see it, but parents can't until an administrator clicks **Publish**.
- **Students who have left:** ADM-007 is marked as inactive. Their old 2025 records are still kept.
- **Teachers can't see fees:** log in as `teacher` and try to open a student's fee page. You'll see "Not allowed".

### Starting the practice over

Practice data you change is kept the next time you start practice mode. To start over with fresh practice data:

1. Close School Manager.
2. In the `build` folder, delete `school-test.db`, plus `school-test.db-wal` and `school-test.db-shm` if you see them.
3. Start practice mode again.

> ⚠️ **Never use the practice logins for real school data.** Their passwords are printed in this guide for anyone to see.

## Using School Manager for real

### Starting it

Double-click **`school.exe`** in the `build` folder. A black window opens. **Keep it open** while you use School Manager, then open **http://127.0.0.1:8080** in your web browser.

**The very first time** you start it, the black window shows an **administrator password**, like this:

```
  Initial administrator account created.
  Username: admin
  Password: xxxxxxxxxxxxxxxx
```

**Write this password down and keep it safe.** It is shown only once. Log in with username `admin` and this password.

### First-time setup (administrator)

Do these steps once, in this order:

1. **Setup:** add your classes (for example "Grade 7"), subjects (for example "Mathematics"), and terms (for example year "2026", term "Term 1").
2. **Staff:** create an account for each teacher and finance staff member. Choose their role and give them their username and password.
3. **Students → Register student:** enter each student's admission number, full name and class.
   - After you save, a **parent access code** (like `K7PQ-M2XD`) is shown **only once**. Write it down and give it to the parent.
   - If the code is lost, open the student's page and click **Reset parent access code** to make a new one. The old code stops working.

### Everyday tasks

**Entering exam results (administrator creates, teacher or administrator enters marks):**
1. Go to **Exams**. An administrator creates the exam by choosing its name, term, class, and the highest possible mark (usually 100).
2. Open the exam, choose a subject, type each student's marks, and click **Save marks**.
   - Marks must be between 0 and the highest possible mark.
   - If any mark is wrong, **nothing is saved**, and the page tells you which student to fix.
   - Clearing a box removes that student's mark for that subject.
3. When all marks are in, an administrator clicks **Publish results to parents**. Until then, parents can't see the exam.

**Fees (administrator or finance staff):**
1. Go to **Fees**, choose a term and class, and click **Show balances**.
2. To set the same fee for the whole class, type the amount and click **Apply to class**.
3. Click a student's name to see their fee page. There you can:
   - change the fee for just that student
   - **record a payment**, with the amount, the date and an optional reference such as an M-Pesa code
   - see the payment history

Each term is calculated separately: **balance = fees for the term − payments for that term**. The status is shown as **Fully paid**, **Partially paid** or **Unpaid**. If someone pays too much, the extra is shown as a **credit**.

**When a student leaves:** open the student, click **Edit**, and untick **Active**. Their records are kept, and their parents can still log in to see past results and fees.

**Searching for a student:** use the search box on the Dashboard or the Students page. You can search by name or admission number.

### For parents

1. Open the address the school gives you (on the school computer it is **http://127.0.0.1:8080/parent/login**).
2. Enter your child's **admission number** and the **access code** from the school. Capital letters, small letters, dashes and spaces don't matter.
3. You'll see your child's name, admission number and class, with links to:
   - **Results:** choose an exam to see the marks and percentage for each subject.
   - **Fees:** choose a term to see the fees, the payments made and the amount still owed.

Parents can only ever see their own child's information.

### Good to know

- **Logging out:** click **Log out** at the top of the page. You're also logged out automatically after 12 hours.
- **Grades:** this version shows marks and percentages only. It doesn't show letter grades or class rankings.
- **Stopping School Manager:** close the black window. Nobody can use the program until it is started again.
- **Forgotten administrator password:** ask the person who maintains the program. Part 2 explains how to reset it.

### Keep your information safe: make backups

All school information is stored in one file called **`school.db`**, kept in the same folder as `school.exe`. To back it up:

1. Close School Manager (close the black window).
2. Copy `school.db` to a USB stick or another safe place. If you also see `school.db-wal` and `school.db-shm`, copy them too.

Do this regularly, for example at the end of each week.

---

# Part 2: Technical information

School Manager is a single Go program. It uses server-rendered HTML pages (Go `html/template`), plain CSS, no JavaScript, and one SQLite database file. It runs on Windows without a C compiler because it uses the pure-Go `modernc.org/sqlite` driver, so CGO is not needed.

## Folders

| Folder | Contents |
|---|---|
| `code/` | All source code (the Go module `school`) |
| `build/` | The compiled program, `school.exe` |

The templates, CSS, database migrations and sample data (`code/test.sql`) are embedded in `school.exe`, so the one file is all you need to run the app.

When you double-click `school.exe`, it creates its database in the `build/` folder. Don't delete `build/` when rebuilding if it holds real data. `.gitignore` excludes `*.db` files so school data is never committed.

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
.\build\school.exe            # normal mode, database school.db
.\build\school.exe --test     # practice mode, database school-test.db with sample data
```

Then open http://127.0.0.1:8080. Parents log in at http://127.0.0.1:8080/parent/login.

**First run:** if the database has no staff accounts, the app creates an administrator named `admin` and prints its password in the console once. To choose the password yourself, set it before the first run:

```powershell
$env:SCHOOL_ADMIN_PASSWORD = "choose-a-strong-password"; .\build\school.exe
```

**Lost admin password:** run `.\build\school.exe -reset-password admin` with the same database (use `-db` if it's elsewhere) while the server is stopped. It prints a new random password, re-enables the account and exits.

### Test mode details

`--test` uses `school-test.db`, unless `-db` is given, and fills it from the embedded `code/test.sql`:

- two classes, five subjects, four terms across 2025–2026
- three staff accounts and seven students
- six exams, one of them a draft and one marked out of 50
- fee assessments and payments

The data is loaded only when the database has no staff accounts, so it is never mixed into a database in use. On later starts it prints a notice and doesn't load it again. The logins are listed in Part 1 and at the top of `test.sql`.

### Options

| Flag | Environment variable | Default | Meaning |
|---|---|---|---|
| `-addr` | `SCHOOL_ADDR` | `127.0.0.1:8080` | Listen address. Use `0.0.0.0:8080` to allow other machines (for example, parents' phones on the school network). |
| `-db` | `SCHOOL_DB` | `school.db` | SQLite database file, relative to the current folder |
| `-secure-cookies` | `SCHOOL_SECURE_COOKIES=1` | off | Turn this on when the site is served over HTTPS |
| `--test` | | off | Uses `school-test.db` (unless `-db` is given) and loads the sample data if it's empty |
| `-reset-password <user>` | | | Sets a new random password for a staff account, prints it and exits |

**Back up your data** by copying the database file while the server is stopped. The app uses SQLite WAL mode, so also copy the `school.db-wal` and `school.db-shm` files if they exist.

## Tests

```powershell
cd code
go test ./...                                                # all tests
go test ./internal/web -run TestTeacherCannotModifyFees -v   # a single test
```

Tests use a temporary database and never touch `school.db`. `TestSampleData` checks that `test.sql` still loads and that its documented logins and fee statuses hold.

## Design decisions and defaults

- **Fees** are set per student per term. Each term's balance is independent: balance = required fees − payments. The status is Fully paid, Partially paid or Unpaid. An overpayment is shown as a credit. On the Fees page, "Apply to class" sets the same amount for every active student in the class. Money is stored as integer cents.
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
