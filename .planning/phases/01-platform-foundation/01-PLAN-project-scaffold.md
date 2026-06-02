---
id: 01-plan-project-scaffold
phase: 1
plan: 01
type: execute
wave: 1
depends_on: []
objective: "Install Go runtime, initialize Go module, create full directory structure, Docker Compose dev stack, Makefile, and .env.example"
files_modified:
  - go.mod
  - .env.example
  - Makefile
  - sqlc.yaml
  - deployments/docker/compose.yml
  - deployments/docker/prometheus.yml
  - deployments/docker/config.yaml
  - cmd/api/main.go
  - internal/app/app.go
  - internal/common/errors.go
  - internal/common/response.go
  - scripts/generate.sh
  - db/migrations/.gitkeep
  - db/queries/.gitkeep
  - db/sqlc/.gitkeep
  - pkg/.gitkeep
requirements_addressed:
  - REQ-025
  - REQ-026
autonomous: false

must_haves:
  truths:
    - "Go 1.25+ is installed and `go version` reports the correct version"
    - "go.mod declares module sudoku-pvp with Go 1.25+"
    - "All directories from BACKEND_ARCHITECTURE.md exist under the project root"
    - "docker compose up -d starts postgres, redis, rabbitmq, prometheus, loki, grafana successfully"
    - "cmd/api/main.go compiles with `go build ./cmd/api/...` exit 0"
    - "No fmt.Println appears anywhere in cmd/ or internal/"
  artifacts:
    - path: "go.mod"
      provides: "Module declaration for sudoku-pvp"
      contains: "module sudoku-pvp"
    - path: "deployments/docker/compose.yml"
      provides: "Dev infra stack"
      contains: "postgres"
    - path: "cmd/api/main.go"
      provides: "Entry point skeleton"
      contains: "func main()"
    - path: "internal/app/app.go"
      provides: "Composition root skeleton"
      contains: "package app"
    - path: "internal/common/errors.go"
      provides: "Typed error definitions"
      contains: "ErrNotFound"
    - path: "internal/common/response.go"
      provides: "Gin JSON response helpers"
      contains: "package common"
    - path: "sqlc.yaml"
      provides: "sqlc code generation config"
      contains: "version: \"2\""
    - path: "Makefile"
      provides: "Developer task runner"
      contains: "docker-up"
  key_links:
    - from: "cmd/api/main.go"
      to: "internal/app/app.go"
      via: "import"
      pattern: "sudoku-pvp/internal/app"
---

<objective>
Install Go runtime, initialize the Go module, create the complete project directory skeleton per BACKEND_ARCHITECTURE.md, wire Docker Compose dev stack, write Makefile, and create the minimal entry point + composition root skeletons. No infra wiring yet — that is Wave 2.

Purpose: Establish the buildable module and dev environment that all subsequent waves depend on.
Output: Compilable cmd/api/main.go, Docker Compose stack startable with `make docker-up`, full directory tree in place.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\.planning\ROADMAP.md
@J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md
@J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
</context>

<tasks>

<task type="checkpoint:human-action" gate="blocking">
  <name>Task 1: Install Go 1.25+ runtime</name>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Environment Availability section — Go runtime is missing, no fallback)
  </read_first>
  <what-built>Nothing automated — Go runtime is not present on this machine per RESEARCH.md Assumption A2. The executor cannot install the OS-level runtime.</what-built>
  <how-to-verify>
    1. Download Go 1.25+ installer from https://go.dev/dl/ for your OS/arch.
    2. Run the installer. Accept default install path.
    3. Open a new terminal and run: go version
    4. Confirm output contains "go1.25" or higher (e.g., "go version go1.25.0 windows/amd64").
    5. Confirm GOPATH is accessible: go env GOPATH
  </how-to-verify>
  <resume-signal>Type "go installed" once `go version` returns go1.25+ in a fresh terminal.</resume-signal>
</task>

<task type="auto">
  <name>Task 2: Initialize Go module and full project directory structure</name>
  <files>
    go.mod,
    cmd/api/main.go,
    internal/app/app.go,
    internal/config/.gitkeep,
    internal/database/.gitkeep,
    internal/redis/.gitkeep,
    internal/rabbitmq/.gitkeep,
    internal/telemetry/.gitkeep,
    internal/logger/.gitkeep,
    internal/middleware/.gitkeep,
    internal/common/errors.go,
    internal/common/response.go,
    internal/auth/.gitkeep,
    internal/user/.gitkeep,
    internal/wallet/.gitkeep,
    internal/sudoku/.gitkeep,
    internal/matchmaking/.gitkeep,
    internal/battle/.gitkeep,
    internal/ranking/.gitkeep,
    internal/mission/.gitkeep,
    internal/shop/.gitkeep,
    internal/analytics/.gitkeep,
    internal/admin/.gitkeep,
    db/migrations/.gitkeep,
    db/queries/.gitkeep,
    db/sqlc/.gitkeep,
    pkg/.gitkeep,
    scripts/generate.sh,
    sqlc.yaml,
    .env.example
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Recommended Project Structure section, Pattern 9 sqlc config, Anti-Patterns section)
    J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (Section 4 Repository Structure)
    J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
  </read_first>
  <action>
    Run `go mod init sudoku-pvp` at the project root (J:\sources\sudoku-pvp). This creates go.mod with module path sudoku-pvp and the installed Go version.

    Create all directories listed in BACKEND_ARCHITECTURE.md Section 4 by placing .gitkeep files: cmd/api/, internal/app/, internal/config/, internal/database/, internal/redis/, internal/rabbitmq/, internal/telemetry/, internal/logger/, internal/middleware/, internal/common/, internal/auth/, internal/user/, internal/wallet/, internal/sudoku/, internal/matchmaking/, internal/battle/, internal/ranking/, internal/mission/, internal/shop/, internal/analytics/, internal/admin/, db/migrations/, db/queries/, db/sqlc/, pkg/, scripts/, deployments/docker/.

    Create cmd/api/main.go with package main and a func main() that prints nothing (no fmt.Println — use os.Exit(0) for the placeholder). The function body is a // TODO: wire app comment. Import path for app package: sudoku-pvp/internal/app. Do NOT call app package yet — just import with blank identifier or leave the import commented so it compiles without the package existing.

    Create internal/app/app.go with package app. Define type App struct with a comment // Composition root - dependencies wired here. Include func New() *App returning &App{}. No imports beyond standard library needed yet.

    Create internal/common/errors.go with package common. Define sentinel errors: ErrNotFound = errors.New("not found"), ErrUnauthorized = errors.New("unauthorized"), ErrForbidden = errors.New("forbidden"), ErrConflict = errors.New("conflict"), ErrBadRequest = errors.New("bad request"), ErrInternal = errors.New("internal server error"). Import "errors".

    Create internal/common/response.go with package common. Define type ErrorResponse struct with fields Code string json:"code" and Message string json:"message". Define func Error(c *gin.Context, status int, code string, msg string) that calls c.JSON with ErrorResponse. Define func OK(c *gin.Context, data any) that calls c.JSON(200, data). Do NOT import gin yet — leave imports as a comment // TODO: import gin after go get in Wave 2; for now define struct types only without the gin handler function, OR import gin with a build tag — actually: since gin is not yet downloaded, define the file with only the ErrorResponse struct and a comment marking the handler functions as pending Wave 2 gin install. Keep the file compilable without gin: do not include gin.Context references in this wave.

    Create sqlc.yaml at project root with version "2", sql entry pointing schema at "db/migrations", queries at "db/queries", engine "postgresql", gen go package "sqlcdb" out "db/sqlc" sql_package "pgx/v5" emit_interface true emit_json_tags true emit_db_tags true emit_pointers_for_null_types true.

    Create scripts/generate.sh: shebang #!/bin/bash, command: sqlc generate. Make executable.

    Create .env.example with the following env var stubs (no real secrets):
    SUDOKU_SERVER_PORT=8080
    SUDOKU_POSTGRES_DSN=postgres://sudoku:sudoku@localhost:5432/sudoku_dev?sslmode=disable
    SUDOKU_POSTGRES_MAX_CONNS=25
    SUDOKU_POSTGRES_MIN_CONNS=5
    SUDOKU_REDIS_ADDR=localhost:6379
    SUDOKU_REDIS_PASSWORD=
    SUDOKU_REDIS_DB=0
    SUDOKU_REDIS_POOL_SIZE=10
    SUDOKU_RABBITMQ_URL=amqp://guest:guest@localhost:5672/
    SUDOKU_TELEMETRY_OTLP_ENDPOINT=
    SUDOKU_LOG_LEVEL=debug
  </action>
  <verify>
    <automated>go build ./cmd/api/... exits 0 (run from J:\sources\sudoku-pvp)</automated>
  </verify>
  <acceptance_criteria>
    - go.mod contains "module sudoku-pvp" on line 1
    - go.mod contains "go 1.25" or higher
    - cmd/api/main.go contains "func main()"
    - internal/app/app.go contains "package app"
    - internal/common/errors.go contains "ErrNotFound"
    - internal/common/response.go contains "ErrorResponse"
    - sqlc.yaml contains "version: \"2\""
    - sqlc.yaml contains "sql_package: \"pgx/v5\""
    - .env.example contains "SUDOKU_POSTGRES_DSN"
    - scripts/generate.sh contains "sqlc generate"
    - db/migrations/ directory exists
    - db/queries/ directory exists
    - db/sqlc/ directory exists
    - go build ./cmd/api/... exits 0
    - grep -r "fmt.Println" internal/ cmd/ returns no matches
  </acceptance_criteria>
  <done>Go module initialized at sudoku-pvp, all directories created, entry point and app skeleton compile cleanly, no fmt.Println in source.</done>
</task>

<task type="auto">
  <name>Task 3: Docker Compose dev stack and Makefile</name>
  <files>
    deployments/docker/compose.yml,
    deployments/docker/prometheus.yml,
    deployments/docker/config.yaml,
    Makefile
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Docker Compose Dev Stack reference under Environment Availability, Open Questions #2 rabbitmq image)
    J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (Section 14 Deployment)
  </read_first>
  <action>
    Create deployments/docker/compose.yml with the following services (per RESEARCH.md dev stack reference):

    postgres: image postgres:17-alpine, environment POSTGRES_USER=sudoku POSTGRES_PASSWORD=sudoku POSTGRES_DB=sudoku_dev, ports 5432:5432, volumes postgres_data:/var/lib/postgresql/data, healthcheck: test ["CMD-SHELL", "pg_isready -U sudoku"] interval 5s timeout 5s retries 5.

    redis: image redis:8-alpine, ports 6379:6379, healthcheck: test ["CMD", "redis-cli", "ping"] interval 5s timeout 3s retries 5.

    rabbitmq: image rabbitmq:3-management-alpine (per RESEARCH.md Open Question #2 recommendation), ports 5672:5672 and 15672:15672, environment RABBITMQ_DEFAULT_USER=guest RABBITMQ_DEFAULT_PASS=guest, healthcheck: test ["CMD", "rabbitmq-diagnostics", "ping"] interval 10s timeout 10s retries 5.

    prometheus: image prom/prometheus:latest, ports 9090:9090, volumes ./prometheus.yml:/etc/prometheus/prometheus.yml:ro, depends_on on nothing.

    loki: image grafana/loki:latest, ports 3100:3100.

    grafana: image grafana/grafana:latest, ports 3000:3000, environment GF_SECURITY_ADMIN_PASSWORD=admin, depends_on [prometheus, loki].

    volumes block: postgres_data: (empty driver default).

    Create deployments/docker/prometheus.yml with global scrape_interval 15s and scrape_configs job_name sudoku-api static_configs targets localhost:8080 metrics_path /metrics.

    Create deployments/docker/config.yaml as the default Viper config file for dev (RESEARCH.md Pattern 1 — viper looks for config.yaml in deployments/docker/). Populate with all nested keys matching the Config struct:
    server.port: 8080
    postgres.dsn: postgres://sudoku:sudoku@localhost:5432/sudoku_dev?sslmode=disable
    postgres.max_conns: 25
    postgres.min_conns: 5
    redis.addr: localhost:6379
    redis.password: ""
    redis.db: 0
    redis.pool_size: 10
    rabbitmq.url: amqp://guest:guest@localhost:5672/
    telemetry.otlp_endpoint: ""
    log_level: debug

    Create Makefile at project root with the following targets:
    - docker-up: docker compose -f deployments/docker/compose.yml up -d
    - docker-down: docker compose -f deployments/docker/compose.yml down
    - docker-logs: docker compose -f deployments/docker/compose.yml logs -f
    - build: go build -o bin/api ./cmd/api/...
    - run: go run ./cmd/api/...
    - test: go test ./... -short -count=1
    - test-integration: go test ./... -count=1 -race -timeout 120s
    - generate: bash scripts/generate.sh
    - lint: golangci-lint run ./...
    - tidy: go mod tidy
    Default target (.DEFAULT_GOAL := build).
  </action>
  <verify>
    <automated>docker compose -f deployments/docker/compose.yml up -d exits 0 and docker compose -f deployments/docker/compose.yml ps shows postgres, redis, rabbitmq as healthy within 30s</automated>
  </verify>
  <acceptance_criteria>
    - deployments/docker/compose.yml contains "postgres:17-alpine"
    - deployments/docker/compose.yml contains "redis:8-alpine"
    - deployments/docker/compose.yml contains "rabbitmq:3-management-alpine"
    - deployments/docker/compose.yml contains "grafana/loki"
    - deployments/docker/prometheus.yml contains "job_name"
    - deployments/docker/config.yaml contains "postgres"
    - Makefile contains "docker-up"
    - Makefile contains "test-integration"
    - docker compose -f deployments/docker/compose.yml up -d exits 0
    - docker compose -f deployments/docker/compose.yml ps output contains "healthy" for postgres, redis, rabbitmq (after 30s)
  </acceptance_criteria>
  <done>Dev infra stack starts successfully with `make docker-up`; all three infra services report healthy; Prometheus, Loki, Grafana services are running.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| env vars → config struct | Secrets enter process as environment variables; never in committed files |
| Docker Compose → host network | Infra ports exposed on 0.0.0.0 in dev; acceptable for local dev only |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-01-01 | Information Disclosure | .env.example | accept | .env.example contains no real secrets — only placeholder values; .env (with real values) is in .gitignore |
| T-01-02 | Tampering | go.mod dependency chain | mitigate | go.sum locks all transitive dependencies; `go mod verify` in CI validates checksums |
| T-01-03 | Tampering | npm/pip/cargo installs | accept | Phase 1 uses only `go get` — package legitimacy audit completed in RESEARCH.md; all packages Approved |
| T-01-SC | Tampering | go get package installs | mitigate | All packages verified in RESEARCH.md Package Legitimacy Audit; no [SLOP] or [SUS] packages; false-positive SLOP on go-redis and otel confirmed legitimate |
| T-01-04 | Information Disclosure | deployments/docker/config.yaml | accept | Contains dev-only non-secret defaults; real prod secrets always via env vars per 12-factor |
</threat_model>

<verification>
- go version reports go1.25 or higher
- go build ./cmd/api/... exits 0 from project root
- go.mod line 1 is "module sudoku-pvp"
- docker compose -f deployments/docker/compose.yml up -d exits 0
- docker compose -f deployments/docker/compose.yml ps shows postgres, redis, rabbitmq as (healthy)
- grep -r "fmt.Println" internal/ cmd/ returns no matches
- sqlc.yaml contains sql_package: "pgx/v5"
- All directories exist: cmd/api/, internal/app/, internal/config/, internal/database/, internal/redis/, internal/rabbitmq/, internal/telemetry/, internal/logger/, internal/middleware/, internal/common/, db/migrations/, db/queries/, db/sqlc/
</verification>

<success_criteria>
- Go 1.25+ installed and verified
- Module sudoku-pvp initialized with correct directory structure
- Docker Compose stack starts cleanly with make docker-up
- cmd/api/main.go compiles without errors
- No fmt.Println anywhere in cmd/ or internal/
- All Wave 2 plans can begin immediately (no blockers)
</success_criteria>

<output>
Create J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-01-SUMMARY.md when done
</output>
