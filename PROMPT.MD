# Build a Simple School Management System

Build a functional school management web application using Go, server-rendered HTML templates, and SQLite.

The goal is to create a simple, working first version that can be expanded later. Prioritize correctness, simplicity, and maintainability over the number of features.

## 1. Required Technology

- **Backend:** Go.
- **HTTP server:** Go's standard `net/http`.
- **HTML rendering:** Go's standard `html/template`.
- **Database:** SQLite.
- **Styling:** Plain CSS.
- **JavaScript:** Vanilla JavaScript only when necessary.

Do not use React, Vue, Angular, a separate frontend application, or a complex frontend build system.

Use normal HTTP requests and HTML forms. Render pages on the server rather than building a single-page application.

Keep the application as a simple monolith that can run on a low-cost VPS. Use as few dependencies as reasonably possible.

## 2. User Types

The system has three types of users.

### Administrator

The administrator can:

- Register and edit students.
- Assign admission numbers and generate parent access codes.
- Create and manage examinations.
- Enter and edit examination results.
- Set the required school fees for each semester.
- Record student fee payments.
- View student academic and financial records.
- Create teacher and finance staff accounts.

### Teacher

Teachers can:

- Log in to the staff dashboard.
- View students and their classes.
- Enter and edit examination results.

Teachers must not be allowed to modify fee payments.

### Parent

Parents log in using:

- Student admission number.
- Access code provided by the school during admission.

A parent can view only the student associated with those credentials.

Parents do not need a separate registration process.

## 3. Student Management

Allow authorized staff to:

- Register a student with a full name and unique admission number.
- Assign the student to a class.
- Edit student information.
- View a student's profile.
- Search for students by name or admission number.
- Mark a student inactive without deleting historical records.

Generate an initial access code when registering a student. Display it to the authorized staff member so it can be provided to the parent.

Store a secure hash of the access code rather than the plaintext code. Provide a way for an administrator to reset the code if necessary.

Keep the student record simple. Add more profile fields only when required.

## 4. Examination Results

Allow administrators to create examinations and specify:

- Examination name.
- Academic year.
- Semester or term.
- Class.
- Maximum marks, where applicable.

Allow authorized staff to enter each student's marks for individual subjects.

For every result, store:

- Student.
- Examination.
- Subject.
- Marks obtained.
- Maximum marks.
- Grade, if the school uses a defined grading system.

Prevent duplicate results for the same student, examination, and subject.

Validate marks on the server.

Parents should be able to view results from previous examinations, grouped by academic year, semester, and examination.

For the initial version, display marks and percentages. Do not implement complex ranking, report-card generation, or configurable grading systems yet.

## 5. School Fee Payments

Allow administrators or authorized finance staff to:

- Set the total required fees for each student's semester.
- Record a payment.
- View the payment history.
- View the total amount paid.
- View the outstanding balance.

Store each payment with:

- Student.
- Academic year.
- Semester or term.
- Amount paid.
- Payment date.
- Optional payment reference.

Calculate the outstanding balance as:

Outstanding balance = Required fees − Total recorded payments.

Display whether the semester is:

- Fully paid.
- Partially paid.
- Unpaid.

If payments exceed the required fees, display the resulting credit rather than incorrectly marking the account as unpaid.

Keep each semester's balance independent.

Parents should be able to view fee balances and payment histories for previous semesters.

Do not implement online payment processing, invoices, receipts in PDF format, or complex accounting features in the first version.

## 6. Parent Portal

Create a simple, mobile-friendly parent portal.

After logging in, parents should see:

- Student's name.
- Admission number.
- Current class.
- A link to examination results.
- A link to fee payment records.

The examination page should let parents select an examination and view subject marks.

The fees page should let parents select an academic year and semester and view required fees, payments, and outstanding balance.

Keep the interface simple and easy to understand.

Parents must not be able to view another student's information by changing a URL or submitting a different admission number after logging in.

Only show examination results that the school has explicitly marked as published. If publication controls are not yet implemented, implement a simple published/draft status rather than exposing every entered result.

## 7. Staff Dashboard

Create a basic dashboard with links to:

- Students.
- Examinations and results.
- Fee payments.
- Staff accounts, for administrators only.

Use ordinary server-rendered HTML pages and forms.

Include student search, clear validation messages, success and error messages, and tables for displaying results and payment records.

Use responsive CSS so the pages work on desktop and mobile devices.

Do not spend time on charts, animations, elaborate dashboard statistics, or decorative UI components.

## 8. Database

Use a single SQLite database file.

Create a normalized schema with the necessary tables for:

- Students.
- Staff users and roles.
- Classes.
- Academic years and semesters.
- Subjects.
- Examinations.
- Examination results.
- Semester fee assessments.
- Fee payments.

Add appropriate foreign keys, unique constraints, and indexes.

Use parameterized SQL queries and database transactions where necessary.

Enable foreign-key enforcement and configure SQLite appropriately for a web application, including a busy timeout and WAL mode where suitable.

Keep write transactions short. Do not create multiple database files to distribute writes.

Use a simple, repeatable database migration or initialization process.

## 9. Security

Implement the essential security measures:

- Hash staff passwords using an appropriate password-hashing algorithm.
- Hash parent access codes.
- Use secure sessions and implement logout.
- Enforce permissions on the server.
- Protect state-changing forms against CSRF.
- Validate all user input on the server.
- Use parameterized SQL.
- Avoid exposing sensitive information in error messages.
- Prevent parents from accessing other students' records.
- Do not allow teachers to modify financial records.

Do not build a complicated security framework. Implement these protections directly and correctly using established Go libraries and patterns.

## 10. Project Structure

Use a simple, maintainable project structure. For example:

- `main.go` or `cmd/server/main.go`
- `internal/` for database access, business logic, authentication, and handlers.
- `templates/` for HTML templates.
- `static/css/` for stylesheets.
- `migrations/` for database schema changes.

Adjust the structure if the existing repository already has a sensible organization.

Keep database queries, business logic, and HTML templates separate.

Use Go's `html/template` for automatic contextual HTML escaping.

## 11. Testing

Write a small set of automated tests for the most important behavior:

- Parent authentication.
- Parents cannot access other students' records.
- Teachers cannot modify fee payments.
- Duplicate admission numbers are rejected.
- Invalid examination marks are rejected.
- Partial payments produce the correct balance.
- Fully paid fees produce a zero balance.
- Payments for different semesters are calculated independently.

Use a temporary test database rather than real school data.

## 12. Implementation Instructions

I am using a limited/free AI coding plan, so work efficiently and avoid unnecessary complexity.

Follow these steps:

1. Inspect the existing repository and understand its current state.
2. Briefly outline the database schema and implementation plan.
3. Implement the core application in small, manageable stages.
4. Prioritize a working database, authentication, student management, results, fee payments, and parent portal.
5. Run the available tests and fix important errors.
6. Provide clear instructions for running the application locally.

Do not generate a lengthy architecture document before writing code.

Do not implement every feature in one enormous response. Make progress in small, verifiable steps.

Do not add features outside this scope unless they are necessary for the core system to work securely.

If something is ambiguous, choose a reasonable simple default and document it.

Do not claim that a feature works unless it has been implemented and tested.

Start by inspecting the repository, then proceed with implementation. The objective is a working first version, not a large unfinished codebase.