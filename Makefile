.PHONY: test lint run db-up db-down clean docker-push

DOCKER_IMAGE ?= fabianofsc/notification-service
TAG ?= latest

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

# docker-push publishes a multi-architecture manifest for $(TAG) (defaults to
# latest).
# Run `docker login` before invoking this target.
docker-push:
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $(DOCKER_IMAGE):$(TAG) .

clean:
	rm -f notification-service
	docker compose down -v