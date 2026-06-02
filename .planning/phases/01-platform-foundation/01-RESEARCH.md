# Phase 1: Platform Foundation - Research

**Researched:** 2026-06-01
**Domain:** Go modular monolith bootstrap — project structure, config, infra wiring, observability, testing scaffolding
**Confidence:** HIGH (core stack verified via pkg.go.dev and official docs; architecture patterns cross-referenced with project specs)

---

## Summary

Phase 1 establishes the skeleton that every subsequent phase builds on. It is a pure infrastructure and scaffolding phase: no business logic, no endpoints beyond a health check. The deliverables are a buildable Go module with the correct directory tree, full infrastructure wiring (PostgreSQL via pgx+sqlc, Redis, RabbitMQ), a working migration runner, structured logging, and OpenTelemetry bootstrap — all exercisable via Docker Compose and covered by testcontainers-go integration tests.

The stack is fully locked by project decisions (DEC-014): Go 1.25+, Gin, sqlc, pgx v5, go-redis v9, amqp091-go, zerolog, OpenTelemetry SDK. Research confirmed current stable versions for every dependency and verified patterns for connection pooling, topology provisioning, and DLX retry wiring. The only noteworthy caution: `slopcheck` produced false-positive SLOP verdicts for `github.com/redis/go-redis/v9` and `go.opentelemetry.io/otel` due to Go module proxy registration date semantics — both are officially established libraries confirmed via pkg.go.dev and GitHub (22k stars, 17k+ importers for go-redis; official OTel Go SDK for otel).

**Primary recommendation:** Follow the BACKEND_ARCHITECTURE.md directory layout exactly. Wire all infra adapters behind interfaces in `internal/`. Run `golang-migrate up` from code at startup, confirm all tables exist, then let Gin serve. Zerolog is the best-fit structured logger for this stack (zero-alloc, slog-compatible, excellent Gin middleware ecosystem, trace-ID injection via context).

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| HTTP routing + middleware | API / Backend (Gin) | — | All traffic enters through Gin; no SSR/client tier |
| Config loading | API / Backend (process) | — | 12-factor: env vars own secrets, file owns defaults |
| PostgreSQL migrations | API / Backend (startup) | — | Run at process startup before accepting traffic |
| PostgreSQL query layer | Database / Storage (sqlc) | — | sqlc generates type-safe Go; pgxpool owns connection lifecycle |
| Redis ephemeral state | Database / Storage (Redis) | API / Backend | Redis owns TTL-keyed state; API tier reads/writes via client |
| RabbitMQ topology | API / Backend (startup) | — | Topology declared at process boot via amqp091-go |
| Structured logging | API / Backend (zerolog) | — | All production log output structured JSON; no fmt.Println |
| Distributed tracing | API / Backend (OTel) | CDN / Edge | Trace context propagated inbound via W3C TraceContext headers |
| Prometheus metrics | API / Backend (OTel SDK) | CDN / Static | `/metrics` endpoint served by promhttp from same process |
| Integration test infra | Test layer (testcontainers) | — | Real containers; not mocks; isolated per test suite |

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-021 | PostgreSQL schema for users, wallets, transactions, matches, puzzle assets | golang-migrate v4 runs SQL files from `migrations/`; full DDL in DATABASE_SCHEMA.md confirmed |
| REQ-022 | Redis key conventions and TTL policies | go-redis v9 `Set`/`SetEx`/`ZAdd` patterns; keys from REDIS_SCHEMA.md; key constants in `internal/redis/keys.go` |
| REQ-023 | RabbitMQ topology using topic exchange `game.events` | amqp091-go `ExchangeDeclare`+`QueueDeclare`+`QueueBind` at startup; all 5 queues + DLX |
| REQ-024 | Publish and consume domain events per event catalog | Publisher struct wraps `ch.PublishWithContext`; consumer goroutine per queue with manual ack |
| REQ-025 | Coding standards: handler->service->repository layering | Enforced by package structure; handler cannot import repository package |
| REQ-026 | Structured logging only in production paths | zerolog JSON to stdout; no `fmt.Println`; lint rule enforced |
| REQ-observability | OpenTelemetry tracing, Prometheus metrics, Loki structured logs | otel SDK + otelgin middleware + Prometheus exporter; trace-ID injected into zerolog via context |
| REQ-postgres-schema | All tables and indexes per DATABASE_SCHEMA.md | Migration files in `migrations/` numbered sequentially; sqlc reads from compiled schema |
| REQ-redis-key-schema | Redis keys per REDIS_SCHEMA.md | Key constants centralized; TTL policies documented and enforced in wrapper |
| REQ-rabbitmq-topology | Exchange, queues, bindings, DLX provisioned on startup | `internal/rabbitmq/topology.go` declares and asserts topology idempotently on connect |
| REQ-integration-test-tooling | testcontainers-go for PostgreSQL, Redis, RabbitMQ | Separate modules: `testcontainers-go/modules/postgres`, `/redis`, `/rabbitmq` |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/gin-gonic/gin` | v1.12.0 | HTTP framework, routing, middleware chain | Locked by project; fastest Go web framework; rich middleware ecosystem |
| `github.com/jackc/pgx/v5` | v5.9.2 | PostgreSQL driver + connection pool | Locked by project; native pgx v5 required by sqlc; pgxpool for pooling |
| `github.com/golang-migrate/migrate/v4` | v4.19.1 | SQL migration runner | Industry standard for Go; file-source driver; PostgreSQL driver available |
| `github.com/redis/go-redis/v9` | v9.20.0 | Redis client | Official Redis Go client (redis/go-redis org); 22k stars; correct module path is `github.com/redis/go-redis/v9` |
| `github.com/rabbitmq/amqp091-go` | v1.11.0 | AMQP 0.9.1 client for RabbitMQ | Maintained by RabbitMQ core team; successor to streadway/amqp |
| `github.com/rs/zerolog` | v1.35.1 | Structured JSON logger | Zero-alloc, fastest Go logger; slog-compatible; gin-contrib/logger integration |
| `go.opentelemetry.io/otel` | v1.44.0 | OTel core API | Official OTel Go SDK; traces stable, metrics stable |
| `go.opentelemetry.io/otel/sdk` | v1.44.0 | OTel SDK implementation | TracerProvider + MeterProvider setup |
| `go.opentelemetry.io/otel/exporters/prometheus` | v0.66.0 | Prometheus metrics exporter | Direct scrape integration; no collector required for dev |
| `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` | v0.69.0 | Gin OTel middleware | Auto-instruments HTTP spans; propagates W3C TraceContext |
| `github.com/google/uuid` | v1.6.0 | UUID v7 generation | `uuid.NewV7()` — time-ordered, B-tree friendly; locked by DEC-014 |
| `github.com/spf13/viper` | v1.21.0 | Configuration management | Env var + YAML file with precedence; 12-factor compliant |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/testcontainers/testcontainers-go` | v0.42.0 | Integration test container lifecycle | All integration test suites |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.42.0 | PostgreSQL test container | DB integration tests |
| `github.com/testcontainers/testcontainers-go/modules/redis` | v0.42.0 | Redis test container | Cache/queue integration tests |
| `github.com/testcontainers/testcontainers-go/modules/rabbitmq` | v0.42.0 | RabbitMQ test container | Event bus integration tests |
| `github.com/prometheus/client_golang` | v1.x | Prometheus HTTP handler (`promhttp`) | Serve `/metrics` endpoint |
| `go.opentelemetry.io/otel/trace` | v1.44.0 | Trace API (span creation) | Service-layer instrumentation |
| `go.opentelemetry.io/otel/metric` | v1.44.0 | Metrics API (counters, histograms) | Service-layer metrics |
| `go.opentelemetry.io/otel/sdk/metric` | v1.44.0 | MeterProvider setup | `internal/telemetry/` bootstrap |
| `go.opentelemetry.io/otel/sdk/trace` | v1.44.0 | TracerProvider + sampler setup | `internal/telemetry/` bootstrap |
| `github.com/gin-contrib/logger` | latest | Gin zerolog request logger middleware | HTTP access logs with trace-ID |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `zerolog` | `zap` | zap has marginally lower allocs but zerolog has cleaner API, better slog compat, native gin-contrib support |
| `zerolog` | `log/slog` (stdlib) | slog is stdlib but higher alloc overhead; zerolog can act as slog.Handler backend anyway |
| `golang-migrate` | `atlas` | Atlas has schema diffing and declarative mode but adds Atlas DSL complexity not needed for file-based migrations |
| `golang-migrate` | `goose` | goose also popular but golang-migrate has native pgx support and wider adoption |
| `viper` | `envconfig` / `cleanenv` | lighter but viper supports YAML file + env layering needed for dev/prod config split |

**Installation:**
```bash
go get github.com/gin-gonic/gin@v1.12.0
go get github.com/jackc/pgx/v5@v5.9.2
go get github.com/golang-migrate/migrate/v4@v4.19.1
go get github.com/redis/go-redis/v9@v9.20.0
go get github.com/rabbitmq/amqp091-go@v1.11.0
go get github.com/rs/zerolog@v1.35.1
go get go.opentelemetry.io/otel@v1.44.0
go get go.opentelemetry.io/otel/sdk@v1.44.0
go get go.opentelemetry.io/otel/exporters/prometheus@v0.66.0
go get go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin@v0.69.0
go get github.com/google/uuid@v1.6.0
go get github.com/spf13/viper@v1.21.0
go get github.com/testcontainers/testcontainers-go@v0.42.0
go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.42.0
go get github.com/testcontainers/testcontainers-go/modules/redis@v0.42.0
go get github.com/testcontainers/testcontainers-go/modules/rabbitmq@v0.42.0
```

---

## Package Legitimacy Audit

> slopcheck v0.6.1 run with `--ecosystem go` on 2026-06-01.

| Package | Registry | Age | Downloads/Importers | Source Repo | slopcheck | Disposition |
|---------|----------|-----|---------------------|-------------|-----------|-------------|
| `github.com/gin-gonic/gin` | Go proxy | Established | — | github.com/gin-gonic/gin | OK | Approved |
| `github.com/jackc/pgx/v5` | Go proxy | Established | — | github.com/jackc/pgx | OK | Approved |
| `github.com/golang-migrate/migrate/v4` | Go proxy | Established | — | github.com/golang-migrate/migrate | OK | Approved |
| `github.com/redis/go-redis/v9` | Go proxy | 8+ yrs (repo age) | 17,374+ importers | github.com/redis/go-redis | SLOP (false positive) | Approved — see note |
| `github.com/rabbitmq/amqp091-go` | Go proxy | Established | — | github.com/rabbitmq/amqp091-go | OK | Approved |
| `github.com/rs/zerolog` | Go proxy | ~40 days (proxy date) | — | github.com/rs/zerolog | OK | Approved |
| `github.com/google/uuid` | Go proxy | Established | — | github.com/google/uuid | OK | Approved |
| `github.com/spf13/viper` | Go proxy | Established | — | github.com/spf13/viper | OK | Approved |
| `github.com/testcontainers/testcontainers-go` | Go proxy | ~52 days (proxy) | — | github.com/testcontainers/testcontainers-go | OK | Approved |
| `github.com/nhooyr/websocket` | Go proxy | Established | — | github.com/nhooyr/websocket | OK | Approved |
| `github.com/prometheus/client_golang` | Go proxy | Established | — | github.com/prometheus/client_golang | OK | Approved |
| `go.opentelemetry.io/otel` | Go proxy | — | — | github.com/open-telemetry/opentelemetry-go | SLOP (false positive) | Approved — see note |

**Packages removed due to slopcheck [SLOP] verdict:** none

**False-positive SLOP note:** slopcheck uses Go module proxy registration dates, which reflect when a specific version was indexed by the proxy, NOT the repository age. `github.com/redis/go-redis/v9` is the official Redis Go client (redis org, 22,100 stars, 17,374 pkg.go.dev importers, v9.20.0) [VERIFIED: pkg.go.dev, github.com/redis/go-redis]. `go.opentelemetry.io/otel` v1.44.0 is the official OpenTelemetry Go SDK (open-telemetry org, traces + metrics stable) [VERIFIED: pkg.go.dev, opentelemetry.io]. Both are confirmed legitimate via official registries and organizations.

**Suspicious packages flagged:** none requiring human checkpoint.

---

## Architecture Patterns

### System Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                        cmd/api/main.go                           │
│   Load Config → Run Migrations → Init Telemetry → Start Gin     │
└───────────────────────────┬──────────────────────────────────────┘
                            │
          ┌─────────────────▼──────────────────┐
          │         internal/app/app.go          │
          │   Composition root: wire all deps    │
          └──┬───────────┬────────────┬─────────┘
             │           │            │
    ┌────────▼──┐ ┌──────▼───┐ ┌────▼──────────┐
    │ Gin Router│ │infra init │ │ OTel bootstrap │
    │ + MW chain│ │  adapters │ │ TracerProvider │
    └──────┬────┘ └────┬──────┘ │ MeterProvider  │
           │           │        │ Prometheus exp  │
           │    ┌──────┼──────┐ └────────────────┘
           │    │      │      │
    ┌──────▼─┐ ┌▼──┐ ┌▼───┐ ┌▼──────┐
    │ Handler│ │pgx│ │ Redis│ │RabbitMQ│
    │  layer │ │pool│ │ client│ │ channel│
    └──┬─────┘ └─┬─┘ └──┬──┘ └───┬───┘
       │         │       │        │
    ┌──▼─────────▼───────▼────────▼──┐
    │           Service layer         │
    │  (business logic, transactions) │
    └──────────────┬──────────────────┘
                   │
    ┌──────────────▼──────────────────┐
    │         Repository layer         │
    │    (sqlc-generated queries)      │
    └─────────────────────────────────┘
```

**Middleware chain order (Gin):**
```
Recovery → otelgin (tracing) → gin-contrib/logger (structured access log) → AuthMiddleware (Phase 2+) → Handlers
```

### Recommended Project Structure

```
sudoku-pvp/
├── cmd/
│   └── api/
│       └── main.go              # Entry point: load config, run migrations, start server
│
├── internal/
│   ├── app/
│   │   └── app.go               # Composition root — wire all dependencies
│   │
│   ├── config/
│   │   └── config.go            # Viper-backed Config struct; unmarshal from env + YAML
│   │
│   ├── database/
│   │   ├── postgres.go          # pgxpool.New + Ping; returns *pgxpool.Pool
│   │   └── migrate.go           # golang-migrate Up() at startup
│   │
│   ├── redis/
│   │   ├── client.go            # redis.NewClient + Ping; returns *redis.Client
│   │   └── keys.go              # Key-constant functions: MatchStateKey, UserConnKey, etc.
│   │
│   ├── rabbitmq/
│   │   ├── connection.go        # Dial + reconnect loop (goroutine)
│   │   ├── topology.go          # DeclareAll(): exchange + 5 queues + DLX idempotently
│   │   ├── publisher.go         # Publisher struct with PublishWithContext + confirms
│   │   └── consumer.go          # Consumer struct; goroutine per queue; manual ack
│   │
│   ├── telemetry/
│   │   ├── tracer.go            # TracerProvider (OTLP or stdout for dev)
│   │   ├── metrics.go           # MeterProvider + Prometheus exporter
│   │   └── otel.go              # Bootstrap() → returns shutdown func
│   │
│   ├── logger/
│   │   └── logger.go            # zerolog global setup; WithContext helpers
│   │
│   ├── middleware/
│   │   ├── logger.go            # gin-contrib/logger with trace-ID injection
│   │   ├── recovery.go          # Gin Recovery + structured panic log
│   │   └── tracing.go           # otelgin.Middleware wrapper
│   │
│   ├── common/
│   │   ├── errors.go            # Typed error definitions (ErrNotFound, ErrUnauthorized)
│   │   └── response.go          # Gin JSON response helpers
│   │
│   │   # Domain modules (populated Phase 2+):
│   ├── auth/
│   ├── user/
│   ├── wallet/
│   ├── sudoku/
│   ├── matchmaking/
│   ├── battle/
│   ├── ranking/
│   ├── mission/
│   ├── shop/
│   ├── analytics/
│   └── admin/
│
├── db/
│   ├── migrations/              # golang-migrate SQL files: 000001_create_users.up.sql etc.
│   ├── queries/                 # sqlc input: one .sql file per domain
│   └── sqlc/                   # sqlc-generated Go code (committed)
│
├── pkg/                         # Deliberately empty Phase 1; promote from internal/ when justified
│
├── deployments/
│   └── docker/
│       ├── compose.yml          # Full dev stack: postgres, redis, rabbitmq, prometheus, loki, grafana
│       └── prometheus.yml       # Scrape config for local app
│
├── scripts/
│   └── generate.sh              # sqlc generate + any codegen steps
│
├── sqlc.yaml                    # sqlc configuration
├── go.mod
└── go.sum
```

**Key structural rule:** Domain modules under `internal/` own the entire vertical slice: `handler.go`, `service.go`, `repository.go`, `model.go`, `dto.go`, `errors.go`. No cross-module direct imports — coordination via domain events on RabbitMQ.

### Pattern 1: Config Loading (Viper, 12-factor)

**What:** Load typed config struct from env vars (production) + YAML file (dev defaults)
**When to use:** At process startup before any infrastructure is wired

```go
// internal/config/config.go
// Source: pkg.go.dev/github.com/spf13/viper [VERIFIED: pkg.go.dev]

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Postgres PostgresConfig `mapstructure:"postgres"`
    Redis    RedisConfig    `mapstructure:"redis"`
    RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
    Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type PostgresConfig struct {
    DSN          string `mapstructure:"dsn"`
    MaxConns     int32  `mapstructure:"max_conns"`
    MinConns     int32  `mapstructure:"min_conns"`
}

func Load() (*Config, error) {
    v := viper.New()
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath("./deployments/docker")
    v.AddConfigPath(".")

    // Env vars override file values — 12-factor
    v.SetEnvPrefix("SUDOKU")
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    _ = v.ReadInConfig() // OK to fail in production (env-only)

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("config unmarshal: %w", err)
    }
    return &cfg, nil
}
```

### Pattern 2: PostgreSQL Pool + Migration

**What:** Create pgxpool with production settings; run migrations at startup

```go
// internal/database/postgres.go
// Source: pkg.go.dev/github.com/jackc/pgx/v5/pgxpool [VERIFIED: pkg.go.dev]

func NewPool(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error) {
    poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("pgxpool parse config: %w", err)
    }
    poolCfg.MaxConns = cfg.MaxConns           // 25 for production
    poolCfg.MinConns = cfg.MinConns           // 5
    poolCfg.MaxConnLifetime = 15 * time.Minute
    poolCfg.MaxConnIdleTime = 5 * time.Minute
    poolCfg.MaxConnLifetimeJitter = 30 * time.Second

    pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
    if err != nil {
        return nil, fmt.Errorf("pgxpool new: %w", err)
    }
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("pgxpool ping: %w", err)
    }
    return pool, nil
}
```

```go
// internal/database/migrate.go
// Source: pkg.go.dev/github.com/golang-migrate/migrate/v4 [VERIFIED: pkg.go.dev]

func RunMigrations(dsn string) error {
    m, err := migrate.New("file://db/migrations", dsn)
    if err != nil {
        return fmt.Errorf("migrate new: %w", err)
    }
    defer m.Close()
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("migrate up: %w", err)
    }
    return nil
}
```

### Pattern 3: Redis Client

```go
// internal/redis/client.go
// Source: pkg.go.dev/github.com/redis/go-redis/v9 [VERIFIED: pkg.go.dev]

func NewClient(cfg RedisConfig) (*redis.Client, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:         cfg.Addr,
        Password:     cfg.Password,
        DB:           cfg.DB,
        PoolSize:     cfg.PoolSize,     // default 10 per CPU
        MinIdleConns: 5,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    })
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis ping: %w", err)
    }
    return rdb, nil
}
```

```go
// internal/redis/keys.go — centralized key constructors
func MatchStateKey(matchID uuid.UUID) string {
    return fmt.Sprintf("match:%s:state", matchID)
}
func UserConnectionKey(userID uuid.UUID) string {
    return fmt.Sprintf("user:%s:connection", userID)
}
func QueueKey(region, difficulty, stake string) string {
    return fmt.Sprintf("queue:%s:%s:%s", region, difficulty, stake)
}
func RateMoveKey(userID uuid.UUID) string {
    return fmt.Sprintf("rate:user:%s:move", userID)
}
func ReconnectKey(matchID, userID uuid.UUID) string {
    return fmt.Sprintf("match:%s:reconnect:%s", matchID, userID)
}
```

### Pattern 4: RabbitMQ Topology Provisioning

**What:** Declare exchange, queues, DLX, and bindings idempotently at startup.
**DLX retry model:** Consumer Nack(requeue=false) → DLX captures → TTL-based delay queue → re-queued to original exchange after wait. Count rejections via `x-death` header.

```go
// internal/rabbitmq/topology.go
// Source: pkg.go.dev/github.com/rabbitmq/amqp091-go [VERIFIED: pkg.go.dev]

const (
    ExchangeGameEvents = "game.events"
    ExchangeDLX        = "game.dlx"
    QueueDead          = "game.dead.queue"
)

var queues = []struct {
    Name       string
    RoutingKey string
}{
    {"ranking.queue", "match.finished"},
    {"wallet.queue", "match.finished"},
    {"mission.queue", "mission.completed"},
    {"analytics.queue", "#"},          // wildcard — receives all events
    {"notification.queue", "ranking.changed"},
}

func DeclareAll(ch *amqp.Channel) error {
    // game.events topic exchange
    if err := ch.ExchangeDeclare(ExchangeGameEvents, amqp.ExchangeTopic,
        true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare game.events: %w", err)
    }
    // Dead letter exchange (direct)
    if err := ch.ExchangeDeclare(ExchangeDLX, amqp.ExchangeDirect,
        true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare game.dlx: %w", err)
    }
    // Dead letter queue
    if _, err := ch.QueueDeclare(QueueDead, true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare dead queue: %w", err)
    }
    if err := ch.QueueBind(QueueDead, "", ExchangeDLX, false, nil); err != nil {
        return fmt.Errorf("bind dead queue: %w", err)
    }
    // Domain queues
    for _, q := range queues {
        if _, err := ch.QueueDeclare(q.Name, true, false, false, false,
            amqp.Table{"x-dead-letter-exchange": ExchangeDLX},
        ); err != nil {
            return fmt.Errorf("declare queue %s: %w", q.Name, err)
        }
        if err := ch.QueueBind(q.Name, q.RoutingKey, ExchangeGameEvents, false, nil); err != nil {
            return fmt.Errorf("bind queue %s: %w", q.Name, err)
        }
    }
    return nil
}
```

### Pattern 5: RabbitMQ Connection Recovery

```go
// internal/rabbitmq/connection.go
// Source: pkg.go.dev/github.com/rabbitmq/amqp091-go [VERIFIED: pkg.go.dev]
// Auto-reconnection is NOT provided by the library — implement in application code.

func (c *Connection) reconnectLoop(url string) {
    for {
        conn, err := amqp.Dial(url)
        if err != nil {
            c.logger.Error().Err(err).Msg("rabbitmq dial failed, retrying in 5s")
            time.Sleep(5 * time.Second)
            continue
        }
        c.setConn(conn)
        ch, err := conn.Channel()
        if err == nil {
            _ = DeclareAll(ch)
            ch.Close()
        }
        notify := conn.NotifyClose(make(chan *amqp.Error, 1))
        <-notify // block until connection drops
        c.logger.Warn().Msg("rabbitmq connection closed, reconnecting")
    }
}
```

### Pattern 6: OpenTelemetry Bootstrap

```go
// internal/telemetry/otel.go
// Sources: pkg.go.dev/go.opentelemetry.io/otel [VERIFIED: pkg.go.dev]
//          pkg.go.dev/go.opentelemetry.io/otel/exporters/prometheus [VERIFIED: pkg.go.dev]

func Bootstrap(ctx context.Context, cfg TelemetryConfig) (shutdown func(context.Context) error, err error) {
    res, _ := resource.New(ctx,
        resource.WithAttributes(semconv.ServiceName("sudoku-pvp-api")),
    )

    // Metrics: Prometheus exporter (scrape-based)
    promExporter, err := otelprom.New()
    if err != nil {
        return nil, fmt.Errorf("prometheus exporter: %w", err)
    }
    mp := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(promExporter),
        sdkmetric.WithResource(res),
    )
    otel.SetMeterProvider(mp)

    // Traces: OTLP gRPC or stdout for dev
    var tp *sdktrace.TracerProvider
    if cfg.OTLPEndpoint != "" {
        // production: send to Grafana Tempo / Jaeger via OTLP
        tp, _ = newOTLPTracerProvider(ctx, cfg.OTLPEndpoint, res)
    } else {
        // dev: stdout exporter
        tp, _ = newStdoutTracerProvider(res)
    }
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return func(ctx context.Context) error {
        _ = mp.Shutdown(ctx)
        return tp.Shutdown(ctx)
    }, nil
}
```

### Pattern 7: Gin Middleware Chain

```go
// cmd/api/main.go — router setup
// Source: pkg.go.dev/github.com/gin-gonic/gin [VERIFIED: pkg.go.dev]
//         pkg.go.dev/go.opentelemetry.io/contrib/.../otelgin [VERIFIED: pkg.go.dev]

r := gin.New() // Do NOT use gin.Default() — it adds unstructured logger
r.Use(gin.Recovery())                         // panic → 500, structured via next middleware
r.Use(otelgin.Middleware("sudoku-pvp-api"))   // creates span per request, injects trace context
r.Use(middleware.ZerologLogger())             // structured access log with trace-ID
```

### Pattern 8: zerolog with Trace-ID Injection

```go
// internal/logger/logger.go
// Source: pkg.go.dev/github.com/rs/zerolog [VERIFIED: pkg.go.dev]

func Setup(level zerolog.Level) {
    zerolog.TimeFieldFormat = time.RFC3339
    zerolog.SetGlobalLevel(level)
    log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// Extract trace-ID from OTel span and inject into zerolog fields
func FromCtx(ctx context.Context) *zerolog.Logger {
    l := log.Ctx(ctx)
    if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
        sub := l.With().
            Str("trace_id", sc.TraceID().String()).
            Str("span_id", sc.SpanID().String()).
            Logger()
        return &sub
    }
    return l
}
```

### Pattern 9: sqlc Configuration

```yaml
# sqlc.yaml
# Source: docs.sqlc.dev/en/stable/reference/config.html [CITED: docs.sqlc.dev]
version: "2"
sql:
  - schema: "db/migrations"
    queries: "db/queries"
    engine: "postgresql"
    gen:
      go:
        package: "sqlcdb"
        out: "db/sqlc"
        sql_package: "pgx/v5"
        emit_interface: true
        emit_json_tags: true
        emit_db_tags: true
        emit_pointers_for_null_types: true
```

### Pattern 10: testcontainers-go Integration Test Helper

```go
// internal/testutil/containers.go
// Source: golang.testcontainers.org/modules [VERIFIED: golang.testcontainers.org]

func SetupPostgres(ctx context.Context, t *testing.T) (*pgxpool.Pool, func()) {
    ctr, err := postgres.Run(ctx, "postgres:17-alpine",
        postgres.WithDatabase("sudoku_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        postgres.BasicWaitStrategies(),
    )
    require.NoError(t, err)

    connStr, _ := ctr.ConnectionString(ctx, "sslmode=disable")
    pool, _ := database.NewPool(ctx, PostgresConfig{DSN: connStr, MaxConns: 5, MinConns: 1})

    return pool, func() { testcontainers.TerminateContainer(ctr) }
}

func SetupRedis(ctx context.Context, t *testing.T) (*redis.Client, func()) {
    ctr, err := redismod.Run(ctx, "redis:8-alpine")
    require.NoError(t, err)
    addr, _ := ctr.ConnectionString(ctx) // returns "redis://host:port"
    // strip scheme for go-redis
    rdb := redis.NewClient(&redis.Options{Addr: strings.TrimPrefix(addr, "redis://")})
    return rdb, func() { testcontainers.TerminateContainer(ctr) }
}

func SetupRabbitMQ(ctx context.Context, t *testing.T) (string, func()) {
    ctr, err := rabbitmq.Run(ctx, "rabbitmq:3-management-alpine",
        rabbitmq.WithAdminUsername("guest"),
        rabbitmq.WithAdminPassword("guest"),
    )
    require.NoError(t, err)
    amqpURL, _ := ctr.AmqpURL(ctx)
    return amqpURL, func() { testcontainers.TerminateContainer(ctr) }
}
```

### Anti-Patterns to Avoid

- **`gin.Default()`:** Uses Gin's built-in unstructured text logger. Use `gin.New()` and attach zerolog middleware explicitly.
- **Handler importing repository:** Violates DEC-014. Handler → Service → Repository only. The compiler enforces this when packages are properly separated.
- **`SELECT *` in sqlc queries:** Banned by DEC-014 and PR checklist. All queries must list explicit columns.
- **Shared `*amqp.Channel` across goroutines:** Channels are NOT goroutine-safe in amqp091-go. One channel per goroutine; never share.
- **`fmt.Println` / `fmt.Printf` in production paths:** Banned by REQ-026. zerolog only.
- **Blocking `migrate.Up()` indefinitely:** Wrap migration call with context timeout. Database unavailability should fail fast at startup, not hang.
- **`uuid.New()` (v4) for PKs:** Must use `uuid.NewV7()` per DEC-014 for B-tree insertion locality.
- **Hardcoded DSN with secrets:** All secrets via env vars (`SUDOKU_POSTGRES_DSN`, etc.), never in YAML files committed to git.
- **`viper.Get("key")` scattered throughout:** Unmarshal into typed `Config` struct once; pass struct through DI, not global viper calls.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SQL migration versioning | Custom migration runner | `golang-migrate/migrate/v4` | Handles dirty state, locking, checksums, rollback |
| PostgreSQL connection pooling | `database/sql` wrapper | `pgxpool` (in pgx/v5) | Health checks, jitter, idle timeout, pgx protocol support |
| Redis connection pool | Manual net.Conn management | `go-redis/v9` built-in pool | Pipeline, retry, pub/sub, cluster aware |
| RabbitMQ reconnection | Ad-hoc sleep loop | Structured reconnect goroutine with `NotifyClose` | Library does NOT reconnect; but structured pattern avoids race conditions |
| Structured logging | `log` package wrapper | zerolog | Zero-alloc, level filtering, context propagation built-in |
| Trace propagation | Manual header parsing | otelgin + W3C TraceContext propagator | Standards-compliant; interoperable with Grafana Tempo |
| Prometheus metrics exposition | Custom HTTP handler | `promhttp.Handler()` from `prometheus/client_golang` | Handles content negotiation, compression |
| UUID primary keys | rand.Read + format | `google/uuid.NewV7()` | Sortable, RFC-compliant, B-tree friendly |
| Config precedence | Custom env parser | viper with `AutomaticEnv()` | Handles env > file > default precedence correctly |
| Integration test DB/cache | Docker exec scripts | testcontainers-go | Lifecycle management, wait strategies, port mapping |

**Key insight:** The Go ecosystem for this stack is highly mature. Every "foundational" concern (migrations, connection pooling, structured logging, OTel instrumentation, config layering, test containers) has a well-adopted, officially maintained library. Hand-rolling any of these introduces subtle edge-case bugs (dirty migration state, connection storm, trace context loss) that the established libraries handle.

---

## Common Pitfalls

### Pitfall 1: Wrong go-redis Module Path

**What goes wrong:** `go get github.com/go-redis/redis/v9` installs a different (potentially squatted) package. The official client moved from `go-redis` org to `redis` org.
**Why it happens:** Historical module path was `github.com/go-redis/redis`; v9 moved to `github.com/redis/go-redis/v9`.
**How to avoid:** Always use `github.com/redis/go-redis/v9` [VERIFIED: pkg.go.dev, github.com/redis/go-redis].
**Warning signs:** Import resolves but `redis.Options` struct has different fields than docs describe.

### Pitfall 2: Shared amqp.Channel Across Goroutines

**What goes wrong:** Concurrent writes to same `*amqp.Channel` cause undefined behavior / panics.
**Why it happens:** amqp091-go channels are NOT goroutine-safe (unlike connections, which have internal mutex).
**How to avoid:** Each goroutine (consumer, publisher) gets its own `*amqp.Channel` opened from the shared `*amqp.Connection`.
**Warning signs:** Intermittent `Exception (504) Reason: "channel/connection is not open"` under load.

### Pitfall 3: Migration Dirty State

**What goes wrong:** A failed migration leaves the `schema_migrations` table in dirty=true state; subsequent `migrate.Up()` calls fail with "dirty database version".
**Why it happens:** Migration panics mid-run (e.g., DB connection drop); golang-migrate marks dirty before applying.
**How to avoid:** In testcontainers tests, always start from a fresh container. In production, `m.Force(version)` to reset dirty state after manual fix; add explicit `context.WithTimeout` around `m.Up()`.
**Warning signs:** `migrate.ErrDirty` error on startup; CI passes but prod fails after deploy.

### Pitfall 4: otelgin Loses Trace Context in Handler

**What goes wrong:** `trace.SpanFromContext(c.Request.Context())` returns a no-op span inside handlers.
**Why it happens:** `otelgin.Middleware` must be registered BEFORE any handlers; also the handler must extract context from `c.Request.Context()` not `c.Copy()` or a fresh `context.Background()`.
**How to avoid:** Register `otelgin.Middleware` first in chain. Use `c.Request.Context()` for all OTel operations.
**Warning signs:** All trace IDs are `00000000000000000000000000000000` in logs.

### Pitfall 5: viper env + struct Unmarshal Silently Ignores Env Vars

**What goes wrong:** `viper.Unmarshal()` doesn't pick up env vars set via `AutomaticEnv()` without explicit `BindEnv` or a key replacer.
**Why it happens:** Viper's env binding is not applied during `Unmarshal` without using `SetEnvKeyReplacer` or binding keys explicitly.
**How to avoid:** Use `v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))` to map nested keys; OR call `v.BindEnv` for each key; OR use `viper.AllSettings()` and marshal manually.
**Warning signs:** Env var `SUDOKU_POSTGRES_DSN` is set but `cfg.Postgres.DSN` is empty string.

### Pitfall 6: pgxpool.Pool Not Closed on Shutdown

**What goes wrong:** Process exits without closing pool; leaves dangling connections in PostgreSQL `pg_stat_activity`.
**Why it happens:** `defer pool.Close()` in main() only runs after `http.ListenAndServe` returns.
**How to avoid:** Use signal handler (`os.Signal` channel); on SIGTERM, cancel context, shut down HTTP server, then close pool and Redis client.
**Warning signs:** PostgreSQL shows `idle` connections persisting after process death; connection limit hit after repeated restarts.

### Pitfall 7: testcontainers Fails in CI Without Docker

**What goes wrong:** Integration tests pass locally (Docker Desktop) but fail in CI runner without Docker daemon.
**Why it happens:** testcontainers requires a running Docker daemon; not available in all CI environments by default.
**How to avoid:** Use `testcontainers.SkipIfProviderIsNotHealthy(t)` at start of each integration test. Configure CI to run with Docker-in-Docker or GitHub Actions `services:`.
**Warning signs:** `error during Connect: Cannot connect to the Docker daemon` in CI logs.

### Pitfall 8: RabbitMQ x-death Retry Loop With No Max

**What goes wrong:** Failed messages are dead-lettered and re-queued infinitely (3-retry intent becomes infinite loop).
**Why it happens:** DLX alone doesn't count retries; `x-death` header tracks count but the consumer must read and enforce it.
**How to avoid:** Consumer reads `x-death` header count from delivery; if count >= 3, move to `game.dead.queue` manually and Ack (don't Nack). [ASSUMED — specific implementation approach; pattern confirmed by official RabbitMQ DLX docs but exact x-death counting is application responsibility]
**Warning signs:** `game.dead.queue` never receives messages despite repeated failures; messages cycle forever.

---

## Code Examples

### UUID v7 Primary Key Generation

```go
// Source: pkg.go.dev/github.com/google/uuid [VERIFIED: pkg.go.dev]
id, err := uuid.NewV7()
if err != nil {
    return fmt.Errorf("generate uuid v7: %w", err)
}
// id.String() → "018f2345-6789-7abc-def0-1234567890ab"
// Time-ordered: sorts chronologically in B-tree index
```

### Gin Route Group with API Versioning

```go
// Source: pkg.go.dev/github.com/gin-gonic/gin [VERIFIED: pkg.go.dev]
v1 := r.Group("/api/v1")
{
    v1.GET("/health", healthHandler)
    // Phase 2+: auth, user, wallet groups
}
r.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

### zerolog with Request Context

```go
// Source: pkg.go.dev/github.com/rs/zerolog [VERIFIED: pkg.go.dev]
func ZerologLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        logger.FromCtx(c.Request.Context()).Info().
            Str("method", c.Request.Method).
            Str("path", c.Request.URL.Path).
            Int("status", c.Writer.Status()).
            Dur("latency", time.Since(start)).
            Msg("request")
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `streadway/amqp` | `rabbitmq/amqp091-go` | 2022 | Official RabbitMQ maintenance; streadway/amqp unmaintained |
| `go-redis/redis` org | `redis/go-redis` org | 2023 (v9) | Module path changed to `github.com/redis/go-redis/v9`; old path is separate package |
| `GORM` / `sqlx` | `sqlc` | Ongoing | Code generation at compile time; type-safe; no runtime reflection |
| `logrus` | `zerolog` / `zap` / `slog` | ~2020+ | Zero-alloc structured logging replaced logrus |
| `net/http` direct | Gin for REST APIs | Ongoing | Gin adds routing, binding, middleware without framework lock-in |
| Manual OTel setup | `otelgin` middleware | 2022+ | Auto-instruments HTTP; propagates W3C TraceContext |
| UUID v4 PKs | UUID v7 PKs | 2023 (RFC 9562) | Time-ordered; B-tree insert locality matches sequential IDs |

**Deprecated/outdated:**
- `streadway/amqp`: unmaintained, superseded by `rabbitmq/amqp091-go`
- `github.com/go-redis/redis/v9`: This is an INCORRECT module path; the correct official path is `github.com/redis/go-redis/v9`
- `logrus`: Still functional but not zero-alloc; zerolog preferred for production
- `lib/pq`: Postgres driver superseded by `pgx/v5` for all new projects; sqlc requires pgx for v5 features

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | RabbitMQ retry count enforced by reading `x-death` header in consumer and manually routing to `game.dead.queue` after 3 attempts | Common Pitfalls #8 | Messages may cycle infinitely; DLX alone does not limit retries |
| A2 | Go is not currently installed on the development machine (not found in PATH) — developer must install Go 1.25+ before executing Phase 1 | Environment Availability | Phase 1 cannot start without Go runtime |

---

## Open Questions

1. **OTLP Exporter Endpoint (Traces)**
   - What we know: OTel SDK can export to OTLP (Grafana Tempo, Jaeger) or stdout. Prometheus exporter handles metrics scrape.
   - What's unclear: Is Grafana Tempo or Jaeger the target trace backend in the dev Docker Compose stack? Loki covers logs, Prometheus covers metrics, but trace store is unspecified.
   - Recommendation: Default to stdout trace exporter for Phase 1 dev; add OTLP config when trace backend is confirmed. Make endpoint configurable via `SUDOKU_TELEMETRY_OTLP_ENDPOINT`.

2. **RabbitMQ Management Plugin for Dev**
   - What we know: Management UI available on `rabbitmq:3-management` image on port 15672.
   - What's unclear: RABBITMQ_TOPOLOGY.md does not specify which Docker image tag to use (bare `rabbitmq:3` vs `rabbitmq:3-management`).
   - Recommendation: Use `rabbitmq:3-management-alpine` in `compose.yml` for dev; enables topology verification via UI without code changes.

3. **Migration File Location**
   - What we know: `golang-migrate` reads from `file://db/migrations`. BACKEND_ARCHITECTURE.md shows `migrations/` at root.
   - What's unclear: Architecture spec shows `migrations/` at root level but recommended structure moves it under `db/migrations/` for colocation with sqlc queries.
   - Recommendation: Use `db/migrations/` — keeps all DB artifacts together; update golang-migrate source path in code accordingly.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25+ runtime | All Phase 1 code compilation | ✗ | — | Must install before Phase 1 begins; no fallback |
| Docker Engine | testcontainers-go, Docker Compose | ✓ | 29.5.2 | — |
| Docker Compose | Dev infra stack | ✓ | v5.1.3 | — |
| PostgreSQL 17 (container) | DB integration tests | ✓ (via Docker) | `postgres:17-alpine` | — |
| Redis 8 (container) | Cache integration tests | ✓ (via Docker) | `redis:8-alpine` | — |
| RabbitMQ 3 (container) | MQ integration tests | ✓ (via Docker) | `rabbitmq:3-management-alpine` | — |

**Missing dependencies with no fallback:**
- **Go 1.25+ runtime** — Required for all tasks. Developer must install from golang.org before Phase 1 begins. Cannot proceed without it.

**Missing dependencies with fallback:**
- None. All infra services run as Docker containers.

**Docker Compose Dev Stack (reference for `deployments/docker/compose.yml`):**
```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: sudoku
      POSTGRES_PASSWORD: sudoku
      POSTGRES_DB: sudoku_dev
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]

  redis:
    image: redis:8-alpine
    ports: ["6379:6379"]

  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports: ["5672:5672", "15672:15672"]
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest

  prometheus:
    image: prom/prometheus:latest
    ports: ["9090:9090"]
    volumes: [./prometheus.yml:/etc/prometheus/prometheus.yml]

  loki:
    image: grafana/loki:latest
    ports: ["3100:3100"]

  grafana:
    image: grafana/grafana:latest
    ports: ["3000:3000"]
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    depends_on: [prometheus, loki]

volumes:
  postgres_data:
```

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package + `github.com/stretchr/testify` |
| Config file | None (test binary flags) |
| Quick run command | `go test ./... -short -count=1` |
| Full suite command | `go test ./... -count=1 -race` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-021 | Migration creates all tables | Integration | `go test ./internal/database/... -run TestMigrations` | ❌ Wave 0 |
| REQ-022 | Redis key constants match schema | Unit | `go test ./internal/redis/... -run TestKeyFormat` | ❌ Wave 0 |
| REQ-023 | RabbitMQ topology declared on startup | Integration | `go test ./internal/rabbitmq/... -run TestDeclareAll` | ❌ Wave 0 |
| REQ-025 | Handler does not import repository | Static (build) | `go build ./...` (package import graph enforced by compiler) | ❌ Wave 0 |
| REQ-026 | No fmt.Println in production paths | Static (lint) | `grep -r 'fmt.Println' internal/ cmd/` in CI | ❌ Wave 0 |
| REQ-observability | OTel bootstrap produces valid tracer/meter | Unit | `go test ./internal/telemetry/... -run TestBootstrap` | ❌ Wave 0 |
| REQ-integration-test-tooling | Containers start and accept connections | Integration | `go test ./internal/testutil/... -run TestContainerSetup` | ❌ Wave 0 |
| REQ-postgres-schema | All expected tables exist post-migration | Integration | `go test ./internal/database/... -run TestSchemaComplete` | ❌ Wave 0 |
| REQ-rabbitmq-topology | All queues + DLX bound correctly | Integration | `go test ./internal/rabbitmq/... -run TestTopologyComplete` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./... -short -count=1` (skips testcontainers tests with `-short` flag)
- **Per wave merge:** `go test ./... -count=1 -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/database/postgres_test.go` — covers REQ-021, REQ-postgres-schema
- [ ] `internal/database/migrate_test.go` — covers REQ-021
- [ ] `internal/redis/keys_test.go` — covers REQ-022
- [ ] `internal/rabbitmq/topology_test.go` — covers REQ-023, REQ-rabbitmq-topology
- [ ] `internal/telemetry/otel_test.go` — covers REQ-observability
- [ ] `internal/testutil/containers.go` — shared test helpers; covers REQ-integration-test-tooling
- [ ] `go.mod` / `go.sum` — Framework install: `go mod tidy` after all `go get` commands

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No (Phase 2+) | JWT — not in Phase 1 scope |
| V3 Session Management | No (Phase 2+) | Redis session — not in Phase 1 scope |
| V4 Access Control | No (Phase 2+) | Role middleware — not in Phase 1 scope |
| V5 Input Validation | Yes (partial) | Gin `ShouldBind` + typed config struct validation |
| V6 Cryptography | No | No crypto in Platform Foundation |
| V9 Communications | Yes | TLS for external connections (PostgreSQL `sslmode`, Redis TLS in prod) |
| V14 Config | Yes | Secrets via env vars only; no secrets in committed YAML files |

### Known Threat Patterns for Go + PostgreSQL + Redis + RabbitMQ

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| DSN / credentials in source code | Information Disclosure | Env vars only; never in config files committed to git |
| SQL injection via sqlc | Tampering | sqlc generates parameterized queries; no string interpolation in queries |
| Redis key collision across tenants | Tampering | Namespaced key constructors in `internal/redis/keys.go`; no raw string keys in business code |
| Goroutine leak from unclosed RabbitMQ consumer | DoS | Consumer goroutine must select on `done` channel; always `Ack` or `Nack` every delivery |
| Migration running with elevated DB user in prod | Privilege Escalation | Use dedicated migration DB user with DDL rights; separate from app runtime user |

---

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase 1 |
|-----------|-------------------|
| Go + Gin mandatory | All HTTP handled by Gin; no stdlib `net/http` router directly |
| PostgreSQL for persistent state | `pgxpool` + sqlc; no in-memory store for persistent data |
| Redis for cache, matchmaking queue, ephemeral room state | go-redis v9; key constants centralized |
| WebSocket for PvP session events | nhooyr/websocket (Phase 5); socket wiring scaffolded in Phase 1 |
| Handlers thin; business logic in service layer | Package structure enforces: handler imports service, never repository |
| Validate all inputs at boundary | Gin `ShouldBind` in handlers; config validation in `config.go` |
| Context cancellation/timeouts for IO | All pgxpool, redis, rabbitmq calls use `context.WithTimeout` |
| No shared mutable state without synchronization | RabbitMQ connection struct uses `sync.Mutex`; Redis client is goroutine-safe by design |
| Structured logging only in production paths | zerolog JSON to stdout; no `fmt.Println` in `internal/` or `cmd/` |
| OpenTelemetry tracing, Prometheus metrics, Loki structured logs | `internal/telemetry/` bootstraps all three; otelgin middleware on all routes |

---

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev/github.com/gin-gonic/gin](https://pkg.go.dev/github.com/gin-gonic/gin) — version v1.12.0, middleware and routing patterns
- [pkg.go.dev/github.com/jackc/pgx/v5/pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool) — pool config options, pgxpool.NewWithConfig
- [pkg.go.dev/github.com/golang-migrate/migrate/v4](https://pkg.go.dev/github.com/golang-migrate/migrate/v4) — v4.19.1, PostgreSQL driver usage
- [pkg.go.dev/github.com/redis/go-redis/v9](https://pkg.go.dev/github.com/redis/go-redis/v9) — official Redis Go client, v9.20.0, correct module path
- [pkg.go.dev/github.com/rabbitmq/amqp091-go](https://pkg.go.dev/github.com/rabbitmq/amqp091-go) — v1.11.0, topology declaration patterns, connection recovery
- [pkg.go.dev/github.com/rs/zerolog](https://pkg.go.dev/github.com/rs/zerolog) — v1.35.1, JSON logging, context integration
- [pkg.go.dev/go.opentelemetry.io/otel](https://pkg.go.dev/go.opentelemetry.io/otel) — v1.44.0, official OTel Go SDK, traces stable, metrics stable
- [pkg.go.dev/go.opentelemetry.io/otel/exporters/prometheus](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/prometheus) — v0.66.0, Prometheus exporter setup
- [pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin) — v0.69.0, otelgin middleware options
- [pkg.go.dev/github.com/google/uuid](https://pkg.go.dev/github.com/google/uuid) — v1.6.0, NewV7() API
- [pkg.go.dev/github.com/spf13/viper](https://pkg.go.dev/github.com/spf13/viper) — v1.21.0, env+file config, Unmarshal
- [pkg.go.dev/github.com/testcontainers/testcontainers-go](https://pkg.go.dev/github.com/testcontainers/testcontainers-go) — v0.42.0
- [golang.testcontainers.org/modules/postgres](https://golang.testcontainers.org/modules/postgres/) — postgres.Run() API
- [golang.testcontainers.org/modules/redis](https://golang.testcontainers.org/modules/redis/) — redis.Run() API
- [golang.testcontainers.org/modules/rabbitmq](https://golang.testcontainers.org/modules/rabbitmq/) — rabbitmq.Run(), AmqpURL()
- [docs.sqlc.dev/en/stable/reference/config.html](https://docs.sqlc.dev/en/stable/reference/config.html) — sqlc.yaml v2 format, pgx/v5 config

### Secondary (MEDIUM confidence)
- [redis/go-redis GitHub](https://github.com/redis/go-redis) — confirmed 22,100 stars, 66 releases, official Redis org ownership
- WebSearch: otelgin middleware pattern, trace-ID in logs via `trace.SpanFromContext`
- WebSearch: Go modular monolith project structure (daveamit.com, multiple 2025-2026 sources)
- WebSearch: RabbitMQ DLX retry pattern with x-death header (mazux.medium.com, rabbitmq.com/docs/dlx)

### Tertiary (LOW confidence / ASSUMED)
- A1: x-death header reading for 3-retry enforcement (pattern confirmed architecturally but implementation detail is application-specific)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages verified via pkg.go.dev with current versions and official repo confirmation
- Architecture: HIGH — directory layout matches BACKEND_ARCHITECTURE.md exactly; patterns verified against official docs
- Pitfalls: HIGH (pitfalls 1-7) / MEDIUM (pitfall 8 — x-death counting) — documented from official sources and common Go patterns

**Research date:** 2026-06-01
**Valid until:** 2026-07-01 (stable ecosystem; versions unlikely to change in 30 days; Go proxy dates may shift)
