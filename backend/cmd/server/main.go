// Command server wires everything together and runs the HTTP server with
// graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"shorturl/internal/config"
	"shorturl/internal/handler"
	"shorturl/internal/middleware"
	"shorturl/internal/repository"
	"shorturl/internal/router"
	"shorturl/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var log *slog.Logger
	if cfg.IsProduction() {
		log = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	} else {
		log = slog.New(slog.NewTextHandler(os.Stderr, opts))
	}
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- database pool ----
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Error("bad DATABASE_URL", "err", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Error("db pool error", "err", err)
		os.Exit(1)
	}
	pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := pool.Ping(pingCtx); err != nil {
		log.Error("database unreachable", "err", err)
		os.Exit(1)
	}
	pingCancel()

	// ---- layers ----
	usersRepo := repository.NewUsers(pool)
	sessionsRepo := repository.NewSessions(pool)
	urlsRepo := repository.NewURLs(pool)

	authSvc := service.NewAuthService(cfg, usersRepo, sessionsRepo, log)
	urlSvc := service.NewURLService(cfg, urlsRepo, log)
	statsSvc := service.NewStatsService(urlsRepo)
	limiter := middleware.NewLimiter()
	trusted := middleware.ParseTrustedProxies(cfg.TrustedProxies)

	ready := func(ctx context.Context) bool {
		pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return pool.Ping(pctx) == nil
	}
	api := handler.NewAPI(cfg, urlSvc, authSvc, statsSvc, limiter, trusted, ready)

	// background workers
	urlSvc.StartClickFlusher(ctx)
	authSvc.CleanupExpiredSessions(ctx)

	root := router.New(cfg, api, log, trusted)
	root = middleware.CORS(cfg.AllowedOrigins)(root)
	root = middleware.CSRF(cfg.BaseURL, cfg.AllowedOrigins)(root)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// ---- serve ----
	go func() {
		log.Info("server starting", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil {
		log.Error("shutdown error", "err", err)
	}

	// flush click buffer before exit (5s budget)
	fCtx, fCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer fCancel()
	if err := urlSvc.FlushNow(fCtx); err != nil {
		log.Error("click buffer flush on shutdown failed", "err", err)
	} else {
		log.Info("click buffer flushed")
	}

	pool.Close()
	log.Info("bye")
}
