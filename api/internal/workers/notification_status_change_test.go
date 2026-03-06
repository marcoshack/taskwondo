package workers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestNotificationStatusChange_Name(t *testing.T) {
	task := &NotificationStatusChangeTask{}
	if task.Name() != "notification.status_change" {
		t.Fatalf("expected notification.status_change, got %s", task.Name())
	}
}

func TestNotificationStatusChange_Execute_SendsIntermediateNotification(t *testing.T) {
	assigneeID := uuid.New()
	actorID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		assigneeID: {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Alice"},
		actorID:    {ID: actorID, Email: "actor@example.com", DisplayName: "Bob"},
	}}

	prefs := model.NotificationPreferences{StatusChangesIntermediate: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(assigneeID, projectID, "notifications"): {UserID: assigneeID, ProjectID: &projectID, Key: "notifications", Value: prefsJSON},
	}}
	sender := &mockEmailSender{}

	task := &NotificationStatusChangeTask{
		users: users, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.StatusChangeEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 42, Title: "Fix bug", AssigneeID: assigneeID,
		ActorID: actorID, OldStatus: "open", NewStatus: "in_progress",
		Category: model.CategoryInProgress,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
	if sender.sent[0].to != "assignee@example.com" {
		t.Errorf("expected to=assignee@example.com, got %s", sender.sent[0].to)
	}
}

func TestNotificationStatusChange_Execute_SendsFinalNotification(t *testing.T) {
	assigneeID := uuid.New()
	actorID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		assigneeID: {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Alice"},
		actorID:    {ID: actorID, Email: "actor@example.com", DisplayName: "Bob"},
	}}

	prefs := model.NotificationPreferences{StatusChangesFinal: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(assigneeID, projectID, "notifications"): {UserID: assigneeID, ProjectID: &projectID, Key: "notifications", Value: prefsJSON},
	}}
	sender := &mockEmailSender{}

	task := &NotificationStatusChangeTask{
		users: users, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.StatusChangeEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 42, Title: "Fix bug", AssigneeID: assigneeID,
		ActorID: actorID, OldStatus: "in_progress", NewStatus: "done",
		Category: model.CategoryDone,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
}

func TestNotificationStatusChange_Execute_SkipsWhenIntermediateDisabled(t *testing.T) {
	assigneeID := uuid.New()
	actorID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{}}

	// Only final enabled, not intermediate
	prefs := model.NotificationPreferences{StatusChangesFinal: true, StatusChangesIntermediate: false}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(assigneeID, projectID, "notifications"): {UserID: assigneeID, ProjectID: &projectID, Key: "notifications", Value: prefsJSON},
	}}
	sender := &mockEmailSender{}

	task := &NotificationStatusChangeTask{
		users: users, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.StatusChangeEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 1, Title: "Test", AssigneeID: assigneeID,
		ActorID: actorID, OldStatus: "open", NewStatus: "in_progress",
		Category: model.CategoryInProgress,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (intermediate disabled), got %d", len(sender.sent))
	}
}

func TestNotificationStatusChange_Execute_SkipsWhenNoSettings(t *testing.T) {
	assigneeID := uuid.New()
	actorID := uuid.New()
	projectID := uuid.New()

	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := &NotificationStatusChangeTask{
		users: &mockUserRepo{users: map[uuid.UUID]*model.User{}},
		settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.StatusChangeEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 1, Title: "Test", AssigneeID: assigneeID,
		ActorID: actorID, OldStatus: "open", NewStatus: "done",
		Category: model.CategoryDone,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (no settings), got %d", len(sender.sent))
	}
}

func TestNotificationStatusChange_Execute_InvalidPayload(t *testing.T) {
	task := &NotificationStatusChangeTask{
		users: &mockUserRepo{users: map[uuid.UUID]*model.User{}},
		settings: &mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		sender: &mockEmailSender{}, baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}
