package server

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/jirugutema/rbac_service/internal/handler"
)

func registerRoutes(r *gin.Engine, health *handler.Health, roles *handler.RoleHandler) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", health.Check)
	r.GET("/roles/:id", roles.GetRole)
}
