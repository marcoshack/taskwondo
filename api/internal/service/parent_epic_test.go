package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestMapParentEpics_SkipsEpicsAndEmpty(t *testing.T) {
	repo := newMockWorkItemRepo()
	svc := &WorkItemService{items: repo}

	epicID := uuid.New()
	childID := uuid.New()
	repo.parentEpics = map[uuid.UUID]model.ParentEpicRef{
		childID: {DisplayID: "TASK-76", Title: "Split pane"},
		epicID:  {DisplayID: "SHOULD-NOT", Title: "ignored"},
	}

	got, err := svc.MapParentEpics(context.Background(), []model.WorkItem{
		{ID: epicID, Type: model.WorkItemTypeEpic},
		{ID: childID, Type: model.WorkItemTypeTicket},
	})
	if err != nil {
		t.Fatalf("MapParentEpics: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 parent epic, got %d", len(got))
	}
	ref, ok := got[childID]
	if !ok || ref.DisplayID != "TASK-76" {
		t.Fatalf("unexpected parent epic map: %#v", got)
	}
	if len(repo.lastParentEpicIDs) != 1 || repo.lastParentEpicIDs[0] != childID {
		t.Fatalf("expected repo lookup for child only, got %#v", repo.lastParentEpicIDs)
	}
}

func TestMapParentEpics_EmptyWhenNoNonEpicIDs(t *testing.T) {
	repo := newMockWorkItemRepo()
	svc := &WorkItemService{items: repo}
	got, err := svc.MapParentEpics(context.Background(), []model.WorkItem{
		{ID: uuid.New(), Type: model.WorkItemTypeEpic},
	})
	if err != nil {
		t.Fatalf("MapParentEpics: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %#v", got)
	}
	if len(repo.lastParentEpicIDs) != 0 {
		t.Fatalf("expected no repo call when only epics, got %#v", repo.lastParentEpicIDs)
	}
}

func TestMapParentEpics_ForwardsOnlyNonEpicIDs(t *testing.T) {
	repo := newMockWorkItemRepo()
	svc := &WorkItemService{items: repo}

	ticketID := uuid.New()
	taskID := uuid.New()
	epicID := uuid.New()
	repo.parentEpics = map[uuid.UUID]model.ParentEpicRef{
		ticketID: {DisplayID: "TASK-76", Title: "Epic"},
		taskID:   {DisplayID: "TASK-76", Title: "Epic"},
	}

	got, err := svc.MapParentEpics(context.Background(), []model.WorkItem{
		{ID: ticketID, Type: model.WorkItemTypeTicket},
		{ID: epicID, Type: model.WorkItemTypeEpic},
		{ID: taskID, Type: model.WorkItemTypeTask},
	})
	if err != nil {
		t.Fatalf("MapParentEpics: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 parent epics, got %#v", got)
	}
	if len(repo.lastParentEpicIDs) != 2 {
		t.Fatalf("expected 2 ids forwarded to repo, got %#v", repo.lastParentEpicIDs)
	}
	for _, id := range repo.lastParentEpicIDs {
		if id == epicID {
			t.Fatalf("epic id must not be forwarded to repo: %#v", repo.lastParentEpicIDs)
		}
	}
}

// MapParentEpics SQL contract (repository layer — host integration / DB tests):
//
//  1. child_of / parent_of relations to an epic beat parent_id (precedence 1 < 2)
//  2. Earliest relation wins when multiple epics are linked
//  3. Non-epic parent_id targets produce no badge (JOIN requires parent.type = 'epic')
//  4. Soft-deleted children/epics are excluded
//  5. Items of type epic never appear in the result set
//
// Repository tests need a live Postgres (or sqlmock); none in this package today.
