package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/httperror"
)

func (s *Handler) Ping(c *gin.Context) error {
	argument := c.Param("argument")

	p := struct {
		Result string `json:"result"`
	}{Result: argument}

	if p.Result == "notfound" {
		return httperror.ReturnWithHTTPStatus(errors.New("Nope"), http.StatusNotFound)
	}

	c.JSON(http.StatusOK, p)
	return nil
}
