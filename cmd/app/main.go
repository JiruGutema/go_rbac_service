package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jirugutema/rbac_service/cmd/app/docs"
	"github.com/jirugutema/rbac_service/config"
	"github.com/jirugutema/rbac_service/internal/server"
)

// @title       RBAC Service API
// @version     1.0
// @description Role-Based Access Control (RBAC) service API.
// @BasePath    /
func main() {
	cfg := *config.LoadConfig()
	ctx := context.Background()
	dbURL := config.ConstructDBConnectionString(cfg)
	rURL := config.ConstructRedisConnectionString(cfg)
	db, err := config.NewPostgres(ctx, dbURL)
	if err != nil {
		slog.Error("postgres unavailable", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := config.NewRedis(ctx, rURL)
	if err != nil {
		slog.Error("redis unavailable", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	srv := server.New(cfg, db, rdb)

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		slog.Info("swagger", "url", fmt.Sprintf("http://localhost:%s/swagger/index.html", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
	}
}
