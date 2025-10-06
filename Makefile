PACKAGES := $(shell go list ./...)
# name := $(shell basename ${PWD})

all: help

.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

## init: initialize project (make init module=github.com/user/project)
.PHONY: init
init:
	go install github.com/air-verse/air@latest
	asdf reshim golang

## vet: vet code
.PHONY: vet
vet:
	go vet $(PACKAGES)

## test: run unit tests
.PHONY: test
test:
	go test -race -cover $(PACKAGES)

## migrate-create: create a new migration (e.g., make migrate-create name=add_indices_to_posts)
.PHONY: migrate-create
migrate-create:
ifndef name
    $(error name is not set. Usage: make migrate-create name=<migration_name>)
endif
    go run cmd/migrate create "$(name)" sql

## migrate-up: migrate up
.PHONY: migrate-up
migrate-up:
	go run cmd/migrate up

## migrate-down: migrate down
.PHONY: migrate-down
migrate-down:
	go run cmd/migrate/main.go down

## migrate-status: migrate status
.PHONY: migrate-status
migrate-status:
	go run cmd/migrate/main.go status

## migrate-version: migrate version
.PHONY: migrate-version
migrate-version:
	go run cmd/migrate/main.go version