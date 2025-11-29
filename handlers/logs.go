package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetLogsHandler(c *gin.Context) error {
	fromStr := c.Param("from")
	toStr := c.Param("to")

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return err
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return err
	}

	// Adjust 'to' date to include the entire day
	to = to.Add(24 * time.Hour).Add(-1 * time.Second)

	logs, err := h.store.GetLogs(from, to)
	if err != nil {
		return err
	}
	fmt.Println(logs)

	c.JSON(http.StatusOK, logs)
	return nil
}
