package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Handler) Pong(c *gin.Context) error {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
	return nil
}
