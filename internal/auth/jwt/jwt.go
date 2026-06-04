package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the JWT payload for sudoku-pvp access tokens.
// Fields: Sub (UUID v7 userId), Role, standard iat/exp.
// D-05: no username or guest flag embedded.
type Claims struct {
	Sub  uuid.UUID `json:"sub"`
	Role string    `json:"role"`
	jwt.RegisteredClaims
}

// NewAccessClaims constructs a Claims value with exp and iat set.
// Used by the auth service to build tokens before signing.
func NewAccessClaims(userID uuid.UUID, role string, ttl time.Duration) Claims {
	now := time.Now().UTC()
	return Claims{
		Sub:  userID,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
}

// SignJWT signs a Claims struct with HS256 and returns the compact token string.
// D-04: algorithm fixed to HS256; secret from caller (loaded from config).
func SignJWT(claims Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("jwt sign: %w", err)
	}
	return signed, nil
}

// ValidateJWT parses and validates a compact token string.
// Returns *Claims on success or a wrapped error on failure.
// D-20: shared by REST middleware and WS middleware — single source of truth.
// T-02-02-01: key func rejects any algorithm other than HS256 to prevent alg:none
// and algorithm confusion attacks.
func ValidateJWT(tokenStr string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt validate: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt validate: invalid token claims")
	}

	return claims, nil
}
