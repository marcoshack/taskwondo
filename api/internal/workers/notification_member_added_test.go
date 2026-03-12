package workers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestNotificationMemberAdded_Name(t *testing.T) {
	task := &NotificationMemberAddedTask{}
	if task.Name() != "notification.member_added" {
		t.Fatalf("expected notification.member_added, got %s", task.Name())
	}
}

func TestNotificationMemberAdded_Execute_SendsToUser(t *testing.T) {
	userID := uuid.New()
	addedByID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		userID:    {ID: userID, Email: "user@example.com", DisplayName: "Alice"},
		addedByID: {ID: addedByID, Email: "admin@example.com", DisplayName: "Bob"},
	}}

	prefs := model.GlobalNotificationPreferences{AddedToProject: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		userID.String() + "global_notifications": {UserID: userID, Key: "global_notifications", Value: prefsJSON},
	}}
	sender := &mockEmailSender{}

	task := &NotificationMemberAddedTask{
		users: users, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.MemberAddedEvent{
		ProjectID: projectID, ProjectKey: "TP", ProjectName: "Test Project",
		UserID: userID, AddedByID: addedByID, Role: "member",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
	if sender.sent[0].to != "user@example.com" {
		t.Errorf("expected to=user@example.com, got %s", sender.sent[0].to)
	}
	if sender.sent[0].subject != "You've been added to project Test Project" {
		t.Errorf("unexpected subject: %s", sender.sent[0].subject)
	}
}

func TestNotificationMemberAdded_Execute_SkipsWhenDisabled(t *testing.T) {
	userID := uuid.New()
	addedByID := uuid.New()
	projectID := uuid.New()

	// No settings — default is false (opt-in)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := &NotificationMemberAddedTask{
		users:    &mockUserRepo{users: map[uuid.UUID]*model.User{}},
		settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.MemberAddedEvent{
		ProjectID: projectID, ProjectKey: "TP", ProjectName: "Test Project",
		UserID: userID, AddedByID: addedByID, Role: "member",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (disabled), got %d", len(sender.sent))
	}
}

func TestNotificationMemberAdded_Execute_InvalidPayload(t *testing.T) {
	task := &NotificationMemberAddedTask{
		users: &mockUserRepo{users: map[uuid.UUID]*model.User{}},
		settings: &mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		sender: &mockEmailSender{}, baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}
