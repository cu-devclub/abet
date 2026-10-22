-- DEV ONLY test data (make seed). Idempotent: safe to run repeatedly. Example e-mails only.
-- Run against main_db:  make seed
BEGIN;

INSERT INTO departments (code, name) VALUES
  ('CS',   'Computer Science'),
  ('MATH', 'Mathematics')
ON CONFLICT (code) DO NOTHING;

INSERT INTO curricula (code, name, department_id) VALUES
  ('CS-B',   'B.Sc. Computer Science',  (SELECT id FROM departments WHERE code = 'CS')),
  ('CS-M',   'M.Sc. Computer Science',  (SELECT id FROM departments WHERE code = 'CS')),
  ('MATH-B', 'B.Sc. Mathematics',       (SELECT id FROM departments WHERE code = 'MATH'))
ON CONFLICT (code) DO NOTHING;

INSERT INTO users (email, role, department_id, curriculum_id) VALUES
  ('super1@chula.ac.th',      'super_admin', NULL, NULL),
  ('admin1@chula.ac.th',      'admin',       (SELECT id FROM departments WHERE code='CS'),   NULL),
  ('admin2@chula.ac.th',      'admin',       (SELECT id FROM departments WHERE code='MATH'), (SELECT id FROM curricula WHERE code='MATH-B')),
  ('instructor1@chula.ac.th', 'instructor',  (SELECT id FROM departments WHERE code='CS'),   (SELECT id FROM curricula WHERE code='CS-B')),
  ('instructor2@chula.ac.th', 'instructor',  (SELECT id FROM departments WHERE code='MATH'), (SELECT id FROM curricula WHERE code='MATH-B')),
  ('staff1@chula.ac.th',      'staff',       (SELECT id FROM departments WHERE code='CS'),   (SELECT id FROM curricula WHERE code='CS-B')),
  ('staff2@chula.ac.th',      'staff',       (SELECT id FROM departments WHERE code='MATH'), (SELECT id FROM curricula WHERE code='MATH-B'))
ON CONFLICT (email) DO NOTHING;

-- admin1: whole CS department; admin2: only curriculum MATH-B
INSERT INTO admin_scopes (user_id, department_id)
  SELECT u.id, d.id FROM users u, departments d WHERE u.email='admin1@chula.ac.th' AND d.code='CS'
ON CONFLICT DO NOTHING;
INSERT INTO admin_scopes (user_id, curriculum_id)
  SELECT u.id, c.id FROM users u, curricula c WHERE u.email='admin2@chula.ac.th' AND c.code='MATH-B'
ON CONFLICT DO NOTHING;

INSERT INTO courses (code, name, curriculum_id) VALUES
  ('2301101', 'Introduction to Programming', (SELECT id FROM curricula WHERE code='CS-B')),
  ('2301102', 'Data Structures',             (SELECT id FROM curricula WHERE code='CS-B')),
  ('2301201', 'Advanced Algorithms',         (SELECT id FROM curricula WHERE code='CS-M')),
  ('2301301', 'Calculus I',                  (SELECT id FROM curricula WHERE code='MATH-B'))
ON CONFLICT (curriculum_id, code) DO NOTHING;

-- offerings for the current academic year (CE): semester 1 for every course, semester 2 for two of them
INSERT INTO course_offerings (course_id, academic_year, semester)
  SELECT id, EXTRACT(YEAR FROM now())::int, 1 FROM courses WHERE code IN ('2301101','2301102','2301201','2301301')
ON CONFLICT (course_id, academic_year, semester) DO NOTHING;
INSERT INTO course_offerings (course_id, academic_year, semester)
  SELECT id, EXTRACT(YEAR FROM now())::int, 2 FROM courses WHERE code IN ('2301101','2301301')
ON CONFLICT (course_id, academic_year, semester) DO NOTHING;

-- explicit access rows (staff of a department get default download without a row)
INSERT INTO course_access (offering_id, user_id, can_upload, can_download, source)
  SELECT o.id, u.id, v.up, v.dl, 'manual'
  FROM (VALUES
    ('2301101', 'instructor1@chula.ac.th', true,  true),
    ('2301102', 'instructor1@chula.ac.th', true,  false),
    ('2301301', 'instructor2@chula.ac.th', true,  true)
  ) AS v(course_code, email, up, dl)
  JOIN courses c ON c.code = v.course_code
  JOIN course_offerings o ON o.course_id = c.id
  JOIN users u ON u.email = v.email
ON CONFLICT (offering_id, user_id) DO NOTHING;

COMMIT;
