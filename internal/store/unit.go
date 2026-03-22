package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/schliz/convoke/internal/db"
)

// unitCols is the column list for unit queries, table-alias-qualified for safe
// use in JOINs. Matches the field order in db.Unit.
const unitCols = `u.id, u.name, u.slug, u.description, u.logo_path, u.contact_email, u.admin_group, u.created_at, u.updated_at`

// ListUnits returns all units ordered by name.
func ListUnits(ctx context.Context, dbtx db.DBTX) ([]db.Unit, error) {
	return db.New(dbtx).ListUnits(ctx)
}

// GetUnitByID returns a single unit by its primary key.
// Returns pgx.ErrNoRows if no unit exists with the given ID.
func GetUnitByID(ctx context.Context, dbtx db.DBTX, id int64) (db.Unit, error) {
	return db.New(dbtx).GetUnitByID(ctx, id)
}

// GetUnitBySlug returns a single unit by its URL slug.
// Returns pgx.ErrNoRows if no unit exists with the given slug.
func GetUnitBySlug(ctx context.Context, dbtx db.DBTX, slug string) (db.Unit, error) {
	return db.New(dbtx).GetUnitBySlug(ctx, slug)
}

// ListUnitsByUserGroups returns units whose group bindings overlap with the
// provided IdP groups. Membership is resolved by joining units with
// unit_group_bindings and matching against the groups array using ANY().
//
// Returns nil, nil for nil or empty groups (no database round-trip).
func ListUnitsByUserGroups(ctx context.Context, dbtx db.DBTX, groups []string) ([]db.Unit, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	rows, err := dbtx.Query(ctx, `
		SELECT DISTINCT `+unitCols+`
		FROM units u
		JOIN unit_group_bindings ugb ON u.id = ugb.unit_id
		WHERE ugb.group_name = ANY($1::text[])
		ORDER BY u.name
	`, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var units []db.Unit
	for rows.Next() {
		var u db.Unit
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Slug, &u.Description,
			&u.LogoPath, &u.ContactEmail, &u.AdminGroup,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return units, nil
}

// IsUnitMember checks if any of the user's IdP groups match the unit's group
// bindings. This resolves membership without a dedicated membership table.
//
// Returns false, nil for nil or empty groups (no database round-trip).
func IsUnitMember(ctx context.Context, dbtx db.DBTX, unitID int64, userGroups []string) (bool, error) {
	if len(userGroups) == 0 {
		return false, nil
	}

	var isMember bool
	err := dbtx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM unit_group_bindings
			WHERE unit_id = $1
			  AND group_name = ANY($2::text[])
		)
	`, unitID, userGroups).Scan(&isMember)
	return isMember, err
}

// CreateUnitWithBindings inserts a new unit and its group bindings in a
// transaction. Returns the created unit. Follows the same transactional
// pattern as CreateCalendarWithViewers.
func (s *Store) CreateUnitWithBindings(
	ctx context.Context,
	params db.CreateUnitParams,
	groupBindings []string,
) (db.Unit, error) {
	var unit db.Unit

	err := s.WithTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
		var txErr error
		unit, txErr = q.CreateUnit(ctx, params)
		if txErr != nil {
			return fmt.Errorf("create unit: %w", txErr)
		}

		for _, g := range groupBindings {
			if txErr = q.InsertUnitGroupBinding(ctx, db.InsertUnitGroupBindingParams{
				UnitID:    unit.ID,
				GroupName: g,
			}); txErr != nil {
				return fmt.Errorf("insert group binding: %w", txErr)
			}
		}

		return nil
	})

	return unit, err
}

// UpdateUnitWithBindings updates a unit and replaces its group bindings in a
// transaction. Always clears existing bindings, then re-inserts the new set.
// Returns the updated unit.
func (s *Store) UpdateUnitWithBindings(
	ctx context.Context,
	params db.UpdateUnitParams,
	groupBindings []string,
) (db.Unit, error) {
	var unit db.Unit

	err := s.WithTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
		var txErr error
		unit, txErr = q.UpdateUnit(ctx, params)
		if txErr != nil {
			return fmt.Errorf("update unit: %w", txErr)
		}

		if txErr = q.DeleteUnitGroupBindings(ctx, unit.ID); txErr != nil {
			return fmt.Errorf("delete group bindings: %w", txErr)
		}

		for _, g := range groupBindings {
			if txErr = q.InsertUnitGroupBinding(ctx, db.InsertUnitGroupBindingParams{
				UnitID:    unit.ID,
				GroupName: g,
			}); txErr != nil {
				return fmt.Errorf("insert group binding: %w", txErr)
			}
		}

		return nil
	})

	return unit, err
}

// IsUnitAdmin checks if the user is an admin for the given unit. A user is
// considered a unit admin if:
//   - They are an association admin (isAssocAdmin == true), OR
//   - Any of their IdP groups matches the unit's admin_group.
//
// Returns true, nil immediately for association admins (no database round-trip).
// Returns false, nil for nil or empty groups when not an association admin.
func IsUnitAdmin(ctx context.Context, dbtx db.DBTX, unitID int64, userGroups []string, isAssocAdmin bool) (bool, error) {
	if isAssocAdmin {
		return true, nil
	}
	if len(userGroups) == 0 {
		return false, nil
	}

	var isAdmin bool
	err := dbtx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM units
			WHERE id = $1
			  AND admin_group IS NOT NULL
			  AND admin_group = ANY($2::text[])
		)
	`, unitID, userGroups).Scan(&isAdmin)
	return isAdmin, err
}
