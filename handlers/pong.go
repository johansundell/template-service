package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/httperror"
)

func (h *Handler) Pong(c *gin.Context) error {
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		return httperror.ReturnWithHTTPStatus(err, http.StatusBadRequest)
	}

	c.JSON(http.StatusOK, gin.H{"message": input})
	return nil
}
