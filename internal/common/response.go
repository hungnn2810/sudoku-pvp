package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard JSON error body returned by all API endpoints.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error writes a JSON error response with the given HTTP status, code, and message.
func Error(c *gin.Context, status int, code string, msg string) {
	c.JSON(status, ErrorResponse{Code: code, Message: msg})
}

// OK writes a 200 JSON response with the given data payload.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}
