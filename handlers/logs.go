package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/httperror"
)

func (h *Handler) GetLogsHandler(c *gin.Context) error {
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

	// Adjust 'to' date to include the entire day
	to = to.Add(24 * time.Hour).Add(-1 * time.Second)

	logs, err := h.store.GetLogs(from, to)
	if err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusInternalServerError)
	}
	//fmt.Println(logs)

	c.JSON(http.StatusOK, logs)
	return nil
}
