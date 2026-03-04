package workers

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// embedIndexer is the minimal interface for the indexer service.
type embedIndexer interface {
	IndexEntity(ctx context.Context, entityType string, entityID uuid.UUID, projectID *uuid.UUID) error
	DeleteEmbedding(ctx context.Context, entityType string, entityID uuid.UUID) error
	BackfillAll(ctx context.Context) (int64, error)
}

// featureFlagChecker reads system settings to check feature flags.
type featureFlagChecker interface {
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
}

// isSemanticSearchEnabled checks the feature flag from system settings.
func isSemanticSearchEnabled(ctx context.Context, settings featureFlagChecker) bool {
	setting, err := settings.Get(ctx, model.SettingFeatureSemanticSearch)
	if err != nil {
		return false
	}
	var enabled bool
	if err := json.Unmarshal(setting.Value, &enabled); err != nil {
		return false
	}
	return enabled
}

// EmbedIndexTask processes embed.index events to generate embeddings.
type EmbedIndexTask struct {
	indexer  embedIndexer
	settings featureFlagChecker
	logger   zerolog.Logger
}

// NewEmbedIndexTask creates a new EmbedIndexTask.
func NewEmbedIndexTask(indexer embedIndexer, settings featureFlagChecker, logger zerolog.Logger) *EmbedIndexTask {
	return &EmbedIndexTask{indexer: indexer, settings: settings, logger: logger}
}

// Name returns the task name used as the NATS subject suffix.
func (t *EmbedIndexTask) Name() string { return "embed.index" }

// Execute processes an embed index event.
func (t *EmbedIndexTask) Execute(ctx context.Context, payload []byte) error {
	if !isSemanticSearchEnabled(ctx, t.settings) {
		return nil // Feature disabled — silently discard
	}

	var evt model.EmbedIndexEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid embed index event payload")
		return nil // Bad payload — no point retrying
	}

	if err := t.indexer.IndexEntity(ctx, evt.EntityType, evt.EntityID, evt.ProjectID); err != nil {
		return err // Retryable error
	}

	t.logger.Debug().
		Str("entity_type", evt.EntityType).
		Str("entity_id", evt.EntityID.String()).
		Msg("embedding indexed")
	return nil
}

// EmbedDeleteTask processes embed.delete events to remove embeddings.
type EmbedDeleteTask struct {
	indexer  embedIndexer
	settings featureFlagChecker
	logger   zerolog.Logger
}

// NewEmbedDeleteTask creates a new EmbedDeleteTask.
func NewEmbedDeleteTask(indexer embedIndexer, settings featureFlagChecker, logger zerolog.Logger) *EmbedDeleteTask {
	return &EmbedDeleteTask{indexer: indexer, settings: settings, logger: logger}
}

// Name returns the task name used as the NATS subject suffix.
func (t *EmbedDeleteTask) Name() string { return "embed.delete" }

// Execute processes an embed delete event.
func (t *EmbedDeleteTask) Execute(ctx context.Context, payload []byte) error {
	if !isSemanticSearchEnabled(ctx, t.settings) {
		return nil
	}

	var evt model.EmbedDeleteEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid embed delete event payload")
		return nil
	}

	if err := t.indexer.DeleteEmbedding(ctx, evt.EntityType, evt.EntityID); err != nil {
		return err
	}

	t.logger.Debug().
		Str("entity_type", evt.EntityType).
		Str("entity_id", evt.EntityID.String()).
		Msg("embedding deleted")
	return nil
}

// EmbedBackfillTask processes embed.backfill events to index all existing entities.
type EmbedBackfillTask struct {
	indexer  embedIndexer
	settings featureFlagChecker
	logger   zerolog.Logger
}

// NewEmbedBackfillTask creates a new EmbedBackfillTask.
func NewEmbedBackfillTask(indexer embedIndexer, settings featureFlagChecker, logger zerolog.Logger) *EmbedBackfillTask {
	return &EmbedBackfillTask{indexer: indexer, settings: settings, logger: logger}
}

// Name returns the task name used as the NATS subject suffix.
func (t *EmbedBackfillTask) Name() string { return "embed.backfill" }

// Execute runs the full backfill process.
func (t *EmbedBackfillTask) Execute(ctx context.Context, _ []byte) error {
	if !isSemanticSearchEnabled(ctx, t.settings) {
		return nil
	}

	t.logger.Info().Msg("starting embedding backfill")

	total, err := t.indexer.BackfillAll(ctx)
	if err != nil {
		t.logger.Error().Err(err).Msg("embedding backfill failed")
		return err
	}

	t.logger.Info().Int64("total_indexed", total).Msg("embedding backfill completed")
	return nil
}
