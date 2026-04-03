package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// QueueTeamRepository handles queue-team assignment persistence.
type QueueTeamRepository struct {
	db *sql.DB
}

// NewQueueTeamRepository creates a new QueueTeamRepository.
func NewQueueTeamRepository(db *sql.DB) *QueueTeamRepository {
	return &QueueTeamRepository{db: db}
}

// Assign links a team to a queue.
func (r *QueueTeamRepository) Assign(ctx context.Context, queueID, teamID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO queue_teams (queue_id, team_id) VALUES ($1, $2)`,
		queueID, teamID)
	if err != nil {
		return fmt.Errorf("assigning team to queue: %w", err)
	}
	return nil
}

// Unassign removes a team from a queue.
func (r *QueueTeamRepository) Unassign(ctx context.Context, queueID, teamID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM queue_teams WHERE queue_id = $1 AND team_id = $2`,
		queueID, teamID)
	if err != nil {
		return fmt.Errorf("unassigning team from queue: %w", err)
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

// ListTeamsByQueue returns all teams assigned to a queue.
func (r *QueueTeamRepository) ListTeamsByQueue(ctx context.Context, queueID uuid.UUID) ([]model.Team, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.id, t.project_id, t.name, t.description, t.created_at, t.updated_at
		 FROM teams t
		 JOIN queue_teams qt ON qt.team_id = t.id
		 WHERE qt.queue_id = $1
		 ORDER BY t.name`, queueID)
	if err != nil {
		return nil, fmt.Errorf("querying teams by queue: %w", err)
	}
	defer rows.Close()

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

// ListQueuesByTeam returns all queues assigned to a team.
func (r *QueueTeamRepository) ListQueuesByTeam(ctx context.Context, teamID uuid.UUID) ([]model.Queue, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT q.id, q.project_id, q.name, q.description, q.queue_type, q.is_public,
		        q.default_priority, q.default_assignee_id, q.workflow_id, q.created_at, q.updated_at
		 FROM queues q
		 JOIN queue_teams qt ON qt.queue_id = q.id
		 WHERE qt.team_id = $1
		 ORDER BY q.name`, teamID)
	if err != nil {
		return nil, fmt.Errorf("querying queues by team: %w", err)
	}
	defer rows.Close()

	return scanQueues(rows)
}
