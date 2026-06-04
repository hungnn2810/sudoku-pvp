// Package middleware provides Gin HTTP middleware for the sudoku-pvp API.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sudoku-pvp/internal/logger"
)

// Recovery returns a Gin middleware that catches panics in downstream handlers,
// logs the panic via structured zerolog (T-04-01: stack trace is not exposed
// to the client), and responds with a generic 500 JSON error.
//
// Uses gin.CustomRecovery (not gin.Recovery) to avoid unstructured text log output.
// Must be the first middleware in the chain so that all downstream panics are caught.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		// Log full panic detail internally — never returned to client (T-04-01).
		logger.FromCtx(c.Request.Context()).Error().
			Interface("panic", err).
			Str("path", c.Request.URL.Path).
			Str("method", c.Request.Method).
			Msg("panic recovered")

		c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
			"code":    "INTERNAL_ERROR",
			"message": "internal server error",
		})
	})
}
