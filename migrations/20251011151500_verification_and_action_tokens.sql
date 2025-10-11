-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS verification_codes (
  id UUID PRIMARY KEY,
  user_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  contact TEXT NOT NULL,
  purpose TEXT NOT NULL,
  channel TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  attempts INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 5,
  last_sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookups
CREATE INDEX IF NOT EXISTS idx_verification_codes_contact ON verification_codes (contact, purpose, channel);
CREATE INDEX IF NOT EXISTS idx_verification_codes_user ON verification_codes (user_id, purpose, channel);
CREATE INDEX IF NOT EXISTS idx_verification_codes_expires_at ON verification_codes (expires_at);

-- Only one active code per contact/purpose/channel (consumed_at IS NULL)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_verification_codes_active_contact
  ON verification_codes (contact, purpose, channel)
  WHERE consumed_at IS NULL;

-- Only one active code per user/purpose/channel (when user_id is set)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_verification_codes_active_user
  ON verification_codes (user_id, purpose, channel)
  WHERE consumed_at IS NULL AND user_id IS NOT NULL;

-- action_tokens for internal short-lived tokens (e.g., password reset)
CREATE TABLE IF NOT EXISTS action_tokens (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose TEXT NOT NULL, -- e.g., 'password_reset'
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_action_tokens_token_hash ON action_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_action_tokens_user_purpose ON action_tokens (user_id, purpose, expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_action_tokens_user_purpose;
DROP INDEX IF EXISTS uidx_action_tokens_token_hash;
DROP TABLE IF EXISTS action_tokens;

DROP INDEX IF EXISTS uidx_verification_codes_active_user;
DROP INDEX IF EXISTS uidx_verification_codes_active_contact;
DROP INDEX IF EXISTS idx_verification_codes_expires_at;
DROP INDEX IF EXISTS idx_verification_codes_user;
DROP INDEX IF EXISTS idx_verification_codes_contact;
DROP TABLE IF EXISTS verification_codes;
-- +goose StatementEnd