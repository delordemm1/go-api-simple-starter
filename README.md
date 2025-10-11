# Go API Simple Starter

Production-ready Go API starter built with Chi + Huma, PostgreSQL, Redis, modular domain architecture, RFC 7807 problem+json errors, and first-class OAuth (Google + Apple form_post).

## Contents
- Quick start
- Tech stack
- Project structure
- Configuration
- Database & migrations
- Modules and layering
- Error handling (RFC 7807)
- Sessions & auth
- Notifications & templates
- OAuth (Google & Apple)
- API routes
- Development workflow
- Extending with new modules
- Deployment notes

---

## Quick start

Prerequisites:
- Go 1.21+ (asdf recommended via [.tool-versions](.tool-versions))
- PostgreSQL
- Redis

1) Clone and configure environment
- Create a .env file (or copy .env.example if provided) and fill values (see Configuration)

2) Install dev tools
- make init

3) Run database migrations
- make migrate-up

4) Start the API
- go run [cmd/api/main.go](cmd/api/main.go)
- The API listens on :8080 by default (configurable)

Health check:
- curl http://localhost:8080/health

---

## Tech stack
- HTTP: Chi router + Huma typed handlers [internal/server/server.go](internal/server/server.go)
- Config: Viper + godotenv [internal/config/config.go](internal/config/config.go)
- Database: PostgreSQL (pgx pool helper) [internal/database/postgres.go](internal/database/postgres.go)
- Cache: Redis client helper [internal/cache/redis.go](internal/cache/redis.go)
- Sessions: Postgres-backed provider [internal/session/postgres.go](internal/session/postgres.go)
- Problem errors: RFC 7807 helpers [internal/httpx/problem.go](internal/httpx/problem.go)
- Notifications: SMTP + SMS + embedded templates [internal/notification](internal/notification)
- User module: repository/service/handlers [internal/modules/user](internal/modules/user)

---

## Project structure

Key layout:
- [cmd/api/main.go](cmd/api/main.go) CLI entrypoint using Huma CLI hooks
- [internal/server/server.go](internal/server/server.go) router + API instance, middleware, health
- [internal/config/config.go](internal/config/config.go) strongly-typed config loader (env-only)
- [internal/httpx/problem.go](internal/httpx/problem.go) problem+json conversions
- [internal/modules/user](internal/modules/user) user bounded context
  - repositories: [internal/modules/user/repository.go](internal/modules/user/repository.go) (+ sub-repos)
  - services: [internal/modules/user/service.go](internal/modules/user/service.go) (+ sub-services)
  - handlers: [internal/modules/user/handler.go](internal/modules/user/handler.go) (+ sub-handlers)
  - domain errors: [internal/modules/user/errors.go](internal/modules/user/errors.go)
  - DTOs/handlers for auth/password/profile/oauth: see files under internal/modules/user
- [migrations](migrations) schema managed by Goose [cmd/migrate/main.go](cmd/migrate/main.go)
- [Makefile](Makefile) developer tasks (migrations, tests)

---

## Configuration

Configuration is loaded from environment only. [.env](.env) is read into the process by godotenv, then Viper binds env keys to the typed struct in [internal/config/config.go](internal/config/config.go).

Important environment variables (examples):
- Server
  - SERVER_PORT=8080
  - SERVER_ENV=development
- Database / cache
  - DATABASE_URL=postgres://user:pass@localhost:5432/app?sslmode=disable
  - REDIS_URL=redis://localhost:6379
- JWT
  - JWT_SECRET=change-me
- Google OAuth
  - GOOGLE_CLIENT_ID=...
  - GOOGLE_CLIENT_SECRET=...
  - GOOGLE_REDIRECT_URL=http://localhost:8080/users/oauth/google/callback
- Apple OAuth
  - APPLE_CLIENT_ID=com.example.app.web
  - APPLE_TEAM_ID=XXXXXXXXXX
  - APPLE_KEY_ID=YYYYYYYYYY
  - APPLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
  - APPLE_REDIRECT_URL=http://localhost:8080/users/oauth/apple/callback
- SMTP (email)
  - SMTP_HOST=smtp.mailtrap.io
  - SMTP_PORT=2525
  - SMTP_USERNAME=...
  - SMTP_PASSWORD=...
  - SMTP_FROM="App Name <no-reply@example.com>"
- Templates
  - EMAIL_TEMPLATES_DIR=./internal/notification/templates/files (optional override in dev)
  - TEMPLATES_RELOAD=false
- Verification & reset tokens
  - VERIFICATION_TTL_MINUTES=10
  - VERIFICATION_RESEND_COOLDOWN_SECONDS=60
  - VERIFICATION_MAX_ATTEMPTS=5
  - RESET_TOKEN_TTL_MINUTES=15

See defaults in [internal/config/config.go](internal/config/config.go).

---

## Database & migrations

Goose is used for migrations via [cmd/migrate/main.go](cmd/migrate/main.go). Core schema:
- Users table, active sessions, and OAuth state: [migrations/20251006101208_initial_tables.sql](migrations/20251006101208_initial_tables.sql)
- Verification codes and action tokens: [migrations/20251011151500_verification_and_action_tokens.sql](migrations/20251011151500_verification_and_action_tokens.sql)

Common tasks (see [Makefile](Makefile)):
- Create migration: make migrate-create name=add_indices_to_posts
- Migrate up: make migrate-up
- Migrate down: make migrate-down
- Status/version: make migrate-status / make migrate-version

Notes:
- oauth_states stores PKCE verifier and anti-CSRF state per provider.
- user_active_sessions tracks device sessions with sliding/absolute TTLs handled in code.
- verification_codes and action_tokens enable email verification and internal token flows.

---

## Modules and layering

Each bounded context lives under internal/modules/<name>. The user module demonstrates the pattern:
- Repository layer (SQL): [internal/modules/user/repository_*.go](internal/modules/user)
- Service layer (business rules): [internal/modules/user/service_*.go](internal/modules/user)
- Handler layer (transport/Huma): [internal/modules/user/handler_*.go](internal/modules/user)
- Domain errors: [internal/modules/user/errors.go](internal/modules/user/errors.go) mapped uniformly to HTTP problems via [internal/httpx/problem.go](internal/httpx/problem.go)

Huma typed handlers bind:
- path:"...", query:"..." for URL pieces
- Body struct with json:"..." for JSON request bodies
- form:"..." for form-encoded fields (e.g., Apple form_post callback)

Router wiring:
- API setup and security schemes: [internal/server/server.go](internal/server/server.go)
- User routes: [internal/modules/user/handler.go](internal/modules/user/handler.go)

---

## Error handling (RFC 7807)

Domain errors are plain Go types carrying HTTP metadata: [internal/modules/user/errors.go](internal/modules/user/errors.go). They implement a small interface read by [internal/httpx/problem.go](internal/httpx/problem.go) which converts any domain error into an RFC 7807 application/problem+json response with these extensions:
- code: stable machine-readable business code (e.g., ErrInvalidResetToken)
- context: optional structured payload (e.g., validation field errors)
- requestId: request correlation via chi middleware

Handlers return domain errors and call httpx.ToProblem(ctx, err) once, ensuring consistent error responses without switch/case per error type.

---

## Sessions & auth

- Session store: Postgres provider [internal/session/postgres.go](internal/session/postgres.go)
- Auth middleware: Huma-compatible bearer auth [internal/middleware/auth_huma.go](internal/middleware/auth_huma.go)
- Protected route group is created in [internal/modules/user/handler.go](internal/modules/user/handler.go) and wired to profile/endpoints.

Login flows:
- Email/password: issues an opaque session token returned to the client, used as a Bearer token
- OAuth (Google/Apple): after callback + token exchange, the service creates the same session type and returns the token

---

## Notifications & templates

Notification service composes:
- SMTP email: [internal/notification/email_smtp.go](internal/notification/email_smtp.go)
- SMS sender (dummy): [internal/notification/sms_sender.go](internal/notification/sms_sender.go)
- Push sender (dummy): [internal/notification/push_sender.go](internal/notification/push_sender.go)
- Template engine (embedded files; dev reload supported): [internal/notification/templates](internal/notification/templates)

Example templates are embedded under [internal/notification/templates/files](internal/notification/templates/files).

---

## OAuth (Google & Apple)

Initiation:
- Endpoint: GET /users/oauth/{provider} via [internal/modules/user/handler_oauth.go](internal/modules/user/handler_oauth.go)
- Service builds AuthCodeURL with PKCE + state in [internal/modules/user/service_oauth.go](internal/modules/user/service_oauth.go)
- Apple specifics: when requesting name/email scopes, Apple requires response_mode=form_post and response_type=code. This is applied only for Apple.

Callback (single path, dual methods):
- GET /users/oauth/{provider}/callback (for Google)
- POST /users/oauth/{provider}/callback (for Apple form_post). Accepts application/x-www-form-urlencoded and optionally JSON.
- Both call into the same service callback for verification, token exchange, and local session creation.

Apple details:
- Client secret is an ES256 JWT generated per request (team ID, key ID, private key) and passed during token exchange.
- The user's email and sub come from id_token in the token response.
- On first sign-in, Apple may include a user field (JSON string with name) in the form POST; capture it optionally in the POST DTO if you need to persist names.
- Ensure your reverse proxy forwards POST to the callback URL and that the Apple redirect URL matches exactly (scheme/host/path).

Data:
- oauth_states table stores the anti-CSRF state and PKCE verifier until consumed.
- Sessions are created with the same mechanism as password login.

---

## API routes (high level)

Public:
- GET /health
- POST /users/register
- POST /users/login
- POST /users/password/forgot
- POST /users/password/code/verify
- POST /users/password/reset
- POST /users/verify/email/request
- POST /users/verify/email/confirm
- GET /users/oauth/{provider}
- GET /users/oauth/{provider}/callback
- POST /users/oauth/{provider}/callback

Protected (Bearer session):
- GET /users/profile
- PATCH /users/profile
- POST /users/logout

See route registration in [internal/modules/user/handler.go](internal/modules/user/handler.go).

---

## Development workflow

Run locally:
- go run [cmd/api/main.go](cmd/api/main.go) -p 8080

Useful make targets:
- make test
- make vet
- make migrate-create name=add_feature
- make migrate-up
- make migrate-down

Live reload (optional): install air via make init and run air (see [.air.toml](.air.toml)).

Logging: JSON structured logs with slog are enabled in the entrypoint. Add fields liberally for observability.

---

## Extending with new modules

1) Create internal/modules/your-domain with repository_, service_, handler_ files
2) Add domain-specific errors like [internal/modules/user/errors.go](internal/modules/user/errors.go)
3) Register routes from your handler in [internal/server/server.go](internal/server/server.go) or the module's RegisterRoutes
4) Follow the patterns:
   - Inputs: typed DTOs with path/query/Body/form tags
   - Validation: central validator (see [internal/validation/validator.go](internal/validation/validator.go))
   - Errors: return domain errors, map once via httpx.ToProblem
   - Persistence: keep SQL in repository layer; keep business rules in service layer

---

## Deployment notes

- Build: go build ./...
- Env-only configuration: configure environment variables; no config files are required in production
- Database migrations should run on startup or via CI/CD using [cmd/migrate/main.go](cmd/migrate/main.go)
- TLS and reverse proxy in front (e.g., Nginx/Caddy); ensure POST is forwarded for Apple callback
- Observability: consider structured logs shipping and request IDs propagated to logs and problems

---

## License

MIT