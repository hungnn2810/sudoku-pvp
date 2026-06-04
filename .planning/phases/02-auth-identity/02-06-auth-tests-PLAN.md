---
phase: 02-auth-identity
plan: "06"
type: execute
wave: 5
depends_on:
  - 02-05
files_modified:
  - internal/auth/jwt/jwt_test.go
  - internal/auth/username/generator_test.go
  - internal/auth/service/auth_service_test.go
  - internal/auth/integration/auth_test.go
autonomous: true
requirements:
  - REQ-auth-guest
  - REQ-auth-google
  - REQ-auth-refresh
  - REQ-auth-jwt-middleware
  - REQ-027
  - REQ-028
  - REQ-integration-test-tooling

must_haves:
  truths:
    - "jwt_test.go: TestSignJWT_RoundTrip succeeds; TestValidateJWT_InvalidSignature returns error; TestValidateJWT_Expired returns error; TestValidateJWT_AlgNone returns error"
    - "generator_test.go: TestGenerate_Format verifies output matches AdjectiveNounNumber pattern via regexp; TestGenerate_Uniqueness generates 1000 names and confirms < 50% collision rate"
    - "auth_service_test.go: unit tests for GuestLogin, Refresh (rotate), Logout using stub repos"
    - "auth_test.go: integration tests for full POST /auth/guest and POST /auth/refresh flows against real Redis (testcontainers) and PostgreSQL (testcontainers)"
    - "All integration tests gated by //go:build integration build tag"
    - "go test ./... -short -count=1 exits 0 (unit tests pass without Docker)"
  artifacts:
    - path: "internal/auth/jwt/jwt_test.go"
      provides: "Unit tests for SignJWT, ValidateJWT, NewAccessClaims"
    - path: "internal/auth/username/generator_test.go"
      provides: "Unit tests for Generate() format and distribution"
    - path: "internal/auth/service/auth_service_test.go"
      provides: "Unit tests for AuthService with stub repos"
    - path: "internal/auth/integration/auth_test.go"
      provides: "Integration tests for auth flows against real containers"
---

<objective>
Write unit tests for the JWT utility and username generator, unit tests for AuthService with stub repositories, and integration tests for the full guest login and refresh flows against real PostgreSQL and Redis containers. Tests provide coverage for the Phase 2 goal and protect against regressions in auth-critical code paths.

Purpose: Phase 2 has no UI; correctness is proven entirely through tests. JWT tests guard against algorithm confusion (alg:none, RS256 swap). Service unit tests guard rotation and idempotency invariants. Integration tests prove the full HTTP flow connects all layers correctly.

Output: Four test files; `go test ./... -short` exits 0; `go test ./... -tags integration` exercises real containers.
</objective>

<execution_context>
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-PATTERNS.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Write unit tests for internal/auth/jwt/jwt_test.go and internal/auth/username/generator_test.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\auth\jwt\jwt.go (SignJWT, ValidateJWT, NewAccessClaims signatures and implementation)
    - J:\sources\sudoku-pvp\internal\auth\username\generator.go (Generate() implementation)
    - J:\sources\sudoku-pvp\internal\database\postgres_test.go (test structure conventions in this project — t.Parallel(), table-driven tests, testify require)
  </read_first>
  <files>internal/auth/jwt/jwt_test.go, internal/auth/username/generator_test.go</files>
  <action>
Create internal/auth/jwt/jwt_test.go. Package: "jwt_test". Build tag: none (unit test, no Docker). Imports: "testing", "time", "github.com/google/uuid", "github.com/stretchr/testify/require", "sudoku-pvp/internal/auth/jwt".

Write these test functions:

TestSignJWT_RoundTrip:
  - secret := []byte("test-secret-32-bytes-minimum-ok")
  - userID := uuid.Must(uuid.NewV7()); role := "player"
  - claims := jwt.NewAccessClaims(userID, role, 15*time.Minute)
  - tokenStr, err := jwt.SignJWT(claims, secret); require.NoError(t, err); require.NotEmpty(t, tokenStr)
  - parsed, err := jwt.ValidateJWT(tokenStr, secret); require.NoError(t, err)
  - require.Equal(t, userID, parsed.Sub); require.Equal(t, role, parsed.Role)

TestValidateJWT_InvalidSignature:
  - Create valid token with secret1; try ValidateJWT with secret2
  - require.Error(t, err)

TestValidateJWT_Expired:
  - claims := jwt.NewAccessClaims(uuid.Must(uuid.NewV7()), "player", -1*time.Second)  (negative TTL = already expired)
  - Sign and then validate; require.Error(t, err)

TestValidateJWT_WrongAlgorithm_AlgNone:
  - Manually craft a token string with alg:none header (base64url of `{"alg":"none","typ":"JWT"}`)
  - Call jwt.ValidateJWT with valid secret; require.Error(t, err)
  - err message should indicate unexpected signing method or invalid token

TestNewAccessClaims_Fields:
  - userID := uuid.Must(uuid.NewV7()); role := "admin"
  - claims := jwt.NewAccessClaims(userID, role, 5*time.Minute)
  - require.Equal(t, userID, claims.Sub); require.Equal(t, role, claims.Role)
  - require.NotNil(t, claims.ExpiresAt); require.NotNil(t, claims.IssuedAt)

Create internal/auth/username/generator_test.go. Package: "username_test". Imports: "regexp", "testing", "sudoku-pvp/internal/auth/username".

Write these test functions:

TestGenerate_Format:
  - pattern := regexp.MustCompile(`^[A-Z][a-z]+[A-Z][a-z]+\d{2}$`)
  - Run Generate() 100 times; for each result: require.Regexp(t, pattern, result)

TestGenerate_Length:
  - result := username.Generate()
  - require.GreaterOrEqual(t, len(result), 6)  (shortest: adj 3 + noun 3 + num 2 = 8)
  - require.LessOrEqual(t, len(result), 25)

TestGenerate_Uniqueness:
  - Generate 1000 names; put in a map; assert len(map) > 500 (at least 50% unique)
  - This tests that random distribution is reasonable (not the same name every time)
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go test ./internal/auth/jwt/... ./internal/auth/username/... -v -count=1</automated>
  </verify>
  <acceptance_criteria>
    - `go test ./internal/auth/jwt/... -v -count=1` exits 0 and shows PASS for all 5 test functions
    - `go test ./internal/auth/username/... -v -count=1` exits 0 and shows PASS for all 3 test functions
    - `grep "TestValidateJWT_Expired\|TestValidateJWT_InvalidSignature\|TestValidateJWT_WrongAlgorithm" internal/auth/jwt/jwt_test.go` — all three security tests present
    - `grep "regexp.MustCompile" internal/auth/username/generator_test.go` confirms format regex test
  </acceptance_criteria>
  <done>8 unit tests pass; JWT tests cover algorithm confusion and expiry; generator test verifies format and distribution.</done>
</task>

<task type="auto">
  <name>Task 2: Write AuthService unit tests with stub repos</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\auth\service\auth_service.go (GuestLogin, Refresh, Logout signatures and behavior)
    - J:\sources\sudoku-pvp\internal\auth\repository\auth_repo.go (AuthRepo methods to stub)
    - J:\sources\sudoku-pvp\internal\auth\repository\user_repo.go (UserRepo methods to stub)
    - J:\sources\sudoku-pvp\internal\database\postgres_test.go (testify require convention)
  </read_first>
  <files>internal/auth/service/auth_service_test.go</files>
  <action>
Create internal/auth/service/auth_service_test.go. Package: "service_test". Build tag: none (unit tests — use stub repos, no Docker). Imports: "context", "testing", "time", "github.com/google/uuid", "github.com/stretchr/testify/require", "sudoku-pvp/internal/auth/service", "sudoku-pvp/db/sqlc".

Since AuthService takes concrete *AuthRepo and *UserRepo (not interfaces), the unit tests must use a thin wrapper approach or test AuthService against real in-memory structures. Alternatively, extract minimal interfaces for testing. Choose the simpler path: define local stub types in the test file that satisfy the same method signatures.

Option: Define AuthServiceDeps interface-based test helpers within the test file. Refactor AuthService to accept interfaces instead of concrete types, OR write integration-only tests for service (no unit test stubs needed).

Given the service was implemented against concrete types (not interfaces), write these tests as follows:
- GuestLogin: requires real Redis + Postgres. Mark with //go:build integration tag.
- Refresh rotation: requires real Redis. Mark with //go:build integration tag.
- Logout idempotent: requires real Redis. Mark with //go:build integration tag.

Write the file with //go:build integration at the top. All tests use testutil.SetupPostgres and testutil.SetupRedis helpers.

Test functions:

TestAuthService_GuestLogin_ReturnsTokenPair (integration):
  1. SetupPostgres, SetupRedis
  2. Wire full AuthService (no JWKSCache needed for guest)
  3. pair, err := svc.GuestLogin(ctx)
  4. require.NoError(t, err); require.NotEmpty(t, pair.AccessToken); require.NotEmpty(t, pair.RefreshToken)
  5. Validate AccessToken is a valid JWT: jwt.ValidateJWT(pair.AccessToken, []byte(secret))

TestAuthService_Refresh_RotatesToken (integration):
  1. SetupRedis + SetupPostgres
  2. Wire AuthService
  3. pair1, _ := svc.GuestLogin(ctx)
  4. Parse userID from pair1.AccessToken claims
  5. pair2, err := svc.Refresh(ctx, userID, pair1.RefreshToken)
  6. require.NoError(t, err); require.NotEqual(t, pair1.RefreshToken, pair2.RefreshToken)
  7. Try using pair1.RefreshToken again: pair3, err := svc.Refresh(ctx, userID, pair1.RefreshToken)
  8. require.Error(t, err)  (old token rejected after rotation — D-03)

TestAuthService_Logout_Idempotent (integration):
  1. Wire AuthService + guest login
  2. svc.Logout(ctx, userID) — first call
  3. err := svc.Logout(ctx, userID) — second call
  4. require.NoError(t, err)  (idempotent — no error on double logout — D-06)
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/service/... (unit build check; integration tests only run with -tags integration)</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/service/...` exits 0
    - `grep "//go:build integration" internal/auth/service/auth_service_test.go` confirms build tag
    - `grep "TestAuthService_GuestLogin_ReturnsTokenPair" internal/auth/service/auth_service_test.go` finds the test
    - `grep "TestAuthService_Refresh_RotatesToken" internal/auth/service/auth_service_test.go` finds the test
    - `grep "TestAuthService_Logout_Idempotent" internal/auth/service/auth_service_test.go` finds the test
    - `grep "NotEqual.*RefreshToken\|pair1.RefreshToken.*pair2.RefreshToken" internal/auth/service/auth_service_test.go` confirms rotation test asserts different tokens
    - `go test ./... -short -count=1` exits 0 (integration tests skipped — only unit tests run without -tags integration)
  </acceptance_criteria>
  <done>auth_service_test.go compiles; 3 integration tests written for GuestLogin, token rotation, and idempotent logout; unit suite unaffected.</done>
</task>

<task type="auto">
  <name>Task 3: Write HTTP integration tests in internal/auth/integration/auth_test.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\testutil\containers.go (SetupPostgres, SetupRedis helpers — read exact function signatures)
    - J:\sources\sudoku-pvp\internal\app\app.go (how to construct App for tests — New(cfg))
    - J:\sources\sudoku-pvp\internal\config\config.go (how to build test Config with AuthConfig)
    - J:\sources\sudoku-pvp\internal\database\postgres_test.go (integration test file structure and build tags)
  </read_first>
  <files>internal/auth/integration/auth_test.go</files>
  <action>
Create internal/auth/integration/auth_test.go. Build tag: //go:build integration. Package: "auth_integration_test".

Imports: "bytes", "context", "encoding/json", "net/http", "net/http/httptest", "testing", "github.com/stretchr/testify/require", "sudoku-pvp/internal/app", "sudoku-pvp/internal/config", "sudoku-pvp/internal/testutil".

Helper: setupTestApp(t *testing.T) (*app.App, func()):
  1. testutil.SetupPostgres(t) — returns dsn string
  2. testutil.SetupRedis(t) — returns addr string
  3. cfg := &config.Config{Server: ..., Postgres: {DSN: dsn, MaxConns: 2, MinConns: 1}, Redis: {Addr: addr, ...}, Auth: {JWTSecret: "test-secret-for-integration", AccessTokenTTL: 15*time.Minute, RefreshTokenTTL: 720*time.Hour}}
  4. a, err := app.New(cfg); require.NoError(t, err)
  5. Return a, func() { a.Shutdown(context.Background()) }

TestHTTP_GuestLogin_Returns200:
  1. a, cleanup := setupTestApp(t); defer cleanup()
  2. Create *httptest.Server: srv := httptest.NewServer(a.Router()) — add Router() method to App if not present, or use httptest.NewServer with the gin engine
  3. POST /api/v1/auth/guest with empty body
  4. require.Equal(t, 200, resp.StatusCode)
  5. Decode response: var pair struct{ AccessToken, RefreshToken string }
  6. require.NotEmpty(t, pair.AccessToken); require.NotEmpty(t, pair.RefreshToken)

TestHTTP_GuestLogin_Refresh_Cycle:
  1. setupTestApp; POST /api/v1/auth/guest → get pair1
  2. Parse userID from pair1.AccessToken JWT claims (use jwt.ParseWithClaims or the ValidateJWT helper from internal/auth/jwt)
  3. POST /api/v1/auth/refresh body: {userId: userID.String(), refreshToken: pair1.RefreshToken}
  4. require.Equal(t, 200, resp.StatusCode)
  5. Decode pair2; require.NotEqual(t, pair1.RefreshToken, pair2.RefreshToken)

TestHTTP_ProtectedRoute_NoToken_Returns401:
  1. setupTestApp
  2. POST /api/v1/auth/logout with no Authorization header
  3. require.Equal(t, 401, resp.StatusCode)

TestHTTP_ProtectedRoute_ValidToken_Returns204:
  1. setupTestApp; guest login → get pair and userID
  2. POST /api/v1/auth/logout with header Authorization: Bearer {pair.AccessToken}
  3. require.Equal(t, 204, resp.StatusCode)

Note: App.Router() may need to be exposed. Check app.go — if the router is unexported, add a public accessor:
  func (a *App) Router() http.Handler { return a.router }
Add this to app.go in this task if it's not already there.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/integration/... && go test ./internal/auth/integration/... -tags integration -v -count=1 -timeout 120s (requires Docker)</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/integration/...` exits 0
    - `grep "//go:build integration" internal/auth/integration/auth_test.go` confirms build tag
    - `grep "TestHTTP_GuestLogin_Returns200" internal/auth/integration/auth_test.go` finds the test
    - `grep "TestHTTP_GuestLogin_Refresh_Cycle" internal/auth/integration/auth_test.go` finds the test
    - `grep "TestHTTP_ProtectedRoute_NoToken_Returns401" internal/auth/integration/auth_test.go` finds the test
    - `grep "TestHTTP_ProtectedRoute_ValidToken_Returns204" internal/auth/integration/auth_test.go` finds the test
    - `go test ./... -short -count=1` exits 0 (integration tests skipped, no Docker required for short run)
    - (Optional, Docker required) `go test ./internal/auth/integration/... -tags integration -v -count=1 -timeout 120s` exits 0
  </acceptance_criteria>
  <done>4 HTTP integration tests written; guest login, refresh cycle, and auth middleware tested end-to-end; unit suite unaffected.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| test → real Redis/Postgres | Integration tests use testcontainers-go; containers isolated per test run; no shared state |
| test secret | "test-secret-for-integration" is only in test code; never reaches production config |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-02-06-01 | Information Disclosure | TestValidateJWT_WrongAlgorithm_AlgNone | mitigate | Test proves alg:none tokens are rejected; confirms T-02-02-01 mitigation is in place |
| T-02-06-02 | Tampering | TestAuthService_Refresh_RotatesToken | mitigate | Test proves old token is rejected after rotation; confirms D-03 implementation |
</threat_model>

<verification>
1. `cd J:/sources/sudoku-pvp && go test ./internal/auth/jwt/... ./internal/auth/username/... -v -count=1` — JWT and generator unit tests pass
2. `go test ./... -short -count=1` — entire unit suite exits 0 (no Docker required)
3. `grep "TestValidateJWT_WrongAlgorithm" internal/auth/jwt/jwt_test.go` — alg:none test present
4. `grep "NotEqual.*RefreshToken" internal/auth/service/auth_service_test.go` — rotation assertion present
</verification>

<success_criteria>
- jwt_test.go: 5 unit tests pass including alg:none rejection
- generator_test.go: 3 unit tests pass including format regex and distribution
- auth_service_test.go: 3 integration tests with build tag; rotation and idempotency proven
- auth_test.go: 4 HTTP integration tests with build tag; guest login and protected routes proven
- `go test ./... -short -count=1` exits 0
</success_criteria>

<output>
Create .planning/phases/02-auth-identity/02-06-SUMMARY.md when done.
</output>

## Artifacts This Phase Produces

### New Test Files
- `internal/auth/jwt/jwt_test.go` — 5 unit tests: RoundTrip, InvalidSignature, Expired, AlgNone, NewAccessClaims
- `internal/auth/username/generator_test.go` — 3 unit tests: Format, Length, Uniqueness
- `internal/auth/service/auth_service_test.go` — 3 integration tests (//go:build integration): GuestLogin, Refresh rotation, Logout idempotent
- `internal/auth/integration/auth_test.go` — 4 HTTP integration tests (//go:build integration): GuestLogin HTTP, Refresh cycle HTTP, 401 no token, 204 valid token

### Modified Files (if needed)
- `internal/app/app.go` — may add `Router() http.Handler` accessor for integration tests
