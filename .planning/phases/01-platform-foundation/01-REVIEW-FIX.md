---
phase: 01-platform-foundation
fixed_at: 2026-06-04T04:45:00Z
review_path: .planning/phases/01-platform-foundation/01-REVIEW.md
iteration: 1
findings_in_scope: 12
fixed: 12
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

**Fixed at:** 2026-06-04T04:45:00Z
**Source review:** .planning/phases/01-platform-foundation/01-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 12 (5 Critical + 7 Warning; IN-03 also addressed alongside CR-05)
- Fixed: 12
- Skipped: 0

## Fixed Issues

### CR-01: HTTP server never receives shutdown signal

**Files modified:** `internal/app/app.go`, `cmd/api/main.go`
**Commit:** 4c0c00a
**Applied fix:** Added `srv *http.Server` field to `App` struct. `App.Run` now constructs an explicit `*http.Server` using `srv.ListenAndServe()` and returns nil on `http.ErrServerClosed`. `App.Shutdown` now calls `a.srv.Shutdown(ctx)` first (before closing infra), ensuring in-flight requests are drained within the 30-second budget. Also added `errors` import.

---

### CR-02: Migration context timeout is dead code

**Files modified:** `cmd/api/main.go`
**Commit:** 4c0c00a
**Applied fix:** Replaced the dead `_ = migCtx` pattern with a goroutine-based timeout wrapper that races `database.RunMigrations` against the 30-second context. A `select` on `migDone` and `migCtx.Done()` now enforces the timeout, exiting with `log.Fatal` if the migration takes more than 30 seconds.

---

### CR-03: MeterProvider shutdown error silently discarded

**Files modified:** `internal/telemetry/otel.go`
**Commit:** 2bb5663
**Applied fix:** Replaced the `_ = mp.Shutdown(ctx)` discard with a pattern that collects both `mp.Shutdown` and `tp.Shutdown` errors into a `[]error` slice, then joins them with `errors.Join`. The caller now receives a combined error if either provider fails to shut down. Added `errors` import.

---

### CR-04: Reconnect loop TOCTOU race on connection reference

**Files modified:** `internal/rabbitmq/connection.go`
**Commit:** dc160be
**Applied fix:** Restructured `reconnectLoop` to take an initial snapshot of `c.conn` once before the outer loop begins, then update the local `conn` variable to `newConn` after each successful reconnect (`conn = newConn` before `break`). The outer loop now calls `NotifyClose` on the current live connection rather than the stale pre-reconnect one. This eliminates the TOCTOU race where the next iteration would re-snapshot and call `NotifyClose` on the already-closed connection, losing track of subsequent disconnections.

---

### CR-05: QueueKey accepts unsanitised strings

**Files modified:** `internal/redis/keys.go`, `internal/redis/keys_test.go`
**Commit:** bdbe22d
**Applied fix:** Changed `QueueKey` signature from `(string, string, string) string` to `(string, string, string) (string, error)`. The function now validates each segment with `strings.ContainsAny(s, ":*?")` and returns an error for any segment containing those characters. Tests updated for the new return signature. A new `TestQueueKey_InvalidSegments` table-driven test covers 6 injection and wildcard cases (also addresses IN-03).

---

### WR-01: Server.Port has no default

**Files modified:** `internal/config/config.go`
**Commit:** 7839aa6
**Applied fix:** Added `v.SetDefault("server.port", 8080)` before config file reading so the port defaults to 8080 when `SUDOKU_SERVER_PORT` is unset. Added validation that rejects ports outside the 1–65535 range with an explicit error message.

---

### WR-02: MaxConns=0 accepted silently

**Files modified:** `internal/database/postgres.go`
**Commit:** 169fc4e
**Applied fix:** Added a guard before setting pool config: `if cfg.MaxConns <= 0 { return nil, fmt.Errorf("postgres max_conns must be > 0, got %d", cfg.MaxConns) }`. This surfaces missing pool size configuration as an explicit startup error rather than silently falling back to pgxpool's internal default of 4.

---

### WR-03: Publisher messages not durable

**Files modified:** `internal/rabbitmq/publisher.go`
**Commit:** 7962ec3
**Applied fix:** Added `DeliveryMode: amqp.Persistent` to the `amqp.Publishing` struct in the `Publish` method. Messages are now written to disk by RabbitMQ and will survive broker restarts, consistent with the durable queue declarations already in place.

---

### WR-04: reconnectLoop sleep is not interruptible

**Files modified:** `internal/rabbitmq/connection.go`
**Commit:** dc160be
**Applied fix:** Added a `done chan struct{}` field to `Connection`, initialised in `New`. `Close()` now calls `close(c.done)` in addition to setting `c.closed = true`. The `time.Sleep(reconnectDelay)` in the retry inner loop is replaced with `select { case <-time.After(reconnectDelay): case <-c.done: return }`, allowing `Close()` to interrupt the delay immediately on shutdown.

---

### WR-05: users timestamps lack DEFAULT NOW()

**Files modified:** `db/migrations/000001_create_users.up.sql`
**Commit:** 6739cfe
**Applied fix:** Added `DEFAULT NOW()` to both `created_at TIMESTAMPTZ NOT NULL` and `updated_at TIMESTAMPTZ NOT NULL` columns. INSERTs that omit these columns now receive the current timestamp automatically.

---

### WR-06: signal.Stop not called after signal receipt

**Files modified:** `cmd/api/main.go`
**Commit:** 4c0c00a
**Applied fix:** Added `signal.Stop(quit)` as the first statement in the signal handler goroutine, immediately after receiving the signal from the `quit` channel. This prevents the Go runtime from delivering additional signals to the channel after shutdown processing has begun.

---

### WR-07: go.mod declares non-existent version 1.26.4

**Files modified:** `go.mod`
**Commit:** 595c15f (initial fix to 1.23.0), 5b3665e (go mod tidy raised to 1.25.0)
**Applied fix:** Corrected `go 1.26.4` to `go 1.23.0` as the starting point. Running `go mod tidy` then determined the actual minimum required by the dependency graph is `1.25.0`, which was committed as a follow-up. The resulting `go 1.25.0` is a valid released Go version that toolchains can satisfy.

---

## Skipped Issues

None — all 12 in-scope findings were successfully fixed.

---

_Fixed: 2026-06-04T04:45:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
