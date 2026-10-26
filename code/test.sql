-- Test data for trying out the application. Loaded by `school.exe --test`
-- into a separate database (school-test.db), and only when that database is
-- empty. Never load this into a real school database.
--
-- Staff logins (username / password):
--   admin   / admin-test     (administrator)
--   teacher / teacher-test   (teacher)
--   finance / finance-test   (finance)
--
-- Parent logins (admission number / access code):
--   ADM-001 / TEST-0001   ...   ADM-007 / TEST-0007
--
-- Passwords and access codes are stored as bcrypt hashes, exactly as the
-- application stores them. Access codes are hashed in normalized form
-- (uppercase, no dash), e.g. TEST-0001 -> TEST0001.

INSERT INTO classes (id, name) VALUES
    (1, 'Grade 7'),
    (2, 'Grade 8');

INSERT INTO subjects (id, name) VALUES
    (1, 'Mathematics'),
    (2, 'English'),
    (3, 'Kiswahili'),
    (4, 'Science'),
    (5, 'Social Studies');

INSERT INTO academic_years (id, name) VALUES
    (1, '2025'),
    (2, '2026');

INSERT INTO terms (id, academic_year_id, name) VALUES
    (1, 1, 'Term 3'),
    (2, 2, 'Term 1'),
    (3, 2, 'Term 2'),
    (4, 2, 'Term 3');

INSERT INTO staff (id, username, full_name, role, password_hash) VALUES
    (1, 'admin',   'Test Administrator', 'admin',   '$2a$10$He3QnVzFJtoAYbpPPtOKQ.T32XeD/2sKOcKp0rLm1tA3kgtStypAG'),
    (2, 'teacher', 'Grace Muthoni',      'teacher', '$2a$10$CPcJLQdUoBb2FVaDnTmEkeKrGyG326TrUww.Opei2Bk/8M6cSwF3S'),
    (3, 'finance', 'Peter Omondi',       'finance', '$2a$10$pjESM8Sl1GyQZ83W2cQV2eg8yY2PPQ9VM5xoxnzVMfPalYXgo9euW');

-- ADM-007 has left the school (inactive) but keeps their 2025 records.
INSERT INTO students (id, admission_no, full_name, class_id, active, access_code_hash) VALUES
    (1, 'ADM-001', 'Amani Otieno',   1, 1, '$2a$10$A/AwlUrlT7NqW0PBzRfGWOHtYmHisz2rfinXYFznMa738r17RgrH2'),
    (2, 'ADM-002', 'Baraka Mwangi',  1, 1, '$2a$10$rM4H9UDaiTGlHy3tsuQwdue7oR93qk63QRFUY2ehVeXePlIyTr5Qq'),
    (3, 'ADM-003', 'Chebet Kiprono', 1, 1, '$2a$10$n5Y6cWKO9ejU9ZJHUagaquZYnAvNCcwjgG.Hg.BBt2eKKR.BUw3vG'),
    (4, 'ADM-004', 'Esther Wambui',  2, 1, '$2a$10$b7KtrnmsG4/BdS5t9bDLZ.VB0ngvI1lj9I7MRpyPKpUtYWKbHYuIq'),
    (5, 'ADM-005', 'Faith Achieng',  2, 1, '$2a$10$uVHLVaNrZ8Dv3WmKXCxLTuKtMBuObHNjbUbjKkLUMeQqcor/mIpta'),
    (6, 'ADM-006', 'Hassan Ali',     2, 1, '$2a$10$7X/3FKSUr9ZMvn7AvHCCSuCzedC0H6cMy9oXQ5eqdRH2029ZMid4O'),
    (7, 'ADM-007', 'Imani Kariuki',  2, 0, '$2a$10$qZHHfzsOEPKycYiXcl2OfOBAtnHrkW8YD99aYGJ2sUbhI8YNxFqEK');

-- Exam 4 is a draft, so parents must not see it. Exam 5 is out of 50.
INSERT INTO exams (id, name, term_id, class_id, max_marks, published) VALUES
    (1, 'End of Term', 1, 1, 100, 1),
    (2, 'Midterm',     2, 1, 100, 1),
    (3, 'End of Term', 2, 1, 100, 1),
    (4, 'Midterm',     3, 1, 100, 0),
    (5, 'End of Term', 2, 2,  50, 1),
    (6, 'Midterm',     3, 2, 100, 1);

-- Deterministic "random" marks between 40% and 97% of each exam's maximum.
-- The draft exam only has Mathematics and English entered so far.
INSERT INTO exam_results (student_id, exam_id, subject_id, marks, max_marks, updated_by)
SELECT s.id, e.id, sub.id,
       CAST(e.max_marks * (40 + (s.id * 37 + sub.id * 17 + e.id * 11) % 58) / 100 AS INTEGER),
       e.max_marks, 2
FROM students s
JOIN exams e ON e.class_id = s.class_id
CROSS JOIN subjects sub
WHERE e.id <> 4 OR sub.id IN (1, 2);

-- Required fees (cents). 2025 Term 3 for the students enrolled then;
-- every 2026 term for active students: Grade 7 = 15,000, Grade 8 = 16,500.
INSERT INTO fee_assessments (student_id, term_id, amount_cents) VALUES
    (1, 1, 1400000), (2, 1, 1400000), (3, 1, 1400000), (7, 1, 1550000);

INSERT INTO fee_assessments (student_id, term_id, amount_cents)
SELECT s.id, t.id, CASE s.class_id WHEN 1 THEN 1500000 ELSE 1650000 END
FROM students s CROSS JOIN terms t
WHERE s.active = 1 AND t.id IN (2, 3, 4);

-- Payments (cents), recorded by the finance user. Current term (2026 Term 3):
-- ADM-001 fully paid, ADM-002 partial, ADM-003 unpaid, ADM-004 partial,
-- ADM-005 overpaid (credit 500.00), ADM-006 unpaid.
INSERT INTO fee_payments (student_id, term_id, amount_cents, paid_on, reference, recorded_by) VALUES
    -- 2025 Term 3
    (1, 1, 1400000, '2025-09-05', 'MPESA-QK1A001', 3),
    (2, 1, 1400000, '2025-09-08', 'BANK-55012',    3),
    (3, 1, 1000000, '2025-09-10', 'MPESA-QK1A014', 3),
    (3, 1,  400000, '2025-10-20', 'CASH',          3),
    (7, 1, 1550000, '2025-09-02', 'BANK-55003',    3),
    -- 2026 Term 1 (ADM-002 overpaid by 500.00)
    (1, 2, 1500000, '2026-01-08', 'MPESA-RB2C101', 3),
    (2, 2, 1550000, '2026-01-12', 'BANK-60110',    3),
    (3, 2, 1500000, '2026-01-15', 'MPESA-RB2C220', 3),
    (4, 2, 1650000, '2026-01-09', 'BANK-60105',    3),
    (5, 2,  800000, '2026-01-10', 'MPESA-RB2C150', 3),
    (5, 2,  850000, '2026-02-20', 'MPESA-RD4F031', 3),
    (6, 2, 1650000, '2026-01-11', 'BANK-60120',    3),
    -- 2026 Term 2 (ADM-003 partial)
    (1, 3, 1500000, '2026-05-04', 'MPESA-SC3A010', 3),
    (2, 3, 1500000, '2026-05-06', 'MPESA-SC3A044', 3),
    (3, 3,  700000, '2026-05-12', 'CASH',          3),
    (4, 3, 1650000, '2026-05-05', 'BANK-61001',    3),
    (5, 3, 1650000, '2026-05-07', 'BANK-61007',    3),
    (6, 3, 1000000, '2026-05-08', 'MPESA-SC3A061', 3),
    (6, 3,  650000, '2026-06-30', 'MPESA-SF6B002', 3),
    -- 2026 Term 3 (current)
    (1, 4, 1500000, '2026-09-02', 'MPESA-TD7K001', 3),
    (2, 4,  600000, '2026-09-03', 'MPESA-TD7K019', 3),
    (4, 4,  500000, '2026-09-04', 'CASH',          3),
    (4, 4,  500000, '2026-10-01', 'MPESA-TF9Q120', 3),
    (5, 4, 1700000, '2026-09-01', 'BANK-62002',    3);
