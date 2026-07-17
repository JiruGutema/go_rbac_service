// Package dto defines the request and response shapes exposed over HTTP
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jirugutema/rbac_service/internal/domain"
)

type RoleResponse struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsDisabled  bool      `json:"is_disabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewRoleResponse(role domain.Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID,
		TenantID:    role.TenantID,
		Name:        role.Name,
		Description: role.Description,
		IsDisabled:  role.IsDisabled,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}
