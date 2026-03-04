package workers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

type mockIndexer struct {
	indexCalls  int
	deleteCalls int
	backfillCalls int
	indexErr    error
	deleteErr   error
	backfillErr error
}

func (m *mockIndexer) IndexEntity(_ context.Context, _ string, _ uuid.UUID, _ *uuid.UUID) error {
	m.indexCalls++
	return m.indexErr
}

func (m *mockIndexer) DeleteEmbedding(_ context.Context, _ string, _ uuid.UUID) error {
	m.deleteCalls++
	return m.deleteErr
}

func (m *mockIndexer) BackfillAll(_ context.Context) (int64, error) {
	m.backfillCalls++
	return 42, m.backfillErr
}

type mockFeatureChecker struct {
	settings map[string]*model.SystemSetting
}

func (m *mockFeatureChecker) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, model.ErrNotFound
}

func enabledSettings() *mockFeatureChecker {
	return &mockFeatureChecker{
		settings: map[string]*model.SystemSetting{
			model.SettingFeatureSemanticSearch: {Key: model.SettingFeatureSemanticSearch, Value: []byte("true")},
		},
	}
}

func disabledSettings() *mockFeatureChecker {
	return &mockFeatureChecker{settings: map[string]*model.SystemSetting{}}
}

func TestEmbedIndexTask_Name(t *testing.T) {
	task := NewEmbedIndexTask(nil, nil, zerolog.Nop())
	if task.Name() != "embed.index" {
		t.Fatalf("expected embed.index, got %s", task.Name())
	}
}

func TestEmbedIndexTask_FeatureDisabled(t *testing.T) {
	indexer := &mockIndexer{}
	task := NewEmbedIndexTask(indexer, disabledSettings(), zerolog.Nop())

	err := task.Execute(context.Background(), []byte("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if indexer.indexCalls != 0 {
		t.Fatal("expected no index calls when feature disabled")
	}
}

func TestEmbedIndexTask_Success(t *testing.T) {
	indexer := &mockIndexer{}
	task := NewEmbedIndexTask(indexer, enabledSettings(), zerolog.Nop())

	projectID := uuid.New()
	evt := model.EmbedIndexEvent{
		EntityType: "work_item",
		EntityID:   uuid.New(),
		ProjectID:  &projectID,
	}
	payload, _ := json.Marshal(evt)

	err := task.Execute(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if indexer.indexCalls != 1 {
		t.Fatalf("expected 1 index call, got %d", indexer.indexCalls)
	}
}

func TestEmbedIndexTask_BadPayload(t *testing.T) {
	indexer := &mockIndexer{}
	task := NewEmbedIndexTask(indexer, enabledSettings(), zerolog.Nop())

	err := task.Execute(context.Background(), []byte("invalid json"))
	if err != nil {
		t.Fatal("bad payload should return nil (no retry)")
	}
	if indexer.indexCalls != 0 {
		t.Fatal("should not call indexer on bad payload")
	}
}

func TestEmbedDeleteTask_Name(t *testing.T) {
	task := NewEmbedDeleteTask(nil, nil, zerolog.Nop())
	if task.Name() != "embed.delete" {
		t.Fatalf("expected embed.delete, got %s", task.Name())
	}
}

func TestEmbedDeleteTask_Success(t *testing.T) {
	indexer := &mockIndexer{}
	task := NewEmbedDeleteTask(indexer, enabledSettings(), zerolog.Nop())

	evt := model.EmbedDeleteEvent{
		EntityType: "work_item",
		EntityID:   uuid.New(),
	}
	payload, _ := json.Marshal(evt)

	err := task.Execute(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if indexer.deleteCalls != 1 {
		t.Fatalf("expected 1 delete call, got %d", indexer.deleteCalls)
	}
}

func TestEmbedBackfillTask_Name(t *testing.T) {
	task := NewEmbedBackfillTask(nil, nil, zerolog.Nop())
	if task.Name() != "embed.backfill" {
		t.Fatalf("expected embed.backfill, got %s", task.Name())
	}
}

func TestEmbedBackfillTask_FeatureDisabled(t *testing.T) {
	indexer := &mockIndexer{}
	task := NewEmbedBackfillTask(indexer, disabledSettings(), zerolog.Nop())

	err := task.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if indexer.backfillCalls != 0 {
		t.Fatal("expected no backfill calls when feature disabled")
	}
}

func TestEmbedBackfillTask_Success(t *testing.T) {
	indexer := &mockIndexer{}
	task := NewEmbedBackfillTask(indexer, enabledSettings(), zerolog.Nop())

	err := task.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if indexer.backfillCalls != 1 {
		t.Fatalf("expected 1 backfill call, got %d", indexer.backfillCalls)
	}
}
