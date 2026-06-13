# ─────────────────────────────────────────────────────────────────────────────
# MultiTrade Platform — Developer Makefile
# Usage: make <target>
# ─────────────────────────────────────────────────────────────────────────────

APP_NAME   := multitrade
BINARY     := bin/$(APP_NAME)
MAIN       := ./cmd/api
DOCS_OUT   := internal/docs
SWAG_BIN   := $(shell go env GOPATH)/bin/swag

.PHONY: all build run test test-unit test-integration lint fmt vet \
        swagger docker-up docker-down migrate clean help

## all: build the binary (default)
all: build

## build: compile the server binary into ./bin/
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY) $(MAIN)

## run: run the server directly (no binary)
run:
	go run $(MAIN)

## test: run all tests with verbose output
test:
	go test -v -timeout 120s ./...

## test-unit: run only unit tests (no DB required)
test-unit:
	go test -v -timeout 30s ./internal/auth/...

## test-integration: run all integration tests (requires Postgres + Redis)
test-integration:
	go test -v -timeout 120s \
		./internal/wallet/... \
		./internal/matchingengine/...

## test-e2e: run the full e2e smoke test (server must be running on :8080)
test-e2e:
	go run scripts/smoke_e2e.go

## lint: run golangci-lint (install: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

## fmt: format all Go source files
fmt:
	gofmt -w -s .

## vet: run go vet
vet:
	go vet ./...

## swagger: regenerate Swagger/OpenAPI docs
swagger:
	@which $(SWAG_BIN) > /dev/null || go install github.com/swaggo/swag/cmd/swag@v1.8.12
	$(SWAG_BIN) init -g cmd/api/main.go -o $(DOCS_OUT) --parseDependency=false
	@echo "Swagger docs generated at $(DOCS_OUT)/"

## docker-up: start all infrastructure containers (Postgres, Redis, Kafka)
docker-up:
	docker compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 3
	docker compose ps

## docker-down: stop and remove all containers
docker-down:
	docker compose down

## docker-build: build the production Docker image
docker-build:
	docker build -t $(APP_NAME):latest .

## migrate: apply all pending database migrations (server handles this on start)
migrate:
	go run $(MAIN) --migrate-only 2>/dev/null || \
		echo "Migrations are run automatically on server start. Use 'make run'."

## clean: remove build artifacts
clean:
	@rm -rf bin/
	@echo "Cleaned build artifacts."

## help: show this help message
help:
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'
