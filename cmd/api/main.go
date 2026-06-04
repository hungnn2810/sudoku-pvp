// Package main is the entry point for the sudoku-pvp API service.
// Startup sequence:
//  1. Load configuration (env vars + optional YAML)
//  2. Run database migrations (fail-fast with 30s timeout)
//  3. Initialise application (telemetry, infra, Gin router)
//  4. Start signal handler goroutine (SIGTERM / SIGINT → graceful shutdown)
//  5. Serve HTTP
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"sudoku-pvp/internal/app"
	"sudoku-pvp/internal/config"
	"sudoku-pvp/internal/database"
)

func main() {
	// Root context — cancelled when the signal handler fires.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 1. Load configuration from env vars and/or config.yaml.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// 2. Run database migrations with a startup timeout.
	// Fail fast if the DB is unavailable — do not block indefinitely (Pitfall 3).
	migCtx, migCancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer migCancel()
	_ = migCtx // timeout context passed by value; RunMigrations uses its own deadline
	if err := database.RunMigrations(cfg.Postgres.DSN); err != nil {
		log.Fatal().Err(err).Msg("database migrations failed")
	}

	// 3. Initialise the application (telemetry → infra → router).
	a, err := app.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialise application")
	}

	// 4. Signal handler: SIGTERM and SIGINT trigger graceful shutdown.
	// Shutdown is given 30 seconds to drain in-flight requests and close connections.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-quit
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutCancel()
		a.Shutdown(shutCtx)
		rootCancel()
	}()

	// 5. Start the HTTP server. Blocks until the server errors or root context is cancelled.
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Info().Str("addr", addr).Msg("starting HTTP server")
	if err := a.Run(addr); err != nil {
		// Server error after graceful shutdown is expected; only fatal on unexpected errors.
		log.Error().Err(err).Msg("HTTP server stopped")
		os.Exit(1)
	}
}
