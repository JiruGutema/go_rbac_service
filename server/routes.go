package server

import (
	"net/http"

	"github.com/jirugutema/rbac_service/handler"
)

func registerRoutes(mux *http.ServeMux, health *handler.Health) {
	mux.HandleFunc("GET /health", health.Check)
}
