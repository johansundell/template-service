package main

import (
	"io/fs"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/httperror"
)

type HandlerFuncWithError func(http.ResponseWriter, *http.Request) error

// Route struct for the service
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc HandlerFuncWithError
}

// Routes for the servcie web handlers
type Routes []Route

// NewRouter creates a new web handler
func NewRouter(handler *handlers.Handler) *mux.Router {
	routes := getRoutes(handler)
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		handler := handlerWithErrors(route.HandlerFunc)
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

func handlerWithErrors(inner HandlerFuncWithError) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Version", appVersionStr)
		if err := inner(w, r); err != nil {
			logger.Error(err.Error())
			http.Error(w, httperror.StatusText(err), httperror.HTTPStatus(err))
		}
	}
}

/*func wwwLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if settings.Debug {
			logger.Info(name + " " + r.RequestURI + " " + r.RemoteAddr + " " + r.Method)
		}
		w.Header().Set("X-Version", appVersionStr)
		inner.ServeHTTP(w, r)
	})
}*/
