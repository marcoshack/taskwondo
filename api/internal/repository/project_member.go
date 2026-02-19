package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectMemberRepository handles project membership persistence.
type ProjectMemberRepository struct {
	db *sql.DB
}

// NewProjectMemberRepository creates a new ProjectMemberRepository.
func NewProjectMemberRepository(db *sql.DB) *ProjectMemberRepository {
	return &ProjectMemberRepository{db: db}
}

// Add inserts a new project membership.
func (r *ProjectMemberRepository) Add(ctx context.Context, member *model.ProjectMember) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO project_members (id, project_id, user_id, role)
		 VALUES ($1, $2, $3, $4)`,
		member.ID, member.ProjectID, member.UserID, member.Role)
	if err != nil {
		return fmt.Errorf("inserting project member: %w", err)
	}
	return nil
}

// GetByProjectAndUser returns a membership record for a specific project and user.
func (r *ProjectMemberRepository) GetByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error) {
	var m model.ProjectMember
	err := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, role, created_at
		 FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID).
		Scan(&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning project member: %w", err)
	}
	return &m, nil
}

// ListByProject returns all members of a project with their user details.
func (r *ProjectMemberRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectMemberWithUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pm.id, pm.project_id, pm.user_id, pm.role, pm.created_at,
		        u.email, u.display_name, u.avatar_url
		 FROM project_members pm
		 INNER JOIN users u ON u.id = pm.user_id
		 WHERE pm.project_id = $1
		 ORDER BY pm.created_at`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying project members: %w", err)
	}
	defer rows.Close()

	var members []model.ProjectMemberWithUser
	for rows.Next() {
		var m model.ProjectMemberWithUser
		var avatarURL sql.NullString

		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.CreatedAt,
			&m.Email, &m.DisplayName, &avatarURL); err != nil {
			return nil, fmt.Errorf("scanning project member row: %w", err)
		}

		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}

		members = append(members, m)
	}

	return members, rows.Err()
}

// UpdateRole changes a member's role within a project.
func (r *ProjectMemberRepository) UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE project_members SET role = $1
		 WHERE project_id = $2 AND user_id = $3`,
		role, projectID, userID)
	if err != nil {
		return fmt.Errorf("updating member role: %w", err)
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

// Remove deletes a project membership (hard delete).
func (r *ProjectMemberRepository) Remove(ctx context.Context, projectID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID)
	if err != nil {
		return fmt.Errorf("removing project member: %w", err)
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

// ListByUser returns all project memberships for a user with project details.
func (r *ProjectMemberRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.ProjectMemberWithProject, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pm.id, pm.project_id, pm.user_id, pm.role, pm.created_at,
		        p.name, p.key
		 FROM project_members pm
		 INNER JOIN projects p ON p.id = pm.project_id
		 WHERE pm.user_id = $1 AND p.deleted_at IS NULL
		 ORDER BY p.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying user project memberships: %w", err)
	}
	defer rows.Close()

	var members []model.ProjectMemberWithProject
	for rows.Next() {
		var m model.ProjectMemberWithProject
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.CreatedAt,
			&m.ProjectName, &m.ProjectKey); err != nil {
			return nil, fmt.Errorf("scanning user project membership row: %w", err)
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

// CountByRole returns the number of members with a given role in a project.
func (r *ProjectMemberRepository) CountByRole(ctx context.Context, projectID uuid.UUID, role string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM project_members WHERE project_id = $1 AND role = $2`,
		projectID, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting members by role: %w", err)
	}
	return count, nil
}
