package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/spf13/cobra"

	"github.com/superset-studio/kapstan/api/internal/controllers"
	"github.com/superset-studio/kapstan/api/internal/jobs"
	"github.com/superset-studio/kapstan/api/internal/platform/config"
	"github.com/superset-studio/kapstan/api/internal/platform/database"
	"github.com/superset-studio/kapstan/api/internal/platform/logging"
	"github.com/superset-studio/kapstan/api/internal/repositories"
	"github.com/superset-studio/kapstan/api/internal/services"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "kapstan",
		Short:   "Kapstan — open-source cloud management platform",
		Version: version,
	}

	root.AddCommand(serverCmd())
	root.AddCommand(migrateCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func serverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start the Kapstan API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			logger := logging.Setup(cfg.LogLevel, cfg.LogFormat)

			db, err := database.Connect(cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("connecting to database: %w", err)
			}
			defer db.Close()

			logger.Info("database connected")

			// Create pgx pool for River.
			pool, err := pgxpool.New(cmd.Context(), cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("creating pgx pool: %w", err)
			}
			defer pool.Close()

			// Run River migrations.
			migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
			if err != nil {
				return fmt.Errorf("creating river migrator: %w", err)
			}
			if _, err := migrator.Migrate(cmd.Context(), rivermigrate.DirectionUp, nil); err != nil {
				return fmt.Errorf("running river migrations: %w", err)
			}
			logger.Info("river migrations complete")

			// Build connection service for River workers.
			connRepo := repositories.NewConnectionRepository(db)
			connService := services.NewConnectionService(connRepo, cfg.EncryptionKey)

			// Create and start River client.
			riverClient, err := jobs.NewJobClient(pool, connService)
			if err != nil {
				return fmt.Errorf("creating river client: %w", err)
			}
			if err := riverClient.Start(cmd.Context()); err != nil {
				return fmt.Errorf("starting river client: %w", err)
			}
			logger.Info("river job worker started")

			e := controllers.NewRouter(db, cfg.JWTSecret, cfg.EncryptionKey)

			// Start server in a goroutine.
			go func() {
				addr := ":" + cfg.Port
				logger.Info("starting server", slog.String("addr", addr))
				if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
					logger.Error("server error", slog.String("error", err.Error()))
					os.Exit(1)
				}
			}()

			// Wait for interrupt signal, then gracefully shut down.
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit

			logger.Info("shutting down server")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Stop River client.
			if err := riverClient.Stop(ctx); err != nil {
				logger.Error("river client stop error", slog.String("error", err.Error()))
			}

			if err := e.Shutdown(ctx); err != nil {
				return fmt.Errorf("server shutdown: %w", err)
			}

			logger.Info("server stopped")
			return nil
		},
	}
}

func migrateCmd() *cobra.Command {
	var migrationsPath string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			logger := logging.Setup(cfg.LogLevel, cfg.LogFormat)

			logger.Info("running migrations", slog.String("path", migrationsPath))

			if err := database.RunMigrations(cfg.DatabaseURL, migrationsPath); err != nil {
				return fmt.Errorf("running migrations: %w", err)
			}

			logger.Info("migrations complete")
			return nil
		},
	}

	cmd.Flags().StringVar(&migrationsPath, "path", "migrations", "path to migration files")

	return cmd
}
