---
name: go-service-template
description: Build, scaffold, or extend Go web services using the template-service design pattern. Use when creating new Go microservices or adding endpoints to services that follow the kardianos/service daemon lifecycle, interface-decoupled storage (WASM SQLite & MySQL), Gin route wrapping with typed error handling, constant-time auth, and audit logging.
---

# Go Service Template Pattern

Production architecture pattern for Go microservices combining cross-platform OS service lifecycle management, interface-decoupled multi-backend persistence, Gin web routing with centralized error handling, and CGO-free containerization.

## Architectural Layers

```
                                    +------------------------------+
                                    |     main.go (Daemon CLI)     |
                                    |      kardianos/service       |
                                    +--------------+---------------+
                                                   |
                                    +--------------v---------------+
                                    |    service.go (Lifecycle)    |
                                    |     Start / Run / Stop       |
                                    +--------------+---------------+
                                                   |
                        +--------------------------+--------------------------+
                        |                                                     |
             +----------v----------+                               +----------v----------+
             |    store.Store      |                               |      gin.Engine     |
             |     (Interface)     |                               |      (router.go)    |
             +----------+----------+                               +----------+----------+
                        |                                                     |
          +-------------+-------------+                                       |
          |                           |                                       |
+---------v---------+       +---------v---------+                             |
| SQLite (Pure WASM)|       |       MySQL       |                             |
| ncruces/go-sqlite3|       | go-sql-driver/mysql                             |
+-------------------+       +-------------------+                             |
                                                                              |
                                     +----------------------------------------+
                                     |
                       +-------------v-------------+
                       |    Middleware Pipeline    |
                       | - Auth (Constant-time)    |
                       | - Logger (Body buffering) |
                       | - WrapHandler (Errors)    |
                       +-------------+-------------+
                                     |
                       +-------------v-------------+
                       |    handlers.Handler       |
                       |  func(*gin.Context) error |
                       +---------------------------+
```

---

## Key Design Rules

1. **Service Runner**: All services implement `service.Interface` (`Start`, `Stop`). The HTTP server runs asynchronously in a goroutine while `Start` returns immediately. `Stop` triggers graceful shutdown via `srv.Shutdown(ctx)` with a 5-second deadline.
2. **Storage Decoupling**: Business logic, handlers, and middlewares **must only** accept the `store.Store` interface, never concrete driver structs (`*store.Storage`).
3. **Pure Go SQLite**: Use `github.com/ncruces/go-sqlite3` (WASM-based) rather than `mattn/go-sqlite3` to avoid CGO compiler toolchain dependencies.
4. **Error-Returning Handlers**: HTTP handlers return `error` (`HandlerFuncWithError func(*gin.Context) error`). Handlers wrap domain errors with status codes using `httperror.ReturnWithHTTPStatus(err, code)`.
5. **Route Registration**: Routes are declared as declarative data structs (`Route{Name, Method, Pattern, HandlerFunc, UseLogger, UseAuth}`).
6. **Constant-Time Auth**: Token authentication always uses `crypto/subtle.ConstantTimeCompare` to eliminate timing attack vectors.
7. **Asset Duality**: Embedded assets (`embed.FS`) are the default for single-binary portability, with a config toggle (`USE_FILE_SYSTEM`) to load from disk during local frontend iteration.

---

## Core Components

### 1. Storage Interface & Pluggable Backends

Define the storage contract in `store/storage.go`:

```go
package store

import (
	"database/sql"
	"time"
	"myproject/types"
)

type Store interface {
	Ping() error
	GetLogs(from, to time.Time) ([]types.UsageLog, error)
	LogRequest(status int, method, errStr, endpoint, createdAt, response, request string) error
}

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

func (s *Storage) Ping() error {
	return s.db.Ping()
}
```

#### Pure Go SQLite Driver (`store/sqlite.go`)
```go
package store

import (
	"database/sql"
	_ "github.com/ncruces/go-sqlite3/driver"
)

func NewSqliteDatabase(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:"+file)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		status INTEGER,
		method TEXT,
		error TEXT,
		endpoint TEXT,
		created_at DATETIME,
		response TEXT,
		request TEXT
	)`)
	return db, err
}
```

---

### 2. Typed HTTP Error Handling (`httperror/httpError.go`)

Allows handlers to return idiomatic Go errors paired with HTTP status codes:

```go
package httperror

import (
	"fmt"
	"net/http"
)

type statusError struct {
	error
	status int
}

func (e statusError) Unwrap() error { return e.error }
func (e statusError) Error() string  { return fmt.Sprintf("status %d: %v", e.status, e.error) }

func ReturnWithHTTPStatus(err error, status int) error {
	return statusError{error: err, status: status}
}

func HTTPStatus(err error) int {
	if se, ok := err.(statusError); ok {
		return se.status
	}
	return http.StatusInternalServerError
}

func StatusText(err error) string {
	if se, ok := err.(statusError); ok {
		return http.StatusText(se.status)
	}
	return http.StatusText(http.StatusInternalServerError)
}
```

---

### 3. Gin Route Pipeline & Declarative Routing (`router.go`)

```go
package main

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"myproject/handlers"
	"myproject/httperror"
	"myproject/store"
	"myproject/types"
)

type HandlerFuncWithError func(*gin.Context) error

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc HandlerFuncWithError
	UseLogger   bool
	UseAuth     bool
}

type Routes []Route

func NewRouter(handler *handlers.Handler, s store.Store, settings types.AppSettings) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	routes := Routes{
		{
			Name:        "HealthCheck",
			Method:      "GET",
			Pattern:     "/",
			HandlerFunc: handler.HealthCheck,
		},
		{
			Name:        "Ping",
			Method:      "GET",
			Pattern:     "/ping/:argument",
			HandlerFunc: handler.Ping,
			UseLogger:   true,
		},
		{
			Name:        "Pong",
			Method:      "POST",
			Pattern:     "/pong",
			HandlerFunc: handler.Pong,
			UseLogger:   true,
			UseAuth:     true,
		},
	}

	for _, route := range routes {
		fn := route.HandlerFunc
		if route.UseAuth {
			fn = AuthMiddleware(settings.AuthToken)(fn)
		}
		if route.UseLogger {
			fn = LoggerMiddleware(s)(fn)
		}
		router.Handle(route.Method, route.Pattern, WrapHandler(fn))
	}

	return router
}

func WrapHandler(inner HandlerFuncWithError) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Version", Version)
		if err := inner(c); err != nil {
			c.String(httperror.HTTPStatus(err), httperror.StatusText(err))
		}
	}
}
```

---

### 4. Constant-Time Auth Middleware

```go
func AuthMiddleware(authToken string) func(HandlerFuncWithError) HandlerFuncWithError {
	return func(inner HandlerFuncWithError) HandlerFuncWithError {
		return func(c *gin.Context) error {
			if authToken == "" {
				return inner(c)
			}
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				return httperror.ReturnWithHTTPStatus(fmt.Errorf("missing authorization header"), http.StatusUnauthorized)
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(token), []byte(authToken)) != 1 {
				return httperror.ReturnWithHTTPStatus(fmt.Errorf("invalid authorization token"), http.StatusUnauthorized)
			}
			return inner(c)
		}
	}
}
```

---

### 5. Service Daemon Lifecycle (`service.go` & `main.go`)

In `main.go`:
```go
package main

import (
	"flag"
	"log"
	"github.com/kardianos/service"
)

const nameOfService = "my-service"
var Version = "dev"

func main() {
	svcFlag := flag.String("service", "", "Control the system service: install, start, stop, uninstall")
	flag.Parse()

	svcConfig := &service.Config{
		Name:        nameOfService,
		DisplayName: nameOfService,
		Description: "High-performance microservice daemon",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(*svcFlag) != 0 {
		if err := service.Control(s, *svcFlag); err != nil {
			log.Fatalf("Control error: %v", err)
		}
		return
	}

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
```

In `service.go`:
```go
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/kardianos/service"
	"myproject/handlers"
	"myproject/store"
)

type program struct {
	exit chan struct{}
}

func (p *program) Start(s service.Service) error {
	p.exit = make(chan struct{})
	go p.run()
	return nil
}

func (p *program) run() error {
	// 1. Initialize DB Store
	var db *sql.DB
	var err error
	if settings.UseMySQL {
		// connect mysql
	} else {
		db, err = store.NewSqliteDatabase("data.db")
	}
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	storage := store.NewStorage(db)
	handler := handlers.NewHandler(storage, settings.UseFileSystem, tpls, nameOfService, Version)
	router := NewRouter(handler, storage, settings)

	srv := &http.Server{
		Addr:    settings.Port,
		Handler: http.TimeoutHandler(router, time.Duration(settings.Timeout)*time.Second, "Timeout"),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server listen error: %v", err)
		}
	}()

	<-p.exit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

func (p *program) Stop(s service.Service) error {
	close(p.exit)
	return nil
}
```

---

## Workflow: Adding an Endpoint

Follow this procedure to add a new endpoint:

1. **Define Handler**:
   Create or open a file in `handlers/`. Implement signature `func(h *Handler) <Action>(c *gin.Context) error`. Return `httperror.ReturnWithHTTPStatus(err, status)` on failure or `c.JSON(...)` / `nil` on success.
2. **Register Route**:
   In `router.go`, add a new `Route` entry into `getRoutes()` with required flags (`UseAuth: true`, `UseLogger: true`).
3. **Add Tests**:
   Write a table-driven test using `gin.CreateTestContext(httptest.NewRecorder())`. Mock the `store.Store` interface without opening file handles or network sockets.
4. **Verify**:
   Run `go test ./...` and verify routing and middleware execution.

---

## Production Dockerfile Standard

```dockerfile
# Build stage (Zero CGO)
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X 'main.Version=${VERSION}'" \
    -o template-service .

# Run stage
FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup \
    && chown -R appuser:appgroup /app

COPY --from=builder --chown=appuser:appgroup /app/template-service .
COPY --from=builder --chown=appuser:appgroup /app/assets ./assets
COPY --from=builder --chown=appuser:appgroup /app/tmpl ./tmpl

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/ || exit 1

CMD ["./template-service"]
```
