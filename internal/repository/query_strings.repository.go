package repository

var GetRoleQuery = `
		SELECT id, tenant_id, name, description, is_disabled, created_at, updated_at
		FROM roles
		WHERE id = $1`
