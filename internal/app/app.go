// Package app is the composition root for the sudoku-pvp API service.
// All dependencies are wired here and exposed through the App struct.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	authgoogle "sudoku-pvp/internal/auth/google"
	"sudoku-pvp/internal/auth/handler"
	"sudoku-pvp/internal/auth/repository"
	"sudoku-pvp/internal/auth/service"
	"sudoku-pvp/internal/config"
	"sudoku-pvp/internal/database"
	"sudoku-pvp/internal/logger"
	"sudoku-pvp/internal/middleware"
	"sudoku-pvp/internal/rabbitmq"
	rdb "sudoku-pvp/internal/redis"
	"sudoku-pvp/internal/telemetry"
)

// App holds all wired application dependencies and the Gin router.
// Shutdown must be called on SIGTERM/SIGINT to drain connections cleanly.
type App struct {
	cfg               *config.Config
	router            *gin.Engine
	srv               *http.Server
	pool              *pgxpool.Pool
	redisClient       *redis.Client
	mqConn            *rabbitmq.Connection
	telemetryShutdown func(context.Context) error
	authHandler       *handler.AuthHandler
}

// New constructs the App by sequentially:
//  1. Setting up structured logging (zerolog global)
//  2. Bootstrapping OTel providers (Prometheus metrics + stdout traces)
//  3. Connecting to PostgreSQL
//  4. Connecting to Redis
//  5. Connecting to RabbitMQ and declaring topology
//  6. Building the Gin router with middleware + routes
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	// 1. Configure global zerolog logger first so subsequent steps are observable.
	logger.Setup(logger.ParseLevel(cfg.LogLevel))

	// 2. Bootstrap OpenTelemetry (MeterProvider + TracerProvider + propagator).
	telShutdown, err := telemetry.Bootstrap(ctx, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("telemetry bootstrap: %w", err)
	}

	// 3. PostgreSQL connection pool.
	pool, err := database.NewPool(ctx, cfg.Postgres)
	if err != nil {
		_ = telShutdown(ctx)
		return nil, fmt.Errorf("postgres pool: %w", err)
	}

	// 4. Redis client.
	redisClient, err := rdb.NewClient(cfg.Redis)
	if err != nil {
		pool.Close()
		_ = telShutdown(ctx)
		return nil, fmt.Errorf("redis client: %w", err)
	}

	// 4.5. Wire auth dependencies (after Redis + Postgres, before router setup).
	jwksCache := authgoogle.NewJWKSCache(cfg.Auth.GoogleClientID)
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
	authHndlr := handler.NewAuthHandler(authSvc)

	// 5. RabbitMQ connection (dial + topology declaration + reconnect loop).
	log := zerolog.Ctx(ctx)
	mqConn, err := rabbitmq.New(cfg.RabbitMQ.URL, *log)
	if err != nil {
		_ = redisClient.Close()
		pool.Close()
		_ = telShutdown(ctx)
		return nil, fmt.Errorf("rabbitmq connection: %w", err)
	}

	a := &App{
		cfg:               cfg,
		pool:              pool,
		redisClient:       redisClient,
		mqConn:            mqConn,
		telemetryShutdown: telShutdown,
		authHandler:       authHndlr,
	}

	// 6. Wire Gin router (must happen after infra connects so handler deps are available).
	a.router = a.setupRouter()

	return a, nil
}

// setupRouter builds a Gin engine with the required middleware chain and routes.
// Uses gin.New() per RESEARCH.md Anti-Patterns — NOT the default constructor,
// which attaches an unstructured text logger violating REQ-026.
//
// Middleware chain order per RESEARCH.md Pattern 7:
// Recovery -> Tracing (otelgin) -> ZerologLogger
func (a *App) setupRouter() *gin.Engine {
	r := gin.New()

	// Middleware registered in order: Recovery first (catches all panics, T-04-04),
	// then Tracing (injects OTel span into request context), then ZerologLogger
	// (reads span context from c.Request.Context() for trace_id injection).
	r.Use(middleware.Recovery())
	r.Use(middleware.Tracing())
	r.Use(middleware.ZerologLogger())

	// Prometheus /metrics endpoint — T-04-02: acceptable in dev; restrict in prod (Phase 7).
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 group.
	v1 := r.Group("/api/v1")
	v1.GET("/health", healthHandler)

	// Auth routes (public — no JWT middleware required).
	auth := v1.Group("/auth")
	auth.POST("/guest", a.authHandler.GuestLogin)
	auth.POST("/google", a.authHandler.GoogleLogin)
	auth.POST("/refresh", a.authHandler.Refresh)

	// Protected routes — JWTMiddleware registered AFTER global Recovery/Tracing/ZerologLogger chain.
	// T-02-05-05: auth middleware on specific route groups, not global, preserving /health and /metrics.
	jwtSecret := []byte(a.cfg.Auth.JWTSecret)
	protected := v1.Group("")
	protected.Use(middleware.JWTMiddleware(jwtSecret))
	protected.POST("/auth/logout", a.authHandler.Logout)

	// WebSocket endpoint — WSJWTMiddleware validates token before upgrade (D-19).
	// Phase 5 will replace wsPlaceholderHandler with the real WebSocket handler.
	r.GET("/ws/connect", middleware.WSJWTMiddleware(jwtSecret), wsPlaceholderHandler)

	return r
}

// healthHandler returns a simple JSON liveness response.
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "sudoku-pvp-api",
	})
}

// wsPlaceholderHandler responds with 503 until the WebSocket handler is implemented in Phase 5.
func wsPlaceholderHandler(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"message": "websocket not yet implemented"})
}

// Router returns the underlying http.Handler for use in tests.
// Integration tests use httptest.NewServer(a.Router()) to exercise the full routing stack.
func (a *App) Router() http.Handler {
	return a.router
}

// Run starts the HTTP server on addr (e.g. ":8080").
// Blocks until the server is shut down or returns an unexpected error.
// Returns nil on clean shutdown (http.ErrServerClosed) or a non-nil error otherwise.
func (a *App) Run(addr string) error {
	a.srv = &http.Server{Addr: addr, Handler: a.router}
	if err := a.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully drains all open connections and flushes telemetry.
// Call with a context derived from a signal handler with a 30-second timeout
// to avoid hanging the process on exit (Pitfall 6 — pgxpool dangling connections).
// The HTTP server is stopped first to stop accepting new requests, then
// infrastructure connections are drained, and finally telemetry is flushed.
func (a *App) Shutdown(ctx context.Context) {
	if a.srv != nil {
		_ = a.srv.Shutdown(ctx)
	}
	if a.mqConn != nil {
		a.mqConn.Close()
	}
	if a.redisClient != nil {
		_ = a.redisClient.Close()
	}
	if a.pool != nil {
		a.pool.Close()
	}
	if a.telemetryShutdown != nil {
		_ = a.telemetryShutdown(ctx)
	}
}
