//go:build integration

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	authjwt "sudoku-pvp/internal/auth/jwt"
	"sudoku-pvp/internal/auth/repository"
	"sudoku-pvp/internal/auth/service"
	"sudoku-pvp/internal/testutil"
)

const testJWTSecret = "test-secret-for-service-unit-32b"

func newTestAuthService(t *testing.T) (*service.AuthService, func()) {
	t.Helper()
	ctx := context.Background()

	pool, pgCleanup := testutil.SetupPostgres(ctx, t)
	rdb, redisCleanup := testutil.SetupRedis(ctx, t)

	userRepo := repository.NewUserRepo(pool)
	authRepo := repository.NewAuthRepo(rdb)

	svc := service.NewAuthService(
		userRepo,
		authRepo,
		nil, // JWKSCache not needed for guest/refresh/logout tests
		testJWTSecret,
		15*time.Minute,
		30*24*time.Hour,
	)

	cleanup := func() {
		pgCleanup()
		redisCleanup()
	}

	return svc, cleanup
}

// TestAuthService_GuestLogin_ReturnsTokenPair verifies that GuestLogin creates a user
// and returns a valid access+refresh token pair. (REQ-auth-guest, D-14, D-17)
func TestAuthService_GuestLogin_ReturnsTokenPair(t *testing.T) {
	svc, cleanup := newTestAuthService(t)
	defer cleanup()

	ctx := context.Background()
	pair, err := svc.GuestLogin(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken, "access token must not be empty")
	require.NotEmpty(t, pair.RefreshToken, "refresh token must not be empty")

	// Validate that the access token is a legitimate HS256 JWT.
	claims, err := authjwt.ValidateJWT(pair.AccessToken, []byte(testJWTSecret))
	require.NoError(t, err, "access token must be a valid JWT")
	require.NotEqual(t, uuid.Nil, claims.Sub, "token sub must be a non-nil UUID")
	require.Equal(t, "player", claims.Role, "guest role must be 'player' (D-17)")
}

// TestAuthService_Refresh_RotatesToken verifies that a refresh rotates the token
// and the old token is rejected on a second use. (D-03, T-02-06-02)
func TestAuthService_Refresh_RotatesToken(t *testing.T) {
	svc, cleanup := newTestAuthService(t)
	defer cleanup()

	ctx := context.Background()

	// Guest login to obtain initial token pair.
	pair1, err := svc.GuestLogin(ctx)
	require.NoError(t, err)

	// Parse the userID from the access token claims.
	claims, err := authjwt.ValidateJWT(pair1.AccessToken, []byte(testJWTSecret))
	require.NoError(t, err)
	userID := claims.Sub

	// Refresh: old token in, new pair out.
	pair2, err := svc.Refresh(ctx, userID, pair1.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, pair1.RefreshToken, pair2.RefreshToken,
		"refresh token must rotate on each use (D-03)")

	// Replay attack: use the old refresh token again — must fail.
	_, err = svc.Refresh(ctx, userID, pair1.RefreshToken)
	require.Error(t, err, "old refresh token must be rejected after rotation (D-03, T-02-04-02)")
}

// TestAuthService_Logout_Idempotent verifies that logout can be called twice
// without returning an error on the second call. (D-06)
func TestAuthService_Logout_Idempotent(t *testing.T) {
	svc, cleanup := newTestAuthService(t)
	defer cleanup()

	ctx := context.Background()

	pair, err := svc.GuestLogin(ctx)
	require.NoError(t, err)

	claims, err := authjwt.ValidateJWT(pair.AccessToken, []byte(testJWTSecret))
	require.NoError(t, err)
	userID := claims.Sub

	// First logout — removes the token from Redis.
	err = svc.Logout(ctx, userID)
	require.NoError(t, err)

	// Second logout — key already gone; must be idempotent (D-06).
	err = svc.Logout(ctx, userID)
	require.NoError(t, err, "logout must be idempotent — second call must not error (D-06)")
}
