---
name: go-service-template
description: Build, scaffold, or extend Go web services using the template-service design pattern. Use when creating new Go microservices or adding endpoints that follow kardianos/service lifecycle management, interface-decoupled SQLite/MySQL storage, Gin routes with typed errors, constant-time authentication, audit logging, and embedded or filesystem assets.
---

# Go Service Template Pattern

Use the repository implementation as the source of truth. The important boundaries are:

```text
                    +-----------------------------+
                    |    main.go (Daemon CLI)     |
                    |      kardianos/service      |
                    +--------------+--------------+
                                   |
                    +--------------v--------------+
                    |   service.go (Lifecycle)    |
                    |     Start / Run / Stop      |
                    +--------------+--------------+
                                   |
                +------------------+------------------+
                |                                     |
     +----------v----------+           +--------------v--------------+
     |     store.Store     |           |    gin.Engine (router.go)   |
     |     (Interface)     |           +--------------+--------------+
     +----------+----------+                          |
                |                      +--------------v--------------+
          +-----+-----+                |     Middleware Pipeline     |
          |           |                |   Auth -> Logger -> Wrap    |
     +----v----+ +----v----+           +--------------+--------------+
     | SQLite  | |  MySQL  |                          |
     | Pure Go | | Driver  |           +--------------v--------------+
     +---------+ +---------+           |      handlers.Handler       |
                                       |  func(*gin.Context) error   |
                                       +-----------------------------+
```

## Rules

1. **Service lifecycle**: `Start` validates configuration and initializes storage, routing, and the listener before returning startup success. Serving runs asynchronously. `Stop` triggers graceful shutdown with a five-second deadline.
2. **Storage boundary**: Handlers and middleware depend on `store.Store`, not concrete database types.
3. **SQLite**: Use `github.com/ncruces/go-sqlite3` to keep builds CGO-free.
4. **Error handlers**: Handlers return `error` and use `httperror.ReturnWithHTTPStatus` for HTTP failures.
5. **Routes**: Declare routes as `Route` values in `getRoutes(handler)`; `NewRouter` applies middleware and registers them.
6. **Authentication**: Protected routes fail closed. An empty `AUTH_TOKEN` must cause startup validation to fail when any route has `UseAuth: true`. Compare tokens with `subtle.ConstantTimeCompare`.
7. **Assets**: Embedded assets are the default. Filesystem mode loads assets and templates from paths relative to the executable directory.
8. **Ownership**: Close database handles on constructor failure and service shutdown. Do not use `log.Fatal` in reusable service or library code.

## Storage

Keep the storage contract small and interface-based:

```go
type Store interface {
    Ping() error
    GetLogs(from, to time.Time) ([]types.UsageLog, error)
    LogRequest(status int, method, errStr, endpoint, createdAt, response, request string) error
}
```

Constructors should close an opened handle if schema creation fails:

```go
func NewSqliteDatabase(file string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", "file:"+file)
    if err != nil {
        return nil, err
    }
    if _, err = db.Exec(createRequestLogsTable); err != nil {
        _ = db.Close()
        return nil, err
    }
    return db, nil
}
```

The service owns a successfully returned `*sql.DB` and should `defer db.Close()` immediately after construction, before pinging or building the router.

## Typed HTTP Errors

`statusError` implements `Unwrap`, so status extraction must use `errors.As`. A direct type assertion loses the status after idiomatic `%w` wrapping:

```go
func HTTPStatus(err error) int {
    var statusErr statusError
    if errors.As(err, &statusErr) {
        return statusErr.status
    }
    return http.StatusInternalServerError
}

func StatusText(err error) string {
    var statusErr statusError
    if errors.As(err, &statusErr) {
        return http.StatusText(statusErr.status)
    }
    return http.StatusText(http.StatusInternalServerError)
}
```

Test both direct and wrapped errors:

```go
baseErr := errors.New("user not found")
err := fmt.Errorf("load user: %w", ReturnWithHTTPStatus(baseErr, http.StatusNotFound))
```

## Routes and Middleware

The route collection is the extension point:

```go
func getRoutes(handler *handlers.Handler) Routes {
    return Routes{
        {
            Name: "HealthCheck", Method: "GET", Pattern: "/",
            HandlerFunc: handler.HealthCheck,
        },
        {
            Name: "Example", Method: "GET", Pattern: "/example",
            HandlerFunc: handler.Example,
            UseAuth: true,
            UseLogger: true,
        },
    }
}
```

`NewRouter` should obtain routes from `getRoutes(handler)`, validate that every `UseAuth` route has a token, apply authentication outside the logger, then register `WrapHandler`:

```go
for _, route := range getRoutes(handler) {
    if route.UseAuth && settings.AuthToken == "" {
        return nil, fmt.Errorf("AUTH_TOKEN must be set for route %q", route.Name)
    }

    fn := route.HandlerFunc
    if route.UseLogger {
        fn = LoggerMiddleware(s)(fn)
    }
    if route.UseAuth {
        fn = AuthMiddleware(settings.AuthToken)(fn)
    }
    router.Handle(route.Method, route.Pattern, WrapHandler(fn))
}
```

The wrapper converts returned errors into HTTP responses:

```go
func WrapHandler(inner HandlerFuncWithError) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Version", Version)
        if err := inner(c); err != nil {
            c.String(httperror.HTTPStatus(err), httperror.StatusText(err))
        }
    }
}
```

### Authentication

Middleware must never turn a protected route into a public route because configuration is missing. Startup validation is the primary guard; the middleware should also fail closed if called directly:

```go
if authToken == "" {
    return httperror.ReturnWithHTTPStatus(
        errors.New("authentication is not configured"),
        http.StatusInternalServerError,
    )
}
```

Accept the configured authorization format consistently, reject missing or invalid credentials with 401, and use `subtle.ConstantTimeCompare` for the final comparison.

### LoggerMiddleware

The logger middleware must restore the request body after reading it so JSON binding still works, and wrap Gin's writer to capture response bytes:

```go
requestBody, err := io.ReadAll(c.Request.Body)
if err != nil {
    return err
}
c.Request.Body = io.NopCloser(bytes.NewReader(requestBody))

writer := &bodyLogWriter{
    ResponseWriter: c.Writer,
    body:            bytes.NewBuffer(nil),
}
c.Writer = writer

handlerErr := inner(c)
// Persist requestBody, writer.body, status, endpoint, and handlerErr.
return handlerErr
```

Apply auth outside the logger when unauthorized request bodies should not be persisted. Persistence failures should be logged without replacing the handler's response error.

## Service Lifecycle

The service worker must report initialization failures to `Start` instead of silently logging them:

```go
func (p *program) Start(s service.Service) error {
    loadSettings()
    if err := settings.Validate(); err != nil {
        return err
    }

    p.exit = make(chan struct{})
    startup := make(chan error, 1)
    go p.run(startup)
    return <-startup
}

func (p *program) run(startup chan<- error) error {
    db, err := openConfiguredDatabase(settings) // repository SQLite/MySQL constructors
    if err != nil {
        startup <- err
        return err
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        startup <- err
        return err
    }

    // Build handler/router, bind the listener, then send startup <- nil.
    // Serve asynchronously, wait for p.exit, and call srv.Shutdown(ctx).
    return nil
}
```

`openConfiguredDatabase` is a descriptive placeholder, not a repository API. The real implementation must propagate database, router, and listener errors, close resources on every exit path, and avoid `log.Fatal` in the worker.

## Adding an Endpoint

1. Create or update a handler in `handlers/` with signature `func (h *Handler) Action(c *gin.Context) error`.
2. Return `httperror.ReturnWithHTTPStatus` for expected HTTP failures.
3. Add a `Route` entry to `getRoutes(handler)` and choose `UseAuth` and `UseLogger` explicitly.
4. Add focused handler and route tests. Mock `store.Store` where possible.
5. Run `go test ./...` and `go build ./...`.

## Filesystem Assets and Templates

When filesystem mode is enabled, package the `assets` and `tmpl` directories beside the compiled binary. Resolve paths from `utils.GetBinaryBasePath()` rather than the process working directory. Embedded mode uses `embed.FS` and remains portable as a single binary.

## Docker Standard

Use a multi-stage CGO-free build. Copy the binary, `assets`, and `tmpl` into the same runtime directory, run as a non-root user, and expose the configured HTTP port. Keep the image health check pointed at the public health endpoint.
