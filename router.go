package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/httperror"
	"github.com/johansundell/template-service/types"
	"github.com/johansundell/template-service/utils"
)

type HandlerFuncWithError func(http.ResponseWriter, *http.Request) error

// Route struct for the service
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc HandlerFuncWithError
	IsAPICall   bool
}

// Routes for the servcie web handlers
type Routes []Route

// NewRouter creates a new web handler
func NewRouter(handler *handlers.Handler) *mux.Router {
	routes := getRoutes(handler)
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		handler := handlerWithErrors(route.HandlerFunc)
		if route.IsAPICall {
			handler = hadlerWithLogger(route.HandlerFunc, route.Name)
		}
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	// Static files
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(getStaticFiles(settings.UseFileSystem))))
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
			Pattern:     "/ping/{argument}",
			HandlerFunc: handler.Ping,
			IsAPICall:   true,
		},
	}
	return routes
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

func handlerWithErrors(inner HandlerFuncWithError) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Version", Version)
		if err := inner(w, r); err != nil {
			logger.Error(err.Error())
			http.Error(w, httperror.StatusText(err), httperror.HTTPStatus(err))
		}
	}
}

func hadlerWithLogger(inner HandlerFuncWithError, name string) http.HandlerFunc {
	//fmt.Println("Adding logger to handler:", name)
	return func(w http.ResponseWriter, r *http.Request) {
		//fmt.Println("SUDDE")
		// Read the request body once
		var requestBody []byte
		if r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			// Reset the request body so it can be read again
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Wrap the original ResponseWriter
		crw := &types.CustomResponseWriter{ResponseWriter: w, Body: new(bytes.Buffer)}

		if err := inner(crw, r); err != nil {
			log := types.UsageLog{
				//IdKey:     k.ID,
				Status:    httperror.HTTPStatus(err),
				Method:    r.Method,
				Error:     err.Error(),
				Endpoint:  utils.GetUrl(r, r.URL.Path),
				CreatedAt: time.Now(),
				Response:  types.RawJSON("{}"),
				Request:   types.RawJSON(requestBody),
			}
			if r.Body != nil {
				bodyBytes, _ := io.ReadAll(r.Body)
				if len(bodyBytes) == 0 {
					log.Request = types.RawJSON("{}")
				} else {
					log.Request = types.RawJSON(bodyBytes)
				}
			} else {
				log.Request = types.RawJSON("{}")
			}
			fmt.Println("Logging error:", log)
			http.Error(w, httperror.StatusText(err), httperror.HTTPStatus(err))
		} else {
			// Capture the response body and status code
			responseBody := crw.Body.String()
			statusCode := crw.StatusCode

			if statusCode == 0 {
				statusCode = http.StatusOK // Default to 200 if no status code was set
			}
			log := types.UsageLog{
				//IdKey:     k.ID,
				Status:    statusCode,
				Method:    r.Method,
				Error:     "",
				Endpoint:  utils.GetUrl(r, r.URL.Path),
				CreatedAt: time.Now(),
				Response:  types.RawJSON(responseBody),
				Request:   types.RawJSON(requestBody),
			}
			fmt.Println("Logging success:", log)
		}
		//fmt.Println(crw.Body)
	}
}
