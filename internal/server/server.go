// Server component for application
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/jirugutema/rbac_service/config"
	"github.com/jirugutema/rbac_service/internal/handler"
	"github.com/jirugutema/rbac_service/internal/repository"
	"github.com/jirugutema/rbac_service/internal/service"
)

func New(cfg config.Config, db *pgxpool.Pool, rdb *redis.Client) *http.Server {
	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}

	health := handler.NewHealth(map[string]handler.Pinger{
		"postgres": db,
		"redis":    redisPinger{rdb},
	})

	roleHandler := handler.NewRoleHandler(service.NewRoleService(repository.NewRoleRepository(db)))

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	registerRoutes(router, health, roleHandler)

	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

type redisPinger struct{ c *redis.Client }

func (r redisPinger) Ping(ctx context.Context) error { return r.c.Ping(ctx).Err() }
