package contextx

// Key is a private type to avoid collisions in request context keys.
type Key string

// UserIDKey is the context key used to store the authenticated user's ID (string).
const UserIDKey Key = "userID"

// SessionIDKey is the context key used to store the current session ID (string).
const SessionIDKey Key = "sessionID"