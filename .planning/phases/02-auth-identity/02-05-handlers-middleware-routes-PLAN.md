---
phase: 02-auth-identity
plan: "05"
type: execute
wave: 4
depends_on:
  - 02-04
files_modified:
  - internal/auth/handler/auth_handler.go
  - internal/middleware/auth.go
  - internal/app/app.go
autonomous: true
requirements:
  - REQ-auth-guest
  - REQ-auth-google
  - REQ-auth-refresh
  - REQ-auth-jwt-middleware
  - REQ-009

must_haves:
  truths:
    - "POST /api/v1/auth/guest returns 200 with {accessToken, refreshToken} for valid request"
    - "POST /api/v1/auth/google returns 200 with {accessToken, refreshToken} for valid Google ID token"
    - "POST /api/v1/auth/refresh returns 200 with new {accessToken, refreshToken} for valid refresh token body"
    - "POST /api/v1/auth/logout returns 204 when authenticated; clears refresh token from Redis"
    - "GET /ws/connect returns HTTP 401 before WebSocket upgrade if Authorization header is missing or token is invalid"
    - "JWT middleware sets c.Set(\"userId\", claims.Sub.String()) and c.Set(\"role\", claims.Role) in Gin context (D-21)"
    - "ValidateJWT is called by the same code path for both REST and WS middleware (D-20)"
    - "Auth middleware registered AFTER tracing middleware in Gin chain"
    - "All handlers are thin — no business logic in handler functions; delegate to AuthService"
  artifacts:
    - path: "internal/auth/handler/auth_handler.go"
      provides: "AuthHandler with GuestLogin, GoogleLogin, Refresh, Logout handlers"
      exports: ["AuthHandler", "NewAuthHandler"]
    - path: "internal/middleware/auth.go"
      provides: "JWTMiddleware (REST) and WSJWTMiddleware (WebSocket handshake)"
      exports: ["JWTMiddleware", "WSJWTMiddleware"]
    - path: "internal/app/app.go"
      provides: "Wires auth dependencies; registers auth routes and protected route groups"
  key_links:
    - from: "internal/auth/handler/auth_handler.go"
      to: "internal/app/app.go"
      via: "AuthHandler registered on Gin router"
      pattern: "AuthHandler"
    - from: "internal/middleware/auth.go"
      to: "internal/app/app.go"
      via: "middleware.JWTMiddleware applied to protected group"
      pattern: "JWTMiddleware"
---

<objective>
Wire the auth HTTP surface: Gin handlers for the four auth endpoints, JWT middleware for REST and WebSocket, and route registration in app.go. This is the final implementation wave — after this plan, all auth endpoints are reachable and the WebSocket handshake is authenticated.

Purpose: Auth handlers delegate to AuthService; middleware validates tokens using ValidateJWT from internal/auth/jwt/. App.go is updated to wire all auth dependencies through the composition root and register routes.

Output: AuthHandler with four Gin handlers; JWTMiddleware and WSJWTMiddleware in internal/middleware/; app.go wired with auth dependencies and protected route groups.
</objective>

<execution_context>
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-PATTERNS.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\docs\API_CONTRACT.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Implement internal/auth/handler/auth_handler.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\common\response.go (OK and Error helper — use these in every handler)
    - J:\sources\sudoku-pvp\internal\common\errors.go (sentinel errors to map to HTTP status codes)
    - J:\sources\sudoku-pvp\internal\middleware\logger.go (c.Request.Context() pattern — NOT c.Copy())
    - J:\sources\sudoku-pvp\internal\auth\service\auth_service.go (GuestLogin, GoogleLogin, Refresh, Logout signatures)
    - J:\sources\sudoku-pvp\docs\API_CONTRACT.md (request/response shapes for auth endpoints)
  </read_first>
  <files>internal/auth/handler/auth_handler.go</files>
  <action>
Create internal/auth/handler/auth_handler.go. Package name: "handler". Imports: "net/http", "github.com/gin-gonic/gin", "github.com/google/uuid", "sudoku-pvp/internal/auth/service", "sudoku-pvp/internal/common", "sudoku-pvp/internal/logger".

Define AuthHandler struct:
  type AuthHandler struct {
      svc *service.AuthService
  }

Define constructor:
  func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

Define GuestLogin handler:
  func (h *AuthHandler) GuestLogin(c *gin.Context)
  1. ctx := c.Request.Context()
  2. pair, err := h.svc.GuestLogin(ctx)
  3. If err != nil: log error via logger.FromCtx(ctx).Error().Err(err).Msg("guest login failed"); common.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "guest login failed"); return
  4. common.OK(c, pair)

Define GoogleLogin handler:
  func (h *AuthHandler) GoogleLogin(c *gin.Context)
  1. ctx := c.Request.Context()
  2. Parse request body: var req struct{ IDToken string `json:"idToken"` }; if c.ShouldBindJSON(&req) != nil or req.IDToken == "": common.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "idToken required"); return
  3. pair, err := h.svc.GoogleLogin(ctx, req.IDToken)
  4. On error: log + common.Error(c, 401, "AUTH_FAILED", "google authentication failed"); return
  5. common.OK(c, pair)

Define Refresh handler:
  func (h *AuthHandler) Refresh(c *gin.Context)
  1. ctx := c.Request.Context()
  2. Parse body: var req struct{ UserID string `json:"userId"`; RefreshToken string `json:"refreshToken"` }; validate both non-empty; return 400 on failure
  3. Parse userID: uuid.Parse(req.UserID); return 400 on invalid UUID
  4. pair, err := h.svc.Refresh(ctx, userID, req.RefreshToken)
  5. On error: common.Error(c, 401, "AUTH_FAILED", "invalid or expired refresh token"); return
  6. common.OK(c, pair)

Define Logout handler (requires auth middleware — userID available from context):
  func (h *AuthHandler) Logout(c *gin.Context)
  1. ctx := c.Request.Context()
  2. Raw userIDStr, _ := c.Get("userId"); userID, err := uuid.Parse(userIDStr.(string)); if err != nil: common.Error(c, 400, "INVALID_TOKEN", "invalid user id in token"); return
  3. h.svc.Logout(ctx, userID) — ignore error (idempotent)
  4. c.Status(http.StatusNoContent)

All handlers use c.Request.Context() — never c.Copy(). All error logging uses logger.FromCtx(ctx).
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/handler/... && go vet ./internal/auth/handler/...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/handler/...` exits 0
    - `go vet ./internal/auth/handler/...` exits 0
    - `grep "func (h \*AuthHandler) GuestLogin" internal/auth/handler/auth_handler.go` finds the handler
    - `grep "func (h \*AuthHandler) GoogleLogin" internal/auth/handler/auth_handler.go` finds the handler
    - `grep "func (h \*AuthHandler) Refresh" internal/auth/handler/auth_handler.go` finds the handler
    - `grep "func (h \*AuthHandler) Logout" internal/auth/handler/auth_handler.go` finds the handler
    - `grep "c.Request.Context()" internal/auth/handler/auth_handler.go` appears at least 4 times (one per handler)
    - `grep "c.Copy()" internal/auth/handler/auth_handler.go` returns nothing (never use c.Copy())
    - `grep "logger.FromCtx" internal/auth/handler/auth_handler.go` confirms structured logging (not fmt.Println)
    - `grep "http.StatusNoContent" internal/auth/handler/auth_handler.go` confirms 204 on logout
  </acceptance_criteria>
  <done>AuthHandler compiles; all four handlers delegate to service; context propagated; structured logging; no business logic in handlers.</done>
</task>

<task type="auto">
  <name>Task 2: Implement internal/middleware/auth.go (REST + WS JWT middleware)</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\middleware\tracing.go (Gin middleware pattern: gin.HandlerFunc return, no struct needed)
    - J:\sources\sudoku-pvp\internal\middleware\logger.go (c.Request.Context() pattern; middleware registration order)
    - J:\sources\sudoku-pvp\internal\auth\jwt\jwt.go (ValidateJWT signature — reads exact parameters before calling)
    - J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md (D-18 to D-21: WS auth header, HTTP 401 before upgrade, shared ValidateJWT, Gin context keys)
  </read_first>
  <files>internal/middleware/auth.go</files>
  <action>
Create internal/middleware/auth.go. Package name: "middleware". Imports: "net/http", "strings", "github.com/gin-gonic/gin", "sudoku-pvp/internal/auth/jwt" (aliased as authjwt to avoid name collision with stdlib).

Define JWTMiddleware (REST auth):
  func JWTMiddleware(secret []byte) gin.HandlerFunc
  Returns a gin.HandlerFunc that:
  1. Reads Authorization header: c.GetHeader("Authorization")
  2. Validates prefix "Bearer ": if not present or no token after split: c.JSON(http.StatusUnauthorized, gin.H{"code": "MISSING_TOKEN", "message": "authorization header required"}); c.Abort(); return
  3. token := strings.TrimPrefix(authHeader, "Bearer ")
  4. claims, err := authjwt.ValidateJWT(token, secret)
  5. If err != nil: c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_TOKEN", "message": "invalid or expired token"}); c.Abort(); return
  6. c.Set("userId", claims.Sub.String())
  7. c.Set("role", claims.Role)
  8. c.Next()

Define WSJWTMiddleware (WebSocket handshake auth):
  func WSJWTMiddleware(secret []byte) gin.HandlerFunc
  Returns a gin.HandlerFunc that:
  1. Same Authorization header extraction as JWTMiddleware (identical code path — D-20: shared validation)
  2. Same ValidateJWT call
  3. Same c.Set("userId") and c.Set("role") on success
  4. On failure: c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_TOKEN", "message": "websocket auth failed"}); c.Abort(); return
  5. c.Next() on success — WebSocket upgrade happens in the next handler

Note: Both middleware functions must NOT call c.Copy(). Use c.Request.Context() for any context-aware logging if added later. Both call c.Abort() on auth failure to prevent downstream handlers from running.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/middleware/... && go vet ./internal/middleware/...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/middleware/...` exits 0
    - `go vet ./internal/middleware/...` exits 0
    - `grep "func JWTMiddleware" internal/middleware/auth.go` finds the exported function
    - `grep "func WSJWTMiddleware" internal/middleware/auth.go` finds the exported function
    - `grep "c.Set(\"userId\"" internal/middleware/auth.go` appears at least twice (once in each middleware)
    - `grep "c.Set(\"role\"" internal/middleware/auth.go` appears at least twice
    - `grep "c.Abort()" internal/middleware/auth.go` appears at least twice (abort on auth failure in each middleware)
    - `grep "authjwt.ValidateJWT\|jwt.ValidateJWT" internal/middleware/auth.go` confirms shared ValidateJWT call (D-20)
    - `grep "c.Copy()" internal/middleware/auth.go` returns nothing
  </acceptance_criteria>
  <done>auth.go compiles; JWTMiddleware and WSJWTMiddleware both use ValidateJWT; Gin context populated with userId and role; c.Abort() called on failure.</done>
</task>

<task type="auto">
  <name>Task 3: Wire auth in internal/app/app.go and register routes</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\app\app.go (FULL FILE — understand current App struct fields, New() wiring sequence, setupRouter() middleware chain and route group structure before making any changes)
    - J:\sources\sudoku-pvp\internal\config\config.go (Auth AuthConfig field added in plan 02-01 — verify field name before accessing cfg.Auth)
    - J:\sources\sudoku-pvp\internal\auth\service\auth_service.go (NewAuthService constructor signature)
    - J:\sources\sudoku-pvp\internal\auth\handler\auth_handler.go (NewAuthHandler constructor signature)
    - J:\sources\sudoku-pvp\internal\auth\repository\auth_repo.go (NewAuthRepo constructor)
    - J:\sources\sudoku-pvp\internal\auth\repository\user_repo.go (NewUserRepo constructor)
    - J:\sources\sudoku-pvp\internal\auth\google\jwks.go (NewJWKSCache constructor)
  </read_first>
  <files>internal/app/app.go</files>
  <action>
Modify internal/app/app.go:

1. Add imports needed for auth wiring (add to import block, do not remove existing imports):
   - "sudoku-pvp/internal/auth/google" (aliased as authgoogle)
   - "sudoku-pvp/internal/auth/handler"
   - "sudoku-pvp/internal/auth/repository"
   - "sudoku-pvp/internal/auth/service"
   - "sudoku-pvp/internal/middleware"

2. Add auth fields to the App struct (after existing fields):
   - authHandler *handler.AuthHandler

3. In New() function, after step 4 (Redis client connected), add step 4.5 — wire auth:
   jwksCache := authgoogle.NewJWKSCache()
   authRepo := repository.NewAuthRepo(redisClient)
   userRepo := repository.NewUserRepo(pool)
   authSvc := service.NewAuthService(
       userRepo,
       authRepo,
       jwksCache,
       cfg.Auth.JWTSecret,
       cfg.Auth.AccessTokenTTL,
       cfg.Auth.RefreshTokenTTL,
   )
   a.authHandler = handler.NewAuthHandler(authSvc)

   Important: wire auth AFTER redis and postgres are connected (steps 3 and 4), BEFORE setupRouter() call (step 6).

4. In setupRouter() function, after the existing v1.GET("/health", healthHandler) line:

   Register auth routes (public — no JWT middleware):
   auth := v1.Group("/auth")
   auth.POST("/guest", a.authHandler.GuestLogin)
   auth.POST("/google", a.authHandler.GoogleLogin)
   auth.POST("/refresh", a.authHandler.Refresh)

   Register protected auth route (requires JWT):
   protected := v1.Group("")
   protected.Use(middleware.JWTMiddleware([]byte(a.cfg.Auth.JWTSecret)))
   protected.POST("/auth/logout", a.authHandler.Logout)

   Register WebSocket endpoint (JWT middleware before upgrade):
   r.GET("/ws/connect", middleware.WSJWTMiddleware([]byte(a.cfg.Auth.JWTSecret)), wsPlaceholderHandler)

   Also add wsPlaceholderHandler as a package-level function in app.go (Phase 5 will replace with real WS handler):
   func wsPlaceholderHandler(c *gin.Context) {
       c.JSON(http.StatusServiceUnavailable, gin.H{"message": "websocket not yet implemented"})
   }

5. The JWTMiddleware is registered on the "protected" group AFTER Recovery, Tracing, and ZerologLogger (those are global on the engine). This satisfies D-20 and the CONTEXT.md middleware order requirement.

Note: Convert cfg.Auth.JWTSecret to []byte when passing to JWTMiddleware and WSJWTMiddleware. Store as []byte in App struct if needed for efficiency: jwtSecretBytes := []byte(cfg.Auth.JWTSecret).
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./... && go vet ./...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./...` exits 0 (entire project compiles)
    - `go vet ./...` exits 0
    - `grep "authHandler" internal/app/app.go` finds the field on App struct
    - `grep "POST.*auth/guest" internal/app/app.go` confirms route registration
    - `grep "POST.*auth/google" internal/app/app.go` confirms route registration
    - `grep "POST.*auth/refresh" internal/app/app.go` confirms route registration
    - `grep "POST.*auth/logout" internal/app/app.go` confirms route registration
    - `grep "GET.*ws/connect" internal/app/app.go` confirms WebSocket route registration
    - `grep "JWTMiddleware\|WSJWTMiddleware" internal/app/app.go` confirms middleware applied
    - `grep "NewJWKSCache\|NewAuthRepo\|NewUserRepo\|NewAuthService\|NewAuthHandler" internal/app/app.go` — all 5 constructors present
    - Running server responds to GET /api/v1/health with 200 (smoke test — existing endpoint must still work)
  </acceptance_criteria>
  <done>app.go compiles with all auth dependencies wired; four auth routes registered; WebSocket placeholder registered; JWT middleware applied to protected group.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| HTTP client → JWTMiddleware | Authorization header is untrusted; validated via ValidateJWT before setting Gin context values |
| WebSocket client → WSJWTMiddleware | Authorization header validated before HTTP upgrade; invalid token = 401 response, upgrade never occurs (D-19) |
| Gin context → Logout handler | userId read from Gin context set by JWTMiddleware — not from client body; not re-parsed from raw JWT in handler |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-02-05-01 | Spoofing | JWTMiddleware — missing token | mitigate | c.Abort() before c.Next(); downstream handlers never execute on auth failure |
| T-02-05-02 | Elevation of Privilege | Logout handler — reads userId from Gin context | mitigate | userId set by JWTMiddleware from validated JWT claims; handler cannot be reached without valid token; client body userId not trusted |
| T-02-05-03 | Spoofing | WSJWTMiddleware — upgrade before auth | mitigate | WSJWTMiddleware registered before WS upgrade handler; c.Abort() on failure prevents upgrade (D-19) |
| T-02-05-04 | Information Disclosure | POST /auth/google — idToken in request body | accept | idToken is short-lived (1hr); transmitted over TLS; not logged in handler |
| T-02-05-05 | Tampering | setupRouter() — middleware order | mitigate | Auth middleware applied to specific route groups after global chain (Recovery → Tracing → ZerologLogger); not global to avoid protecting /metrics or /health |
</threat_model>

<verification>
1. `cd J:/sources/sudoku-pvp && go build ./...` — entire project compiles
2. `go vet ./...` — no suspicious constructs
3. `grep "c.Abort()" internal/middleware/auth.go` — abort called on auth failure
4. `grep "POST.*auth/guest\|POST.*auth/google\|POST.*auth/refresh\|POST.*auth/logout" internal/app/app.go` — all 4 routes registered
5. `grep "ws/connect" internal/app/app.go` — WS route registered
</verification>

<success_criteria>
- Entire project compiles with `go build ./...`
- POST /api/v1/auth/guest, /auth/google, /auth/refresh, /auth/logout routes registered
- GET /ws/connect registered with WSJWTMiddleware
- JWTMiddleware and WSJWTMiddleware set userId and role in Gin context
- Auth middleware correctly registered after tracing middleware
- GET /api/v1/health still responds 200 (no regression)
</success_criteria>

<output>
Create .planning/phases/02-auth-identity/02-05-SUMMARY.md when done.
</output>

## Artifacts This Phase Produces

### New Types
- `handler.AuthHandler` struct (internal/auth/handler/auth_handler.go) — fields: svc *service.AuthService

### New Functions
- `handler.NewAuthHandler(svc *service.AuthService) *AuthHandler`
- `handler.(*AuthHandler).GuestLogin(c *gin.Context)`
- `handler.(*AuthHandler).GoogleLogin(c *gin.Context)`
- `handler.(*AuthHandler).Refresh(c *gin.Context)`
- `handler.(*AuthHandler).Logout(c *gin.Context)`
- `middleware.JWTMiddleware(secret []byte) gin.HandlerFunc`
- `middleware.WSJWTMiddleware(secret []byte) gin.HandlerFunc`
- `app.wsPlaceholderHandler(c *gin.Context)` (package-level, replaced in Phase 5)

### Modified Files
- `internal/app/app.go` — adds authHandler field; wires JWKSCache, AuthRepo, UserRepo, AuthService, AuthHandler; registers 5 routes; applies middleware to protected group

### Routes Registered
- POST /api/v1/auth/guest (public)
- POST /api/v1/auth/google (public)
- POST /api/v1/auth/refresh (public)
- POST /api/v1/auth/logout (protected — JWTMiddleware)
- GET /ws/connect (WSJWTMiddleware + placeholder)
