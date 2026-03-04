package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// IndexerWorkItemRepository is the minimal interface for fetching work items for indexing.
type IndexerWorkItemRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.WorkItem, error)
	ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error)
}

// IndexerCommentRepository is the minimal interface for fetching comments for indexing.
type IndexerCommentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Comment, error)
	ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error)
}

// IndexerProjectRepository is the minimal interface for fetching projects for indexing.
type IndexerProjectRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error)
}

// IndexerMilestoneRepository is the minimal interface for fetching milestones for indexing.
type IndexerMilestoneRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Milestone, error)
	ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error)
}

// IndexerQueueRepository is the minimal interface for fetching queues for indexing.
type IndexerQueueRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Queue, error)
	ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error)
}

// IndexerAttachmentRepository is the minimal interface for fetching attachments for indexing.
type IndexerAttachmentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Attachment, error)
	ListAllIDs(ctx context.Context, limit, offset int) ([]uuid.UUID, error)
}

// IndexerEmbeddingRepository is the minimal interface for embedding persistence.
type IndexerEmbeddingRepository interface {
	Upsert(ctx context.Context, e *model.Embedding) error
	Delete(ctx context.Context, entityType string, entityID uuid.UUID) error
}

// IndexerService handles text extraction per entity type and embedding orchestration.
type IndexerService struct {
	embedding    *EmbeddingService
	embeddings   IndexerEmbeddingRepository
	workItems    IndexerWorkItemRepository
	comments     IndexerCommentRepository
	projects     IndexerProjectRepository
	milestones   IndexerMilestoneRepository
	queues       IndexerQueueRepository
	attachments  IndexerAttachmentRepository
}

// NewIndexerService creates a new IndexerService.
func NewIndexerService(
	embedding *EmbeddingService,
	embeddings IndexerEmbeddingRepository,
	workItems IndexerWorkItemRepository,
	comments IndexerCommentRepository,
	projects IndexerProjectRepository,
	milestones IndexerMilestoneRepository,
	queues IndexerQueueRepository,
	attachments IndexerAttachmentRepository,
) *IndexerService {
	return &IndexerService{
		embedding:   embedding,
		embeddings:  embeddings,
		workItems:   workItems,
		comments:    comments,
		projects:    projects,
		milestones:  milestones,
		queues:      queues,
		attachments: attachments,
	}
}

// IndexEntity fetches the entity, builds text, generates embedding, and upserts.
func (s *IndexerService) IndexEntity(ctx context.Context, entityType string, entityID uuid.UUID, projectID *uuid.UUID) error {
	text, err := s.extractText(ctx, entityType, entityID)
	if err != nil {
		return fmt.Errorf("extracting text for %s/%s: %w", entityType, entityID, err)
	}

	vector, err := s.embedding.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("generating embedding for %s/%s: %w", entityType, entityID, err)
	}

	e := &model.Embedding{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		ProjectID:  projectID,
		Content:    text,
		Embedding:  vector,
	}

	if err := s.embeddings.Upsert(ctx, e); err != nil {
		return fmt.Errorf("upserting embedding for %s/%s: %w", entityType, entityID, err)
	}

	return nil
}

// DeleteEmbedding removes an embedding for the given entity.
func (s *IndexerService) DeleteEmbedding(ctx context.Context, entityType string, entityID uuid.UUID) error {
	return s.embeddings.Delete(ctx, entityType, entityID)
}

// BackfillAll batch-processes all entities to generate embeddings.
// Returns the total number of entities indexed.
func (s *IndexerService) BackfillAll(ctx context.Context) (int64, error) {
	var total int64

	types := []struct {
		name    string
		listIDs func(ctx context.Context, limit, offset int) ([]uuid.UUID, error)
		getProj func(ctx context.Context, id uuid.UUID) *uuid.UUID
	}{
		{
			name:    model.EntityTypeWorkItem,
			listIDs: s.workItems.ListAllIDs,
			getProj: func(ctx context.Context, id uuid.UUID) *uuid.UUID {
				item, err := s.workItems.GetByID(ctx, id)
				if err != nil {
					return nil
				}
				return &item.ProjectID
			},
		},
		{
			name:    model.EntityTypeComment,
			listIDs: s.comments.ListAllIDs,
			getProj: func(ctx context.Context, id uuid.UUID) *uuid.UUID {
				c, err := s.comments.GetByID(ctx, id)
				if err != nil {
					return nil
				}
				item, err := s.workItems.GetByID(ctx, c.WorkItemID)
				if err != nil {
					return nil
				}
				return &item.ProjectID
			},
		},
		{
			name:    model.EntityTypeProject,
			listIDs: s.projects.ListAllIDs,
			getProj: func(_ context.Context, id uuid.UUID) *uuid.UUID { return nil },
		},
		{
			name:    model.EntityTypeMilestone,
			listIDs: s.milestones.ListAllIDs,
			getProj: func(ctx context.Context, id uuid.UUID) *uuid.UUID {
				m, err := s.milestones.GetByID(ctx, id)
				if err != nil {
					return nil
				}
				return &m.ProjectID
			},
		},
		{
			name:    model.EntityTypeQueue,
			listIDs: s.queues.ListAllIDs,
			getProj: func(ctx context.Context, id uuid.UUID) *uuid.UUID {
				q, err := s.queues.GetByID(ctx, id)
				if err != nil {
					return nil
				}
				return &q.ProjectID
			},
		},
		{
			name:    model.EntityTypeAttachment,
			listIDs: s.attachments.ListAllIDs,
			getProj: func(ctx context.Context, id uuid.UUID) *uuid.UUID {
				a, err := s.attachments.GetByID(ctx, id)
				if err != nil {
					return nil
				}
				// Attachments are associated with work items; get the project from the work item
				item, err := s.workItems.GetByID(ctx, a.WorkItemID)
				if err != nil {
					return nil
				}
				return &item.ProjectID
			},
		},
	}

	for _, t := range types {
		offset := 0
		batchSize := 50
		for {
			ids, err := t.listIDs(ctx, batchSize, offset)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Str("entity_type", t.name).Msg("backfill: error listing IDs")
				break
			}
			if len(ids) == 0 {
				break
			}

			for _, id := range ids {
				projectID := t.getProj(ctx, id)
				if err := s.IndexEntity(ctx, t.name, id, projectID); err != nil {
					log.Ctx(ctx).Warn().Err(err).
						Str("entity_type", t.name).
						Str("entity_id", id.String()).
						Msg("backfill: failed to index entity")
					continue
				}
				total++

				// Pause between Ollama calls to avoid overwhelming it
				time.Sleep(50 * time.Millisecond)
			}

			offset += batchSize
		}

		log.Ctx(ctx).Info().Str("entity_type", t.name).Int64("total_indexed", total).Msg("backfill: entity type complete")
	}

	return total, nil
}

// extractText builds a searchable text string from the entity's fields.
func (s *IndexerService) extractText(ctx context.Context, entityType string, entityID uuid.UUID) (string, error) {
	switch entityType {
	case model.EntityTypeWorkItem:
		item, err := s.workItems.GetByID(ctx, entityID)
		if err != nil {
			return "", err
		}
		return extractWorkItemText(item), nil

	case model.EntityTypeComment:
		c, err := s.comments.GetByID(ctx, entityID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Comment:\n\n%s", c.Body), nil

	case model.EntityTypeProject:
		p, err := s.projects.GetByID(ctx, entityID)
		if err != nil {
			return "", err
		}
		return extractProjectText(p), nil

	case model.EntityTypeMilestone:
		m, err := s.milestones.GetByID(ctx, entityID)
		if err != nil {
			return "", err
		}
		return extractMilestoneText(m), nil

	case model.EntityTypeQueue:
		q, err := s.queues.GetByID(ctx, entityID)
		if err != nil {
			return "", err
		}
		return extractQueueText(q), nil

	case model.EntityTypeAttachment:
		a, err := s.attachments.GetByID(ctx, entityID)
		if err != nil {
			return "", err
		}
		return extractAttachmentText(a), nil

	default:
		return "", fmt.Errorf("unknown entity type: %s", entityType)
	}
}

func extractWorkItemText(item *model.WorkItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s", item.Type, item.Title)
	if item.Description != nil && *item.Description != "" {
		fmt.Fprintf(&b, "\n\n%s", *item.Description)
	}
	if len(item.Labels) > 0 {
		fmt.Fprintf(&b, "\n\nLabels: %s", strings.Join(item.Labels, ", "))
	}
	fmt.Fprintf(&b, "\nPriority: %s", item.Priority)
	return b.String()
}

func extractProjectText(p *model.Project) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project %s (%s)", p.Name, p.Key)
	if p.Description != nil && *p.Description != "" {
		fmt.Fprintf(&b, "\n\n%s", *p.Description)
	}
	return b.String()
}

func extractMilestoneText(m *model.Milestone) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Milestone %s", m.Name)
	if m.Description != nil && *m.Description != "" {
		fmt.Fprintf(&b, "\n\n%s", *m.Description)
	}
	fmt.Fprintf(&b, "\nStatus: %s", m.Status)
	return b.String()
}

func extractQueueText(q *model.Queue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Queue %s (%s)", q.Name, q.QueueType)
	if q.Description != nil && *q.Description != "" {
		fmt.Fprintf(&b, "\n\n%s", *q.Description)
	}
	return b.String()
}

func extractAttachmentText(a *model.Attachment) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Attachment %s", a.Filename)
	if a.Comment != "" {
		fmt.Fprintf(&b, "\nComment: %s", a.Comment)
	}
	fmt.Fprintf(&b, "\nType: %s", a.ContentType)
	return b.String()
}
