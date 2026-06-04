---
phase: 01-platform-foundation
reviewed: 2026-06-04T00:00:00Z
depth: standard
files_reviewed: 28
files_reviewed_list:
  - .golangci.yml
  - cmd/api/main.go
  - go.mod
  - internal/app/app.go
  - internal/config/config.go
  - internal/database/migrate.go
  - internal/database/postgres.go
  - internal/database/postgres_test.go
  - internal/logger/logger.go
  - internal/middleware/logger.go
  - internal/middleware/recovery.go
  - internal/middleware/tracing.go
  - internal/rabbitmq/connection.go
  - internal/rabbitmq/consumer.go
  - internal/rabbitmq/publisher.go
  - internal/rabbitmq/topology.go
  - internal/rabbitmq/topology_test.go
  - internal/redis/client.go
  - internal/redis/keys.go
  - internal/redis/keys_test.go
  - internal/telemetry/metrics.go
  - internal/telemetry/otel.go
  - internal/telemetry/otel_test.go
  - internal/telemetry/tracer.go
  - internal/testutil/containers.go
  - internal/testutil/containers_test.go
  - Makefile
  - db/migrations/000001_create_users.up.sql
findings:
  critical: 5
  warning: 7
  info: 4
  total: 16
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-06-04T00:00:00Z
**Depth:** standard
**Files Reviewed:** 28
**Status:** issues_found

## Summary

The platform foundation layer covers configuration loading, PostgreSQL/Redis/RabbitMQ adapters, OpenTelemetry bootstrap, Gin middleware chain, and integration test helpers. The overall structure is clean and follows Go idioms well. However, five critical defects were found that will cause incorrect runtime behaviour: the HTTP server never actually shuts down gracefully (the Gin `Run` wrapper has no `http.Server` handle to stop), the migration runner ignores the context timeout that was explicitly created for it, the MeterProvider shutdown error is silently discarded, the reconnect loop has a channel-swap race that can block permanently after reconnect, and the `QueueKey` function accepts unsanitised caller-supplied strings allowing Redis key injection. Seven warnings around missing defaults, unguarded pool settings, missing signal cleanup, and missing persistence flags in the Publisher are also documented.

---

## Critical Issues

### CR-01: HTTP server never receives shutdown signal — graceful drain is impossible

**File:** `internal/app/app.go:130-132` and `cmd/api/main.go:58-61`

**Issue:** `App.Run` calls `a.router.Run(addr)` which delegates to Gin's built-in `http.ListenAndServe`. Gin's `Engine.Run` does not expose an `*http.Server` handle, so there is no way for `App.Shutdown` to call `server.Shutdown(ctx)`. When the signal handler fires, `a.Shutdown(shutCtx)` closes infrastructure connections and then calls `rootCancel()`, but the HTTP server is still accepting and processing requests — it is never told to stop. `a.Run(addr)` then blocks forever (or until the process is killed), and `os.Exit(1)` is reached only on an unexpected server error. In-flight requests are dropped without the 30-second drain the code comments promise.

**Fix:** Replace `a.router.Run` with an explicit `*http.Server` and call `server.Shutdown(ctx)` from `App.Shutdown`:

```go
// In App struct
srv *http.Server

// In App.Run
func (a *App) Run(addr string) error {
    a.srv = &http.Server{Addr: addr, Handler: a.router}
    if err := a.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        return err
    }
    return nil
}

// In App.Shutdown — call first, before closing infra
if a.srv != nil {
    _ = a.srv.Shutdown(ctx)
}
```

---

### CR-02: Migration context timeout is created but never passed — 30-second guard is dead code

**File:** `cmd/api/main.go:38-43`

**Issue:** A `context.WithTimeout(rootCtx, 30*time.Second)` is created for the migration step, but `RunMigrations` receives only a DSN string — the context is explicitly discarded on line 40 (`_ = migCtx`). If the database is unreachable the `golang-migrate` library will block until the underlying TCP dial times out (OS-level, potentially minutes), defeating the fast-fail startup requirement documented in the comment on line 37.

**Fix:** Propagate the context. `golang-migrate` does not accept a context in its `Up()` call, but the dial itself respects DSN connection parameters. Add a `connect_timeout` parameter to the DSN and remove the misleading dead context:

```go
// Option A: Remove the dead context entirely and document the DSN-level timeout
// dsn should contain: ?connect_timeout=10

// Option B: Wire timeout via a wrapper that polls m.Up() in a goroutine
migCtx, migCancel := context.WithTimeout(rootCtx, 30*time.Second)
defer migCancel()
done := make(chan error, 1)
go func() { done <- database.RunMigrations(cfg.Postgres.DSN) }()
select {
case err := <-done:
    if err != nil { log.Fatal()... }
case <-migCtx.Done():
    log.Fatal().Msg("database migration timed out")
}
```

Either way, the `_ = migCtx` line must be removed — it currently communicates false confidence that a timeout is in effect.

---

### CR-03: MeterProvider shutdown error is silently discarded — metric flush failures are invisible

**File:** `internal/telemetry/otel.go:54`

**Issue:** The shutdown closure returns only the TracerProvider shutdown error:

```go
shutdown := func(ctx context.Context) error {
    _ = mp.Shutdown(ctx)          // error thrown away
    return tp.Shutdown(ctx)
}
```

If `mp.Shutdown` fails (e.g., Prometheus exporter can't flush), the error is silently discarded and the caller receives `nil` or only the tracer error. The Prometheus SDK may not have completed flushing metrics to the registry, which can cause partial metric loss on shutdown and masks a real failure.

**Fix:**

```go
shutdown := func(ctx context.Context) error {
    var errs []error
    if err := mp.Shutdown(ctx); err != nil {
        errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
    }
    if err := tp.Shutdown(ctx); err != nil {
        errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
    }
    return errors.Join(errs...)
}
```

---

### CR-04: Reconnect loop has a TOCTOU race on the connection reference — consumer goroutines can block permanently after reconnect

**File:** `internal/rabbitmq/connection.go:100-152`

**Issue:** After a successful reconnect (`c.setConn(newConn)` on line 141), the outer `for` loop's next iteration reads `conn := c.conn` under `RLock` (line 103) and then calls `conn.NotifyClose(make(chan *amqp.Error, 1))` (line 112). However, `NotifyClose` is called on the **old** `conn` value captured at the top of the loop before the reconnect happened — not the new connection stored via `setConn`. This means after one successful reconnect the loop registers for close events on the stale previous connection (which is already closed), receives an immediate notification on that stale channel, and then tries to reconnect again immediately in a tight loop. More critically: if `newConn` itself disconnects, `NotifyClose` was never called on it, so the reconnect loop never detects the second disconnection. The reconnect loop effectively stops monitoring after the first successful reconnect.

**Fix:** Move the `conn` snapshot to after the reconnect succeeds, so `NotifyClose` is always called on the current live connection:

```go
func (c *Connection) reconnectLoop() {
    for {
        c.mu.RLock()
        conn := c.conn
        isClosed := c.closed
        c.mu.RUnlock()
        if isClosed {
            return
        }

        notify := conn.NotifyClose(make(chan *amqp.Error, 1))
        <-notify

        // ... check isClosed, log, retry dial ...
        for {
            // ... retry logic ...
            c.setConn(newConn)
            conn = newConn   // <-- update local reference for next NotifyClose
            break
        }
        // loop back: NotifyClose will now be called on newConn
    }
}
```

---

### CR-05: `QueueKey` accepts unsanitised strings — Redis key injection via caller-supplied region/difficulty/stake

**File:** `internal/redis/keys.go:27-29`

**Issue:** `QueueKey(region, difficulty, stake string)` formats these strings directly into a Redis key without any validation or sanitisation:

```go
return fmt.Sprintf("queue:%s:%s:%s", region, difficulty, stake)
```

If the calling code forwards any of these values from user input (HTTP request parameters, WebSocket messages) without prior validation, an attacker can inject colon-separated segments that collide with other key namespaces — for example `region="asia:match"` produces `queue:asia:match:medium:100` which structurally overlaps the `match:` key namespace. Redis does not interpret key names, so this is a key namespace collision/confusion vulnerability rather than a code-execution injection, but it allows one user to read or write another user's match state key.

**Fix:** Validate each segment at the function boundary, or enforce validation before calling `QueueKey`. At minimum, reject strings containing `:`:

```go
func QueueKey(region, difficulty, stake string) (string, error) {
    for _, s := range []string{region, difficulty, stake} {
        if strings.ContainsAny(s, ":*?") {
            return "", fmt.Errorf("invalid QueueKey segment: %q", s)
        }
    }
    return fmt.Sprintf("queue:%s:%s:%s", region, difficulty, stake), nil
}
```

---

## Warnings

### WR-01: `Server.Port` has no default — silently binds to `:0` (random ephemeral port) if unset

**File:** `internal/config/config.go:54-87`

**Issue:** The `Load` function validates that `Postgres.DSN`, `Redis.Addr`, and `RabbitMQ.URL` are non-empty, but does not validate `Server.Port`. If `SUDOKU_SERVER_PORT` is not set and no config file is found, `cfg.Server.Port` defaults to `0`. `fmt.Sprintf(":%d", 0)` produces `":0"`, and `net.Listen("tcp", ":0")` succeeds by binding to a random ephemeral port chosen by the OS. The service will start without error but will not be reachable on any predictable port.

**Fix:** Add a default and/or validate the port:

```go
v.SetDefault("server.port", 8080)
// and/or
if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
    return nil, fmt.Errorf("server.port must be between 1 and 65535, got %d", cfg.Server.Port)
}
```

---

### WR-02: `pgxpool` pool sizing is not validated — `MaxConns=0` is accepted and causes pgxpool to use its default of 4, silently

**File:** `internal/database/postgres.go:26-27`

**Issue:** `poolCfg.MaxConns = cfg.MaxConns` and `poolCfg.MinConns = cfg.MinConns` are set unconditionally. If the config values are zero (which they will be if no config file exists and env vars are unset), pgxpool interprets `MaxConns=0` as "use default" (4 connections) and `MinConns=0` as no minimum. This silently contradicts whatever pool sizing was intended, and the behaviour is not surfaced in logs.

**Fix:** Add a config validation and/or a log statement that records the effective pool size after construction:

```go
if cfg.MaxConns <= 0 {
    return nil, fmt.Errorf("postgres max_conns must be > 0, got %d", cfg.MaxConns)
}
```

---

### WR-03: Publisher messages are not durable — messages are lost if RabbitMQ restarts between publish and consume

**File:** `internal/rabbitmq/publisher.go:30-46`

**Issue:** `amqp.Publishing` does not set `DeliveryMode: amqp.Persistent`. Messages published without this flag are stored in memory only. If RabbitMQ restarts (or is restarted for a deployment), all in-flight messages in domain queues are lost even though the queues themselves are declared as durable. Durable queues only survive restarts; messages in them do not unless `DeliveryMode=2` (Persistent) is set on each published message.

**Fix:**

```go
amqp.Publishing{
    ContentType:  "application/json",
    DeliveryMode: amqp.Persistent,
    Body:         body,
}
```

---

### WR-04: `reconnectLoop` uses `time.Sleep` in the hot path — the shutdown signal cannot interrupt a sleep in progress

**File:** `internal/rabbitmq/connection.go:137`

**Issue:** When a reconnect dial fails, the loop sleeps for `reconnectDelay` (5 seconds) using `time.Sleep`. During this sleep, a call to `Connection.Close()` sets `c.closed = true`, but the goroutine remains blocked in `time.Sleep` for up to 5 seconds before it checks `isClosed` again. On shutdown, this causes a 0–5 second hang before the process can exit cleanly.

**Fix:** Replace `time.Sleep` with a context-aware select that can be interrupted immediately:

```go
// Add a done channel to Connection, closed in Close():
select {
case <-time.After(reconnectDelay):
case <-c.done:
    return
}
```

---

### WR-05: `users.created_at` and `users.updated_at` have no `DEFAULT` — inserts without explicit values will fail at the database level

**File:** `db/migrations/000001_create_users.up.sql:9-10`

**Issue:** Both `created_at TIMESTAMPTZ NOT NULL` and `updated_at TIMESTAMPTZ NOT NULL` lack a `DEFAULT` clause. Any INSERT statement that omits these columns will fail with a `NOT NULL constraint` violation at the database level. This forces every INSERT in application code to explicitly supply both timestamps, which is error-prone when new code paths are added. Standard practice is `DEFAULT NOW()`.

**Fix:**

```sql
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

---

### WR-06: `signal.Notify` is not paired with `signal.Stop` — double-signal processing is possible

**File:** `cmd/api/main.go:53-62`

**Issue:** `signal.Notify(quit, os.Interrupt, syscall.SIGTERM)` is called but `signal.Stop(quit)` is never called after the signal is received. While this does not cause a crash in normal single-signal shutdown, it means the Go runtime will continue delivering signals to the `quit` channel after the goroutine has already processed one, potentially interfering with signal handling in tests or in environments that send a second SIGTERM. Best practice is to call `signal.Stop(quit)` immediately after receiving the signal.

**Fix:**

```go
go func() {
    sig := <-quit
    signal.Stop(quit)   // stop further deliveries
    log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
    // ...
}()
```

---

### WR-07: `go.mod` declares `go 1.26.4` — this version does not exist and will cause toolchain errors

**File:** `go.mod:3`

**Issue:** The `go` directive reads `go 1.26.4`. As of the knowledge cutoff (August 2025), Go's latest release is in the 1.23.x/1.24.x range. `go 1.26.4` is a future (or typoed) version. When a developer or CI system has an older toolchain installed, `go build` will print `note: module requires Go >= 1.26.4` and may refuse to build. If a `toolchain` directive is absent, the Go toolchain will attempt a download that may fail in air-gapped environments.

**Fix:** Correct to the actual minimum Go version required by the dependencies, e.g.:

```
go 1.23.0
```

---

## Info

### IN-01: `go 1.26.4` declared in `go.mod` with no `toolchain` directive

**File:** `go.mod:3`

**Issue:** Related to WR-07. When a non-existent Go version is declared without a `toolchain` directive, `go` commands that check version compatibility will behave unpredictably across developer machines. A `toolchain` line would at least pin the expected toolchain explicitly.

**Fix:** Add a `toolchain` directive or correct the `go` version directive to a released version.

---

### IN-02: `migCtx` variable created but immediately discarded with `_ = migCtx`

**File:** `cmd/api/main.go:38-40`

**Issue:** The comment and variable declaration imply that a migration timeout is enforced, but the code acknowledges it is not (`_ = migCtx`). This is misleading to future readers who may assume the timeout is active. See CR-02 for the fix — but regardless of whether the full fix is applied, the `_ = migCtx` line and surrounding variable should be removed rather than left as dead scaffolding.

**Fix:** Remove the dead variable or implement the timeout properly (see CR-02).

---

### IN-03: `QueueKey` has no test for the key injection case

**File:** `internal/redis/keys_test.go`

**Issue:** `TestQueueKey` only tests a happy-path call with clean inputs. There is no test asserting that inputs containing `:` are rejected (or even that the current format produces an ambiguous key). When input sanitisation is added (CR-05), a test covering invalid characters should be added.

**Fix:** Add a table-driven test with `:`, `*`, and empty-string inputs that asserts the expected rejection behaviour.

---

### IN-04: Dead-letter consumer silently discards the poison message body with no logging

**File:** `internal/rabbitmq/consumer.go:59-64`

**Issue:** When `xDeathCount(d) >= maxXDeathCount`, the message is acknowledged and silently discarded. There is no log entry containing the message body, routing key, or x-death headers. This makes post-incident debugging difficult because there is no record of what was discarded.

**Fix:** Add a structured log at Warn level before the Ack:

```go
if xDeathCount(d) >= maxXDeathCount {
    // Log the discard for post-incident debugging
    log.Warn().
        Str("routing_key", d.RoutingKey).
        Int64("x_death_count", xDeathCount(d)).
        Bytes("body", d.Body).
        Msg("message exceeded max dead-letter count, discarding")
    _ = d.Ack(false)
    continue
}
```

---

_Reviewed: 2026-06-04T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
