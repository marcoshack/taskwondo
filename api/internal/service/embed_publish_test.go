package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

type mockFlagSettings struct {
	mu       sync.Mutex
	settings map[string]*model.SystemSetting
}

func (m *mockFlagSettings) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, model.ErrNotFound
}

func (m *mockFlagSettings) Upsert(_ context.Context, s *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[s.Key] = s
	return nil
}

func (m *mockFlagSettings) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.settings, key)
	return nil
}

func (m *mockFlagSettings) List(_ context.Context) ([]model.SystemSetting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []model.SystemSetting
	for _, s := range m.settings {
		result = append(result, *s)
	}
	return result, nil
}

func TestFeatureFlagCache_Disabled(t *testing.T) {
	settings := &mockFlagSettings{
		settings: map[string]*model.SystemSetting{},
	}

	cache := NewFeatureFlagCache("test_flag", 100*time.Millisecond, settings)
	if cache.isEnabled(context.Background()) {
		t.Fatal("expected flag to be disabled when setting doesn't exist")
	}
}

func TestFeatureFlagCache_Enabled(t *testing.T) {
	settings := &mockFlagSettings{
		settings: map[string]*model.SystemSetting{
			"test_flag": {Key: "test_flag", Value: []byte("true")},
		},
	}

	cache := NewFeatureFlagCache("test_flag", 100*time.Millisecond, settings)
	if !cache.isEnabled(context.Background()) {
		t.Fatal("expected flag to be enabled")
	}
}

func TestFeatureFlagCache_TTLExpiry(t *testing.T) {
	settings := &mockFlagSettings{
		settings: map[string]*model.SystemSetting{
			"test_flag": {Key: "test_flag", Value: []byte("true")},
		},
	}

	cache := NewFeatureFlagCache("test_flag", 50*time.Millisecond, settings)

	// First call should cache the value
	if !cache.isEnabled(context.Background()) {
		t.Fatal("expected flag to be enabled on first call")
	}

	// Change the value
	settings.mu.Lock()
	settings.settings["test_flag"] = &model.SystemSetting{Key: "test_flag", Value: []byte("false")}
	settings.mu.Unlock()

	// Cached value should still be true
	if !cache.isEnabled(context.Background()) {
		t.Fatal("expected cached value to still be true")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Now should read the new value
	if cache.isEnabled(context.Background()) {
		t.Fatal("expected flag to be disabled after TTL expiry")
	}
}

type mockPublisher struct {
	mu       sync.Mutex
	events   []publishedEvent
	err      error
}

type publishedEvent struct {
	subject string
	payload any
}

func (m *mockPublisher) Publish(subject string, payload any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, publishedEvent{subject: subject, payload: payload})
	return m.err
}

func TestPublishEmbedIndex_NilPublisher(t *testing.T) {
	// Should not panic
	publishEmbedIndex(context.Background(), nil, nil, "work_item", uuid.New(), nil)
}

func TestPublishEmbedIndex_Disabled(t *testing.T) {
	pub := &mockPublisher{}
	settings := &mockFlagSettings{settings: map[string]*model.SystemSetting{}}
	cache := NewFeatureFlagCache("test_flag", time.Hour, settings)

	publishEmbedIndex(context.Background(), pub, cache, "work_item", uuid.New(), nil)

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.events) != 0 {
		t.Fatal("expected no events when feature is disabled")
	}
}

func TestPublishEmbedIndex_Enabled(t *testing.T) {
	pub := &mockPublisher{}
	settings := &mockFlagSettings{
		settings: map[string]*model.SystemSetting{
			"test_flag": {Key: "test_flag", Value: []byte("true")},
		},
	}
	cache := NewFeatureFlagCache("test_flag", time.Hour, settings)
	projectID := uuid.New()
	entityID := uuid.New()

	publishEmbedIndex(context.Background(), pub, cache, "work_item", entityID, &projectID)

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.events))
	}
	if pub.events[0].subject != "embed.index" {
		t.Fatalf("expected subject embed.index, got %s", pub.events[0].subject)
	}
}

func TestPublishEmbedDelete_Enabled(t *testing.T) {
	pub := &mockPublisher{}
	settings := &mockFlagSettings{
		settings: map[string]*model.SystemSetting{
			"test_flag": {Key: "test_flag", Value: []byte("true")},
		},
	}
	cache := NewFeatureFlagCache("test_flag", time.Hour, settings)

	publishEmbedDelete(context.Background(), pub, cache, "work_item", uuid.New())

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.events))
	}
	if pub.events[0].subject != "embed.delete" {
		t.Fatalf("expected subject embed.delete, got %s", pub.events[0].subject)
	}
}
