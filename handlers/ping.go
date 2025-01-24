package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	p := struct {
		Result string `json:"result"`
	}{Result: vars["argument"]}

	w.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	enc.Encode(p)
}
