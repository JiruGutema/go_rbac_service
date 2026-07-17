// Package repository provides PostgreSQL-backed implementations of the domain repository interfaces.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jirugutema/rbac_service/internal/domain"
)

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) GetRoleRepository(ctx context.Context, id uuid.UUID) (domain.Role, error) {
	var role domain.Role
	err := r.db.QueryRow(ctx, GetRoleQuery, id).Scan(
		&role.ID,
		&role.TenantID,
		&role.Name,
		&role.Description,
		&role.IsDisabled,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Role{}, domain.ErrRoleNotFound
		}
		return domain.Role{}, fmt.Errorf("get role %s: %w", id, err)
	}

	return role, nil
}
