-- +goose Up
-- +goose StatementBegin
-- Users table
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  password_reset_token TEXT NOT NULL DEFAULT '',
  password_reset_token_expiry TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index to optimize lookup by password_reset_token
CREATE INDEX IF NOT EXISTS idx_users_password_reset_token ON users (password_reset_token);

-- Trigger to coalesce NULL password_reset_token to empty string
CREATE OR REPLACE FUNCTION users_password_reset_token_not_null() RETURNS TRIGGER AS $$
BEGIN
  IF NEW.password_reset_token IS NULL THEN
    NEW.password_reset_token := '';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_password_reset_token_not_null ON users;
CREATE TRIGGER trg_users_password_reset_token_not_null
BEFORE INSERT OR UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION users_password_reset_token_not_null();

-- user_active_sessions table
CREATE TABLE IF NOT EXISTS user_active_sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_token TEXT NOT NULL UNIQUE,
  user_agent TEXT,
  ip_address TEXT,
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_active_sessions_user_id ON user_active_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_user_active_sessions_last_active_at ON user_active_sessions (last_active_at);

-- oauth_states table
CREATE TABLE IF NOT EXISTS oauth_states (
  state TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  verifier TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at ON oauth_states (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_oauth_states_expires_at;
DROP TABLE IF EXISTS oauth_states;

DROP INDEX IF EXISTS idx_user_active_sessions_last_active_at;
DROP INDEX IF EXISTS idx_user_active_sessions_user_id;
DROP TABLE IF EXISTS user_active_sessions;

DROP TRIGGER IF EXISTS trg_users_password_reset_token_not_null ON users;
DROP FUNCTION IF EXISTS users_password_reset_token_not_null();

DROP INDEX IF EXISTS idx_users_password_reset_token;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
