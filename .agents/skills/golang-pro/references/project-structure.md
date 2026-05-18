# Project Structure and Module Management

## Standard Project Layout

```
myproject/
├── cmd/                    # Main applications
│   ├── server/
│   │   └── main.go        # Entry point for server
│   └── cli/
│       └── main.go        # Entry point for CLI tool
├── internal/              # Private application code
│   ├── api/              # API handlers
│   ├── service/          # Business logic
│   └── repository/       # Data access layer
├── pkg/                   # Public library code
│   └── models/           # Shared models
├── api/                   # API definitions
│   ├── openapi.yaml      # OpenAPI spec
│   └── proto/            # Protocol buffers
├── web/                   # Web assets
├── scripts/               # Build and install scripts
├── configs/              # Configuration files
├── deployments/          # Docker, K8s configs
├── go.mod               # Module definition
├── go.sum               # Dependency checksums
├── Makefile             # Build automation
└── README.md
```

## go.mod Basics

```go
module github.com/user/myproject

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/lib/pq v1.10.9
	go.uber.org/zap v1.26.0
)
```

## Internal Packages

```go
// internal/ packages can only be imported by code in the parent tree
myproject/
├── internal/
│   ├── auth/           # Can only be imported by myproject
│   │   └── jwt.go
│   └── database/
│       └── postgres.go
└── pkg/
    └── models/         # Can be imported by anyone
        └── user.go

// This works (same project):
import "github.com/user/myproject/internal/auth"

// This fails (different project):
import "github.com/other/project/internal/auth" // Error!
```

## Package Organization

```go
// user/user.go - Domain package
package user

import (
	"context"
	"time"
)

// User represents a user entity
type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

// Repository defines data access interface
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
}

// Service handles business logic
type Service struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}
```

## Makefile Example

```makefile
.PHONY: build test lint clean run

BINARY_NAME=myapp
BUILD_DIR=bin
GO=go
GOFLAGS=-v

build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

test:
	$(GO) test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

run:
	$(GO) run ./cmd/server

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
```

## Dockerfile Multi-Stage Build

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

## Configuration Management

```go
// config/config.go
package config

import (
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Host        string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port        int           `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"10s"`
}

type DatabaseConfig struct {
	URL          string `envconfig:"DATABASE_URL" required:"true"`
	MaxOpenConns int    `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
}

// Load loads configuration from environment
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

## Quick Reference

| Command | Description |
|---------|-------------|
| `go mod init` | Initialize module |
| `go mod tidy` | Add/remove dependencies |
| `go build -ldflags "-X ..."` | Set version info |
| `GOOS=linux go build` | Cross-compile |