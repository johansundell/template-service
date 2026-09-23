# Code Review: `master...HEAD` (`router-module`)

**Review Scope:** Commits `master..HEAD`
- `9878acd` docs: update SKILL.md to reference router/routes.go
- `6bec63a` feat(router): extract router into dedicated package

---

## Standards

### Documented Repo Standards Compliance ([`.agents/skills/go-service-template/SKILL.md`](.agents/skills/go-service-template/SKILL.md))
- **Hard Violations**: **None (PASS)**.
  - **Rule 1 (Service lifecycle) & Rule 8 (Ownership)**: [`service.go`](service.go) preserves synchronous startup and 5-second graceful shutdown semantics; constructor errors close handles properly; no `log.Fatal` in package code.
  - **Rule 2 (Storage boundary)**: Router middleware and configuration depend strictly on [`store.Store`](store/storage.go).
  - **Rule 4 (Error handlers)**: Middleware uses [`httperror.ReturnWithHTTPStatus`](httperror/httpError.go) for HTTP error mapping.
  - **Rule 5 (Routes)**: Default routes are declared in [`router.GetRoutes(handler)`](router/routes.go) and registered via [`router.NewRouter(cfg)`](router/router.go). Documentation in [`.agents/skills/go-service-template/SKILL.md`](.agents/skills/go-service-template/SKILL.md) is fully synchronized.
  - **Rule 6 (Authentication)**: Startup fails fast if protected routes lack `AuthToken`; [`AuthMiddleware`](router/router.go) uses `subtle.ConstantTimeCompare` and fails closed (HTTP 500 when unconfigured).
  - **Rule 7 (Assets)**: Local asset resolution in [`router/router.go`](router/router.go) now properly uses `filepath.Join(utils.GetBinaryBasePath(), "assets")`.

### Baseline Smells (Judgement Calls)
- **Speculative Generality** — [`router/router.go`](router/router.go):
  ```go
  type Config struct {
      ...
      Routes Routes // Optional: defaults to GetRoutes(Handler) when empty
  }
  ...
  routes := cfg.Routes
  if len(routes) == 0 && cfg.Handler != nil {
      routes = GetRoutes(cfg.Handler)
  }
  ```
  Rule 5 specifies declaring routes in `router.GetRoutes(handler)`. In production (`service.go`), `Config.Routes` is unused. The field serves as an abstraction hook only used by `TestCustomRoutes`.
- **Duplicated Code** — [`router/router_test.go`](router/router_test.go):
  Identical request building, body buffering, and recorder setup are repeated across 4 subtests in `TestAuthCheck` instead of using a table-driven test or test helper.

---

## Spec

### (a) Missing or Partial Requirements
- **Logger Dependency Injection**:
  - *Spec Quote:* `"clean dependency injection (version, assets, logger) and no package main globals"`
  - *Finding:* [`router.Config`](router/router.go) injects `Version` and `Assets`, but no `Logger` field was added. Instead, [`router/router.go`](router/router.go) relies on standard `log.Printf` inside `LoggerMiddleware` and removed logging from `AuthMiddleware`.
- **Missing Handler Validation on Route Defaulting**:
  - *Spec Quote:* `"Uses cfg.Routes or defaults to GetRoutes(cfg.Handler)."`
  - *Finding:* In [`router/router.go`](router/router.go), if `cfg.Routes` is omitted and `cfg.Handler` is `nil`, `NewRouter` silently initializes an engine with 0 routes registered rather than validating that either a handler or explicit routes were supplied.

### (b) Behaviour Not Asked For (Scope Creep)
- **SQLite Database Path Refactoring in [`service.go`](service.go)**:
  - *Spec Quote:* `"Update NewRouter call at lines 124–133"`
  - *Finding:* SQLite file resolution was refactored to `filepath.Join(utils.GetBinaryBasePath(), "test.db")`. While fixing a previous review smell, touching SQLite initialization in `service.go` was outside the scope of extracting the router module.

### (c) Implementation Looks Wrong
- **Silent Suppression of Route Logging on Nil Store**:
  - *Spec Quote:* `"- LoggerMiddleware(s store.Store) func(HandlerFuncWithError) HandlerFuncWithError: Request/response logging and persistence."`
  - *Finding:* In [`router/router.go`](router/router.go), `if route.UseLogger && cfg.Store != nil` silently skips logging when `route.UseLogger` is `true` if `cfg.Store` was not provided, masking configuration mistakes rather than returning a validation error.
- **Silent Skip of Static Asset Mount**:
  - *Spec Quote:* `"- Mounts static files from cfg.Assets (or filesystem directory relative to utils.GetBinaryBasePath())."`
  - *Finding:* In [`router/router.go`](router/router.go), static route mounting is guarded by `if cfg.Assets != nil || cfg.Settings.UseFileSystem`. When `cfg.Assets` is `nil` in embedded mode, `/assets` is skipped without error, silently diverging from expected service endpoint availability.

---

## Summary

- **Standards**: 0 hard violations, 2 baseline smells (worst: speculative generality with `Routes` override on `Config`).
- **Spec**: 2 missing/partial requirements, 1 scope creep item, 2 implementation subtleties (worst: silent omission of logging when `Store == nil` despite `route.UseLogger: true`).
