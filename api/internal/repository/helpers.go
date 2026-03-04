package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// listAllIDs returns all non-deleted entity IDs from the given table with pagination.
// Used by backfill operations to iterate through all entities.
func listAllIDs(ctx context.Context, db *sql.DB, table string, limit, offset int) ([]uuid.UUID, error) {
	query := fmt.Sprintf(
		`SELECT id FROM %s WHERE deleted_at IS NULL ORDER BY id LIMIT $1 OFFSET $2`, table)
	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing %s IDs: %w", table, err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning %s ID: %w", table, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// listAllIDsNoSoftDelete returns all entity IDs from the given table with pagination.
// Used for tables that don't have a deleted_at column.
func listAllIDsNoSoftDelete(ctx context.Context, db *sql.DB, table string, limit, offset int) ([]uuid.UUID, error) {
	query := fmt.Sprintf(
		`SELECT id FROM %s ORDER BY id LIMIT $1 OFFSET $2`, table)
	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing %s IDs: %w", table, err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning %s ID: %w", table, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
