package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Health struct {
	deps map[string]Pinger
}

func NewHealth(deps map[string]Pinger) *Health {
	return &Health{deps: deps}
}

func (h *Health) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := gin.H{"status": "ok"}
	code := http.StatusOK

	for name, dep := range h.deps {
		if err := dep.Ping(ctx); err != nil {
			status[name] = "down"
			status["status"] = "degraded"
			code = http.StatusServiceUnavailable
			continue
		}
		status[name] = "up"
	}

	c.JSON(code, status)
}
