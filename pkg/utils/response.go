package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message,omitempty"`
	Code    int         `json:"code"`
	Details interface{} `json:"details,omitempty"`
}

type SuccessResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

func RespondWithError(c *gin.Context, code int, err error, message ...string) {
	var msg string
	if len(message) > 0 {
		msg = message[0]
	}

	response := ErrorResponse{
		Error:   err.Error(),
		Message: msg,
		Code:    code,
	}

	c.JSON(code, response)
}

func RespondWithSuccess(c *gin.Context, data interface{}, message ...string) {
	var msg string
	if len(message) > 0 {
		msg = message[0]
	}

	response := SuccessResponse{
		Data:    data,
		Message: msg,
	}

	c.JSON(http.StatusOK, response)
}

func RespondWithCreated(c *gin.Context, data interface{}, message ...string) {
	var msg string
	if len(message) > 0 {
		msg = message[0]
	}

	response := SuccessResponse{
		Data:    data,
		Message: msg,
	}

	c.JSON(http.StatusCreated, response)
}

func RespondWithValidationError(c *gin.Context, err error) {
	response := ErrorResponse{
		Error:   "Validation failed",
		Message: err.Error(),
		Code:    http.StatusBadRequest,
	}

	c.JSON(http.StatusBadRequest, response)
}

func RespondWithNotFound(c *gin.Context, resource string) {
	response := ErrorResponse{
		Error:   "Not found",
		Message: resource + " not found",
		Code:    http.StatusNotFound,
	}

	c.JSON(http.StatusNotFound, response)
}

func RespondWithInternalError(c *gin.Context, err error) {
	response := ErrorResponse{
		Error:   "Internal server error",
		Message: err.Error(),
		Code:    http.StatusInternalServerError,
	}

	c.JSON(http.StatusInternalServerError, response)
}