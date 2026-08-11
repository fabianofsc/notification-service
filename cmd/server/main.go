package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/clock"
	"github.com/nexus-shopping/notification-service/internal/config"
	httppkg "github.com/nexus-shopping/notification-service/internal/http"
	"github.com/nexus-shopping/notification-service/internal/postgres"
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
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	slog.Info("notification-service starting", "addr", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}