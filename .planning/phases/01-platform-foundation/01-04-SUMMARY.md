---
phase: 1
plan: 04
subsystem: observability-server
tags: [observability, telemetry, gin, zerolog, opentelemetry, prometheus, middleware]
dependency_graph:
  requires:
    - 01-02-config-postgres   # config.Load(), database.NewPool(), RunMigrations()
    - 01-03-redis-rabbitmq    # rdb.NewClient(), rabbitmq.New()
  provides:
    - telemetry.Bootstrap     # OTel TracerProvider + MeterProvider bootstrap
    - logger.Setup            # zerolog global logger setup
    - logger.FromCtx          # trace-ID injecting context logger
    - middleware.ZerologLogger # Gin access log middleware
    - middleware.Tracing      # otelgin wrapper
    - middleware.Recovery     # structured panic recovery
    - app.New                 # composition root with full dependency wiring
    - cmd/api/main.go         # complete entry point
  affects:
    - all downstream phases   # gin router and health endpoint underpin all future handlers
tech_stack:
  added:
    - github.com/gin-gonic/gin@v1.12.0
    - go.opentelemetry.io/otel@v1.44.0
    - go.opentelemetry.io/otel/sdk@v1.44.0
    - go.opentelemetry.io/otel/sdk/trace@v1.44.0
    - go.opentelemetry.io/otel/sdk/metric@v1.44.0
    - go.opentelemetry.io/otel/exporters/prometheus@v0.66.0
    - go.opentelemetry.io/otel/exporters/stdout/stdouttrace@v1.44.0
    - go.opentelemetry.io/otel/semconv/v1.26.0
    - go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin@v0.69.0
    - github.com/prometheus/client_golang@v1.23.2
  patterns:
    - zerolog JSON global logger with OTel trace-ID injection via context
    - OTel Bootstrap pattern (stdout tracer + Prometheus meter in Phase 1)
    - gin.New() with explicit middleware chain (Recovery -> Tracing -> ZerologLogger)
    - Composition root pattern in internal/app/app.go
    - Graceful shutdown via SIGTERM/SIGINT with 30-second timeout
key_files:
  created:
    - internal/logger/logger.go
    - internal/telemetry/tracer.go
    - internal/telemetry/metrics.go
    - internal/telemetry/otel.go
    - internal/telemetry/otel_test.go
    - internal/middleware/recovery.go
    - internal/middleware/tracing.go
    - internal/middleware/logger.go
  modified:
    - internal/app/app.go
    - cmd/api/main.go
    - go.mod
    - go.sum
decisions:
  - "stdout trace exporter used for Phase 1 (OTLP deferred to Phase 7 per RESEARCH.md open question)"
  - "gin.New() enforced over gin.Default() to prevent unstructured text log output (REQ-026)"
  - "CustomRecovery used over gin.Recovery() for structured panic logging (T-04-01)"
  - "c.Request.Context() used in ZerologLogger (not c.Copy()) to preserve OTel span context (Pitfall 4)"
  - "prometheus/client_golang v1.23.2 used (not v1.22.0) to maintain otel/exporters/prometheus@v0.66.0 compatibility"
metrics:
  duration: "8m 18s"
  tasks_completed: 2
  tasks_total: 2
  files_created: 8
  files_modified: 4
  completed_date: "2026-06-04T04:06:33Z"
---

# Phase 1 Plan 04: Observability and Server Bootstrap Summary

**One-liner:** Zerolog global logger with OTel trace-ID injection, Prometheus-backed MeterProvider, stdout TracerProvider, Gin server with Recovery->Tracing->ZerologLogger middleware chain, /api/v1/health and /metrics endpoints, and full cmd/api/main.go startup sequence.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Install OTel/Gin dependencies and implement logger + telemetry bootstrap | 96eb413 | logger.go, tracer.go, metrics.go, otel.go, otel_test.go |
| 2 | Gin middleware chain and complete server entry point | 0f8b593 | recovery.go, tracing.go, middleware/logger.go, app.go, main.go |

## What Was Built

### Task 1: Logger and Telemetry Bootstrap

**internal/logger/logger.go:**
- `Setup(level zerolog.Level)`: configures zerolog global with JSON output to stdout (RFC3339 timestamps, no console writer)
- `ParseLevel(s string) zerolog.Level`: string-to-level conversion with InfoLevel default
- `FromCtx(ctx context.Context) *zerolog.Logger`: extracts `trace_id` and `span_id` from OTel span context and injects into zerolog fields; returns global logger unchanged when no valid span exists

**internal/telemetry/tracer.go:** `newStdoutTracerProvider` builds a batched stdout exporter TracerProvider with service name resource attribute `sudoku-pvp-api`.

**internal/telemetry/metrics.go:** `newMeterProvider` wires the Prometheus exporter to OTel MeterProvider, registering with the default Prometheus registry for scraping by promhttp.

**internal/telemetry/otel.go:** `Bootstrap(ctx, cfg)` orchestrates: resource creation -> MeterProvider (Prometheus) -> TracerProvider (stdout Phase 1) -> W3C TraceContext+Baggage propagator. Returns a shutdown func that drains both providers.

**internal/telemetry/otel_test.go:** `TestBootstrap` verifies providers are non-nil post-Bootstrap and shutdown completes without error. Uses `//go:build !integration` to run in normal test suite.

### Task 2: Middleware Chain and Server Entry Point

**internal/middleware/recovery.go:** `Recovery()` uses `gin.CustomRecovery` — logs full panic internally via zerolog, returns generic `{"code":"INTERNAL_ERROR"}` to client (T-04-01: no stack trace exposed).

**internal/middleware/tracing.go:** `Tracing()` wraps `otelgin.Middleware("sudoku-pvp-api")` — creates W3C-propagated OTel span per request.

**internal/middleware/logger.go:** `ZerologLogger()` writes structured access log after handler chain. Uses `c.Request.Context()` (not `c.Copy()`) to preserve otelgin-injected trace context (Pitfall 4).

**internal/app/app.go:** Composition root. `New(cfg)` sequences: logger.Setup -> telemetry.Bootstrap -> database.NewPool -> rdb.NewClient -> rabbitmq.New -> setupRouter. `setupRouter` builds `gin.New()` engine with `Recovery -> Tracing -> ZerologLogger` middleware order, registers `/metrics` (promhttp) and `/api/v1/health` (200 JSON). `Shutdown(ctx)` closes mqConn, redisClient, pool, then flushes telemetry.

**cmd/api/main.go:** Full startup: root context with cancel, `config.Load()`, `database.RunMigrations()` (with 30s timeout context), `app.New(cfg)`, SIGTERM/SIGINT signal handler goroutine with 30s shutdown timeout, `app.Run(":PORT")`. All logging via zerolog — no fmt.Println.

## Verification Results

All acceptance criteria passed:

- `go build ./...` exits 0 (full module)
- `go test ./internal/telemetry/... -run TestBootstrap -count=1` passes
- `grep -r "fmt.Println" internal/ cmd/` returns no matches
- `grep -r "gin.Default()" internal/ cmd/` returns no matches
- `internal/middleware/logger.go` uses `c.Request.Context()` (not `c.Copy()`)
- `internal/app/app.go` uses `gin.New()` with correct middleware order
- All 10 key files created/modified successfully

## Deviations from Plan

### Minor Dependency Version Adjustment

**[Rule 1 - Bug] prometheus/client_golang version pinned to v1.23.2 (not v1.22.0)**
- **Found during:** Task 1 dependency installation
- **Issue:** Installing `github.com/prometheus/client_golang@v1.22.0` as specified in the plan caused a downgrade of `go.opentelemetry.io/otel/exporters/prometheus` from v0.66.0 to v0.59.1 (the two are version-coupled). The plan listed v1.22.0 but the RESEARCH.md Standard Stack table does not pin a specific prometheus client_golang version.
- **Fix:** Used `github.com/prometheus/client_golang@v1.23.2` which is the version required by `otel/exporters/prometheus@v0.66.0`. All functionality is identical for the purposes of this plan.
- **Files modified:** go.mod, go.sum
- **Commit:** 96eb413

### Worktree Dependency Isolation

**[Rule 3 - Blocking] All go get commands run in both main repo and worktree**
- **Found during:** Task 1 initial build
- **Issue:** The worktree has its own go.mod/go.sum, so `go get` commands needed to run inside the worktree (J:/sources/sudoku-pvp/.claude/worktrees/agent-a56aa9b7c16ac4df9), not the main repo.
- **Fix:** Re-ran all `go get` and `go mod tidy` commands with `cd $WT_ROOT` as the working directory.
- **Impact:** None on produced artifacts; both go.mod files now contain the required dependencies.

## Threat Surface Scan

No new network endpoints or security-relevant surfaces beyond what the plan's threat model specifies:

| Flag | File | Description |
|------|------|-------------|
| No new surfaces | — | All surfaces (HTTP /health, /metrics, middleware chain) are explicitly modelled in T-04-01 through T-04-05 |

The /metrics endpoint (T-04-02) is accepted in Phase 1 dev and flagged for Phase 7 hardening (network restriction or auth).

## Known Stubs

None. All implemented functions are fully wired:
- `healthHandler` returns static 200 JSON (no stub — this is the complete health endpoint)
- `app.New()` wires all real infrastructure (no mock data flows)
- `/metrics` serves real Prometheus metrics via promhttp.Handler()

## Self-Check: PASSED

**Files exist:**
- FOUND: internal/logger/logger.go
- FOUND: internal/telemetry/otel.go
- FOUND: internal/telemetry/tracer.go
- FOUND: internal/telemetry/metrics.go
- FOUND: internal/telemetry/otel_test.go
- FOUND: internal/middleware/logger.go
- FOUND: internal/middleware/recovery.go
- FOUND: internal/middleware/tracing.go
- FOUND: internal/app/app.go
- FOUND: cmd/api/main.go

**Commits exist:**
- 96eb413: feat(01-04): implement zerolog logger and OTel telemetry bootstrap
- 0f8b593: feat(01-04): Gin middleware chain, composition root, and complete server entry point
