// Package service holds the business-logic layer
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/jirugutema/rbac_service/internal/domain"
)

type RoleService struct {
	roles domain.RoleReader
}

func NewRoleService(roles domain.RoleReader) *RoleService {
	return &RoleService{roles: roles}
}

func (s *RoleService) GetRole(ctx context.Context, id uuid.UUID) (domain.Role, error) {
	if id == uuid.Nil {
		return domain.Role{}, ErrInvalidRoleID
	}

	role, err := s.roles.GetRoleRepository(ctx, id)
	if err != nil {
		return domain.Role{}, fmt.Errorf("get role %s: %w", id, err)
	}

	return role, nil
}
