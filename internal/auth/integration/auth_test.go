//go:build integration

// Package auth_integration_test exercises the full auth HTTP flow against
// real PostgreSQL and Redis containers (via testcontainers-go). All tests are
// gated by the "integration" build tag and require Docker.
package auth_integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authjwt "sudoku-pvp/internal/auth/jwt"
	"sudoku-pvp/internal/app"
	"sudoku-pvp/internal/config"
	"sudoku-pvp/internal/testutil"
)

const integrationJWTSecret = "test-secret-for-integration-32b!"

// setupTestApp starts real Postgres, Redis, and RabbitMQ containers, builds the
// full App, and returns the server and a cleanup function.
func setupTestApp(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	ctx := context.Background()

	pool, pgCleanup := testutil.SetupPostgres(ctx, t)
	rdb, redisCleanup := testutil.SetupRedis(ctx, t)
	rabbitURL, mqCleanup := testutil.SetupRabbitMQ(ctx, t)

	// Extract DSN and Redis addr from the real containers.
	// The pool was created by testutil.SetupPostgres using the container DSN.
	// We need to pass the same DSN to app.New via config. Re-read it from the pool.
	pgDSN := pool.Config().ConnString()
	_ = pool // pool itself is not used by app.New — it creates its own pool

	// Extract Redis addr from the client created by SetupRedis.
	redisAddr := rdb.Options().Addr

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
		Postgres: config.PostgresConfig{
			DSN:      pgDSN,
			MaxConns: 2,
			MinConns: 1,
		},
		Redis: config.RedisConfig{
			Addr:     redisAddr,
			Password: "",
			DB:       0,
			PoolSize: 2,
		},
		RabbitMQ: config.RabbitMQConfig{
			URL: rabbitURL,
		},
		Telemetry: config.TelemetryConfig{},
		Auth: config.AuthConfig{
			JWTSecret:       integrationJWTSecret,
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 30 * 24 * time.Hour,
		},
		LogLevel: "warn", // suppress noisy startup logs during tests
	}

	a, err := app.New(cfg)
	require.NoError(t, err, "app.New failed")

	srv := httptest.NewServer(a.Router())

	cleanup := func() {
		srv.Close()
		a.Shutdown(context.Background())
		pgCleanup()
		redisCleanup()
		mqCleanup()
	}

	return srv, cleanup
}

// post sends a POST request to the test server and returns the response.
func post(t *testing.T, srv *httptest.Server, path string, body any, authToken string) *http.Response {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// decodeTokenPair decodes the standard token pair JSON body.
func decodeTokenPair(t *testing.T, resp *http.Response) (accessToken, refreshToken string) {
	t.Helper()
	defer resp.Body.Close()

	var pair struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&pair))
	return pair.AccessToken, pair.RefreshToken
}

// TestHTTP_GuestLogin_Returns200 verifies POST /api/v1/auth/guest returns 200
// with a non-empty access and refresh token pair.
func TestHTTP_GuestLogin_Returns200(t *testing.T) {
	srv, cleanup := setupTestApp(t)
	defer cleanup()

	resp := post(t, srv, "/api/v1/auth/guest", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	accessToken, refreshToken := decodeTokenPair(t, resp)
	require.NotEmpty(t, accessToken, "access token must not be empty")
	require.NotEmpty(t, refreshToken, "refresh token must not be empty")

	// Validate JWT structure.
	claims, err := authjwt.ValidateJWT(accessToken, []byte(integrationJWTSecret))
	require.NoError(t, err, "access token must be a valid JWT")
	require.Equal(t, "player", claims.Role)
}

// TestHTTP_GuestLogin_Refresh_Cycle verifies the full guest-login → refresh cycle:
// new tokens are issued and the old refresh token is not reused.
func TestHTTP_GuestLogin_Refresh_Cycle(t *testing.T) {
	srv, cleanup := setupTestApp(t)
	defer cleanup()

	// Step 1: guest login.
	resp1 := post(t, srv, "/api/v1/auth/guest", nil, "")
	require.Equal(t, http.StatusOK, resp1.StatusCode)
	access1, refresh1 := decodeTokenPair(t, resp1)

	// Step 2: parse userID from access token.
	claims, err := authjwt.ValidateJWT(access1, []byte(integrationJWTSecret))
	require.NoError(t, err)
	userID := claims.Sub.String()

	// Step 3: refresh.
	resp2 := post(t, srv, "/api/v1/auth/refresh", map[string]string{
		"userId":       userID,
		"refreshToken": refresh1,
	}, "")
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	_, refresh2 := decodeTokenPair(t, resp2)
	require.NotEqual(t, refresh1, refresh2, "refresh token must rotate on each use (D-03)")
}

// TestHTTP_ProtectedRoute_NoToken_Returns401 verifies that POST /api/v1/auth/logout
// without an Authorization header returns 401.
func TestHTTP_ProtectedRoute_NoToken_Returns401(t *testing.T) {
	srv, cleanup := setupTestApp(t)
	defer cleanup()

	resp := post(t, srv, "/api/v1/auth/logout", nil, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestHTTP_ProtectedRoute_ValidToken_Returns204 verifies that POST /api/v1/auth/logout
// with a valid Bearer token returns 204 No Content.
func TestHTTP_ProtectedRoute_ValidToken_Returns204(t *testing.T) {
	srv, cleanup := setupTestApp(t)
	defer cleanup()

	// Guest login to obtain a valid token.
	resp1 := post(t, srv, "/api/v1/auth/guest", nil, "")
	require.Equal(t, http.StatusOK, resp1.StatusCode)
	accessToken, _ := decodeTokenPair(t, resp1)

	// Logout with the valid access token.
	resp2 := post(t, srv, "/api/v1/auth/logout", nil, accessToken)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusNoContent, resp2.StatusCode)
}
