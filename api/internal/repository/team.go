package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/marcoshack/taskwondo/internal/model"
)

// TeamRepository handles team persistence.
type TeamRepository struct {
	db *sql.DB
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// ListAllIDs returns all team IDs with pagination (for backfill).
func (r *TeamRepository) ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error) {
	return listAllIDsNoSoftDelete(ctx, r.db, "teams", limit, offset)
}

// SearchFTS performs a full-text search on teams filtered by accessible project IDs.
func (r *TeamRepository) SearchFTS(ctx context.Context, query string, fullProjectIDs []uuid.UUID, limit int) ([]model.SearchResult, error) {
	if len(fullProjectIDs) == 0 {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT t.id, t.project_id, t.name,
		        p.key AS project_key,
		        COALESCE(n.slug, 'default') AS namespace_slug,
		        ts_rank(t.search_vector, plainto_tsquery('english', $1)) +
		        ts_rank(t.search_vector, plainto_tsquery('simple', $1)) AS rank
		 FROM teams t
		 JOIN projects p ON p.id = t.project_id
		 LEFT JOIN namespaces n ON n.id = p.namespace_id
		 WHERE (t.search_vector @@ plainto_tsquery('english', $1)
		     OR t.search_vector @@ plainto_tsquery('simple', $1))
		   AND t.project_id = ANY($2)
		 ORDER BY rank DESC, t.updated_at DESC
		 LIMIT $3`,
		query, pq.Array(fullProjectIDs), limit)
	if err != nil {
		return nil, fmt.Errorf("fts search teams: %w", err)
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		var (
			id            uuid.UUID
			projectID     uuid.UUID
			name          string
			projectKey    string
			namespaceSlug string
			rank          float64
		)
		if err := rows.Scan(&id, &projectID, &name, &projectKey, &namespaceSlug, &rank); err != nil {
			return nil, fmt.Errorf("scanning team fts result: %w", err)
		}
		_ = rank
		results = append(results, model.SearchResult{
			EntityType:    model.EntityTypeTeam,
			EntityID:      id,
			ProjectID:     &projectID,
			Content:       fmt.Sprintf("Team: %s", name),
			ProjectKey:    projectKey,
			NamespaceSlug: namespaceSlug,
		})
	}
	return results, rows.Err()
}

// Create inserts a new team.
func (r *TeamRepository) Create(ctx context.Context, t *model.Team) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO teams (id, project_id, name, description)
		 VALUES ($1, $2, $3, $4)`,
		t.ID, t.ProjectID, t.Name, t.Description)
	if err != nil {
		return fmt.Errorf("inserting team: %w", err)
	}
	return nil
}

// GetByID returns a team by ID.
func (r *TeamRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Team, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, created_at, updated_at
		 FROM teams WHERE id = $1`, id)
	return scanTeam(row)
}

// List returns all teams for a project.
func (r *TeamRepository) List(ctx context.Context, projectID uuid.UUID) ([]model.Team, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, created_at, updated_at
		 FROM teams WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying teams: %w", err)
	}
	defer rows.Close()

	return scanTeams(rows)
}

// Update modifies a team's mutable fields.
func (r *TeamRepository) Update(ctx context.Context, t *model.Team) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE teams SET name = $1, description = $2, updated_at = now()
		 WHERE id = $3`,
		t.Name, t.Description, t.ID)
	if err != nil {
		return fmt.Errorf("updating team: %w", err)
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

// Delete removes a team.
func (r *TeamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM teams WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting team: %w", err)
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

// AddMember adds a user to a team.
func (r *TeamRepository) AddMember(ctx context.Context, teamID, userID uuid.UUID) (*model.TeamMember, error) {
	m := &model.TeamMember{
		ID:     uuid.New(),
		TeamID: teamID,
		UserID: userID,
	}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO team_members (id, team_id, user_id) VALUES ($1, $2, $3)
		 RETURNING created_at`,
		m.ID, m.TeamID, m.UserID).Scan(&m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding team member: %w", err)
	}
	return m, nil
}

// RemoveMember removes a user from a team.
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`,
		teamID, userID)
	if err != nil {
		return fmt.Errorf("removing team member: %w", err)
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

// ListMembers returns all members of a team with user details.
func (r *TeamRepository) ListMembers(ctx context.Context, teamID uuid.UUID) ([]model.TeamMemberWithUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT tm.id, tm.team_id, tm.user_id, tm.created_at,
		        u.email, u.display_name, u.avatar_url
		 FROM team_members tm
		 JOIN users u ON u.id = tm.user_id
		 WHERE tm.team_id = $1
		 ORDER BY u.display_name`, teamID)
	if err != nil {
		return nil, fmt.Errorf("querying team members: %w", err)
	}
	defer rows.Close()

	var members []model.TeamMemberWithUser
	for rows.Next() {
		var m model.TeamMemberWithUser
		var avatarURL sql.NullString
		if err := rows.Scan(&m.ID, &m.TeamID, &m.UserID, &m.CreatedAt,
			&m.Email, &m.DisplayName, &avatarURL); err != nil {
			return nil, fmt.Errorf("scanning team member row: %w", err)
		}
		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func scanTeam(row *sql.Row) (*model.Team, error) {
	var t model.Team
	var description sql.NullString

	err := row.Scan(&t.ID, &t.ProjectID, &t.Name, &description,
		&t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning team: %w", err)
	}

	if description.Valid {
		t.Description = &description.String
	}

	return &t, nil
}

func scanTeams(rows *sql.Rows) ([]model.Team, error) {
	var teams []model.Team
	for rows.Next() {
		var t model.Team
		var description sql.NullString

		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Name, &description,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning team row: %w", err)
		}

		if description.Valid {
			t.Description = &description.String
		}

		teams = append(teams, t)
	}
	return teams, rows.Err()
}
