package google

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWK represents a single JSON Web Key from the JWKS endpoint.
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSResponse is the JSON structure returned by the Google JWKS endpoint.
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWKSCache caches Google public keys in memory with a TTL.
// D-09: ~1hr TTL; re-fetches on kid mismatch; goroutine-safe via sync.RWMutex.
type JWKSCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	ttl       time.Duration
	endpoint  string
	client    *http.Client
}

// NewJWKSCache creates a JWKSCache with the Google JWKS endpoint and a 1-hour TTL.
// The cache is lazily populated on first key lookup.
func NewJWKSCache() *JWKSCache {
	return &JWKSCache{
		ttl:      time.Hour,
		endpoint: "https://www.googleapis.com/oauth2/v3/certs",
		client:   &http.Client{Timeout: 10 * time.Second},
		keys:     make(map[string]*rsa.PublicKey),
	}
}

// fetchLocked fetches and parses the JWKS endpoint, populating the cache.
// Caller MUST hold the write lock before calling this method.
func (c *JWKSCache) fetchLocked(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return fmt.Errorf("jwks fetch: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("jwks fetch: %w", err)
	}
	defer resp.Body.Close()

	var jwksResp JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwksResp); err != nil {
		return fmt.Errorf("jwks fetch: decode: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey, len(jwksResp.Keys))
	for _, key := range jwksResp.Keys {
		if key.Kty != "RSA" {
			continue
		}

		// Decode the base64url-encoded modulus (N) and exponent (E).
		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			return fmt.Errorf("jwks fetch: decode N for kid %q: %w", key.Kid, err)
		}

		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			return fmt.Errorf("jwks fetch: decode E for kid %q: %w", key.Kid, err)
		}

		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)

		newKeys[key.Kid] = &rsa.PublicKey{
			N: n,
			E: int(e.Int64()),
		}
	}

	c.keys = newKeys
	c.fetchedAt = time.Now()
	return nil
}

// getKey returns the RSA public key for the given kid.
// It reads from cache if fresh; otherwise fetches from the JWKS endpoint.
// D-09: kid mismatch triggers an immediate re-fetch.
func (c *JWKSCache) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Fast path: read lock — check if kid is present and cache is still fresh.
	c.mu.RLock()
	if time.Since(c.fetchedAt) < c.ttl {
		if key, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return key, nil
		}
	}
	c.mu.RUnlock()

	// Slow path: write lock — re-fetch JWKS from the endpoint.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check under write lock in case another goroutine already fetched.
	if time.Since(c.fetchedAt) < c.ttl {
		if key, ok := c.keys[kid]; ok {
			return key, nil
		}
	}

	if err := c.fetchLocked(ctx); err != nil {
		return nil, err
	}

	key, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("jwks: kid %q not found", kid)
	}
	return key, nil
}

// googleClaims are the JWT claims contained in a Google ID token.
type googleClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// VerifyGoogleIDToken verifies a Google ID token against cached JWKS and returns
// the subject (sub claim) and email.
// T-02-04-01: RS256 algorithm is enforced in the key func; forged or tampered tokens
// fail signature verification before any user lookup occurs.
func (c *JWKSCache) VerifyGoogleIDToken(ctx context.Context, idToken string) (sub, email string, err error) {
	// Step 1: extract kid from the unverified header.
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(idToken, &googleClaims{})
	if err != nil {
		return "", "", fmt.Errorf("google id token: parse header: %w", err)
	}

	kid, ok := unverified.Header["kid"].(string)
	if !ok || kid == "" {
		return "", "", fmt.Errorf("google id token: missing kid in header")
	}

	// Step 2: resolve the public key for this kid.
	pubKey, err := c.getKey(ctx, kid)
	if err != nil {
		return "", "", fmt.Errorf("google id token: %w", err)
	}

	// Step 3: parse and verify the full token.
	claims := &googleClaims{}
	token, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (any, error) {
		// T-02-04-01: enforce RS256; reject any other algorithm.
		if t.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("google id token: unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return "", "", fmt.Errorf("google id token: verify: %w", err)
	}
	if !token.Valid {
		return "", "", fmt.Errorf("google id token: invalid token")
	}

	return claims.Sub, claims.Email, nil
}
