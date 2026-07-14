// Server component for application
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/jirugutema/rbac_service/config"
	"github.com/jirugutema/rbac_service/handler"
)

func New(cfg config.Config, db *pgxpool.Pool, rdb *redis.Client) *http.Server {
	health := handler.NewHealth(map[string]handler.Pinger{
		"postgres": db,
		"redis":    redisPinger{rdb},
	})

	mux := http.NewServeMux()
	registerRoutes(mux, health)

	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

type redisPinger struct{ c *redis.Client }

func (p redisPinger) Ping(ctx context.Context) error { return p.c.Ping(ctx).Err() }
