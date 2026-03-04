package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// FeatureFlagCache caches a boolean feature flag with a TTL.
type FeatureFlagCache struct {
	mu       sync.RWMutex
	value    bool
	expiry   time.Time
	ttl      time.Duration
	key      string
	settings SystemSettingRepositoryInterface
}

func NewFeatureFlagCache(key string, ttl time.Duration, settings SystemSettingRepositoryInterface) *FeatureFlagCache {
	return &FeatureFlagCache{key: key, ttl: ttl, settings: settings}
}

func (c *FeatureFlagCache) isEnabled(ctx context.Context) bool {
	c.mu.RLock()
	if time.Now().Before(c.expiry) {
		val := c.value
		c.mu.RUnlock()
		return val
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Now().Before(c.expiry) {
		return c.value
	}

	setting, err := c.settings.Get(ctx, c.key)
	if err != nil {
		c.value = false
		c.expiry = time.Now().Add(c.ttl)
		return false
	}

	var enabled bool
	if err := json.Unmarshal(setting.Value, &enabled); err != nil {
		c.value = false
	} else {
		c.value = enabled
	}
	c.expiry = time.Now().Add(c.ttl)
	return c.value
}

// publishEmbedIndex publishes an embed.index event if the feature is enabled.
// Best-effort: logs a warning on error but does not fail the caller.
func publishEmbedIndex(ctx context.Context, publisher EventPublisher, cache *FeatureFlagCache, entityType string, entityID uuid.UUID, projectID *uuid.UUID) {
	if publisher == nil || cache == nil {
		return
	}
	if !cache.isEnabled(ctx) {
		return
	}
	evt := model.EmbedIndexEvent{
		EntityType: entityType,
		EntityID:   entityID,
		ProjectID:  projectID,
	}
	if err := publisher.Publish("embed.index", evt); err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Str("entity_type", entityType).
			Str("entity_id", entityID.String()).
			Msg("failed to publish embed.index event")
	}
}

// publishEmbedDelete publishes an embed.delete event if the feature is enabled.
func publishEmbedDelete(ctx context.Context, publisher EventPublisher, cache *FeatureFlagCache, entityType string, entityID uuid.UUID) {
	if publisher == nil || cache == nil {
		return
	}
	if !cache.isEnabled(ctx) {
		return
	}
	evt := model.EmbedDeleteEvent{
		EntityType: entityType,
		EntityID:   entityID,
	}
	if err := publisher.Publish("embed.delete", evt); err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Str("entity_type", entityType).
			Str("entity_id", entityID.String()).
			Msg("failed to publish embed.delete event")
	}
}
