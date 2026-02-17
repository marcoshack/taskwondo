package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/trackforge/internal/model"
)

// WorkItemEventRepository handles work item event persistence.
type WorkItemEventRepository struct {
	db *sql.DB
}

// NewWorkItemEventRepository creates a new WorkItemEventRepository.
func NewWorkItemEventRepository(db *sql.DB) *WorkItemEventRepository {
	return &WorkItemEventRepository{db: db}
}

// Create inserts a new work item event.
func (r *WorkItemEventRepository) Create(ctx context.Context, event *model.WorkItemEvent) error {
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling event metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO work_item_events (
			id, work_item_id, actor_id, event_type, field_name,
			old_value, new_value, metadata, visibility
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		event.ID, event.WorkItemID, event.ActorID, event.EventType, event.FieldName,
		event.OldValue, event.NewValue, metadataJSON, event.Visibility)
	if err != nil {
		return fmt.Errorf("inserting work item event: %w", err)
	}

	return nil
}

// ListByWorkItem returns all events for a given work item, ordered by creation time.
func (r *WorkItemEventRepository) ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemEvent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, work_item_id, actor_id, event_type, field_name,
		        old_value, new_value, metadata, visibility, created_at
		 FROM work_item_events
		 WHERE work_item_id = $1
		 ORDER BY created_at ASC`, workItemID)
	if err != nil {
		return nil, fmt.Errorf("querying work item events: %w", err)
	}
	defer rows.Close()

	var events []model.WorkItemEvent
	for rows.Next() {
		var event model.WorkItemEvent
		var (
			actorID     uuid.NullUUID
			fieldName   sql.NullString
			oldValue    sql.NullString
			newValue    sql.NullString
			metadataRaw []byte
		)

		if err := rows.Scan(
			&event.ID, &event.WorkItemID, &actorID, &event.EventType, &fieldName,
			&oldValue, &newValue, &metadataRaw, &event.Visibility, &event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning work item event: %w", err)
		}

		if actorID.Valid {
			event.ActorID = &actorID.UUID
		}
		if fieldName.Valid {
			event.FieldName = &fieldName.String
		}
		if oldValue.Valid {
			event.OldValue = &oldValue.String
		}
		if newValue.Valid {
			event.NewValue = &newValue.String
		}

		event.Metadata = make(map[string]interface{})
		if len(metadataRaw) > 0 {
			json.Unmarshal(metadataRaw, &event.Metadata)
		}

		events = append(events, event)
	}

	return events, rows.Err()
}
