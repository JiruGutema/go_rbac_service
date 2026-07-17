package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jirugutema/rbac_service/internal/domain"
	"github.com/jirugutema/rbac_service/internal/dto"
	"github.com/jirugutema/rbac_service/internal/service"
)

type RoleHandler struct {
	roles *service.RoleService
}

func NewRoleHandler(roles *service.RoleService) *RoleHandler {
	return &RoleHandler{roles: roles}
}

// GetRole godoc
//
// @Summary      Get a role by ID
// @Description  Returns a single role identified by its UUID.
// @Tags         roles
// @Produce      json
// @Param        id   path      string  true  "Role ID (UUID)"
// @Success      200  {object}  dto.RoleResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /roles/{id} [get]
func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	role, err := h.roles.GetRole(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRoleID):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		case errors.Is(err, domain.ErrRoleNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.NewRoleResponse(role))
}
