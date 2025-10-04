/*
  # Create User Sessions and OAuth State Tables

  ## Overview
  This migration creates tables for managing user sessions and OAuth authentication flows.

  ## New Tables

  ### 1. `user_active_sessions`
  Tracks active user sessions for session management and security auditing.

  **Columns:**
  - `id` (uuid, primary key) - Unique session identifier
  - `user_id` (uuid, foreign key) - References the user who owns this session
  - `session_token` (text, unique) - JWT or session token string
  - `user_agent` (text) - Browser/client user agent string
  - `ip_address` (text) - IP address of the client
  - `last_active_at` (timestamptz) - Last time this session was used
  - `created_at` (timestamptz) - Session creation timestamp

  ### 2. `oauth_states`
  Stores OAuth flow state data for CSRF protection and flow management.

  **Columns:**
  - `state` (text, primary key) - Unique state string for CSRF protection
  - `provider` (text) - OAuth provider name (google, facebook, github, etc.)
  - `user_id` (uuid, nullable) - Optional user ID if linking existing account
  - `verifier` (text) - PKCE code verifier for enhanced security
  - `expires_at` (timestamptz) - When this state expires
  - `created_at` (timestamptz) - State creation timestamp
  - `updated_at` (timestamptz) - Last update timestamp

  ## Security
  - Enable RLS on both tables
  - Sessions can only be accessed by the owning user
  - OAuth states are protected and only accessible server-side

  ## Indexes
  - Index on `session_token` for fast session lookups
  - Index on `user_id` for user session queries
  - Index on `expires_at` for cleanup operations
  - Index on `state` (primary key provides this automatically)

  ## Important Notes
  - OAuth states should be short-lived (typically 15 minutes)
  - Expired sessions and states should be cleaned up periodically
  - Session tokens should be hashed or encrypted in production
*/

-- Create user_active_sessions table
CREATE TABLE IF NOT EXISTS user_active_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_token text UNIQUE NOT NULL,
  user_agent text DEFAULT '',
  ip_address text DEFAULT '',
  last_active_at timestamptz DEFAULT now(),
  created_at timestamptz DEFAULT now()
);

-- Create indexes for user_active_sessions
CREATE INDEX IF NOT EXISTS idx_user_active_sessions_user_id ON user_active_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_active_sessions_session_token ON user_active_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_user_active_sessions_last_active_at ON user_active_sessions(last_active_at);

-- Enable RLS on user_active_sessions
ALTER TABLE user_active_sessions ENABLE ROW LEVEL SECURITY;

-- RLS Policies for user_active_sessions
CREATE POLICY "Users can view own sessions"
  ON user_active_sessions
  FOR SELECT
  TO authenticated
  USING (auth.uid()::text = user_id::text);

CREATE POLICY "Users can delete own sessions"
  ON user_active_sessions
  FOR DELETE
  TO authenticated
  USING (auth.uid()::text = user_id::text);

-- Create oauth_states table
CREATE TABLE IF NOT EXISTS oauth_states (
  state text PRIMARY KEY,
  provider text NOT NULL,
  user_id uuid,
  verifier text DEFAULT '',
  expires_at timestamptz NOT NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- Create indexes for oauth_states
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at ON oauth_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_states_user_id ON oauth_states(user_id) WHERE user_id IS NOT NULL;

-- Enable RLS on oauth_states
ALTER TABLE oauth_states ENABLE ROW LEVEL SECURITY;

-- RLS Policies for oauth_states
-- OAuth states are typically managed server-side only
-- We use restrictive policies to prevent client access
CREATE POLICY "Service role can manage oauth states"
  ON oauth_states
  FOR ALL
  TO service_role
  USING (true)
  WITH CHECK (true);
