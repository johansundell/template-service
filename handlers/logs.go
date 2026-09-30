package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/httperror"
)

func (h *Handler) GetLogsHandler(c *gin.Context) error {
	// Dates are whole UTC days: from 00:00Z up to, not including, the day after to.
	fromStr := c.Param("from")
	toStr := c.Param("to")

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return httperror.ReturnWithHTTPStatus(errors.New("wrong date format in from"), http.StatusBadRequest)
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return httperror.ReturnWithHTTPStatus(errors.New("wrong date format in to"), http.StatusBadRequest)
	}

	logs, err := h.store.GetLogs(c.Request.Context(), from, to.AddDate(0, 0, 1))
	if err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusInternalServerError)
	}

	c.JSON(http.StatusOK, logs)
	return nil
}
