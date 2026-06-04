---
phase: 02-auth-identity
plan: "04"
type: execute
wave: 3
depends_on:
  - 02-02
  - 02-03
files_modified:
  - internal/auth/google/jwks.go
  - internal/auth/service/auth_service.go
autonomous: true
requirements:
  - REQ-001
  - REQ-auth-guest
  - REQ-auth-google
  - REQ-auth-refresh

must_haves:
  truths:
    - "JWKSCache fetches Google JWKS from https://www.googleapis.com/oauth2/v3/certs on first use and caches in memory with ~1hr TTL"
    - "JWKSCache re-fetches when a kid is not found in the cache (D-09)"
    - "JWKSCache is goroutine-safe via sync.RWMutex"
    - "VerifyGoogleIDToken returns the Google subject (sub claim) and email from a verified ID token"
    - "AuthService.GuestLogin creates a user row and returns an access token + refresh token pair"
    - "AuthService.GoogleLogin verifies the Google ID token, looks up or creates user+provider, returns token pair"
    - "AuthService.Refresh rotates the refresh token: validates stored token matches, deletes old, issues new pair (D-03)"
    - "AuthService.Logout deletes refresh:{userId} from Redis (D-06)"
    - "All AuthService methods are context-first"
  artifacts:
    - path: "internal/auth/google/jwks.go"
      provides: "JWKSCache struct with VerifyGoogleIDToken"
      exports: ["JWKSCache", "NewJWKSCache", "VerifyGoogleIDToken"]
    - path: "internal/auth/service/auth_service.go"
      provides: "AuthService with GuestLogin, GoogleLogin, Refresh, Logout"
      exports: ["AuthService", "NewAuthService", "TokenPair", "GuestLogin", "GoogleLogin", "Refresh", "Logout"]
  key_links:
    - from: "internal/auth/google/jwks.go"
      to: "internal/auth/service/auth_service.go"
      via: "JWKSCache injected into AuthService for Google ID token verification"
      pattern: "JWKSCache"
    - from: "internal/auth/service/auth_service.go"
      to: "internal/auth/handler/auth_handler.go (wave 4)"
      via: "AuthService injected into AuthHandler"
      pattern: "AuthService"
---

<objective>
Implement the Google JWKS cache (internal/auth/google/jwks.go) for offline ID token verification and the auth service (internal/auth/service/auth_service.go) that orchestrates all four auth flows: guest login, Google login, token refresh, and logout.

Purpose: AuthService is the business logic hub that calls jwt package (signing/validation), Google JWKS cache (ID token verification), AuthRepo (refresh token storage), and UserRepo (user creation). The handler layer (wave 4) depends on AuthService for all business operations.

Output: JWKSCache with goroutine-safe in-memory caching; AuthService with GuestLogin/GoogleLogin/Refresh/Logout returning TokenPair.
</objective>

<execution_context>
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-PATTERNS.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Implement internal/auth/google/jwks.go (JWKS cache)</name>
  <read_first>
    - J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md (D-08, D-09: client-side ID token verification; in-memory JWKS cache; kid mismatch triggers re-fetch)
    - J:\sources\sudoku-pvp\internal\config\config.go (no GoogleClientID in AuthConfig yet — check if needed; D-08 only requires signature verification, not audience check; confirm what the spec says)
  </read_first>
  <files>internal/auth/google/jwks.go</files>
  <action>
Create internal/auth/google/jwks.go. Package name: "google". Imports: "context", "crypto/rsa", "encoding/json", "fmt", "math/big", "net/http", "sync", "time", "github.com/golang-jwt/jwt/v5".

Define JWK struct (JSON Web Key):
  type JWK struct {
      Kid string `json:"kid"`
      Kty string `json:"kty"`
      N   string `json:"n"`
      E   string `json:"e"`
  }

Define JWKSResponse struct:
  type JWKSResponse struct {
      Keys []JWK `json:"keys"`
  }

Define JWKSCache struct:
  type JWKSCache struct {
      mu        sync.RWMutex
      keys      map[string]*rsa.PublicKey
      fetchedAt time.Time
      ttl       time.Duration
      endpoint  string
      client    *http.Client
  }

Define constructor:
  func NewJWKSCache() *JWKSCache
  Sets ttl = time.Hour, endpoint = "https://www.googleapis.com/oauth2/v3/certs", client = &http.Client{Timeout: 10 * time.Second}, keys = map.

Define private fetchLocked(ctx context.Context) error:
  - Called while write lock is held
  - Fetches endpoint with http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
  - Decodes JSON into JWKSResponse
  - For each JWK with Kty=="RSA": decode N and E (base64url, big.Int) to build *rsa.PublicKey
  - Store in c.keys map keyed by kid
  - Set c.fetchedAt = time.Now()
  - Wrap errors: fmt.Errorf("jwks fetch: %w", err)

Define getKey(ctx context.Context, kid string) (*rsa.PublicKey, error):
  1. Read lock — check if kid exists in c.keys AND cache is fresh (time.Since(c.fetchedAt) < c.ttl):
     - If both true: return key
  2. Upgrade to write lock — re-fetch (calls fetchLocked(ctx))
  3. Look up kid again — if missing after fetch: return nil, fmt.Errorf("jwks: kid %q not found", kid)
  4. Return key

Define VerifyGoogleIDToken:
  func (c *JWKSCache) VerifyGoogleIDToken(ctx context.Context, idToken string) (sub, email string, err error)
  1. Parse header without verifying to extract kid (jwt.ParseHeader or manual base64 decode of header segment)
  2. Call c.getKey(ctx, kid) to get *rsa.PublicKey
  3. Parse full token: jwt.ParseWithClaims(idToken, &googleClaims{}, func(t *jwt.Token) (any, error) { ... })
     - Key func: verify t.Method == jwt.SigningMethodRS256; return the *rsa.PublicKey
  4. Extract googleClaims (define unexported struct with Sub, Email string, jwt.RegisteredClaims)
  5. Return claims.Sub, claims.Email, nil on success

Use jwt.ParseWithClaims from golang-jwt/jwt/v5 (already in go.mod from plan 02-02).

Important: kid extraction from unverified header. Use jwt.NewParser().ParseUnverified to get the token object and read t.Header["kid"] as string. Do NOT validate signature in this step.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/google/... && go vet ./internal/auth/google/...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/google/...` exits 0
    - `go vet ./internal/auth/google/...` exits 0
    - `grep "sync.RWMutex" internal/auth/google/jwks.go` confirms goroutine safety (D-09)
    - `grep "https://www.googleapis.com/oauth2/v3/certs" internal/auth/google/jwks.go` confirms JWKS endpoint
    - `grep "func (c \*JWKSCache) VerifyGoogleIDToken" internal/auth/google/jwks.go` finds the exported method
    - `grep "kid.*not found" internal/auth/google/jwks.go` confirms kid-miss re-fetch behavior
    - `grep "SigningMethodRS256" internal/auth/google/jwks.go` confirms RS256 is required for Google tokens
  </acceptance_criteria>
  <done>JWKSCache compiles; VerifyGoogleIDToken verifies RS256 signature; cache re-fetches on kid miss; goroutine-safe.</done>
</task>

<task type="auto">
  <name>Task 2: Implement internal/auth/service/auth_service.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\auth\jwt\jwt.go (SignJWT, ValidateJWT, NewAccessClaims — read exact signatures before calling)
    - J:\sources\sudoku-pvp\internal\auth\repository\auth_repo.go (StoreRefreshToken, GetRefreshToken, DeleteRefreshToken signatures)
    - J:\sources\sudoku-pvp\internal\auth\repository\user_repo.go (CreateGuestUser, CreateGoogleUser, GetUserProviderByProvider signatures)
    - J:\sources\sudoku-pvp\internal\auth\google\jwks.go (VerifyGoogleIDToken signature)
    - J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md (D-01 through D-17 — all service-level decisions)
  </read_first>
  <files>internal/auth/service/auth_service.go</files>
  <action>
Create internal/auth/service/auth_service.go. Package name: "service". Imports: "context", "crypto/rand", "encoding/hex", "errors", "fmt", "time", "github.com/google/uuid", "github.com/jackc/pgx/v5", "sudoku-pvp/db/sqlc", "sudoku-pvp/internal/auth/google", "sudoku-pvp/internal/auth/jwt", "sudoku-pvp/internal/auth/repository", "sudoku-pvp/internal/common".

Define TokenPair struct (response model, exported):
  type TokenPair struct {
      AccessToken  string `json:"accessToken"`
      RefreshToken string `json:"refreshToken"`
  }

Define AuthService struct:
  type AuthService struct {
      userRepo  *repository.UserRepo
      authRepo  *repository.AuthRepo
      jwksCache *google.JWKSCache
      jwtSecret []byte
      accessTTL  time.Duration
      refreshTTL time.Duration
  }

Define constructor:
  func NewAuthService(userRepo *repository.UserRepo, authRepo *repository.AuthRepo, jwksCache *google.JWKSCache, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService

Define private generateOpaqueToken() (string, error):
  Generates 32 random bytes via crypto/rand.Read, returns hex.EncodeToString. Used for refresh token value.
  Wrap error: fmt.Errorf("generate opaque token: %w", err).

Define private issueTokenPair(ctx, userID uuid.UUID, role string) (TokenPair, error):
  1. Generate opaque refresh token string via generateOpaqueToken()
  2. Create access claims via jwt.NewAccessClaims(userID, role, s.accessTTL)
  3. Sign access token via jwt.SignJWT(claims, s.jwtSecret)
  4. Store refresh token via s.authRepo.StoreRefreshToken(ctx, userID, refreshToken, s.refreshTTL)
  5. Return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}
  Wrap each error with its operation: "issue token pair: ..."

Define GuestLogin:
  func (s *AuthService) GuestLogin(ctx context.Context) (TokenPair, error)
  1. s.userRepo.CreateGuestUser(ctx) — creates user + wallet in transaction
  2. s.issueTokenPair(ctx, user.ID, "player") — role is always "player" for guests (D-17)
  Wrap: fmt.Errorf("guest login: %w", err)

Define GoogleLogin:
  func (s *AuthService) GoogleLogin(ctx context.Context, idToken string) (TokenPair, error)
  1. s.jwksCache.VerifyGoogleIDToken(ctx, idToken) — returns googleSub, email
  2. s.userRepo.GetUserProviderByProvider(ctx, "google", googleSub) — look up existing user
  3. If found: use provider.UserID. If not found (pgx.ErrNoRows or errors.Is): call s.userRepo.CreateGoogleUser(ctx, googleSub, email, "Player") — username is generic "Player" as placeholder (username uniqueness enforcement is UserRepo responsibility; service passes a base name)
  4. s.issueTokenPair(ctx, userID, "player")
  Wrap: fmt.Errorf("google login: %w", err)
  Note: GetUserProviderByProvider may return pgx.ErrNoRows when user is new — check with errors.Is(err, pgx.ErrNoRows) to branch between existing and new user.

Define Refresh:
  func (s *AuthService) Refresh(ctx context.Context, userID uuid.UUID, providedToken string) (TokenPair, error)
  1. s.authRepo.GetRefreshToken(ctx, userID) — retrieve stored token
  2. Compare providedToken == storedToken (constant-time is ideal but not required at MVP)
  3. If mismatch: return zero, fmt.Errorf("refresh: invalid refresh token")
  4. Delete old token: s.authRepo.DeleteRefreshToken(ctx, userID) — revoke before issuing new (D-03)
  5. s.issueTokenPair(ctx, userID, "player") — role always "player" (no role escalation from refresh)
  Note: refresh token does not carry role information; service hard-codes "player" on refresh. Admin flows are out of scope for Phase 2.

Define Logout:
  func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error
  1. s.authRepo.DeleteRefreshToken(ctx, userID) — idempotent (D-06)
  Wrap: fmt.Errorf("logout: %w", err)
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/service/... && go vet ./internal/auth/service/...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/service/...` exits 0
    - `go vet ./internal/auth/service/...` exits 0
    - `grep "func (s \*AuthService) GuestLogin" internal/auth/service/auth_service.go` finds the method
    - `grep "func (s \*AuthService) GoogleLogin" internal/auth/service/auth_service.go` finds the method
    - `grep "func (s \*AuthService) Refresh" internal/auth/service/auth_service.go` finds the method
    - `grep "func (s \*AuthService) Logout" internal/auth/service/auth_service.go` finds the method
    - `grep "crypto/rand" internal/auth/service/auth_service.go` confirms cryptographic refresh token generation
    - `grep "DeleteRefreshToken" internal/auth/service/auth_service.go` appears at least twice (Refresh revokes before issuing; Logout deletes)
    - `grep "GetUserProviderByProvider" internal/auth/service/auth_service.go` confirms Google existing-user lookup
  </acceptance_criteria>
  <done>AuthService compiles; all four business flows implemented; refresh rotates token; logout is idempotent; Google login handles new and existing users.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| client → GoogleLogin | idToken from untrusted mobile client; verified against Google JWKS before any user lookup |
| client → Refresh | providedToken compared to stored opaque token; mismatch returns error without revealing stored token |
| Google JWKS endpoint | External HTTP; 10s timeout; RS256 algorithm enforced in key func |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-02-04-01 | Spoofing | GoogleLogin — forged ID token | mitigate | VerifyGoogleIDToken verifies RS256 signature against Google JWKS before trusting sub/email claims |
| T-02-04-02 | Tampering | Refresh — token replay after rotation | mitigate | D-03: old token deleted before new token issued; simultaneous rotation attempts cause one to fail with "invalid refresh token" |
| T-02-04-03 | Information Disclosure | generateOpaqueToken — predictable token | mitigate | 32-byte crypto/rand output; 256-bit entropy; hex-encoded = 64-char string |
| T-02-04-04 | Elevation of Privilege | Refresh — role escalation | mitigate | Role hard-coded to "player" on refresh; no role carried in refresh token; admin roles out of Phase 2 scope |
| T-02-04-05 | Denial of Service | JWKSCache — JWKS endpoint unavailable | accept | TTL-based cache; existing keys still usable until TTL expires; failure on kid-miss returns 401 to client (correct behavior) |
</threat_model>

<verification>
1. `cd J:/sources/sudoku-pvp && go build ./internal/auth/...` — all auth sub-packages compile together
2. `grep "crypto/rand" internal/auth/service/auth_service.go` — opaque token uses secure random
3. `grep "SigningMethodRS256" internal/auth/google/jwks.go` — Google token algorithm enforced
4. `grep "DeleteRefreshToken" internal/auth/service/auth_service.go | wc -l` — at least 2 (Refresh + Logout)
</verification>

<success_criteria>
- JWKSCache compiles; goroutine-safe; re-fetches on kid miss; 1hr TTL; RS256 enforced
- AuthService compiles; GuestLogin/GoogleLogin/Refresh/Logout all implemented
- Refresh rotates token atomically (delete old, issue new)
- Logout is idempotent (Del on missing key is not an error)
- GoogleLogin handles both new and existing Google users
</success_criteria>

<output>
Create .planning/phases/02-auth-identity/02-04-SUMMARY.md when done.
</output>

## Artifacts This Phase Produces

### New Types
- `service.TokenPair` struct (internal/auth/service/auth_service.go) — fields: AccessToken, RefreshToken string
- `service.AuthService` struct (internal/auth/service/auth_service.go) — fields: userRepo, authRepo, jwksCache, jwtSecret, accessTTL, refreshTTL
- `google.JWKSCache` struct (internal/auth/google/jwks.go) — fields: mu sync.RWMutex, keys map[string]*rsa.PublicKey, fetchedAt time.Time, ttl time.Duration, endpoint string, client *http.Client
- `google.JWK` struct (internal/auth/google/jwks.go) — JSON Web Key representation
- `google.JWKSResponse` struct (internal/auth/google/jwks.go) — JWKS endpoint response

### New Functions
- `google.NewJWKSCache() *JWKSCache`
- `google.(*JWKSCache).VerifyGoogleIDToken(ctx, idToken string) (sub, email string, err error)`
- `service.NewAuthService(userRepo, authRepo, jwksCache, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService`
- `service.(*AuthService).GuestLogin(ctx) (TokenPair, error)`
- `service.(*AuthService).GoogleLogin(ctx, idToken string) (TokenPair, error)`
- `service.(*AuthService).Refresh(ctx, userID uuid.UUID, providedToken string) (TokenPair, error)`
- `service.(*AuthService).Logout(ctx, userID uuid.UUID) error`

### New Files
- `internal/auth/google/jwks.go`
- `internal/auth/service/auth_service.go`
