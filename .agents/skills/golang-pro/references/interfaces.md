# Interface Design and Composition

## Small, Focused Interfaces

```go
// Single-method interfaces (idiomatic Go)
type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type Closer interface {
	Close() error
}

// Interface composition
type ReadCloser interface {
	Reader
	Closer
}

type WriteCloser interface {
	Writer
	Closer
}

type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}
```

## Accept Interfaces, Return Structs

```go
package storage

import "io"

// Storage is the concrete type (struct)
type Storage struct {
	baseDir string
}

// NewStorage returns a concrete type
func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

// SaveFile accepts an interface for flexibility
func (s *Storage) SaveFile(filename string, data io.Reader) error {
	// Implementation can work with any Reader
	// (file, network, buffer, etc.)
	return nil
}

// Usage allows dependency injection
type Uploader interface {
	SaveFile(filename string, data io.Reader) error
}

type Service struct {
	uploader Uploader // Accept interface
}

// NewService accepts interface for testing flexibility
func NewService(uploader Uploader) *Service {
	return &Service{uploader: uploader}
}
```

## Embedding for Composition

```go
import "sync"

// Embed to extend behavior
type SafeCounter struct {
	mu sync.Mutex
	m  map[string]int
}

func (sc *SafeCounter) Inc(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.m[key]++
}

// Embed interface to add default behavior
type Logger interface {
	Log(msg string)
}

type NoOpLogger struct{}

func (NoOpLogger) Log(msg string) {}

type Service struct {
	Logger // Embedded interface (default implementation can be provided)
}

func NewService(logger Logger) *Service {
	if logger == nil {
		logger = NoOpLogger{} // Provide default
	}
	return &Service{Logger: logger}
}

// Now Service.Log() is available
```

## Interface Segregation

```go
// Bad: Fat interface
type BadRepository interface {
	Create(item Item) error
	Read(id string) (Item, error)
	Update(item Item) error
	Delete(id string) error
	List() ([]Item, error)
	Search(query string) ([]Item, error)
	Count() (int, error)
}

// Good: Segregated interfaces
type Creator interface {
	Create(item Item) error
}

type Reader interface {
	Read(id string) (Item, error)
}

type Updater interface {
	Update(item Item) error
}

type Deleter interface {
	Delete(id string) error
}

type Lister interface {
	List() ([]Item, error)
}

// Compose only what you need
type ReadWriter interface {
	Reader
	Creator
}
```

## Functional Options Pattern

```go
package server

import "time"

type Server struct {
	host         string
	port         int
	timeout      time.Duration
	maxConns     int
	enableLogger bool
}

// Option is a functional option for configuring Server
type Option func(*Server)

func WithHost(host string) Option {
	return func(s *Server) {
		s.host = host
	}
}

func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.timeout = timeout
	}
}

func WithMaxConnections(max int) Option {
	return func(s *Server) {
		s.maxConns = max
	}
}

func WithLogger(enabled bool) Option {
	return func(s *Server) {
		s.enableLogger = enabled
	}
}

// NewServer creates a server with functional options
func NewServer(opts ...Option) *Server {
	// Defaults
	s := &Server{
		host:     "localhost",
		port:     8080,
		timeout:  30 * time.Second,
		maxConns: 100,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}
```

## Dependency Injection via Interfaces

```go
package app

import "context"

// Define interfaces for dependencies
type UserRepository interface {
	GetUser(ctx context.Context, id string) (*User, error)
	SaveUser(ctx context.Context, user *User) error
}

type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// Service depends on interfaces
type UserService struct {
	repo   UserRepository
	mailer EmailSender
}

func NewUserService(repo UserRepository, mailer EmailSender) *UserService {
	return &UserService{
		repo:   repo,
		mailer: mailer,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, email string) error {
	user := &User{Email: email}
	if err := s.repo.SaveUser(ctx, user); err != nil {
		return err
	}
	return s.mailer.SendEmail(ctx, email, "Welcome", "Thanks for registering!")
}

// Easy to mock in tests
type MockUserRepository struct{}

func (m *MockUserRepository) GetUser(ctx context.Context, id string) (*User, error) {
	return &User{ID: id}, nil
}

func (m *MockUserRepository) SaveUser(ctx context.Context, user *User) error {
	return nil
}
```

## Quick Reference

| Pattern | Use Case | Key Principle |
|---------|----------|---------------|
| Small interfaces | Flexibility | Single-method interfaces |
| Accept interfaces | Testability | Depend on abstractions |
| Return structs | Clarity | Concrete return types |
| io.Reader/Writer | I/O operations | Standard library integration |
| Embedding | Composition | Extend behavior without inheritance |
| Functional options | Configuration | Flexible constructors |