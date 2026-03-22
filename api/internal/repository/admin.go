package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// AdminRepository handles admin inspection queries.
type AdminRepository struct {
	db *sql.DB
}

// NewAdminRepository creates a new AdminRepository.
func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// ListProjects returns a paginated list of all projects with aggregated counts.
func (r *AdminRepository) ListProjects(ctx context.Context, search string, cursor string, limit int) (*model.AdminProjectList, error) {
	args := []interface{}{search}
	argIdx := 2

	// Base query
	query := `
SELECT p.id, p.key, p.name, p.created_at,
  COALESCE(n.slug, 'default') AS namespace_slug,
  COALESCE(n.display_name, 'Default') AS namespace_display_name,
  COALESCE(owner_u.display_name, '') AS owner_display_name,
  COALESCE(owner_u.email, '') AS owner_email,
  (SELECT COUNT(*) FROM project_members pm WHERE pm.project_id = p.id) AS member_count,
  (SELECT COUNT(*) FROM work_items wi WHERE wi.project_id = p.id) AS item_count,
  COALESCE((SELECT SUM(a.size_bytes) FROM attachments a
    JOIN work_items wi ON a.work_item_id = wi.id
    WHERE wi.project_id = p.id AND a.deleted_at IS NULL), 0) AS storage_bytes
FROM projects p
LEFT JOIN namespaces n ON p.namespace_id = n.id
LEFT JOIN LATERAL (
  SELECT u.display_name, u.email FROM project_members pm
  JOIN users u ON pm.user_id = u.id
  WHERE pm.project_id = p.id AND pm.role = 'owner'
  LIMIT 1
) owner_u ON true
WHERE p.deleted_at IS NULL
  AND ($1 = '' OR p.name ILIKE '%' || $1 || '%' OR p.key ILIKE '%' || $1 || '%')`

	// Cursor pagination
	if cursor != "" {
		cursorName, cursorID, err := decodeProjectCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND (p.name, p.id) > ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, cursorName, cursorID)
		argIdx += 2
	}

	query += fmt.Sprintf(" ORDER BY p.name ASC, p.id ASC LIMIT $%d", argIdx)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying admin projects: %w", err)
	}
	defer rows.Close()

	var items []model.AdminProject
	for rows.Next() {
		var p model.AdminProject
		if err := rows.Scan(
			&p.ID, &p.Key, &p.Name, &p.CreatedAt,
			&p.NamespaceSlug, &p.NamespaceDisplayName,
			&p.OwnerDisplayName, &p.OwnerEmail,
			&p.MemberCount, &p.ItemCount, &p.StorageBytes,
		); err != nil {
			return nil, fmt.Errorf("scanning admin project row: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating admin project rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeProjectCursor(last.Name, last.ID)
	}

	return &model.AdminProjectList{
		Items:   items,
		Cursor:  nextCursor,
		HasMore: hasMore,
	}, nil
}

// ListNamespaces returns a paginated list of all namespaces with aggregated counts.
func (r *AdminRepository) ListNamespaces(ctx context.Context, search string, cursor string, limit int) (*model.AdminNamespaceList, error) {
	args := []interface{}{search}
	argIdx := 2

	query := `
SELECT n.id, n.slug, n.display_name, n.is_default, n.created_at,
  (SELECT COUNT(*) FROM projects p WHERE p.namespace_id = n.id AND p.deleted_at IS NULL) AS project_count,
  (SELECT COUNT(DISTINCT nm.user_id) FROM namespace_members nm WHERE nm.namespace_id = n.id) AS member_count,
  COALESCE((SELECT SUM(a.size_bytes) FROM attachments a
    JOIN work_items wi ON a.work_item_id = wi.id
    JOIN projects p ON wi.project_id = p.id
    WHERE p.namespace_id = n.id AND p.deleted_at IS NULL AND a.deleted_at IS NULL), 0) AS storage_bytes
FROM namespaces n
WHERE ($1 = '' OR n.slug ILIKE '%' || $1 || '%' OR n.display_name ILIKE '%' || $1 || '%')`

	// Cursor pagination
	if cursor != "" {
		cursorName, cursorID, err := decodeNamespaceCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND (n.display_name, n.id) > ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, cursorName, cursorID)
		argIdx += 2
	}

	query += fmt.Sprintf(" ORDER BY n.display_name ASC, n.id ASC LIMIT $%d", argIdx)
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying admin namespaces: %w", err)
	}
	defer rows.Close()

	var items []model.AdminNamespace
	for rows.Next() {
		var ns model.AdminNamespace
		if err := rows.Scan(
			&ns.ID, &ns.Slug, &ns.DisplayName, &ns.IsDefault, &ns.CreatedAt,
			&ns.ProjectCount, &ns.MemberCount, &ns.StorageBytes,
		); err != nil {
			return nil, fmt.Errorf("scanning admin namespace row: %w", err)
		}
		items = append(items, ns)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating admin namespace rows: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeNamespaceCursor(last.DisplayName, last.ID)
	}

	return &model.AdminNamespaceList{
		Items:   items,
		Cursor:  nextCursor,
		HasMore: hasMore,
	}, nil
}

// GetStats returns aggregated system-wide counts.
func (r *AdminRepository) GetStats(ctx context.Context) (*model.AdminStats, error) {
	var stats model.AdminStats
	err := r.db.QueryRowContext(ctx, `
SELECT
  (SELECT COUNT(*) FROM projects WHERE deleted_at IS NULL),
  (SELECT COUNT(*) FROM namespaces),
  (SELECT COUNT(*) FROM users),
  COALESCE((SELECT SUM(size_bytes) FROM attachments WHERE deleted_at IS NULL), 0)
`).Scan(&stats.Projects, &stats.Namespaces, &stats.Users, &stats.StorageBytes)
	if err != nil {
		return nil, fmt.Errorf("querying admin stats: %w", err)
	}
	return &stats, nil
}

// encodeProjectCursor encodes a project cursor as base64 of "name|id".
func encodeProjectCursor(name string, id uuid.UUID) string {
	return base64.StdEncoding.EncodeToString([]byte(name + "|" + id.String()))
}

// decodeProjectCursor decodes a base64 project cursor into (name, id).
func decodeProjectCursor(cursor string) (string, uuid.UUID, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("decoding cursor: %w", err)
	}
	parts := strings.SplitN(string(data), "|", 2)
	if len(parts) != 2 {
		return "", uuid.Nil, fmt.Errorf("invalid cursor format")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("parsing cursor ID: %w", err)
	}
	return parts[0], id, nil
}

// encodeNamespaceCursor encodes a namespace cursor as base64 of "display_name|id".
func encodeNamespaceCursor(displayName string, id uuid.UUID) string {
	return base64.StdEncoding.EncodeToString([]byte(displayName + "|" + id.String()))
}

// decodeNamespaceCursor decodes a base64 namespace cursor into (display_name, id).
func decodeNamespaceCursor(cursor string) (string, uuid.UUID, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("decoding cursor: %w", err)
	}
	parts := strings.SplitN(string(data), "|", 2)
	if len(parts) != 2 {
		return "", uuid.Nil, fmt.Errorf("invalid cursor format")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("parsing cursor ID: %w", err)
	}
	return parts[0], id, nil
}
