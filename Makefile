.PHONY: build test lint clean run deps docker-up docker-down

BINARY_NAME=wa-server
BUILD_DIR=bin
GO=go
GOFLAGS=-v

build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-api ./cmd/api
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-worker ./cmd/worker
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-websocket ./cmd/websocket

test:
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test
	$(GO) tool cover -html=coverage.out

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

run-api:
	$(GO) run ./cmd/api

run-worker:
	$(GO) run ./cmd/worker

run-websocket:
	$(GO) run ./cmd/websocket

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

deps:
	$(GO) mod download
	$(GO) mod tidy

docker-up:
	docker-compose -f deployments/docker-compose.yml up -d

docker-down:
	docker-compose -f deployments/docker-compose.yml down

migrate:
	$(GO) run ./cmd/migrate

help:
	@echo "Available targets:"
	@echo "  build         - Build all applications"
	@echo "  test          - Run tests with race detector"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linters"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  run-api       - Run API server"
	@echo "  run-worker    - Run worker"
	@echo "  run-websocket - Run WebSocket server"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  docker-up     - Start Docker services"
	@echo "  docker-down   - Stop Docker services"
	@echo "  migrate       - Run database migrations"