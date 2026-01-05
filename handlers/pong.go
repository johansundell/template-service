package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Pong(c *gin.Context) error {
	data, err := io.ReadAll(c.Request.Body)
	defer c.Request.Body.Close()
	if err != nil {
		return err
	}

	var input map[string]interface{}
	err = json.Unmarshal(data, &input)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, gin.H{"message": input})
	return nil
}
