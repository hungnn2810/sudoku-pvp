// Package handler provides thin Gin HTTP handlers for auth endpoints.
// Handlers delegate all business logic to AuthService; no auth rules live here.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sudoku-pvp/internal/auth/service"
	"sudoku-pvp/internal/common"
	"sudoku-pvp/internal/logger"
)

// AuthHandler holds the auth service dependency.
// All handlers are thin — they parse input, call the service, and write the response.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler creates an AuthHandler wrapping the given AuthService.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// GuestLogin handles POST /api/v1/auth/guest.
// Creates a new guest user and returns an access + refresh token pair.
// D-14: no request body needed — identity is UUID only.
func (h *AuthHandler) GuestLogin(c *gin.Context) {
	ctx := c.Request.Context()

	pair, err := h.svc.GuestLogin(ctx)
	if err != nil {
		logger.FromCtx(ctx).Error().Err(err).Msg("guest login failed")
		common.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "guest login failed")
		return
	}

	common.OK(c, pair)
}

// GoogleLogin handles POST /api/v1/auth/google.
// Verifies a Google ID token and returns an access + refresh token pair.
// D-08: client completes Google Sign-In, sends idToken to this endpoint.
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		IDToken string `json:"idToken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.IDToken == "" {
		common.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "idToken required")
		return
	}

	pair, err := h.svc.GoogleLogin(ctx, req.IDToken)
	if err != nil {
		logger.FromCtx(ctx).Error().Err(err).Msg("google login failed")
		common.Error(c, http.StatusUnauthorized, "AUTH_FAILED", "google authentication failed")
		return
	}

	common.OK(c, pair)
}

// Refresh handles POST /api/v1/auth/refresh.
// Rotates an existing refresh token and issues a new token pair.
// D-03: old refresh token is deleted before new one is issued.
func (h *AuthHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		UserID       string `json:"userId"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" || req.RefreshToken == "" {
		common.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "userId and refreshToken required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "userId must be a valid UUID")
		return
	}

	pair, err := h.svc.Refresh(ctx, userID, req.RefreshToken)
	if err != nil {
		common.Error(c, http.StatusUnauthorized, "AUTH_FAILED", "invalid or expired refresh token")
		return
	}

	common.OK(c, pair)
}

// Logout handles POST /api/v1/auth/logout.
// Deletes the user's refresh token from Redis. Requires JWT middleware.
// D-06: idempotent — deleting a non-existent key is not an error.
// T-02-05-02: userId is read from Gin context set by JWTMiddleware, NOT from request body.
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	rawUserID, ok := c.Get("userId")
	if !ok {
		common.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "userId missing from context")
		return
	}
	userIDStrVal, ok := rawUserID.(string)
	if !ok {
		common.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "userId context value is not a string")
		return
	}
	userID, err := uuid.Parse(userIDStrVal)
	if err != nil {
		common.Error(c, http.StatusBadRequest, "INVALID_TOKEN", "invalid user id in token")
		return
	}

	if err := h.svc.Logout(ctx, userID); err != nil {
		logger.FromCtx(ctx).Warn().Err(err).Str("userId", userID.String()).Msg("logout redis error")
		// Still return 204 to client — best-effort logout per D-06,
		// but log the operational failure.
	}
	c.Status(http.StatusNoContent)
}
