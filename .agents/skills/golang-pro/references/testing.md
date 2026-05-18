# Testing Patterns

## Table-Driven Tests

```go
package calculator

import "testing"

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive", 1, 2, 3},
		{"negative", -1, -2, -3},
		{"mixed", -1, 2, 1},
		{"zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
```

## Benchmarking

```go
package benchmark

import "testing"

func BenchmarkStringConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := ""
		for j := 0; j < 100; j++ {
			s += "a"
		}
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		for j := 0; j < 100; j++ {
			sb.WriteString("a")
		}
		_ = sb.String()
	}
}
```

## Running Tests

```bash
# Run all tests with race detector
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run specific test
go test -run TestAdd ./...

# Run tests with verbose output
go test -v ./...
```

## Test Helper Functions

```go
package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func SetupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "...")
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func SeedUsers(t *testing.T, db *sql.DB, users []User) {
	for _, user := range users {
		_, err := db.Exec(...)
		require.NoError(t, err)
	}
}
```

## Quick Reference

| Command | Description |
|---------|-------------|
| `go test -race` | Run with race detector |
| `go test -cover` | Show coverage |
| `go test -bench=.` | Run benchmarks |
| `go test -v` | Verbose output |
| `go test -run` | Run specific test |
| `go test -subtest` | Run subtest |