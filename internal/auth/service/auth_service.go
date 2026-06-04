package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	authjwt "sudoku-pvp/internal/auth/jwt"
	"sudoku-pvp/internal/auth/google"
	"sudoku-pvp/internal/auth/repository"
)

// TokenPair holds the access token and refresh token returned by all login flows.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// AuthService orchestrates the four auth flows: guest login, Google login,
// token refresh, and logout.
// Handler layer (wave 4) depends on AuthService for all business operations.
type AuthService struct {
	userRepo   *repository.UserRepo
	authRepo   *repository.AuthRepo
	jwksCache  *google.JWKSCache
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewAuthService creates an AuthService with the required dependencies.
func NewAuthService(
	userRepo *repository.UserRepo,
	authRepo *repository.AuthRepo,
	jwksCache *google.JWKSCache,
	jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		authRepo:   authRepo,
		jwksCache:  jwksCache,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// generateOpaqueToken generates a 32-byte cryptographically random token
// encoded as a 64-character hex string.
// T-02-04-03: 256-bit entropy prevents refresh token prediction.
func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate opaque token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// issueTokenPair creates a signed access token and an opaque refresh token,
// stores the refresh token in Redis, and returns the pair.
func (s *AuthService) issueTokenPair(ctx context.Context, userID uuid.UUID, role string) (TokenPair, error) {
	// Generate opaque refresh token value.
	refreshToken, err := generateOpaqueToken()
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue token pair: %w", err)
	}

	// Build and sign the JWT access token.
	claims := authjwt.NewAccessClaims(userID, role, s.accessTTL)
	accessToken, err := authjwt.SignJWT(claims, s.jwtSecret)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue token pair: %w", err)
	}

	// Persist refresh token in Redis with the configured TTL.
	if err := s.authRepo.StoreRefreshToken(ctx, userID, refreshToken, s.refreshTTL); err != nil {
		return TokenPair{}, fmt.Errorf("issue token pair: %w", err)
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GuestLogin creates a new guest user (users + wallets rows in a single DB
// transaction) and returns a token pair.
// D-14: no user_providers row for guests.
// D-17: role is always "player" for guest accounts.
func (s *AuthService) GuestLogin(ctx context.Context) (TokenPair, error) {
	user, err := s.userRepo.CreateGuestUser(ctx)
	if err != nil {
		return TokenPair{}, fmt.Errorf("guest login: %w", err)
	}

	// Convert pgtype.UUID to uuid.UUID for issueTokenPair.
	userID, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		return TokenPair{}, fmt.Errorf("guest login: parse user id: %w", err)
	}

	pair, err := s.issueTokenPair(ctx, userID, "player")
	if err != nil {
		return TokenPair{}, fmt.Errorf("guest login: %w", err)
	}
	return pair, nil
}

// GoogleLogin verifies a Google ID token, looks up or creates the associated
// user, and returns a token pair.
// D-08: client-side ID token flow — client sends Google ID token to the backend.
// D-09: token verified against cached JWKS; Google JWKS re-fetched on kid miss.
// D-12: new Google users get users + user_providers + wallets rows in one transaction.
func (s *AuthService) GoogleLogin(ctx context.Context, idToken string) (TokenPair, error) {
	// Step 1: verify the Google ID token and extract sub + email.
	googleSub, email, err := s.jwksCache.VerifyGoogleIDToken(ctx, idToken)
	if err != nil {
		return TokenPair{}, fmt.Errorf("google login: %w", err)
	}

	// Step 2: look up an existing user by Google provider record.
	var userID uuid.UUID
	provider, err := s.userRepo.GetUserProviderByProvider(ctx, "google", googleSub)
	if err != nil {
		// WR-04: repository translates pgx.ErrNoRows → repository.ErrNotFound.
		if !errors.Is(err, repository.ErrNotFound) {
			return TokenPair{}, fmt.Errorf("google login: %w", err)
		}
		// New Google user — create user + provider + wallet atomically.
		// WR-03: generate a unique username instead of hardcoding "Player".
		user, _, createErr := s.userRepo.CreateGoogleUser(ctx, googleSub, email)
		if createErr != nil {
			return TokenPair{}, fmt.Errorf("google login: %w", createErr)
		}
		var parseErr error
		userID, parseErr = uuid.FromBytes(user.ID.Bytes[:])
		if parseErr != nil {
			return TokenPair{}, fmt.Errorf("google login: parse user id: %w", parseErr)
		}
	} else {
		// Existing Google user — use the stored user ID.
		var parseErr error
		userID, parseErr = uuid.FromBytes(provider.UserID.Bytes[:])
		if parseErr != nil {
			return TokenPair{}, fmt.Errorf("google login: parse user id: %w", parseErr)
		}
	}

	pair, err := s.issueTokenPair(ctx, userID, "player")
	if err != nil {
		return TokenPair{}, fmt.Errorf("google login: %w", err)
	}
	return pair, nil
}

// Refresh validates the provided refresh token, rotates it, and returns a new
// token pair.
// D-03: old token deleted before new token issued; simultaneous rotation
// attempts cause one to fail with "invalid refresh token".
// T-02-04-02: token replay after rotation is rejected because the old key is
// deleted before the new key is stored.
// T-02-04-04: role is hard-coded to "player" — no role escalation via refresh.
func (s *AuthService) Refresh(ctx context.Context, userID uuid.UUID, providedToken string) (TokenPair, error) {
	// Retrieve the stored refresh token for this user.
	storedToken, err := s.authRepo.GetRefreshToken(ctx, userID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("refresh: %w", err)
	}

	// Compare provided token against stored token using constant-time comparison
	// to prevent timing side-channel attacks (CR-03).
	// D-03: mismatch returns an error without revealing the stored token.
	if subtle.ConstantTimeCompare([]byte(providedToken), []byte(storedToken)) != 1 {
		return TokenPair{}, fmt.Errorf("refresh: invalid refresh token")
	}

	// Delete the old token BEFORE issuing the new one (D-03: rotate on use).
	if err := s.authRepo.DeleteRefreshToken(ctx, userID); err != nil {
		return TokenPair{}, fmt.Errorf("refresh: %w", err)
	}

	// Issue a fresh token pair. Role is hard-coded — no escalation from refresh.
	pair, err := s.issueTokenPair(ctx, userID, "player")
	if err != nil {
		return TokenPair{}, fmt.Errorf("refresh: %w", err)
	}
	return pair, nil
}

// Logout deletes the user's refresh token from Redis.
// D-06: idempotent — deleting a non-existent key is not an error.
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := s.authRepo.DeleteRefreshToken(ctx, userID); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}
