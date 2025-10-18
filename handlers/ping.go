package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/johansundell/template-service/httperror"
)

func (s *Handler) Ping(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)

	p := struct {
		Result string `json:"result"`
	}{Result: vars["argument"]}

	if p.Result == "notfound" {
		return httperror.ReturnWithHTTPStatus(errors.New("Nope"), http.StatusNotFound)
	}

	w.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	//enc.SetIndent("", "\t")
	enc.Encode(p)
	return nil
}
