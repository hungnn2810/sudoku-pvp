package jwt_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	authjwt "sudoku-pvp/internal/auth/jwt"
)

func TestSignJWT_RoundTrip(t *testing.T) {
	secret := []byte("test-secret-32-bytes-minimum-ok!")
	userID := uuid.Must(uuid.NewV7())
	role := "player"

	claims := authjwt.NewAccessClaims(userID, role, 15*time.Minute)
	tokenStr, err := authjwt.SignJWT(claims, secret)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	parsed, err := authjwt.ValidateJWT(tokenStr, secret)
	require.NoError(t, err)
	require.Equal(t, userID, parsed.Sub)
	require.Equal(t, role, parsed.Role)
}

func TestValidateJWT_InvalidSignature(t *testing.T) {
	secret1 := []byte("test-secret-32-bytes-minimum-ok!")
	secret2 := []byte("different-secret-32-bytes-min-ok")

	userID := uuid.Must(uuid.NewV7())
	claims := authjwt.NewAccessClaims(userID, "player", 15*time.Minute)
	tokenStr, err := authjwt.SignJWT(claims, secret1)
	require.NoError(t, err)

	_, err = authjwt.ValidateJWT(tokenStr, secret2)
	require.Error(t, err)
}

func TestValidateJWT_Expired(t *testing.T) {
	secret := []byte("test-secret-32-bytes-minimum-ok!")
	userID := uuid.Must(uuid.NewV7())

	// Negative TTL = already expired at signing time.
	claims := authjwt.NewAccessClaims(userID, "player", -1*time.Second)
	tokenStr, err := authjwt.SignJWT(claims, secret)
	require.NoError(t, err)

	_, err = authjwt.ValidateJWT(tokenStr, secret)
	require.Error(t, err)
}

func TestValidateJWT_WrongAlgorithm_AlgNone(t *testing.T) {
	// Manually craft a token with alg:none header.
	// Header: {"alg":"none","typ":"JWT"}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	// Payload: minimal valid JWT claims
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"test","exp":9999999999}`))
	// alg:none token has empty signature
	algNoneToken := strings.Join([]string{header, payload, ""}, ".")

	secret := []byte("test-secret-32-bytes-minimum-ok!")
	_, err := authjwt.ValidateJWT(algNoneToken, secret)
	require.Error(t, err)
}

func TestNewAccessClaims_Fields(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	role := "admin"
	ttl := 5 * time.Minute

	claims := authjwt.NewAccessClaims(userID, role, ttl)

	require.Equal(t, userID, claims.Sub)
	require.Equal(t, role, claims.Role)
	require.NotNil(t, claims.ExpiresAt)
	require.NotNil(t, claims.IssuedAt)

	// ExpiresAt should be approximately now + ttl.
	expectedExpiry := time.Now().Add(ttl)
	diff := claims.ExpiresAt.Time.Sub(expectedExpiry)
	if diff < 0 {
		diff = -diff
	}
	require.Less(t, diff, 2*time.Second, "ExpiresAt should be close to now+ttl")
}
