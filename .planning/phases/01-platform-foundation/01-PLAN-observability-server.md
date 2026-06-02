---
id: 01-plan-observability-server
phase: 1
plan: 04
type: execute
wave: 3
depends_on:
  - 01-plan-config-postgres
  - 01-plan-redis-rabbitmq
objective: "Bootstrap zerolog logger, OpenTelemetry (traces + Prometheus metrics), Gin server with middleware chain, health endpoint, and /metrics endpoint"
files_modified:
  - internal/logger/logger.go
  - internal/telemetry/tracer.go
  - internal/telemetry/metrics.go
  - internal/telemetry/otel.go
  - internal/telemetry/otel_test.go
  - internal/middleware/logger.go
  - internal/middleware/recovery.go
  - internal/middleware/tracing.go
  - internal/app/app.go
  - cmd/api/main.go
requirements_addressed:
  - REQ-026
  - REQ-observability
  - REQ-nfr-api-latency
autonomous: true

must_haves:
  truths:
    - "zerolog is set up as global logger with JSON output and trace-ID injection from OTel span context"
    - "OTel Bootstrap() returns a valid shutdown function; TracerProvider and MeterProvider are set globally"
    - "Prometheus /metrics endpoint returns HTTP 200 with Go runtime metrics"
    - "GET /api/v1/health returns HTTP 200 JSON with status=ok"
    - "Request logs contain trace_id field when OTel span is active"
    - "No fmt.Println in any production path; zerolog only"
    - "gin.New() used (not gin.Default()); middleware chain order: Recovery → otelgin → ZerologLogger"
  artifacts:
    - path: "internal/logger/logger.go"
      provides: "zerolog global setup and context helpers"
      exports: ["Setup", "FromCtx"]
    - path: "internal/telemetry/otel.go"
      provides: "OTel bootstrap with Prometheus exporter"
      exports: ["Bootstrap"]
    - path: "internal/middleware/logger.go"
      provides: "Gin zerolog access log middleware"
      exports: ["ZerologLogger"]
    - path: "internal/middleware/tracing.go"
      provides: "otelgin wrapper"
      exports: ["Tracing"]
    - path: "internal/middleware/recovery.go"
      provides: "Gin recovery middleware with structured panic log"
      exports: ["Recovery"]
    - path: "cmd/api/main.go"
      provides: "Full entry point: config → migrations → telemetry → infra → gin → serve"
      contains: "func main()"
    - path: "internal/app/app.go"
      provides: "Composition root with all deps wired"
      contains: "func New("
  key_links:
    - from: "cmd/api/main.go"
      to: "internal/telemetry/otel.go"
      via: "Bootstrap() called before Gin starts"
      pattern: "telemetry.Bootstrap"
    - from: "internal/middleware/logger.go"
      to: "internal/logger/logger.go"
      via: "FromCtx(c.Request.Context()) in handler"
      pattern: "logger.FromCtx"
    - from: "internal/middleware/tracing.go"
      to: "otelgin.Middleware"
      via: "wraps otelgin with service name constant"
      pattern: "otelgin.Middleware"
---

<objective>
Install OTel SDK + otelgin, configure zerolog global logger with trace-ID injection, bootstrap Prometheus metrics exporter, wire Gin server with the full middleware chain, expose /api/v1/health and /metrics endpoints, and complete cmd/api/main.go to sequence: config load → migrations → telemetry bootstrap → infra connect → gin start → graceful shutdown.

Purpose: The observability layer and server entry point must exist before the integration test harness can run smoke tests. This plan completes the running application skeleton.
Output: Running Gin server at :8080 serving /api/v1/health (200) and /metrics (200); structured JSON access logs with trace IDs on every request.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md
@J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md

<interfaces>
From plan 02 (internal/config/config.go):

type Config struct {
    Server    ServerConfig    // Port int
    Postgres  PostgresConfig  // DSN string, MaxConns int32, MinConns int32
    Redis     RedisConfig     // Addr, Password, DB, PoolSize
    RabbitMQ  RabbitMQConfig  // URL string
    Telemetry TelemetryConfig // OTLPEndpoint string
    LogLevel  string
}
func Load() (*Config, error)

From plan 02 (internal/database/):
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error)
func RunMigrations(dsn string) error

From plan 03 (internal/redis/):
func NewClient(cfg config.RedisConfig) (*redis.Client, error)

From plan 03 (internal/rabbitmq/):
func New(url string, logger zerolog.Logger) (*Connection, error)

Exported by this plan (downstream plans depend on these):

// internal/telemetry/otel.go
func Bootstrap(ctx context.Context, cfg config.TelemetryConfig) (shutdown func(context.Context) error, err error)

// internal/logger/logger.go
func Setup(level zerolog.Level)
func FromCtx(ctx context.Context) *zerolog.Logger

// internal/middleware/logger.go
func ZerologLogger() gin.HandlerFunc

// internal/middleware/tracing.go
func Tracing() gin.HandlerFunc

// internal/middleware/recovery.go
func Recovery() gin.HandlerFunc

// internal/app/app.go
type App struct { Router *gin.Engine; ... }
func New(cfg *config.Config) (*App, error)
func (a *App) Run(addr string) error
func (a *App) Shutdown(ctx context.Context) error
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Install OTel/Gin dependencies and implement logger + telemetry bootstrap</name>
  <files>
    internal/logger/logger.go,
    internal/telemetry/tracer.go,
    internal/telemetry/metrics.go,
    internal/telemetry/otel.go,
    internal/telemetry/otel_test.go
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Pattern 6 OTel Bootstrap, Pattern 8 zerolog trace-ID injection, Pitfall 4 otelgin context, Standard Stack table versions)
  </read_first>
  <action>
    Install dependencies:
    go get github.com/gin-gonic/gin@v1.12.0
    go get go.opentelemetry.io/otel@v1.44.0
    go get go.opentelemetry.io/otel/sdk@v1.44.0
    go get go.opentelemetry.io/otel/trace@v1.44.0
    go get go.opentelemetry.io/otel/metric@v1.44.0
    go get go.opentelemetry.io/otel/sdk/metric@v1.44.0
    go get go.opentelemetry.io/otel/sdk/trace@v1.44.0
    go get go.opentelemetry.io/otel/exporters/prometheus@v0.66.0
    go get go.opentelemetry.io/otel/exporters/stdout/stdouttrace@v1.44.0
    go get go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin@v0.69.0
    go get go.opentelemetry.io/otel/semconv/v1.26.0
    go get github.com/prometheus/client_golang@v1.22.0
    go mod tidy

    Create internal/logger/logger.go in package logger. Imports: zerolog "github.com/rs/zerolog", log "github.com/rs/zerolog/log", trace "go.opentelemetry.io/otel/trace", context, os, time.

    Implement func Setup(level zerolog.Level): set zerolog.TimeFieldFormat = time.RFC3339, call zerolog.SetGlobalLevel(level), set log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger(). No console writer in production — JSON to stdout only.

    Implement func FromCtx(ctx context.Context) *zerolog.Logger: get l := log.Ctx(ctx). If sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() — create sub := l.With().Str("trace_id", sc.TraceID().String()).Str("span_id", sc.SpanID().String()).Logger(); return &sub. Otherwise return l.

    Implement func ParseLevel(s string) zerolog.Level: use zerolog.ParseLevel(s), default to zerolog.InfoLevel on error.

    Create internal/telemetry/tracer.go in package telemetry. Imports: sdktrace "go.opentelemetry.io/otel/sdk/trace", stdouttrace "go.opentelemetry.io/otel/exporters/stdout/stdouttrace", resource "go.opentelemetry.io/otel/sdk/resource", semconv "go.opentelemetry.io/otel/semconv/v1.26.0".

    Implement func newResource(ctx context.Context) (*resource.Resource, error): call resource.New(ctx, resource.WithAttributes(semconv.ServiceName("sudoku-pvp-api"))).

    Implement func newStdoutTracerProvider(res *resource.Resource) (*sdktrace.TracerProvider, error): use stdouttrace.New() for dev, wrap in sdktrace.NewTracerProvider with sdktrace.WithBatcher. Return provider and error.

    Create internal/telemetry/metrics.go in package telemetry. Imports: sdkmetric "go.opentelemetry.io/otel/sdk/metric", otelprom "go.opentelemetry.io/otel/exporters/prometheus".

    Implement func newMeterProvider(res *resource.Resource) (*sdkmetric.MeterProvider, error): call otelprom.New() to create Prometheus exporter. Create sdkmetric.NewMeterProvider with sdkmetric.WithReader(promExporter) and sdkmetric.WithResource(res). Return provider and error.

    Create internal/telemetry/otel.go in package telemetry. Imports: otel "go.opentelemetry.io/otel", propagation "go.opentelemetry.io/otel/propagation", config "sudoku-pvp/internal/config".

    Implement func Bootstrap(ctx context.Context, cfg config.TelemetryConfig) (func(context.Context) error, error):
    1. Call newResource(ctx).
    2. Call newMeterProvider(res) — set otel.SetMeterProvider(mp).
    3. If cfg.OTLPEndpoint != "" — call future OTLP provider (stub: return error "OTLP not configured in Phase 1; set SUDOKU_TELEMETRY_OTLP_ENDPOINT empty for stdout"). For Phase 1 always use stdout: call newStdoutTracerProvider(res) — set otel.SetTracerProvider(tp).
    4. Set otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})).
    5. Return shutdown func that calls mp.Shutdown(ctx) then tp.Shutdown(ctx).

    Actually simplify: always use stdout trace exporter in Phase 1 regardless of OTLPEndpoint (open question from RESEARCH.md — default to stdout). OTLP support can be added in Phase 7 when trace backend is confirmed.

    Create internal/telemetry/otel_test.go with build tag //go:build !integration (runs in normal test suite without containers). Test func TestBootstrap: call Bootstrap with empty config, assert returned shutdown func is non-nil, assert otel.GetTracerProvider() is not nil, call shutdown.
  </action>
  <verify>
    <automated>go build ./internal/logger/... ./internal/telemetry/... exits 0</automated>
  </verify>
  <acceptance_criteria>
    - internal/logger/logger.go contains "func Setup(level zerolog.Level)"
    - internal/logger/logger.go contains "func FromCtx(ctx context.Context) *zerolog.Logger"
    - internal/logger/logger.go contains "trace_id"
    - internal/logger/logger.go contains "span_id"
    - internal/telemetry/otel.go contains "func Bootstrap(ctx context.Context"
    - internal/telemetry/otel.go contains "SetMeterProvider"
    - internal/telemetry/otel.go contains "SetTracerProvider"
    - internal/telemetry/otel.go contains "SetTextMapPropagator"
    - internal/telemetry/otel_test.go contains "TestBootstrap"
    - go test ./internal/telemetry/... -run TestBootstrap -count=1 exits 0
    - go build ./internal/logger/... ./internal/telemetry/... exits 0
    - go.mod contains "go.opentelemetry.io/otel"
    - go.mod contains "github.com/gin-gonic/gin"
  </acceptance_criteria>
  <done>zerolog global setup with trace-ID injection implemented; OTel Bootstrap wires TracerProvider (stdout) and MeterProvider (Prometheus); unit test for Bootstrap passes.</done>
</task>

<task type="auto">
  <name>Task 2: Gin middleware chain and complete server entry point</name>
  <files>
    internal/middleware/logger.go,
    internal/middleware/recovery.go,
    internal/middleware/tracing.go,
    internal/app/app.go,
    cmd/api/main.go
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Pattern 7 Gin Middleware Chain, Pattern 8 zerolog request logging, Pitfall 4 otelgin context, Pitfall 6 pgxpool shutdown, Anti-Patterns gin.Default)
    J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (middleware chain order diagram, Section 4 repository structure)
    J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
  </read_first>
  <action>
    Create internal/middleware/recovery.go in package middleware. Import gin "github.com/gin-gonic/gin", logger "sudoku-pvp/internal/logger". Implement func Recovery() gin.HandlerFunc: return gin.CustomRecovery(func(c *gin.Context, err any) { logger.FromCtx(c.Request.Context()).Error().Interface("panic", err).Msg("panic recovered"); c.AbortWithStatusJSON(500, map[string]string{"code": "INTERNAL_ERROR", "message": "internal server error"}) }). Do NOT use gin.Recovery() — that uses unstructured text log.

    Create internal/middleware/tracing.go in package middleware. Import otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin". Implement func Tracing() gin.HandlerFunc: return otelgin.Middleware("sudoku-pvp-api"). This is a thin wrapper that sets the service name constant.

    Create internal/middleware/logger.go in package middleware. Import gin, time, logger "sudoku-pvp/internal/logger". Implement func ZerologLogger() gin.HandlerFunc:
    Return a func(c *gin.Context) that: captures start := time.Now(), calls c.Next(), then calls logger.FromCtx(c.Request.Context()).Info().Str("method", c.Request.Method).Str("path", c.Request.URL.Path).Int("status", c.Writer.Status()).Dur("latency", time.Since(start)).Str("ip", c.ClientIP()).Msg("request").
    CRITICAL per Pitfall 4: use c.Request.Context() (not c.Copy()) so the OTel trace context from otelgin is preserved for trace_id injection.

    Update internal/app/app.go to become the composition root. Package app. Import gin, pgxpool, redis (go-redis), rabbitmq "sudoku-pvp/internal/rabbitmq", rdb "sudoku-pvp/internal/redis", database "sudoku-pvp/internal/database", telemetry "sudoku-pvp/internal/telemetry", logger "sudoku-pvp/internal/logger", middleware "sudoku-pvp/internal/middleware", config "sudoku-pvp/internal/config", common "sudoku-pvp/internal/common", promhttp "github.com/prometheus/client_golang/prometheus/promhttp".

    Define type App struct with fields: cfg *config.Config, router *gin.Engine, pool *pgxpool.Pool, redisClient *redis.Client, mqConn *rabbitmq.Connection, telemetryShutdown func(context.Context) error.

    Implement func New(cfg *config.Config) (*App, error):
    1. Setup logger: logger.Setup(logger.ParseLevel(cfg.LogLevel))
    2. Bootstrap telemetry: telemetry.Bootstrap(ctx, cfg.Telemetry) — store shutdown func
    3. Connect Postgres: database.NewPool(ctx, cfg.Postgres)
    4. Connect Redis: rdb.NewClient(cfg.Redis)
    5. Connect RabbitMQ: rabbitmq.New(cfg.RabbitMQ.URL, zerolog.Ctx(ctx))
    6. Wire Gin router (call setupRouter)
    Return &App{...}, nil.

    Implement private func (a *App) setupRouter() *gin.Engine:
    Create r := gin.New() — NOT gin.Default() (per Anti-Patterns).
    Register middleware in order per RESEARCH.md Pattern 7: r.Use(middleware.Recovery()), r.Use(middleware.Tracing()), r.Use(middleware.ZerologLogger()).
    Register /metrics endpoint: r.GET("/metrics", gin.WrapH(promhttp.Handler())).
    Register API group: v1 := r.Group("/api/v1"). Register health check: v1.GET("/health", healthHandler).
    Define healthHandler inline as func(c *gin.Context) { c.JSON(200, map[string]string{"status": "ok", "service": "sudoku-pvp-api"}) }.
    Return r.

    Implement func (a *App) Run(addr string) error: call a.router.Run(addr).
    Implement func (a *App) Shutdown(ctx context.Context): close pool, close redis client, close rabbitmq connection, call telemetryShutdown(ctx).

    Update cmd/api/main.go with the complete sequenced entry point:
    1. Create root context with cancel.
    2. Call config.Load() — log.Fatal on error.
    3. Call database.RunMigrations(cfg.Postgres.DSN) with context.WithTimeout(5s) — log.Fatal on error.
    4. Call app.New(cfg) — log.Fatal on error.
    5. Start signal handler in goroutine: listen for os.Interrupt and syscall.SIGTERM, call app.Shutdown(context.WithTimeout(30s)).
    6. Call app.Run(fmt.Sprintf(":%d", cfg.Server.Port)).

    Every log call in main.go must use zerolog (log.Fatal().Err(err).Msg(...)). No fmt.Println anywhere.
  </action>
  <verify>
    <automated>go build ./... exits 0</automated>
  </verify>
  <acceptance_criteria>
    - internal/middleware/tracing.go contains "otelgin.Middleware"
    - internal/middleware/logger.go contains "c.Request.Context()" (NOT c.Copy())
    - internal/middleware/logger.go contains "logger.FromCtx"
    - internal/middleware/recovery.go contains "CustomRecovery"
    - internal/app/app.go contains "func New(cfg *config.Config) (*App, error)"
    - internal/app/app.go contains "gin.New()" and does NOT contain "gin.Default()"
    - internal/app/app.go contains "middleware.Recovery()"
    - internal/app/app.go contains "middleware.Tracing()"
    - internal/app/app.go contains "middleware.ZerologLogger()"
    - internal/app/app.go contains "/api/v1"
    - internal/app/app.go contains "/metrics"
    - internal/app/app.go contains "promhttp.Handler()"
    - cmd/api/main.go contains "database.RunMigrations"
    - cmd/api/main.go contains "app.New(cfg)"
    - cmd/api/main.go contains "SIGTERM"
    - cmd/api/main.go does NOT contain "fmt.Println"
    - go build ./... exits 0
    - grep -r "fmt.Println" internal/ cmd/ returns no matches
  </acceptance_criteria>
  <done>Gin server wires Recovery → Tracing → ZerologLogger middleware chain; /api/v1/health and /metrics endpoints registered; cmd/api/main.go sequences full startup with graceful shutdown; no fmt.Println anywhere.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| HTTP request → Gin handler | All inbound HTTP traffic; untrusted headers, paths, bodies |
| OTel trace context → log fields | W3C TraceContext headers from inbound requests propagated into spans and injected into log fields |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-04-01 | Information Disclosure | Panic recovery response | mitigate | CustomRecovery logs full panic internally via zerolog but returns only generic "INTERNAL_ERROR" message to client — stack trace never exposed |
| T-04-02 | Information Disclosure | /metrics endpoint | accept | Prometheus metrics endpoint exposes Go runtime stats; acceptable in dev; production deployment should restrict /metrics to internal network or require auth (Phase 7 hardening) |
| T-04-03 | Spoofing | W3C TraceContext headers | accept | Inbound trace context headers could be spoofed by clients; acceptable in dev/tracing context — OTel SDK generates new root span if no valid context |
| T-04-04 | DoS | Gin panic without Recovery middleware | mitigate | Recovery() registered first in middleware chain; catches all handler panics and returns 500 |
| T-04-05 | Information Disclosure | zerolog log output | mitigate | No DSN, passwords, or secrets logged; connection events log only "connected to X" messages without credentials |
</threat_model>

<verification>
- go build ./... exits 0 (entire module)
- grep -r "fmt.Println" internal/ cmd/ returns no matches
- grep -r "gin.Default()" internal/ cmd/ returns no matches
- internal/app/app.go contains "gin.New()"
- internal/middleware/logger.go contains "c.Request.Context()"
- cmd/api/main.go contains "RunMigrations"
- go test ./internal/telemetry/... -run TestBootstrap -count=1 exits 0
- When `make docker-up` is running: `go run ./cmd/api/...` starts and curl http://localhost:8080/api/v1/health returns {"status":"ok"}
- When server running: curl http://localhost:8080/metrics returns HTTP 200 with go_goroutines in body
</verification>

<success_criteria>
- Full module compiles with go build ./... exit 0
- No fmt.Println anywhere in source
- Gin uses gin.New() with correct middleware order
- /api/v1/health returns 200 {"status":"ok"}
- /metrics returns Prometheus-format metrics
- Request logs include trace_id field when OTel span is active
- Graceful shutdown on SIGTERM drains connections
</success_criteria>

<output>
Create J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-04-SUMMARY.md when done
</output>
