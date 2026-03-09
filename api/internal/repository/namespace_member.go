package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// NamespaceMemberRepository handles namespace membership persistence.
type NamespaceMemberRepository struct {
	db *sql.DB
}

// NewNamespaceMemberRepository creates a new NamespaceMemberRepository.
func NewNamespaceMemberRepository(db *sql.DB) *NamespaceMemberRepository {
	return &NamespaceMemberRepository{db: db}
}

// Add inserts a new namespace membership.
func (r *NamespaceMemberRepository) Add(ctx context.Context, member *model.NamespaceMember) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO namespace_members (namespace_id, user_id, role)
		 VALUES ($1, $2, $3)`,
		member.NamespaceID, member.UserID, member.Role)
	if err != nil {
		return fmt.Errorf("inserting namespace member: %w", err)
	}
	return nil
}

// GetByNamespaceAndUser returns a membership record for a specific namespace and user.
func (r *NamespaceMemberRepository) GetByNamespaceAndUser(ctx context.Context, namespaceID, userID uuid.UUID) (*model.NamespaceMember, error) {
	var m model.NamespaceMember
	err := r.db.QueryRowContext(ctx,
		`SELECT namespace_id, user_id, role, created_at
		 FROM namespace_members WHERE namespace_id = $1 AND user_id = $2`,
		namespaceID, userID).
		Scan(&m.NamespaceID, &m.UserID, &m.Role, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning namespace member: %w", err)
	}
	return &m, nil
}

// ListByNamespace returns all members of a namespace with their user details.
func (r *NamespaceMemberRepository) ListByNamespace(ctx context.Context, namespaceID uuid.UUID) ([]model.NamespaceMemberWithUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT nm.namespace_id, nm.user_id, nm.role, nm.created_at,
		        u.display_name, u.email, u.avatar_url
		 FROM namespace_members nm
		 INNER JOIN users u ON u.id = nm.user_id
		 WHERE nm.namespace_id = $1
		 ORDER BY nm.created_at`, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("querying namespace members: %w", err)
	}
	defer rows.Close()

	var members []model.NamespaceMemberWithUser
	for rows.Next() {
		var m model.NamespaceMemberWithUser
		var avatarURL sql.NullString

		if err := rows.Scan(&m.NamespaceID, &m.UserID, &m.Role, &m.CreatedAt,
			&m.DisplayName, &m.Email, &avatarURL); err != nil {
			return nil, fmt.Errorf("scanning namespace member row: %w", err)
		}

		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}

		members = append(members, m)
	}

	return members, rows.Err()
}

// UpdateRole changes a member's role within a namespace.
func (r *NamespaceMemberRepository) UpdateRole(ctx context.Context, namespaceID, userID uuid.UUID, role string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE namespace_members SET role = $1
		 WHERE namespace_id = $2 AND user_id = $3`,
		role, namespaceID, userID)
	if err != nil {
		return fmt.Errorf("updating namespace member role: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}

	return nil
}

// Remove deletes a namespace membership (hard delete).
func (r *NamespaceMemberRepository) Remove(ctx context.Context, namespaceID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM namespace_members WHERE namespace_id = $1 AND user_id = $2`,
		namespaceID, userID)
	if err != nil {
		return fmt.Errorf("removing namespace member: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}

	return nil
}

// RemoveAllByNamespace deletes all members of a namespace (used before namespace deletion).
func (r *NamespaceMemberRepository) RemoveAllByNamespace(ctx context.Context, namespaceID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM namespace_members WHERE namespace_id = $1`, namespaceID)
	if err != nil {
		return fmt.Errorf("removing all namespace members: %w", err)
	}
	return nil
}

// CountOwnedByUser returns the number of namespaces where the given user is an owner.
func (r *NamespaceMemberRepository) CountOwnedByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM namespace_members WHERE user_id = $1 AND role = 'owner'`,
		userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting namespaces owned by user: %w", err)
	}
	return count, nil
}

// CountByRole returns the number of members with a given role in a namespace.
func (r *NamespaceMemberRepository) CountByRole(ctx context.Context, namespaceID uuid.UUID, role string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM namespace_members WHERE namespace_id = $1 AND role = $2`,
		namespaceID, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting namespace members by role: %w", err)
	}
	return count, nil
}
