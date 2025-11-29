package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/httperror"
)

func (h *Handler) HealthCheck(c *gin.Context) error {
	const tmplFile = "health.html"

	tmpl, err := h.getTemplate(true, tmplFile)
	if err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusInternalServerError)
	}

	if err := h.store.Ping(); err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusInternalServerError)
	}

	dbStatus := "OK"
	if err := h.store.Ping(); err != nil {
		dbStatus = err.Error()
	}

	data := map[string]interface{}{
		"title":    "Health Check",
		"name":     h.nameOfService,
		"version":  h.versionOfService,
		"dbStatus": dbStatus,
	}

	if err := tmpl.ExecuteTemplate(c.Writer, "base", data); err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusInternalServerError)
	}
	return nil
}
