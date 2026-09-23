package router

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/httperror"
	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/types"
	"github.com/johansundell/template-service/utils"
)

// HandlerFuncWithError defines a handler function that returns an error
type HandlerFuncWithError func(*gin.Context) error

// Route defines the configuration for a single HTTP endpoint
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc HandlerFuncWithError
	UseLogger   bool
	UseAuth     bool
}

// Routes is a collection of Route definitions
type Routes []Route

// Config contains the dependencies and settings required to construct a router
type Config struct {
	Handler  *handlers.Handler
	Store    store.Store
	Settings types.AppSettings
	Assets   fs.FS
	Version  string
	Routes   Routes // Optional: defaults to GetRoutes(Handler) when empty
}

// NewRouter creates a new web handler with middleware and registered routes
func NewRouter(cfg Config) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())

	routes := cfg.Routes
	if len(routes) == 0 && cfg.Handler != nil {
		routes = GetRoutes(cfg.Handler)
	}

	for _, route := range routes {
		if route.UseAuth && cfg.Settings.AuthToken == "" {
			return nil, fmt.Errorf("AUTH_TOKEN must be configured for route %q", route.Name)
		}

		fn := route.HandlerFunc

		// Apply Logger Middleware first (innermost), so it only runs after auth passes.
		// Wrapping order is inside-out: the last wrapper applied is the first to execute.
		if route.UseLogger && cfg.Store != nil {
			fn = LoggerMiddleware(cfg.Store)(fn)
		}

		// Apply Auth Middleware second (outermost), so it executes first and rejects
		// unauthenticated requests before the logger reads or stores the body.
		if route.UseAuth {
			fn = AuthMiddleware(cfg.Store, cfg.Settings.AuthToken)(fn)
		}

		router.Handle(route.Method, route.Pattern, WrapHandler(fn, cfg.Version))
	}

	// Static files
	if cfg.Assets != nil || cfg.Settings.UseFileSystem {
		fsys, err := getStaticFiles(cfg.Assets, cfg.Settings.UseFileSystem)
		if err != nil {
			return nil, err
		}
		router.StaticFS("/assets", fsys)
	}

	return router, nil
}

// AuthMiddleware returns a middleware that validates the Authorization header
func AuthMiddleware(s store.Store, authToken string) func(HandlerFuncWithError) HandlerFuncWithError {
	return func(inner HandlerFuncWithError) HandlerFuncWithError {
		return func(c *gin.Context) error {

			if authToken == "" {
				return httperror.ReturnWithHTTPStatus(
					errors.New("authentication is not configured"),
					http.StatusInternalServerError,
				)
			}

			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				return httperror.ReturnWithHTTPStatus(
					fmt.Errorf("missing authorization header"),
					http.StatusUnauthorized,
				)
			}

			// Support both "Bearer <token>" and plain "<token>" formats
			var token string
			if strings.HasPrefix(authHeader, "Bearer ") && len(authHeader) > 7 {
				token = authHeader[7:]
			} else {
				token = authHeader
			}

			// Use constant time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(token), []byte(authToken)) != 1 {
				return httperror.ReturnWithHTTPStatus(
					fmt.Errorf("invalid authorization token"),
					http.StatusUnauthorized,
				)
			}
			return inner(c)
		}
	}
}

func getStaticFiles(assets fs.FS, useLocal bool) (http.FileSystem, error) {
	if useLocal {
		assetDir := filepath.Join(utils.GetBinaryBasePath(), "assets")
		return http.FS(os.DirFS(assetDir)), nil
	}

	if assets == nil {
		return nil, errors.New("embedded assets filesystem is nil")
	}

	fsys, err := fs.Sub(assets, "assets")
	if err != nil {
		return nil, err
	}
	return http.FS(fsys), nil
}

// WrapHandler wraps a HandlerFuncWithError into a Gin HandlerFunc and injects X-Version
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

// LoggerMiddleware logs requests and responses using the provided Store
func LoggerMiddleware(s store.Store) func(HandlerFuncWithError) HandlerFuncWithError {
	return func(inner HandlerFuncWithError) HandlerFuncWithError {
		return func(c *gin.Context) error {
			// Read the request body once
			var requestBody []byte
			if c.Request.Body != nil {
				var readErr error
				requestBody, readErr = io.ReadAll(c.Request.Body)
				c.Request.Body.Close()
				if readErr != nil {
					log.Printf("failed to read request body: %v", readErr)
				}

				// Reset the request body so it can be read again
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// Wrap the original ResponseWriter with our Gin-compatible wrapper
			blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
			c.Writer = blw

			err := inner(c)

			// Log the request/response
			var status int
			var errMsg string
			if err != nil {
				status = httperror.HTTPStatus(err)
				errMsg = err.Error()
			} else {
				status = c.Writer.Status()
				errMsg = ""
			}

			usageLog := types.UsageLog{
				Status:    status,
				Method:    c.Request.Method,
				Error:     errMsg,
				Endpoint:  utils.GetUrl(c.Request, c.Request.URL.Path),
				CreatedAt: time.Now(),
				Response:  types.RawJSON(blw.body.String()),
				Request:   types.RawJSON(requestBody),
			}

			if len(requestBody) == 0 {
				usageLog.Request = types.RawJSON("{}")
			}

			// Persist request log; if persistence fails, record the error to std log
			if persistErr := s.LogRequest(usageLog.Status, usageLog.Method, usageLog.Error, usageLog.Endpoint, usageLog.CreatedAt.Format(time.RFC3339), string(usageLog.Response), string(usageLog.Request)); persistErr != nil {
				log.Printf("failed to persist request log: %v", persistErr)
			} else {
				log.Printf("request logged: %s %s %d", usageLog.Method, usageLog.Endpoint, usageLog.Status)
			}

			return err
		}
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
