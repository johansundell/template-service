package main

import (
	"errors"
	"io/fs"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/johansundell/template-service/handlers"
)

// Route struct for the service
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes for the servcie web handlers
type Routes []Route

// NewRouter creates a new web handler
func NewRouter(handler *handlers.Handler) *mux.Router {
	routes := getRoutes(handler)
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = wwwLogger(handler, route.Name)
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

func handlerWithErrors(f func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Version", appVersionStr)
		if err := f(w, r); err != nil {
			var status int
			var statusErr interface {
				error
				HTTPStatus() int
			}
			if errors.As(err, &statusErr) {
				status = statusErr.HTTPStatus()
			}
			http.Error(w, err.Error(), status)
		}
	}
}

func wwwLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if settings.Debug {
			logger.Info(name + " " + r.RequestURI + " " + r.RemoteAddr + " " + r.Method)
		}
		w.Header().Set("X-Version", appVersionStr)
		inner.ServeHTTP(w, r)
	})
}
