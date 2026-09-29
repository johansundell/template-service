---
name: go-service-template
description: Square Moon template-service for Go daemons - kardianos/service lifecycle, store.Store with WASM SQLite/MySQL, Gin routes with error-returning handlers, embedded or filesystem assets. Use when working in a repo that already follows template-service, or when explicitly asked to scaffold a new Square Moon Go service. Not general Go or general Gin guidance.
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
     |     store.Store     |           |          gin.Engine         |
     |     (Interface)     |           |      (router/router.go)     |
     +----------+----------+           +--------------+--------------+
                |                                     |
                |                      +--------------v--------------+
                |                      |     Middleware Pipeline     |
                |                      |   Auth -> Logger -> Wrap    |
                |                      +--------------+--------------+
          +-----+-----+                               |
          |           |                +--------------v--------------+
     +----v----+ +----v----+           |      handlers.Handler       |
     | SQLite  | |  MySQL  |           |  func(*gin.Context) error   |
     | Pure Go | | Driver  |           +-----------------------------+
     +---------+ +---------+
```

## Rules

1. **Service lifecycle**: `Start` validates configuration and initializes storage, routing, and the listener before returning startup success. Serving runs asynchronously. `Stop` triggers graceful shutdown with a five-second deadline.
2. **Storage boundary**: Handlers and middleware depend on `store.Store`, not concrete database types.
3. **SQLite**: Use `github.com/ncruces/go-sqlite3` to keep builds CGO-free.
4. **Error handlers**: Handlers return `error` and use `httperror.ReturnWithHTTPStatus` for HTTP failures.
5. **Routes**: Declare routes as `Route` values in `router.GetRoutes(handler)`; `router.NewRouter(cfg)` applies middleware and registers them.
6. **Authentication**: Protected routes fail closed. `router.NewRouter` rejects an empty `AUTH_TOKEN` when any route has `UseAuth: true`. When `AUTH_TOKEN` is unset, the service generates a random temporary token and logs it before building the router; real deployments must set `AUTH_TOKEN`. Compare tokens with `subtle.ConstantTimeCompare`.
7. **Assets**: Embedded assets are the default and require non-nil `cfg.Assets`. Filesystem mode loads assets and templates from paths relative to the executable directory.
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

Constructors should close an opened handle if schema creation fails. Set SQLite pragmas in the DSN, not with `db.Exec`: the driver then applies them to every pooled connection, and a bad pragma surfaces as an error on the first statement:

```go
func NewSqliteDatabase(file string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", "file:"+file+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
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

`router.Config` encapsulates dependencies and settings required to construct the router:

```go
type Config struct {
    Handler  *handlers.Handler
    Store    store.Store
    Settings types.AppSettings
    Assets   fs.FS
    Version  string
    Logger   Logger // Optional: defaults to standard logger when nil
}
```

Middleware logs through the leveled `router.Logger` interface. The service passes an adapter over the `kardianos/service` logger, so errors reach the system log at the right level:

```go
type Logger interface {
    Infof(format string, v ...interface{})
    Warningf(format string, v ...interface{})
    Errorf(format string, v ...interface{})
}
```

The route collection in `router/routes.go` is the extension point:

```go
func GetRoutes(handler *handlers.Handler) Routes {
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

`router.NewRouter(cfg)` validates that `cfg.Handler` is provided, obtains routes via `router.GetRoutes(cfg.Handler)`, validates that every `UseAuth` route has a token and every `UseLogger` route has a store (returning an error instead of registering an unsafe route), applies authentication outside the logger, then registers `WrapHandler`:

```go
if cfg.Handler == nil {
    return nil, errors.New("handler must be provided")
}

routes := GetRoutes(cfg.Handler)

for _, route := range routes {
    if route.UseAuth && cfg.Settings.AuthToken == "" {
        return nil, fmt.Errorf("AUTH_TOKEN must be configured for route %q", route.Name)
    }
    if route.UseLogger && cfg.Store == nil {
        return nil, fmt.Errorf("store must be configured for logged route %q", route.Name)
    }

    fn := route.HandlerFunc
    if route.UseLogger {
        fn = LoggerMiddleware(cfg.Store, l)(fn)
    }
    if route.UseAuth {
        fn = AuthMiddleware(cfg.Settings.AuthToken, l)(fn)
    }
    router.Handle(route.Method, route.Pattern, WrapHandler(fn, cfg.Version))
}
```

The wrapper converts returned errors into HTTP responses and sets the `X-Version` header:

```go
func WrapHandler(inner HandlerFuncWithError, version string) gin.HandlerFunc {
    return func(c *gin.Context) {
        if version != "" {
            c.Header("X-Version", version)
        }
        if err := inner(c); err != nil {
            c.String(httperror.HTTPStatus(err), httperror.StatusText(err))
        }
    }
}
```

### Authentication

Middleware must never turn a protected route into a public route because configuration is missing. The service guarantees a token before building the router (`ensureAuthToken`), and `NewRouter` rejects protected routes without one; the middleware should also fail closed if called directly:

```go
if authToken == "" {
    return httperror.ReturnWithHTTPStatus(
        errors.New("authentication is not configured"),
        http.StatusInternalServerError,
    )
}
```

Accept the configured authorization format consistently, reject missing or invalid credentials with 401, and use `subtle.ConstantTimeCompare` for the final comparison. Log rejected requests through the injected logger without the token value.

### LoggerMiddleware

The logger middleware must restore the request body after reading it so JSON binding still works, and wrap Gin's writer to capture response bytes. A failed body read is logged, not returned, so request logging never changes the response:

```go
requestBody, readErr := io.ReadAll(c.Request.Body)
c.Request.Body.Close()
if readErr != nil {
    l.Errorf("failed to read request body: %v", readErr)
}
c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

writer := &bodyLogWriter{
    ResponseWriter: c.Writer,
    body:            bytes.NewBuffer(nil),
}
c.Writer = writer

handlerErr := inner(c)
// Persist requestBody, writer.body, status, endpoint, and handlerErr.
return handlerErr
```

Apply auth outside the logger when unauthorized request bodies should not be persisted. Persistence failures should be logged with `l.Errorf` without replacing the handler's response error.

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
    ensureAuthToken() // random temporary token when AUTH_TOKEN is unset

    // Build handler/router, bind the listener, then send startup <- nil.
    // Serve asynchronously, wait for p.exit, and call srv.Shutdown(ctx).
    return nil
}
```

`openConfiguredDatabase` is a descriptive placeholder, not a repository API. The real implementation must propagate database, router, and listener errors, close resources on every exit path, and avoid `log.Fatal` in the worker.

## Adding an Endpoint

1. Create or update a handler in `handlers/` with signature `func (h *Handler) Action(c *gin.Context) error`.
2. Return `httperror.ReturnWithHTTPStatus` for expected HTTP failures.
3. Add a `Route` entry to `router.GetRoutes(handler)` in `router/routes.go` and choose `UseAuth` and `UseLogger` explicitly.
4. Add focused handler and route tests. Mock `store.Store` where possible.
5. Run `go test ./...` and `go build ./...`.

## Filesystem Assets and Templates

When filesystem mode is enabled, package the `assets` and `tmpl` directories beside the compiled binary. Resolve paths from `utils.GetBinaryBasePath()` rather than the process working directory. Embedded mode uses `embed.FS` and remains portable as a single binary.

## Docker Standard

Use a multi-stage CGO-free build. Copy the binary, `assets`, and `tmpl` into the same runtime directory, run as a non-root user, and expose the configured HTTP port. Keep the image health check pointed at the public health endpoint. When deploying with SQLite, mount a directory (e.g. `./data:/app/data`) for `SQLITE_PATH` so WAL and SHM files persist.
