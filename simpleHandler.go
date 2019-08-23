package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	routes = append(routes, Route{"simpleHandler", "GET", "/{argument}", simpleHandler})
}

func simpleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if settings.Debug {
		logger.Info(vars)
	}
	p := struct {
		Result string `json:"result"`
	}{Result: vars["argument"]}

	w.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if settings.Debug {
		enc.SetIndent("", "\t")
	}
	enc.Encode(p)
}
