---
id: 01-plan-redis-rabbitmq
phase: 1
plan: 03
type: execute
wave: 2
depends_on:
  - 01-01-project-scaffold
objective: "Wire Redis client with centralized key constants, and RabbitMQ connection recovery + topology provisioning + publisher + consumer skeletons"
files_modified:
  - internal/redis/client.go
  - internal/redis/keys.go
  - internal/redis/keys_test.go
  - internal/rabbitmq/connection.go
  - internal/rabbitmq/topology.go
  - internal/rabbitmq/publisher.go
  - internal/rabbitmq/consumer.go
  - internal/rabbitmq/topology_test.go
requirements_addressed:
  - REQ-022
  - REQ-023
  - REQ-024
  - REQ-redis-key-schema
  - REQ-rabbitmq-topology
autonomous: true

must_haves:
  truths:
    - "Redis client connects to dev Redis container and Ping succeeds"
    - "All 5 key constructor functions produce keys matching REDIS_SCHEMA.md patterns"
    - "RabbitMQ connection reconnects automatically after a broker restart (reconnect goroutine via NotifyClose)"
    - "DeclareAll provisions game.events exchange, 5 domain queues with DLX bindings, game.dlx exchange, and game.dead.queue"
    - "Publisher sends messages via PublishWithContext with routing key and content type application/json"
    - "Consumer struct receives deliveries from a named queue with manual Ack/Nack"
  artifacts:
    - path: "internal/redis/client.go"
      provides: "Redis client factory"
      exports: ["NewClient"]
    - path: "internal/redis/keys.go"
      provides: "Centralized key constructors"
      exports: ["MatchStateKey", "UserConnectionKey", "QueueKey", "RateMoveKey", "ReconnectKey"]
    - path: "internal/rabbitmq/connection.go"
      provides: "AMQP connection with reconnect loop"
      exports: ["Connection", "New", "Channel"]
    - path: "internal/rabbitmq/topology.go"
      provides: "Idempotent topology provisioning"
      exports: ["DeclareAll"]
    - path: "internal/rabbitmq/publisher.go"
      provides: "Message publisher"
      exports: ["Publisher", "NewPublisher", "Publish"]
    - path: "internal/rabbitmq/consumer.go"
      provides: "Message consumer skeleton"
      exports: ["Consumer", "NewConsumer", "Consume"]
  key_links:
    - from: "internal/rabbitmq/connection.go"
      to: "internal/rabbitmq/topology.go"
      via: "DeclareAll called after every reconnect"
      pattern: "DeclareAll"
    - from: "internal/redis/keys.go"
      to: "REDIS_SCHEMA.md"
      via: "key format constants"
      pattern: "match:%s:state"
---

<objective>
Install go-redis v9 and amqp091-go, implement Redis client with all key constructor functions per REDIS_SCHEMA.md, and implement the full RabbitMQ adapter: connection recovery loop, idempotent topology provisioning (exchange + 5 queues + DLX), publisher, and consumer skeleton.

Purpose: All infra adapters except Postgres are wired in this plan. Redis key constants prevent typos across all later phases. RabbitMQ topology is provisioned idempotently so Phase 6 event consumers can bind immediately.
Output: internal/redis/ and internal/rabbitmq/ packages, unit tests for key format correctness, integration test for topology provisioning.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md
@J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md
@J:\sources\sudoku-pvp\docs\RABBITMQ_TOPOLOGY.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md

<interfaces>
From internal/config/config.go (created in plan 02 — these types are available):

type RedisConfig struct {
    Addr     string
    Password string
    DB       int
    PoolSize int
}

type RabbitMQConfig struct {
    URL string
}

Exported by this plan (downstream plans depend on these):

// internal/redis/client.go
func NewClient(cfg config.RedisConfig) (*redis.Client, error)

// internal/redis/keys.go
func MatchStateKey(matchID uuid.UUID) string          // → "match:{id}:state"
func UserConnectionKey(userID uuid.UUID) string        // → "user:{id}:connection"
func QueueKey(region, difficulty, stake string) string // → "queue:{region}:{difficulty}:{stake}"
func RateMoveKey(userID uuid.UUID) string              // → "rate:user:{id}:move"
func ReconnectKey(matchID, userID uuid.UUID) string    // → "match:{matchId}:reconnect:{userId}"

// internal/rabbitmq/connection.go
type Connection struct { ... }
func New(url string, logger zerolog.Logger) *Connection
func (c *Connection) Channel() (*amqp.Channel, error)
func (c *Connection) Close()

// internal/rabbitmq/topology.go
const ExchangeGameEvents = "game.events"
const ExchangeDLX        = "game.dlx"
const QueueDead          = "game.dead.queue"
func DeclareAll(ch *amqp.Channel) error

// internal/rabbitmq/publisher.go
type Publisher struct { ... }
func NewPublisher(conn *Connection) (*Publisher, error)
func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error

// internal/rabbitmq/consumer.go
type Consumer struct { ... }
func NewConsumer(conn *Connection, queueName string) (*Consumer, error)
func (c *Consumer) Consume(ctx context.Context, handler func(amqp.Delivery) error) error
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Redis client and key constants with unit tests</name>
  <files>
    internal/redis/client.go,
    internal/redis/keys.go,
    internal/redis/keys_test.go
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Pattern 3 Redis Client, Pitfall 1 correct module path, key constant examples)
    J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md
    J:\sources\sudoku-pvp\.planning\PROJECT.md (DEC-011 rate limit key, DEC-015 queue key schema)
  </read_first>
  <behavior>
    - TestMatchStateKey: MatchStateKey(uuid) returns string matching "match:"+uuid.String()+":state"
    - TestUserConnectionKey: UserConnectionKey(uuid) returns "user:"+uuid.String()+":connection"
    - TestQueueKey: QueueKey("asia","medium","100") returns "queue:asia:medium:100"
    - TestRateMoveKey: RateMoveKey(uuid) returns "rate:user:"+uuid.String()+":move"
    - TestReconnectKey: ReconnectKey(matchUUID, userUUID) returns "match:"+matchUUID.String()+":reconnect:"+userUUID.String()
    - TestKeyFormat_NoRawStrings: All five functions use fmt.Sprintf with named segments; no raw string literals with hardcoded UUIDs
  </behavior>
  <action>
    Install dependencies:
    go get github.com/redis/go-redis/v9@v9.20.0

    CRITICAL — use module path github.com/redis/go-redis/v9 (not github.com/go-redis/redis/v9). See RESEARCH.md Pitfall 1. Wrong path installs different package.

    Create internal/redis/client.go in package redis. Import "github.com/redis/go-redis/v9" as redis. Import config "sudoku-pvp/internal/config".

    Implement func NewClient(cfg config.RedisConfig) (*redis.Client, error):
    Create redis.NewClient with Options: Addr=cfg.Addr, Password=cfg.Password, DB=cfg.DB, PoolSize=cfg.PoolSize, MinIdleConns=5, DialTimeout=5*time.Second, ReadTimeout=3*time.Second, WriteTimeout=3*time.Second.
    Create context.WithTimeout(context.Background(), 5*time.Second), call rdb.Ping(ctx).Err(). Return fmt.Errorf("redis ping: %w", err) on failure. Return rdb, nil on success.

    Create internal/redis/keys.go in package redis. Import fmt, uuid "github.com/google/uuid".

    Implement all 5 key constructor functions per REDIS_SCHEMA.md and DEC-015:
    MatchStateKey(matchID uuid.UUID) string → fmt.Sprintf("match:%s:state", matchID)
    UserConnectionKey(userID uuid.UUID) string → fmt.Sprintf("user:%s:connection", userID)
    QueueKey(region, difficulty, stake string) string → fmt.Sprintf("queue:%s:%s:%s", region, difficulty, stake) — note the 3-segment format per DEC-015 (REDIS_SCHEMA.md authoritative; includes region unlike BACKEND_ARCHITECTURE.md shorthand)
    RateMoveKey(userID uuid.UUID) string → fmt.Sprintf("rate:user:%s:move", userID)
    ReconnectKey(matchID, userID uuid.UUID) string → fmt.Sprintf("match:%s:reconnect:%s", matchID, userID)

    Create internal/redis/keys_test.go in package redis (internal test). Write unit tests for all 5 functions verifying exact string format. Tests must NOT require Docker (pure string formatting, no network). Use a fixed UUID constant for deterministic assertions.
  </action>
  <verify>
    <automated>go test ./internal/redis/... -run TestMatchStateKey -count=1 exits 0</automated>
  </verify>
  <acceptance_criteria>
    - internal/redis/client.go contains "func NewClient(cfg config.RedisConfig) (*redis.Client, error)"
    - internal/redis/client.go contains "github.com/redis/go-redis/v9" (NOT go-redis/redis)
    - internal/redis/keys.go contains "func MatchStateKey"
    - internal/redis/keys.go contains "func UserConnectionKey"
    - internal/redis/keys.go contains "func QueueKey"
    - internal/redis/keys.go contains "func RateMoveKey"
    - internal/redis/keys.go contains "func ReconnectKey"
    - internal/redis/keys.go contains "queue:%s:%s:%s" (3 segments per DEC-015)
    - internal/redis/keys_test.go contains "TestMatchStateKey"
    - internal/redis/keys_test.go contains "TestQueueKey"
    - go test ./internal/redis/... -run Test -count=1 exits 0
    - go.mod contains "github.com/redis/go-redis/v9"
  </acceptance_criteria>
  <done>Redis client factory and all 5 key constructors implemented; unit tests for all key formats pass; correct go-redis module path confirmed in go.mod.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: RabbitMQ connection recovery, topology, publisher, and consumer</name>
  <files>
    internal/rabbitmq/connection.go,
    internal/rabbitmq/topology.go,
    internal/rabbitmq/publisher.go,
    internal/rabbitmq/consumer.go,
    internal/rabbitmq/topology_test.go
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Pattern 4 RabbitMQ Topology, Pattern 5 Connection Recovery, Pitfall 2 shared channels, Pitfall 8 x-death retry loop, Anti-Patterns)
    J:\sources\sudoku-pvp\docs\RABBITMQ_TOPOLOGY.md
    J:\sources\sudoku-pvp\.planning\PROJECT.md (DEC-012 topology spec)
  </read_first>
  <behavior>
    - TestDeclareAll_ExchangeExists: After DeclareAll on fresh RabbitMQ container, passive declare of "game.events" succeeds (no error)
    - TestDeclareAll_AllQueuesExist: After DeclareAll, passive declare of each of the 5 queues succeeds: ranking.queue, wallet.queue, mission.queue, analytics.queue, notification.queue
    - TestDeclareAll_DLXExists: After DeclareAll, game.dlx exchange and game.dead.queue exist
    - TestDeclareAll_Idempotent: Calling DeclareAll twice on same channel returns nil error both times
    - TestPublisher_Publish: Publisher.Publish with routing key "match.finished" and body []byte completes without error
    - TestConsumer_ReceivesMessage: After publishing to game.events with routing key bound to ranking.queue, Consumer.Consume receives the delivery
  </behavior>
  <action>
    Install:
    go get github.com/rabbitmq/amqp091-go@v1.11.0
    go get github.com/rs/zerolog@v1.35.1

    Create internal/rabbitmq/connection.go in package rabbitmq. Import amqp "github.com/rabbitmq/amqp091-go", zerolog "github.com/rs/zerolog", sync, time.

    Define type Connection struct with fields: mu sync.RWMutex, conn *amqp.Connection, url string, logger zerolog.Logger. The mutex protects conn.

    Implement func New(url string, logger zerolog.Logger) *Connection: return &Connection{url: url, logger: logger}. Call go c.reconnectLoop() before returning so the connection is established asynchronously. Alternatively, do a synchronous first dial to fail fast at startup: dial in New() returning (*Connection, error), then launch reconnectLoop() in a goroutine watching NotifyClose.

    Choose the synchronous initial dial pattern (fail fast at startup per RESEARCH.md): New() dials once synchronously and returns error if dial fails. After successful dial, call DeclareAll in a new channel, then launch reconnectLoop as a goroutine.

    Implement func (c *Connection) reconnectLoop(): loop: wait on notify channel (conn.NotifyClose). On close, log warning. Retry dial with 5s sleep between attempts. On successful reconnect, call setConn(newConn), open new channel, call DeclareAll, close that channel. The retry loop is infinite — context cancellation via Close() method.

    Implement func (c *Connection) setConn(conn *amqp.Connection): acquire mu.Lock(), set c.conn = conn, unlock.
    Implement func (c *Connection) Channel() (*amqp.Channel, error): acquire mu.RLock(), return error if c.conn == nil or c.conn.IsClosed(). Open new channel from c.conn. Each caller gets its own channel — NEVER share channels across goroutines (Pitfall 2).
    Implement func (c *Connection) Close(): close c.conn if not nil.

    Create internal/rabbitmq/topology.go in package rabbitmq. Import amqp "github.com/rabbitmq/amqp091-go".

    Define exported constants:
    ExchangeGameEvents = "game.events"
    ExchangeDLX = "game.dlx"
    QueueDead = "game.dead.queue"

    Define var queues as slice of anonymous struct{Name string; RoutingKey string} with entries per DEC-012:
    {"ranking.queue", "match.finished"},
    {"wallet.queue", "match.finished"},
    {"mission.queue", "mission.completed"},
    {"analytics.queue", "#"},
    {"notification.queue", "ranking.changed"}

    Note: wallet.queue binding uses "match.finished" per RABBITMQ_TOPOLOGY.md routing key list; analytics.queue uses "#" wildcard to receive all events per RESEARCH.md pattern.

    Implement func DeclareAll(ch *amqp.Channel) error:
    1. Declare game.events as topic exchange: ch.ExchangeDeclare(ExchangeGameEvents, amqp.ExchangeTopic, durable=true, autoDelete=false, internal=false, noWait=false, nil)
    2. Declare game.dlx as direct exchange: ch.ExchangeDeclare(ExchangeDLX, amqp.ExchangeDirect, durable=true, autoDelete=false, internal=false, noWait=false, nil)
    3. Declare game.dead.queue: ch.QueueDeclare(QueueDead, durable=true, autoDelete=false, exclusive=false, noWait=false, nil)
    4. Bind game.dead.queue to game.dlx with routing key "": ch.QueueBind(QueueDead, "", ExchangeDLX, false, nil)
    5. For each queue in queues: QueueDeclare with durable=true and args amqp.Table{"x-dead-letter-exchange": ExchangeDLX}; then QueueBind to ExchangeGameEvents with the queue's RoutingKey.
    Return any error wrapped with fmt.Errorf.

    Create internal/rabbitmq/publisher.go in package rabbitmq. Import amqp, context, fmt.

    Define type Publisher struct with field ch *amqp.Channel. NOTE: each Publisher owns its own channel — not shared.
    Implement func NewPublisher(conn *Connection) (*Publisher, error): call conn.Channel() to get a new channel. Return &Publisher{ch: ch}, nil.
    Implement func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error: call p.ch.PublishWithContext(ctx, ExchangeGameEvents, routingKey, mandatory=false, immediate=false, amqp.Publishing{ContentType: "application/json", Body: body}). Return wrapped error on failure.
    Implement func (p *Publisher) Close(): p.ch.Close().

    Create internal/rabbitmq/consumer.go in package rabbitmq. Import amqp, context, fmt.

    Define type Consumer struct with fields ch *amqp.Channel, queueName string.
    Implement func NewConsumer(conn *Connection, queueName string) (*Consumer, error): call conn.Channel(). Return &Consumer{ch: ch, queueName: queueName}, nil.
    Implement func (c *Consumer) Consume(ctx context.Context, handler func(amqp.Delivery) error) error:
    Call c.ch.Consume(c.queueName, consumer="", autoAck=false, exclusive=false, noLocal=false, noWait=false, nil) to get deliveries channel.
    Range over deliveries in goroutine: for each delivery, check x-death count — if count >= 3 call d.Ack(false) and skip (dead letter manually to avoid infinite loop per RESEARCH.md Pitfall 8). Otherwise call handler(d); if handler returns error call d.Nack(multiple=false, requeue=false) else call d.Ack(multiple=false).
    The Consume loop blocks until ctx is cancelled or channel closes. Return ctx.Err() or nil.
    Helper for x-death count: extract d.Headers["x-death"] as []interface{}, range entries casting each to amqp.Table, sum "count" field values.

    Create internal/rabbitmq/topology_test.go in package rabbitmq_test with build tag //go:build integration. Tests spin up RabbitMQ via testcontainers-go rabbitmq module, call DeclareAll, then verify exchange and queue existence via passive declare. Include TestDeclareAll_Idempotent.
  </action>
  <verify>
    <automated>go build ./internal/rabbitmq/... exits 0 and go build ./internal/redis/... exits 0</automated>
  </verify>
  <acceptance_criteria>
    - internal/rabbitmq/connection.go contains "func New("
    - internal/rabbitmq/connection.go contains "NotifyClose"
    - internal/rabbitmq/connection.go contains "reconnectLoop"
    - internal/rabbitmq/connection.go contains "sync.RWMutex"
    - internal/rabbitmq/topology.go contains "ExchangeGameEvents"
    - internal/rabbitmq/topology.go contains "ExchangeDLX"
    - internal/rabbitmq/topology.go contains "func DeclareAll(ch *amqp.Channel) error"
    - internal/rabbitmq/topology.go contains "x-dead-letter-exchange"
    - internal/rabbitmq/topology.go contains "analytics.queue"
    - internal/rabbitmq/publisher.go contains "func (p *Publisher) Publish("
    - internal/rabbitmq/publisher.go contains "PublishWithContext"
    - internal/rabbitmq/consumer.go contains "func (c *Consumer) Consume("
    - internal/rabbitmq/consumer.go contains "x-death"
    - internal/rabbitmq/topology_test.go contains "//go:build integration"
    - internal/rabbitmq/topology_test.go contains "TestDeclareAll"
    - go build ./internal/rabbitmq/... exits 0
    - go build ./internal/redis/... exits 0
    - go.mod contains "github.com/rabbitmq/amqp091-go"
    - go.mod contains "github.com/rs/zerolog"
  </acceptance_criteria>
  <done>RabbitMQ connection with reconnect loop, DeclareAll topology provisioning, publisher, and consumer implemented; channel-per-goroutine safety enforced; x-death count limit prevents infinite retry loops; integration test scaffold written.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| env var → RabbitMQConfig.URL | AMQP credentials enter via SUDOKU_RABBITMQ_URL; never committed |
| consumer delivery → handler | AMQP delivery bodies are untrusted input from the broker; each handler must validate payload before processing |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-03-01 | Information Disclosure | AMQP URL in logs | mitigate | Log only connection events (connected/disconnected), never log the full AMQP URL which contains credentials |
| T-03-02 | Tampering | Redis key collision | mitigate | All keys built via centralized constructor functions in keys.go; no raw string keys allowed in business code |
| T-03-03 | DoS | Goroutine leak from unclosed consumer | mitigate | Consumer.Consume selects on ctx.Done(); handler goroutine exits when context cancelled; always Ack or Nack every delivery |
| T-03-04 | DoS | Infinite DLX retry loop | mitigate | Consumer reads x-death count; after 3 rejections routes to game.dead.queue via Ack and skips requeue (per Pitfall 8) |
| T-03-05 | Tampering | Shared amqp.Channel across goroutines | mitigate | Connection.Channel() creates a new channel per call; Publisher and Consumer each own exactly one channel; no sharing |
| T-01-SC | Tampering | go get amqp091-go, go-redis installs | mitigate | Both packages verified in RESEARCH.md Package Legitimacy Audit as Approved |
</threat_model>

<verification>
- go build ./internal/redis/... exits 0
- go build ./internal/rabbitmq/... exits 0
- go test ./internal/redis/... -run TestMatchStateKey -count=1 exits 0
- go test ./internal/redis/... -run TestQueueKey -count=1 exits 0 (verifies 3-segment format)
- internal/rabbitmq/topology.go contains all 5 queue names: ranking.queue, wallet.queue, mission.queue, analytics.queue, notification.queue
- internal/rabbitmq/consumer.go contains "x-death" (Pitfall 8 guard)
- go.mod contains github.com/redis/go-redis/v9 (correct path, not go-redis org)
- go.mod contains github.com/rabbitmq/amqp091-go
</verification>

<success_criteria>
- Redis client factory compiles and ping test passes against Docker-started redis
- All 5 key constructors produce correct format per REDIS_SCHEMA.md; unit tests pass
- RabbitMQ topology declared idempotently on connect and reconnect
- Publisher sends to correct exchange with application/json content type
- Consumer handles Ack/Nack per delivery with x-death guard at 3 retries
- No shared channel between goroutines
</success_criteria>

<output>
Create J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-03-SUMMARY.md when done
</output>
