-- Initial schema. Money is stored as integer cents to avoid rounding errors.

CREATE TABLE classes (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE
);

CREATE TABLE subjects (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE
);

CREATE TABLE academic_years (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

-- A term is one semester/term within an academic year.
CREATE TABLE terms (
    id               INTEGER PRIMARY KEY,
    academic_year_id INTEGER NOT NULL REFERENCES academic_years(id),
    name             TEXT NOT NULL COLLATE NOCASE,
    UNIQUE (academic_year_id, name)
);

CREATE TABLE staff (
    id            INTEGER PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE COLLATE NOCASE,
    full_name     TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'teacher', 'finance')),
    password_hash TEXT NOT NULL,
    active        INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE students (
    id               INTEGER PRIMARY KEY,
    admission_no     TEXT NOT NULL UNIQUE COLLATE NOCASE,
    full_name        TEXT NOT NULL,
    class_id         INTEGER NOT NULL REFERENCES classes(id),
    access_code_hash TEXT NOT NULL,
    active           INTEGER NOT NULL DEFAULT 1,
    created_at       TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_students_class ON students(class_id);
CREATE INDEX idx_students_name ON students(full_name);

CREATE TABLE exams (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    term_id    INTEGER NOT NULL REFERENCES terms(id),
    class_id   INTEGER NOT NULL REFERENCES classes(id),
    max_marks  REAL NOT NULL DEFAULT 100 CHECK (max_marks > 0),
    published  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name, term_id, class_id)
);
CREATE INDEX idx_exams_term ON exams(term_id);
CREATE INDEX idx_exams_class ON exams(class_id);

CREATE TABLE exam_results (
    id         INTEGER PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students(id),
    exam_id    INTEGER NOT NULL REFERENCES exams(id),
    subject_id INTEGER NOT NULL REFERENCES subjects(id),
    marks      REAL NOT NULL CHECK (marks >= 0),
    max_marks  REAL NOT NULL CHECK (max_marks > 0),
    grade      TEXT,
    updated_by INTEGER REFERENCES staff(id),
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (marks <= max_marks),
    UNIQUE (student_id, exam_id, subject_id)
);
CREATE INDEX idx_results_exam ON exam_results(exam_id, subject_id);

CREATE TABLE fee_assessments (
    id           INTEGER PRIMARY KEY,
    student_id   INTEGER NOT NULL REFERENCES students(id),
    term_id      INTEGER NOT NULL REFERENCES terms(id),
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    updated_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (student_id, term_id)
);
CREATE INDEX idx_fee_assessments_term ON fee_assessments(term_id);

CREATE TABLE fee_payments (
    id           INTEGER PRIMARY KEY,
    student_id   INTEGER NOT NULL REFERENCES students(id),
    term_id      INTEGER NOT NULL REFERENCES terms(id),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    paid_on      TEXT NOT NULL,
    reference    TEXT NOT NULL DEFAULT '',
    recorded_by  INTEGER NOT NULL REFERENCES staff(id),
    created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_fee_payments_student_term ON fee_payments(student_id, term_id);
CREATE INDEX idx_fee_payments_term ON fee_payments(term_id);

-- Exactly one of staff_id / student_id is set: staff sessions or parent sessions.
CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    staff_id   INTEGER REFERENCES staff(id) ON DELETE CASCADE,
    student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    CHECK ((staff_id IS NULL) <> (student_id IS NULL))
);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
