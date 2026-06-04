---
phase: 02-auth-identity
plan: "02"
subsystem: auth
tags: [jwt, auth, username, security]
dependency_graph:
  requires:
    - 02-01
  provides:
    - internal/auth/jwt/jwt.go (Claims, SignJWT, ValidateJWT, NewAccessClaims)
    - internal/auth/username/generator.go (Generate)
    - go.mod: github.com/golang-jwt/jwt/v5 v5.3.1
  affects:
    - internal/auth/service/auth_service.go (wave 3 — calls SignJWT, ValidateJWT)
    - internal/auth/repository/user_repo.go (wave 3 — calls username.Generate in collision-retry loop)
    - internal/middleware/auth.go (wave 3 — calls ValidateJWT)
tech_stack:
  added:
    - github.com/golang-jwt/jwt/v5 v5.3.1
  patterns:
    - package-level pure function pattern (analog: internal/redis/keys.go)
    - HS256 algorithm lock in key func (T-02-02-01: alg:none / confusion attack mitigation)
    - embedded word lists for username generation (D-15: no external library)
key_files:
  created:
    - internal/auth/jwt/jwt.go
    - internal/auth/username/generator.go
  modified:
    - go.mod
    - go.sum
decisions:
  - "HS256 locked in ValidateJWT key func via t.Method == jwt.SigningMethodHS256 check (D-04, T-02-02-01)"
  - "Claims carries Sub uuid.UUID and Role string only — no username or guest flag (D-05)"
  - "NewAccessClaims sets ExpiresAt and IssuedAt using time.Now().UTC() for consistent UTC timestamps"
  - "Generate() uses math/rand/v2 rand.IntN (not deprecated math/rand Intn)"
  - "25 adjectives and 25 nouns embedded — exceeds minimum 20 each per D-15"
  - "Number range 10-99 (two-digit) for brevity per D-15"
metrics:
  duration: "~15 minutes"
  completed_date: "2026-06-04"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 2
---

# Phase 2 Plan 02: JWT Service Summary

## One-liner

HS256 JWT sign/validate with alg-confusion guard and embedded-word-list username generator; both packages are pure-logic with no I/O dependencies, ready for wave 3 service layer consumption.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add golang-jwt/v5 dependency and implement internal/auth/jwt/jwt.go | f97e85f | internal/auth/jwt/jwt.go, go.mod, go.sum |
| 2 | Implement internal/auth/username/generator.go | 768ec22 | internal/auth/username/generator.go |

## Artifacts Produced

### New Types
- `jwt.Claims` struct — fields: `Sub uuid.UUID \`json:"sub"\``, `Role string \`json:"role"\``, embedded `jwt.RegisteredClaims`; no username or guest flag per D-05

### New Functions
- `jwt.NewAccessClaims(userID uuid.UUID, role string, ttl time.Duration) Claims` — constructs Claims with exp/iat from UTC now + ttl
- `jwt.SignJWT(claims Claims, secret []byte) (string, error)` — HS256 signing, wraps error as `"jwt sign: %w"`
- `jwt.ValidateJWT(tokenStr string, secret []byte) (*Claims, error)` — ParseWithClaims with explicit HS256 method check in key func; wraps error as `"jwt validate: %w"`
- `username.Generate() string` — returns AdjectiveNounNumber format (e.g. "SwiftKing42"); uses 25 adjectives, 25 nouns, range 10-99

### Modified Files
- `go.mod` — adds `require github.com/golang-jwt/jwt/v5 v5.3.1` (direct)
- `go.sum` — updated with jwt/v5 hashes

## Verification Results

All plan verification checks passed:
- `go build ./internal/auth/...` exits 0
- `grep "golang-jwt/jwt/v5" go.mod` finds `v5.3.1` (direct dependency after go mod tidy)
- `grep "SigningMethodHS256" internal/auth/jwt/jwt.go` finds both creation and validation guards
- `grep "unexpected signing method" internal/auth/jwt/jwt.go` confirms alg confusion guard
- `grep "func Generate" internal/auth/username/generator.go` confirms exported function
- `go vet ./internal/auth/jwt/...` exits 0
- `go vet ./internal/auth/username/...` exits 0

## Deviations from Plan

None — plan executed exactly as written.

## Threat Model Compliance

| Threat ID | Status | Notes |
|-----------|--------|-------|
| T-02-02-01 | Mitigated | key func checks `t.Method != jwt.SigningMethodHS256`, returns error on any other method; prevents alg:none and RSA confusion attacks |
| T-02-02-02 | Accepted | Secret strength is operational concern; documented in .env.example (done in 02-01) |
| T-02-02-03 | Accepted | Claims carries only Sub (UUID) and Role string; no email, username, or guest flag |
| T-02-02-04 | Accepted (wave 3) | Role read from c.Get("role") in middleware — to be enforced in wave 3 middleware implementation |
| T-02-02-SC | Mitigated | golang-jwt/jwt/v5 is the official successor to dgrijalva/jwt-go on pkg.go.dev; well-known high-integrity package |

## Known Stubs

None — both packages are pure-logic utilities with no data rendering paths, no placeholder text, and no mock data.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes introduced. Both files are pure in-memory computation utilities.

## Self-Check: PASSED
