#!/bin/bash

# --- Configuration ---
PROJECT_NAME="go-api-starter"
GIT_MODULE_NAME="github.com/delordemm1/go-api-starter" # Change this to your actual module path

# --- Script Start ---
echo "🚀 Starting project scaffolding for '$PROJECT_NAME'..."

# Check if the directory already exists
# if [ -d "$PROJECT_NAME" ]; then
#     echo "❌ Error: Directory '$PROJECT_NAME' already exists. Aborting."
#     exit 1
# fi

# # Create the root project directory and navigate into it
# mkdir "$PROJECT_NAME"
# cd "$PROJECT_NAME" || exit

# --- Create All Directories ---
echo "📦 Creating directories..."
mkdir -p \
    cmd/api \
    cmd/migrate \
    internal/cache \
    internal/config \
    internal/database \
    internal/middleware \
    internal/modules/post \
    internal/modules/user \
    internal/server \
    migrations

# --- Create All Go Files ---
echo "📄 Creating Go files..."
touch \
    cmd/api/main.go \
    cmd/migrate/main.go \
    internal/cache/redis.go \
    internal/config/config.go \
    internal/database/postgres.go \
    internal/server/server.go \
    internal/modules/post/post.go \
    internal/modules/post/errors.go \
    internal/modules/post/handler.go \
    internal/modules/post/handler_admin.go \
    internal/modules/post/handler_public.go \
    internal/modules/post/service.go \
    internal/modules/post/service_admin.go \
    internal/modules/post/service_public.go \
    internal/modules/post/repository.go \
    internal/modules/post/repository_admin.go \
    internal/modules/post/repository_public.go \
    internal/modules/user/user.go \
    internal/modules/user/errors.go \
    internal/modules/user/handler.go \
    internal/modules/user/handler_auth.go \
    internal/modules/user/handler_password.go \
    internal/modules/user/handler_oauth.go \
    internal/modules/user/handler_profile.go \
    internal/modules/user/service.go \
    internal/modules/user/service_helpers.go \
    internal/modules/user/service_auth.go \
    internal/modules/user/service_password.go \
    internal/modules/user/service_oauth.go \
    internal/modules/user/service_profile.go \
    internal/modules/user/repository.go \
    internal/modules/user/repository_user.go

# --- Create Configuration and Documentation Files with Content ---
echo "📝 Creating config and doc files..."

# go.mod
echo "module $GIT_MODULE_NAME" > go.mod

# .gitignore
cat <<EOF > .gitignore
# Binaries
/bin
*.exe
*.test

# Local environment variables
.env

# OS-specific files
.DS_Store
Thumbs.db
EOF

# README.md
echo "# $PROJECT_NAME" > README.md

# .air.toml
# cat <<EOF > .air.toml
# root = "."
# tmp_dir = "tmp"

# [build]
#   cmd = "go build -o ./tmp/main ./cmd/api"
#   bin = "./tmp/main"
#   full_bin = "air-full"
#   delay = 1000
#   include_ext = ["go", "tpl", "tmpl", "html"]
#   exclude_dir = ["assets", "tmp", "vendor"]
#   log = "air.log"

# [log]
#   time = true

# [misc]
#   clean_on_exit = true
# EOF

# .env.example
# cat <<EOF > .env.example
# # --- Server Configuration ---
# SERVER_PORT="8080"
# SERVER_ENV="development" # development | staging | production

# # --- Database Configuration (PostgreSQL) ---
# DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"

# # --- Cache Configuration (Redis) ---
# REDIS_URL="redis://localhost:6379/0"
# EOF

# docker-compose.yml
# cat <<EOF > docker-compose.yml
# version: '3.8'

# services:
#   postgres:
#     image: postgres:15-alpine
#     container_name: my-postgres
#     ports:
#       - "5432:5432"
#     environment:
#       POSTGRES_USER: user
#       POSTGRES_PASSWORD: password
#       POSTGRES_DB: dbname
#     volumes:
#       - postgres_data:/var/lib/postgresql/data

#   redis:
#     image: redis:7-alpine
#     container_name: my-redis
#     ports:
#       - "6379:6379"
#     volumes:
#       - redis_data:/data

# volumes:
#   postgres_data:
#   redis_data:
# EOF

# Makefile
# cat <<EOF > Makefile
# .PHONY: help dev build migrate-create migrate-up migrate-down migrate-status

# ## help: Show this help message
# help:
# 	@echo "Available commands:"
# 	@sed -n 's/^##//p' \$(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'

# ## dev: Run the app in development with live-reloading
# dev:
# 	@air

# ## build: Build the production binary
# build:
# 	@go build -o ./bin/api ./cmd/api

# ## migrate-create: Create a new migration (e.g., make migrate-create name=add_indices_to_posts)
# migrate-create:
# ifndef name
# 	$(error name is not set. Usage: make migrate-create name=<migration_name>)
# endif
# 	@go run cmd/migrate/main.go create "$(name)" sql

# ## migrate-up: Apply all pending migrations
# migrate-up:
# 	@go run cmd/migrate/main.go up

# ## migrate-down: Rollback the latest migration
# migrate-down:
# 	@go run cmd/migrate/main.go down

# ## migrate-status: Show migration status
# migrate-status:
# 	@go run cmd/migrate/main.go status
# EOF

# Create an initial migration file with a dynamic timestamp
# TIMESTAMP=$(date +%Y%m%d%H%M%S)
# touch "migrations/${TIMESTAMP}_create_initial_tables.sql"

echo "✅ Project '$PROJECT_NAME' created successfully!"
echo "➡️  Next steps:"
echo "    1. cd $PROJECT_NAME"
echo "    2. Update the module path in go.mod"
echo "    3. Run 'go mod tidy'"
echo "    4. Start coding!"