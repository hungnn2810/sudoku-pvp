// Package middleware provides Gin middleware for JWT authentication.
// Both REST and WebSocket middleware share the same ValidateJWT call path (D-20).
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	authjwt "sudoku-pvp/internal/auth/jwt"
)

// JWTMiddleware validates the Bearer token in the Authorization header for REST routes.
// On success, sets "userId" (string UUID) and "role" (string) in the Gin context (D-21).
// On failure, returns HTTP 401 and calls c.Abort() — downstream handlers never execute (T-02-05-01).
// D-20: calls authjwt.ValidateJWT — same code path as WSJWTMiddleware.
// CRITICAL: registered on route groups AFTER global Recovery/Tracing/ZerologLogger chain (T-02-05-05).
func JWTMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "MISSING_TOKEN", "message": "authorization header required"})
			c.Abort()
			return
		}

		claims, err := authjwt.ValidateJWT(token, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_TOKEN", "message": "invalid or expired token"})
			c.Abort()
			return
		}

		// D-21: set userId as string UUID and role string in Gin context.
		c.Set("userId", claims.Sub.String())
		c.Set("role", claims.Role)
		c.Next()
	}
}

// WSJWTMiddleware validates the Bearer token before a WebSocket upgrade.
// D-18: Authorization header used (not query param) — mobile-only clients confirmed.
// D-19: returns HTTP 401 without upgrading on auth failure; c.Abort() prevents upgrade.
// D-20: uses the same authjwt.ValidateJWT and extractBearerToken as JWTMiddleware (T-02-05-03).
func WSJWTMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "MISSING_TOKEN", "message": "authorization header required"})
			c.Abort()
			return
		}

		claims, err := authjwt.ValidateJWT(token, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_TOKEN", "message": "websocket auth failed"})
			c.Abort()
			return
		}

		// D-21: same context keys as JWTMiddleware for uniform handler access.
		c.Set("userId", claims.Sub.String())
		c.Set("role", claims.Role)
		c.Next()
	}
}

// extractBearerToken extracts the token value from an "Authorization: Bearer <token>" header.
// Returns an empty string if the header is missing or not in Bearer format.
func extractBearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}
