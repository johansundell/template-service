package handlers

import (
	"log"
	"net/http"
)

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	const tmplFile = "health.html"

	tmpl, err := h.getTemplate(true, tmplFile)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":   "Health Check",
		"Name":    h.nameOfService,
		"Version": h.versionOfService,
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Println(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		//reportError(w, r, err)
		return
	}
}
