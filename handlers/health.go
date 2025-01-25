package handlers

import (
	"net/http"

	"github.com/johansundell/template-service/httperror"
)

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	const tmplFile = "health.html"

	tmpl, err := h.getTemplate(true, tmplFile)
	if err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusInternalServerError)
	}

	data := map[string]interface{}{
		"title":   "Health Check",
		"name":    h.nameOfService,
		"version": h.versionOfService,
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusInternalServerError)
	}
	return nil
}
