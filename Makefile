.PHONY: dev test test-integration lint run db-up db-down clean

SHELL := /bin/bash

# --- Development ---

dev:
	@echo "Starting PostgreSQL..."
	docker compose up -d postgres
	@echo "Waiting for PostgreSQL..."
	@sleep 2
	@echo "Running the service..."
	@set -o allexport && source .env 2>/dev/null; set +o allexport; go run ./cmd/server

# --- Testing ---

test:
	go test -race -count=1 ./internal/...

test-integration:
	go test -race -count=1 -tags=integration ./internal/...

# --- Linting ---

lint:
	go vet ./...
	@test -z "$$(go fmt ./...)" || (echo "Files not formatted:" && go fmt ./... && echo "Run 'go fmt ./...' to fix" && exit 1)

# --- Run ---

run:
	@set -o alexport && source .env 2>/dev/null; set +o allexport; go run ./cmd/server

# --- Infrastructure ---

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

# --- Cleanup ---

clean:
	rm -f notification-service
	docker compose down -v