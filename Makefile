.PHONY: test lint run db-up db-down clean

SHELL := /bin/bash

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	@test -z "$$(go fmt ./...)" || (echo "Files not formatted:" && go fmt ./... && echo "Run 'go fmt ./...' to fix" && exit 1)

run:
	@set -o allexport && source .env 2>/dev/null; set +o allexport; go run ./cmd/server

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

clean:
	rm -f notification-service
	docker compose down -v