.PHONY: build test lint clean run deps docker-up docker-down

BINARY_NAME=wa-server
BUILD_DIR=bin
GO=go
GOFLAGS=-v

build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

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

run:
	$(GO) run ./cmd/server

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

deps:
	$(GO) mod download
	$(GO) mod tidy

docker-up:
	docker compose --env-file .env -f deployments/docker-compose.yml up -d

docker-down:
	docker compose --env-file .env -f deployments/docker-compose.yml down

migrate:
	$(GO) run ./cmd/migrate

help:
	@echo "Available targets:"
	@echo "  build         - Build server binary"
	@echo "  test          - Run tests with race detector"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linters"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  run           - Run server"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  docker-up     - Start Docker services"
	@echo "  docker-down   - Stop Docker services"
	@echo "  migrate       - Run database migrations"