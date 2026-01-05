package main

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/httperror"
	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/types"
	"github.com/johansundell/template-service/utils"
)

type HandlerFuncWithError func(*gin.Context) error

// Route struct for the service
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc HandlerFuncWithError
	UseLogger   bool
	UseAuth     bool
}

// Routes for the servcie web handlers
type Routes []Route

// NewRouter creates a new web handler
func NewRouter(handler *handlers.Handler, s *store.Storage) *gin.Engine {
	gin.SetMode(gin.ReleaseMode) // Set mode before creating the router

	//router := gin.Default()
	router := gin.New()
	router.Use(gin.Recovery())

	routes := getRoutes(handler)

	for _, route := range routes {
		// Apply Auth Middleware
		if route.UseAuth {
			route.HandlerFunc = AuthMiddleware(settings.AuthToken)(route.HandlerFunc)
		}

		// Apply Logger Middleware
		if route.UseLogger {
			route.HandlerFunc = LoggerMiddleware(s)(route.HandlerFunc)
		}

		// Convert to Gin Handler and register
		router.Handle(route.Method, route.Pattern, WrapHandler(route.HandlerFunc))
	}

	// Static files
	router.StaticFS("/assets", getStaticFiles(settings.UseFileSystem))

	return router
}

func getRoutes(handler *handlers.Handler) Routes {
	routes := Routes{
		Route{
			Name:        "HealthCheck",
			Method:      "GET",
			Pattern:     "/",
			HandlerFunc: handler.HealthCheck,
		},
		Route{
			Name:        "Ping",
			Method:      "GET",
			Pattern:     "/ping/:argument",
			HandlerFunc: handler.Ping,
			UseLogger:   true,
		},
		Route{
			Name:        "Pong",
			Method:      "POST",
			Pattern:     "/pong",
			HandlerFunc: handler.Pong,
			UseLogger:   true,
			UseAuth:     true,
		},
		Route{
			Name:        "GetLogs",
			Method:      "GET",
			Pattern:     "/logs/:from/:to",
			HandlerFunc: handler.GetLogsHandler,
			UseAuth:     true,
		},
	}
	return routes
}

// checkAuthHeader validates the Authorization header against the configured auth token
// AuthMiddleware returns a middleware that validates the Authorization header
func AuthMiddleware(authToken string) func(HandlerFuncWithError) HandlerFuncWithError {
	return func(inner HandlerFuncWithError) HandlerFuncWithError {
		return func(c *gin.Context) error {
			if authToken == "" {
				// If no auth token is configured, skip authentication
				return inner(c)
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

func getStaticFiles(useLocal bool) http.FileSystem {
	if useLocal {
		return http.FS(os.DirFS("assets"))
	}

	fsys, err := fs.Sub(embededFiles, "assets")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

func WrapHandler(inner HandlerFuncWithError) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Version", Version)
		if err := inner(c); err != nil {
			c.String(httperror.HTTPStatus(err), httperror.StatusText(err))
		}
	}
}

func LoggerMiddleware(s *store.Storage) func(HandlerFuncWithError) HandlerFuncWithError {
	return func(inner HandlerFuncWithError) HandlerFuncWithError {
		return func(c *gin.Context) error {
			// Read the request body once
			var requestBody []byte
			if c.Request.Body != nil {
				requestBody, _ = io.ReadAll(c.Request.Body)
				c.Request.Body.Close()

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

			log := types.UsageLog{
				Status:    status,
				Method:    c.Request.Method,
				Error:     errMsg,
				Endpoint:  utils.GetUrl(c.Request, c.Request.URL.Path),
				CreatedAt: time.Now(),
				Response:  types.RawJSON(blw.body.String()),
				Request:   types.RawJSON(requestBody),
			}

			if len(requestBody) == 0 {
				log.Request = types.RawJSON("{}")
			}

			if err != nil {
				fmt.Println("Logging error:", log)
			} else {
				fmt.Println("Logging success:", log)
			}
			s.LogRequest(log.Status, log.Method, log.Error, log.Endpoint, log.CreatedAt.Format(time.RFC3339), string(log.Response), string(log.Request))

			return err
		}
	}
}

// Add this struct at the end of the file
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
