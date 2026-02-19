package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	message := "internal server error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if m, ok := he.Message.(string); ok {
			message = m
		} else {
			message = http.StatusText(code)
		}
	}

	resp := errorResponse{
		Error:   http.StatusText(code),
		Message: message,
	}

	_ = c.JSON(code, resp)
}
