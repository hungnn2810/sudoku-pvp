---
phase: 02-auth-identity
plan: "02"
type: execute
wave: 2
depends_on:
  - 02-01
files_modified:
  - internal/auth/jwt/jwt.go
  - internal/auth/username/generator.go
autonomous: true
requirements:
  - REQ-auth-jwt-middleware
  - REQ-001

must_haves:
  truths:
    - "SignJWT produces a valid HS256 compact token from a Claims struct and a secret byte slice"
    - "ValidateJWT parses the compact token and returns *Claims on success; returns wrapped error on invalid signature, expired token, or wrong algorithm"
    - "Claims embeds jwt.RegisteredClaims and carries Sub uuid.UUID and Role string; no username or guest flag (per D-05)"
    - "username.Generate() returns a string matching the pattern AdjectiveNounNumber (e.g. SwiftKing42) using embedded word lists only"
  artifacts:
    - path: "internal/auth/jwt/jwt.go"
      provides: "Claims, SignJWT, ValidateJWT"
      exports: ["Claims", "SignJWT", "ValidateJWT"]
    - path: "internal/auth/username/generator.go"
      provides: "Generate() string"
      exports: ["Generate"]
  key_links:
    - from: "internal/auth/jwt/jwt.go"
      to: "internal/auth/service/auth_service.go (wave 3)"
      via: "jwt.SignJWT + jwt.ValidateJWT called in service methods"
      pattern: "jwt\\.SignJWT|jwt\\.ValidateJWT"
    - from: "internal/auth/username/generator.go"
      to: "internal/auth/repository/user_repo.go (wave 3)"
      via: "username.Generate() called in collision-retry loop"
      pattern: "username\\.Generate"
---

<objective>
Implement the JWT utility package (internal/auth/jwt/) and the username generator utility (internal/auth/username/). These are pure-logic packages with no I/O dependencies.

Purpose: JWT sign/validate is the cryptographic core consumed by both the REST middleware and the WS middleware (D-20). It must be implemented before any service or middleware can be written. The username generator is required by the guest login path in the repository layer.

Output: internal/auth/jwt/jwt.go exporting Claims, SignJWT, ValidateJWT; internal/auth/username/generator.go exporting Generate(); go.mod updated with github.com/golang-jwt/jwt/v5.
</objective>

<execution_context>
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-PATTERNS.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\internal\redis\keys.go
@J:\sources\sudoku-pvp\internal\config\config.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add golang-jwt dependency and implement internal/auth/jwt/jwt.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\go.mod (verify github.com/golang-jwt/jwt/v5 is absent — must run go get before writing the file)
    - J:\sources\sudoku-pvp\internal\redis\keys.go (package-level pure function pattern — same structure for jwt package)
  </read_first>
  <files>internal/auth/jwt/jwt.go, go.mod, go.sum</files>
  <action>
Step 1 — Add the dependency (REQUIRED before writing the file):
  cd J:/sources/sudoku-pvp && go get github.com/golang-jwt/jwt/v5

This updates go.mod and go.sum. Confirm the dependency appears in go.mod after running.

Step 2 — Create internal/auth/jwt/jwt.go. Package name is "jwt" (matches directory name). Imports needed: "fmt", "time", "github.com/golang-jwt/jwt/v5", "github.com/google/uuid".

Define Claims struct embedding jwt.RegisteredClaims (per D-05 — no username or guest flag):
  type Claims struct {
      Sub  uuid.UUID `json:"sub"`
      Role string    `json:"role"`
      jwt.RegisteredClaims
  }

Define SignJWT with this exact signature:
  func SignJWT(claims Claims, secret []byte) (string, error)

Implementation rules:
  - Use jwt.NewWithClaims(jwt.SigningMethodHS256, claims) — algorithm HS256 per D-04
  - Call token.SignedString(secret)
  - Wrap error: fmt.Errorf("jwt sign: %w", err)

Define ValidateJWT with this exact signature:
  func ValidateJWT(tokenStr string, secret []byte) (*Claims, error)

Implementation rules:
  - Use jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) { ... })
  - In the key func, verify t.Method == jwt.SigningMethodHS256. If not, return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
  - Return the secret as the key on match
  - After parsing: check err != nil — wrap as fmt.Errorf("jwt validate: %w", err)
  - Extract claims with ok assertion on token.Claims.(*Claims)
  - Verify token.Valid before returning
  - Return (*Claims, nil) on success

Also define a helper used by tests and service:
  func NewAccessClaims(userID uuid.UUID, role string, ttl time.Duration) Claims

  Sets Sub = userID, Role = role, RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().UTC().Add(ttl)), RegisteredClaims.IssuedAt = jwt.NewNumericDate(time.Now().UTC()).

Do not put fenced code blocks in this action. The above prose describes the complete implementation contract.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/jwt/... && go vet ./internal/auth/jwt/...</automated>
  </verify>
  <acceptance_criteria>
    - `go get github.com/golang-jwt/jwt/v5` exits 0 and `grep "golang-jwt/jwt/v5" go.mod` finds the entry
    - `go build ./internal/auth/jwt/...` exits 0
    - `go vet ./internal/auth/jwt/...` exits 0 (no suspicious constructs)
    - `grep "SigningMethodHS256" internal/auth/jwt/jwt.go` confirms HS256 is used (D-04)
    - `grep "Sub.*uuid.UUID" internal/auth/jwt/jwt.go` confirms Sub is uuid.UUID (D-05)
    - `grep "username\|guest" internal/auth/jwt/jwt.go` returns nothing — no username or guest flag in Claims (D-05)
    - `grep "jwt sign:" internal/auth/jwt/jwt.go` and `grep "jwt validate:" internal/auth/jwt/jwt.go` confirm error wrapping pattern
  </acceptance_criteria>
  <done>jwt.go compiles cleanly; Claims has Sub+Role+RegisteredClaims; SignJWT and ValidateJWT exported; go.mod has golang-jwt/jwt/v5.</done>
</task>

<task type="auto">
  <name>Task 2: Implement internal/auth/username/generator.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\redis\keys.go (pure function pattern — same no-struct approach for generator)
  </read_first>
  <files>internal/auth/username/generator.go</files>
  <action>
Create internal/auth/username/generator.go. Package name is "username". Imports: "fmt", "math/rand/v2".

Define two package-level var slices (embedded word lists, per D-15 — no external library):
  var adjectives = []string{ at least 20 entries including: "Swift", "Bold", "Calm", "Dark", "Epic", "Iron", "Jade", "Keen", "Lone", "Mist", "Nova", "Onyx", "Pure", "Rune", "Sage", "Tide", "Umber", "Void", "Wild", "Zen" }
  var nouns      = []string{ at least 20 entries including: "King", "Wolf", "Fox", "Bear", "Hawk", "Crow", "Dragon", "Eagle", "Falcon", "Ghost", "Hunter", "Knight", "Lion", "Monk", "Ninja", "Oracle", "Panda", "Quest", "Rider", "Sage" }

Define Generate function:
  func Generate() string

Implementation rules:
  - Pick adjective: adjectives[rand.IntN(len(adjectives))]
  - Pick noun: nouns[rand.IntN(len(nouns))]
  - Pick number: 10 + rand.IntN(90)  (range 10-99 for two-digit number, keeping name short per D-15)
  - Return fmt.Sprintf("%s%s%d", adj, noun, num)

This function is pure — no IO, no context. Collision retry is handled by the caller (UserRepo.CreateGuestUser) which calls Generate() in a loop.

No other exported symbols needed in this package.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/username/... && go vet ./internal/auth/username/...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/username/...` exits 0
    - `go vet ./internal/auth/username/...` exits 0
    - `grep "func Generate" internal/auth/username/generator.go` confirms exported function
    - `grep "math/rand/v2" internal/auth/username/generator.go` confirms correct rand package (not deprecated math/rand)
    - `grep "rand.IntN" internal/auth/username/generator.go` finds the call (not deprecated Intn)
    - Length of adjectives slice >= 20: `grep -o '"[A-Za-z]*"' internal/auth/username/generator.go | grep -c .` — at least 40 strings total (20 adj + 20 noun)
    - No import for any external dictionary/word-list library (only stdlib)
  </acceptance_criteria>
  <done>generator.go compiles; Generate() returns AdjectiveNounNumber format using embedded word lists; no external library dependency.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| caller → ValidateJWT | Token string arrives from untrusted HTTP header; ValidateJWT must reject invalid signatures, expired tokens, and wrong algorithms before returning Claims |
| ValidateJWT → jwt.ParseWithClaims | Library boundary: signing method must be explicitly checked inside the key func to prevent alg=none attacks |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-02-02-01 | Spoofing | ValidateJWT — alg:none / algorithm confusion | mitigate | Key func checks t.Method == jwt.SigningMethodHS256 and returns error on any other value; prevents alg:none and RSA confusion attacks |
| T-02-02-02 | Tampering | SignJWT — weak secret | accept | Secret strength is an operational concern (documented in .env.example as "change-me-in-production"); enforced by Load() non-empty guard in 02-01 |
| T-02-02-03 | Information Disclosure | Claims — no PII in token body | accept | Claims carries only Sub (UUID) and Role string per D-05; no email, username, or guest flag embedded |
| T-02-02-04 | Elevation of Privilege | Role claim in JWT | mitigate | Role is set at token issuance only (service layer); handlers must read role from c.Get("role") set by middleware, never from raw JWT re-parsed in handlers |
| T-02-02-SC | Tampering | go get golang-jwt/jwt/v5 | mitigate | Verify package on pkg.go.dev before executing go get; golang-jwt/jwt is the official successor to dgrijalva/jwt-go (well-known, high-integrity package) |
</threat_model>

<verification>
1. `cd J:/sources/sudoku-pvp && go build ./internal/auth/...` — both jwt and username packages compile
2. `grep "golang-jwt/jwt/v5" J:/sources/sudoku-pvp/go.mod` — dependency registered
3. `grep "SigningMethodHS256" J:/sources/sudoku-pvp/internal/auth/jwt/jwt.go` — algorithm locked to HS256
4. `grep "unexpected signing method" J:/sources/sudoku-pvp/internal/auth/jwt/jwt.go` — alg confusion guard present
5. `grep "func Generate" J:/sources/sudoku-pvp/internal/auth/username/generator.go` — exported
</verification>

<success_criteria>
- go.mod includes github.com/golang-jwt/jwt/v5
- internal/auth/jwt/jwt.go exports Claims (Sub uuid.UUID, Role string), SignJWT, ValidateJWT, NewAccessClaims
- ValidateJWT rejects wrong algorithm, expired tokens, invalid signatures
- internal/auth/username/generator.go exports Generate() returning AdjectiveNounNumber format
- Both packages build and vet cleanly
</success_criteria>

<output>
Create .planning/phases/02-auth-identity/02-02-SUMMARY.md when done.
</output>

## Artifacts This Phase Produces

### New Types
- `jwt.Claims` struct (internal/auth/jwt/jwt.go) — fields: Sub uuid.UUID `json:"sub"`, Role string `json:"role"`, embedded jwt.RegisteredClaims

### New Functions
- `jwt.SignJWT(claims Claims, secret []byte) (string, error)` — HS256 signing
- `jwt.ValidateJWT(tokenStr string, secret []byte) (*Claims, error)` — parse + algorithm check + validity check
- `jwt.NewAccessClaims(userID uuid.UUID, role string, ttl time.Duration) Claims` — constructs Claims with exp/iat
- `username.Generate() string` — returns e.g. "SwiftKing42"

### Modified Files
- `go.mod` — adds require github.com/golang-jwt/jwt/v5
- `go.sum` — updated with jwt/v5 hashes

### New Files
- `internal/auth/jwt/jwt.go`
- `internal/auth/username/generator.go`
