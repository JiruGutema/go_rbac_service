package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
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

func (h *Health) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := map[string]string{"status": "ok"}
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(status)
}
