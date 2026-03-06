package workers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestNotificationCommentAssigned_Name(t *testing.T) {
	task := &NotificationCommentOnAssignedTask{}
	if task.Name() != "notification.comment_assigned" {
		t.Fatalf("expected notification.comment_assigned, got %s", task.Name())
	}
}

func TestNotificationCommentAssigned_Execute_SendsToAssignee(t *testing.T) {
	assigneeID := uuid.New()
	commenterID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		assigneeID:  {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Alice"},
		commenterID: {ID: commenterID, Email: "commenter@example.com", DisplayName: "Bob"},
	}}

	prefs := model.NotificationPreferences{CommentsOnAssigned: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(assigneeID, projectID, "notifications"): {UserID: assigneeID, ProjectID: &projectID, Key: "notifications", Value: prefsJSON},
	}}
	sender := &mockEmailSender{}

	task := &NotificationCommentOnAssignedTask{
		users: users, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.CommentOnAssignedEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 42, Title: "Fix bug", AssigneeID: assigneeID,
		CommenterID: commenterID, Preview: "Looks good!",
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
	if sender.sent[0].subject != "[TP] New comment on #42: Fix bug" {
		t.Errorf("unexpected subject: %s", sender.sent[0].subject)
	}
}

func TestNotificationCommentAssigned_Execute_SkipsWhenDisabled(t *testing.T) {
	assigneeID := uuid.New()
	commenterID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		assigneeID: {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Alice"},
	}}

	// No settings — default is false
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := &NotificationCommentOnAssignedTask{
		users: users, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.CommentOnAssignedEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 1, Title: "Test", AssigneeID: assigneeID,
		CommenterID: commenterID, Preview: "test",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (disabled), got %d", len(sender.sent))
	}
}

func TestNotificationCommentAssigned_Execute_InvalidPayload(t *testing.T) {
	task := &NotificationCommentOnAssignedTask{
		users: &mockUserRepo{users: map[uuid.UUID]*model.User{}},
		settings: &mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		sender: &mockEmailSender{}, baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}
