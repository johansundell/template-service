package httperror

import (
	"errors"
	"fmt"
	"net/http"
)

type statusError struct {
	error
	status int
}

func (e statusError) Unwrap() error {
	return e.error
}

func HTTPStatus(err error) int {
	if err == nil {
		return 0
	}
	var statusErr interface {
		error
		HTTPStatus() int
	}
	if errors.As(err, &statusErr) {
		return statusErr.HTTPStatus()
	}
	return http.StatusInternalServerError
}

func StatusText(err error) string {
	if myerr, ok := err.(statusError); ok {
		return http.StatusText(myerr.status)
	}
	return http.StatusText(http.StatusInternalServerError)
}

func (se statusError) Error() string {
	return fmt.Sprintf("status %d: err %v", se.status, se.error)
}

func ReturnWithHTTPStatus(err error, status int) error {
	return statusError{
		error:  err,
		status: status,
	}
}
