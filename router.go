package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
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
	IsAPICall   bool
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
		handlerFunc := handlerWithLogger(route.HandlerFunc, route.IsAPICall, s)
		router.Handle(route.Method, route.Pattern, handlerFunc)
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
			IsAPICall:   true,
		},
	}
	return routes
}

// TODO: Add auth header check function calls here

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

func handlerWithLogger(inner HandlerFuncWithError, logUsage bool, s *store.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Version", Version)

		// Read the request body once
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			//fmt.Println("Request body:", string(requestBody))
			// Reset the request body so it can be read again
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Wrap the original ResponseWriter with our Gin-compatible wrapper
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		if err := inner(c); err != nil {
			if logUsage {
				log := types.UsageLog{
					Status:    httperror.HTTPStatus(err),
					Method:    c.Request.Method,
					Error:     err.Error(),
					Endpoint:  utils.GetUrl(c.Request, c.Request.URL.Path),
					CreatedAt: time.Now(),
					Response:  types.RawJSON("{}"),
					Request:   types.RawJSON(requestBody),
				}
				if len(requestBody) == 0 {
					log.Request = types.RawJSON("{}")
				} else {
					log.Request = types.RawJSON(requestBody)
				}
				fmt.Println("Logging error:", log)
				s.LogRequest(log.Status, log.Method, log.Error, log.Endpoint, log.CreatedAt.Format(time.RFC3339), string(log.Response), string(log.Request))
			}
			c.String(httperror.HTTPStatus(err), httperror.StatusText(err))
		} else {
			// Capture the response body and status code
			if logUsage {
				responseBody := blw.body.String()
				statusCode := c.Writer.Status()

				log := types.UsageLog{
					Status:    statusCode,
					Method:    c.Request.Method,
					Error:     "",
					Endpoint:  utils.GetUrl(c.Request, c.Request.URL.Path),
					CreatedAt: time.Now(),
					Response:  types.RawJSON(responseBody),
					Request:   types.RawJSON(requestBody),
				}
				fmt.Println("Logging success:", log)
				s.LogRequest(log.Status, log.Method, log.Error, log.Endpoint, log.CreatedAt.Format(time.RFC3339), string(log.Response), string(log.Request))
			}
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
