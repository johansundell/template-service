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

// Logger defines a leveled logging interface for router middleware
type Logger interface {
	Infof(format string, v ...interface{})
	Warningf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

type stdLogger struct{}

func (stdLogger) Infof(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (stdLogger) Warningf(format string, v ...interface{}) {
	log.Printf("WARNING: "+format, v...)
}

func (stdLogger) Errorf(format string, v ...interface{}) {
	log.Printf("ERROR: "+format, v...)
}

// LogSink receives request log entries. It must not block; logqueue.Queue
// writes them to the store in the background.
type LogSink interface {
	Enqueue(entry types.UsageLog)
}

// Config contains the dependencies and settings required to construct a router
type Config struct {
	Handler  *handlers.Handler
	LogSink  LogSink // Required when any route has UseLogger
	Settings types.AppSettings
	Assets   fs.FS
	Version  string
	Logger   Logger // Optional: defaults to standard logger when nil
}

// NewRouter creates a new web handler with middleware and registered routes
func NewRouter(cfg Config) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)

	if cfg.Handler == nil {
		return nil, errors.New("handler must be provided")
	}

	router := gin.New()
	router.Use(gin.Recovery())

	routes := GetRoutes(cfg.Handler)

	l := cfg.Logger
	if l == nil {
		l = stdLogger{}
	}

	for _, route := range routes {
		if route.UseAuth && cfg.Settings.AuthToken == "" {
			return nil, fmt.Errorf("AUTH_TOKEN must be configured for route %q", route.Name)
		}
		if route.UseLogger && cfg.LogSink == nil {
			return nil, fmt.Errorf("log sink must be configured for logged route %q", route.Name)
		}

		fn := route.HandlerFunc

		// Apply Logger Middleware first (innermost), so it only runs after auth passes.
		// Wrapping order is inside-out: the last wrapper applied is the first to execute.
		if route.UseLogger {
			fn = LoggerMiddleware(cfg.LogSink, l)(fn)
		}

		// Apply Auth Middleware second (outermost), so it executes first and rejects
		// unauthenticated requests before the logger reads or stores the body.
		if route.UseAuth {
			fn = AuthMiddleware(cfg.Settings.AuthToken, l)(fn)
		}

		router.Handle(route.Method, route.Pattern, WrapHandler(fn, cfg.Version))
	}

	// Static files
	fsys, err := getStaticFiles(cfg.Assets, cfg.Settings.UseFileSystem)
	if err != nil {
		return nil, err
	}
	router.StaticFS("/assets", fsys)

	return router, nil
}

// AuthMiddleware returns a middleware that validates the Authorization header
func AuthMiddleware(authToken string, l Logger) func(HandlerFuncWithError) HandlerFuncWithError {
	if l == nil {
		l = stdLogger{}
	}
	return func(inner HandlerFuncWithError) HandlerFuncWithError {
		return func(c *gin.Context) error {

			if authToken == "" {
				l.Warningf("AUTH_TOKEN is not set")
				return httperror.ReturnWithHTTPStatus(
					errors.New("authentication is not configured"),
					http.StatusInternalServerError,
				)
			}

			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				l.Warningf("unauthorized request: %s %s from %s: missing authorization header", c.Request.Method, c.Request.URL.Path, c.ClientIP())
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
				l.Warningf("unauthorized request: %s %s from %s: invalid authorization token", c.Request.Method, c.Request.URL.Path, c.ClientIP())
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

// maxRequestBodyBytes caps how much of a request body LoggerMiddleware reads
// and stores; larger requests are rejected with 413.
const maxRequestBodyBytes = 1 << 20

// LoggerMiddleware captures each request and response and hands the entry to
// sink; persisting it happens in the background.
func LoggerMiddleware(sink LogSink, l Logger) func(HandlerFuncWithError) HandlerFuncWithError {
	if l == nil {
		l = stdLogger{}
	}
	return func(inner HandlerFuncWithError) HandlerFuncWithError {
		return func(c *gin.Context) error {
			// Read the request body once, capped so large requests can't
			// exhaust memory or bloat the request log
			var requestBody []byte
			if c.Request.Body != nil {
				body := http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
				var readErr error
				requestBody, readErr = io.ReadAll(body)
				body.Close()
				if readErr != nil {
					var tooLarge *http.MaxBytesError
					if errors.As(readErr, &tooLarge) {
						l.Warningf("request body too large: %s %s from %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())
						return httperror.ReturnWithHTTPStatus(readErr, http.StatusRequestEntityTooLarge)
					}
					l.Errorf("failed to read request body: %v", readErr)
					return readErr
				}

				// Reset the request body so it can be read again
				c.Request.Body = io.NopCloser(bytes.NewReader(requestBody))
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
				CreatedAt: time.Now().UTC(),
				Response:  types.RawJSON(blw.body.String()),
				Request:   types.RawJSON(requestBody),
			}

			if len(requestBody) == 0 {
				usageLog.Request = types.RawJSON("{}")
			}

			sink.Enqueue(usageLog)

			return err
		}
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
