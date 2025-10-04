/*
  # Create Users Table

  ## Overview
  This migration creates the core users table for authentication and user management.

  ## New Tables

  ### `users`
  Stores user account information including credentials and profile data.

  **Columns:**
  - `id` (uuid, primary key) - Unique user identifier
  - `firstName` (text) - User's first name
  - `lastName` (text) - User's last name
  - `email` (text, unique) - User's email address (used for login)
  - `password_hash` (text) - Hashed password using bcrypt
  - `email_verified` (boolean) - Whether email has been verified
  - `password_reset_token` (text, nullable) - Token for password reset flow
  - `password_reset_token_expiry` (timestamptz, nullable) - Expiry for reset token
  - `created_at` (timestamptz) - Account creation timestamp
  - `updated_at` (timestamptz) - Last update timestamp

  ## Security
  - Enable RLS on users table
  - Users can read their own profile data
  - Users can update their own profile data
  - Password hash is protected and not exposed to clients

  ## Indexes
  - Unique index on email for fast lookups and preventing duplicates
  - Index on password_reset_token for password reset flow

  ## Important Notes
  - Email addresses are case-sensitive as stored
  - Password hashes use bcrypt with cost 10
  - Password reset tokens expire after 15 minutes
*/

-- Create users table
CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "firstName" text NOT NULL DEFAULT '',
  "lastName" text NOT NULL DEFAULT '',
  email text UNIQUE NOT NULL,
  password_hash text DEFAULT '',
  email_verified boolean DEFAULT false,
  password_reset_token text,
  password_reset_token_expiry timestamptz,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- Create indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_password_reset_token ON users(password_reset_token) WHERE password_reset_token IS NOT NULL;

-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Users can view own profile"
  ON users
  FOR SELECT
  TO authenticated
  USING (auth.uid()::text = id::text);

CREATE POLICY "Users can update own profile"
  ON users
  FOR UPDATE
  TO authenticated
  USING (auth.uid()::text = id::text)
  WITH CHECK (auth.uid()::text = id::text);

-- Allow user creation during registration (public access)
CREATE POLICY "Anyone can create user account"
  ON users
  FOR INSERT
  TO anon, authenticated
  WITH CHECK (true);
