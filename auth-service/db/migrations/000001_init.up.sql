CREATE TABLE sessions (
  id           UUID PRIMARY KEY,
  user_id      UUID NOT NULL,
  email        TEXT NOT NULL,
  user_agent   TEXT NOT NULL DEFAULT '',
  ip           TEXT NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at   TIMESTAMPTZ NOT NULL,
  revoked_at   TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
