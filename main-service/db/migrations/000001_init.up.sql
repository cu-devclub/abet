CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE departments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT NOT NULL UNIQUE,                     -- stored UPPERCASE
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE curricula (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  department_id UUID NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
  cover_filename TEXT,                           -- original filename; NULL = no cover
  cover_size_bytes BIGINT,
  cover_updated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_curricula_department ON curricula(department_id);

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE CHECK (email = lower(email)),
  name TEXT NOT NULL DEFAULT '',                 -- filled from Google on login; '' until then
  picture_url TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL CHECK (role IN ('super_admin','admin','instructor','staff')),
  department_id UUID REFERENCES departments(id) ON DELETE RESTRICT,
  curriculum_id UUID REFERENCES curricula(id) ON DELETE RESTRICT,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Admin management scope (D: admin may span multiple departments/curricula). Only meaningful for role=admin.
CREATE TABLE admin_scopes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  department_id UUID REFERENCES departments(id) ON DELETE CASCADE,
  curriculum_id UUID REFERENCES curricula(id) ON DELETE CASCADE,
  CHECK ((department_id IS NULL) <> (curriculum_id IS NULL))
);
CREATE UNIQUE INDEX uq_admin_scope_dept ON admin_scopes(user_id, department_id) WHERE department_id IS NOT NULL;
CREATE UNIQUE INDEX uq_admin_scope_curr ON admin_scopes(user_id, curriculum_id) WHERE curriculum_id IS NOT NULL;

CREATE TABLE courses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT NOT NULL,                            -- e.g. 2301173
  name TEXT NOT NULL,
  curriculum_id UUID NOT NULL REFERENCES curricula(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (curriculum_id, code)
);

CREATE TABLE course_offerings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  academic_year INT NOT NULL CHECK (academic_year BETWEEN 2000 AND 2200),   -- CE (A2)
  semester SMALLINT NOT NULL CHECK (semester IN (1,2,3)),                  -- 3 = Summer (A1)
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (course_id, academic_year, semester)
);
CREATE INDEX idx_offerings_year ON course_offerings(academic_year, semester);

CREATE TABLE course_access (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  can_upload BOOLEAN NOT NULL,
  can_download BOOLEAN NOT NULL,
  source TEXT NOT NULL CHECK (source IN ('manual','csv')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (offering_id, user_id),
  CHECK (can_upload OR can_download)
);
CREATE INDEX idx_access_user ON course_access(user_id);

CREATE TABLE submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
  uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
  original_filename TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('invalid','queued','processing','generated','failed')),
  job_id UUID NOT NULL UNIQUE,
  pdf_size_bytes BIGINT,
  error_count INT NOT NULL DEFAULT 0,
  warning_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);
CREATE INDEX idx_submissions_offering_created ON submissions(offering_id, created_at DESC);

CREATE TABLE submission_issues (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  severity TEXT NOT NULL CHECK (severity IN ('error','warning')),
  stage TEXT NOT NULL CHECK (stage IN ('missing','structure','naming','csv','build')),
  code TEXT NOT NULL,
  title TEXT NOT NULL,                          -- exact SRS §7 message (with placeholders filled)
  description TEXT NOT NULL DEFAULT '',          -- SRS "Cause / Explanation"
  location TEXT NOT NULL DEFAULT '',             -- relpath / "row N"
  suggested_fix TEXT NOT NULL DEFAULT '',        -- SRS "User Action Required"
  sort_order INT NOT NULL
);
CREATE INDEX idx_issues_submission ON submission_issues(submission_id, sort_order);
