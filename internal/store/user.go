package store

import (
	"context"

	"github.com/schliz/convoke/internal/model"
)

// GetOrCreateUser upserts a user by email, updating groups and admin status on conflict.
func GetOrCreateUser(ctx context.Context, db DBTX, email string, displayName string, groups []string, isAdmin bool) (*model.User, error) {
	var u model.User
	err := db.QueryRow(ctx, `
		INSERT INTO users (email, display_name, groups, is_admin)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			groups = EXCLUDED.groups,
			is_admin = EXCLUDED.is_admin,
			updated_at = NOW()
		RETURNING id, email, display_name, groups, is_admin, created_at, updated_at
	`, email, displayName, groups, isAdmin).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.Groups, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
