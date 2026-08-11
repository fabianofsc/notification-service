package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/clock"
	"github.com/nexus-shopping/notification-service/internal/config"
	"github.com/nexus-shopping/notification-service/internal/email"
	httppkg "github.com/nexus-shopping/notification-service/internal/http"
	"github.com/nexus-shopping/notification-service/internal/postgres"
	"github.com/nexus-shopping/notification-service/internal/sms"
	"github.com/nexus-shopping/notification-service/internal/worker"
)

func main() {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := postgres.RunMigrations(ctx, cfg.DatabaseURL); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	repo := postgres.NewRepository(pool)
	realClock := clock.Real{}
	idGen := clock.UUIDv7Generator{}
	emailProvider := &email.FakeProvider{Log: logger}
	smsProvider := &sms.FakeProvider{Log: logger}

	router := httppkg.NewRouter(httppkg.Dependencies{
		SendNotification: app.SendNotificationDeps{
			Notifications: repo,
			Clock:         realClock,
			IDs:           idGen,
		},
		GetNotification: app.GetNotificationDeps{
			Notifications: repo,
		},
		BasicAuthUser: cfg.BasicAuthUser,
		BasicAuthPass: cfg.BasicAuthPass,
		Pool:          pool,
	})

	w := worker.New(worker.WorkerDeps{
		Notifications: repo,
		Deliveries:    repo,
		Email:         emailProvider,
		Sms:           smsProvider,
		Clock:         realClock,
		IDs:           idGen,
		Log:           logger,
	}, worker.WorkerConfig{
		PollInterval:   cfg.PollInterval,
		BatchSize:      cfg.BatchSize,
		LeaseDuration:  cfg.LeaseDuration,
		MaxConcurrency: cfg.MaxConcurrency,
	})

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go func() {
		if err := w.Run(workerCtx); err != nil {
			slog.Error("worker error", "error", err)
		}
	}()

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		slog.Info("shutting down...")

		workerCancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("notification-service starting", "addr", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("notification-service stopped")
}