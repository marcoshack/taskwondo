package workers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// mockProjectMemberRepo is a test double for projectMemberRepository.
type mockProjectMemberRepo struct {
	members map[uuid.UUID][]model.ProjectMemberWithUser
}

func (m *mockProjectMemberRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]model.ProjectMemberWithUser, error) {
	return m.members[projectID], nil
}

func TestNotificationNewItem_Name(t *testing.T) {
	task := &NotificationNewItemTask{}
	if task.Name() != "notification.new_item" {
		t.Fatalf("expected notification.new_item, got %s", task.Name())
	}
}

func TestNotificationNewItem_Execute_SendsToMembers(t *testing.T) {
	creatorID := uuid.New()
	member1ID := uuid.New()
	member2ID := uuid.New()
	projectID := uuid.New()

	members := &mockProjectMemberRepo{members: map[uuid.UUID][]model.ProjectMemberWithUser{
		projectID: {
			{ProjectMember: model.ProjectMember{UserID: creatorID}, Email: "creator@example.com", DisplayName: "Creator"},
			{ProjectMember: model.ProjectMember{UserID: member1ID}, Email: "member1@example.com", DisplayName: "Member1"},
			{ProjectMember: model.ProjectMember{UserID: member2ID}, Email: "member2@example.com", DisplayName: "Member2"},
		},
	}}

	// Enable new_item_created for member1 and member2
	prefs := model.NotificationPreferences{NewItemCreated: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(member1ID, projectID, "notifications"): {UserID: member1ID, ProjectID: &projectID, Key: "notifications", Value: prefsJSON},
		settingKey(member2ID, projectID, "notifications"): {UserID: member2ID, ProjectID: &projectID, Key: "notifications", Value: prefsJSON},
	}}
	sender := &mockEmailSender{}

	task := &NotificationNewItemTask{
		members:  members,
		settings: settings,
		sender:   sender,
		baseURL:  "https://example.com",
		logger:   zerolog.Nop(),
	}

	evt := model.NewItemEvent{
		WorkItemID: uuid.New(),
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 42,
		Title:      "New feature",
		CreatorID:  creatorID,
		Type:       "task",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should send to member1 and member2, not creator
	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(sender.sent))
	}
}

func TestNotificationNewItem_Execute_SkipsCreator(t *testing.T) {
	creatorID := uuid.New()
	projectID := uuid.New()

	members := &mockProjectMemberRepo{members: map[uuid.UUID][]model.ProjectMemberWithUser{
		projectID: {
			{ProjectMember: model.ProjectMember{UserID: creatorID}, Email: "creator@example.com", DisplayName: "Creator"},
		},
	}}

	prefs := model.NotificationPreferences{NewItemCreated: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(creatorID, projectID, "notifications"): {UserID: creatorID, ProjectID: &projectID, Key: "notifications", Value: prefsJSON},
	}}
	sender := &mockEmailSender{}

	task := &NotificationNewItemTask{
		members: members, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.NewItemEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 1, Title: "Test", CreatorID: creatorID, Type: "task",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (creator excluded), got %d", len(sender.sent))
	}
}

func TestNotificationNewItem_Execute_SkipsWhenDisabled(t *testing.T) {
	creatorID := uuid.New()
	memberID := uuid.New()
	projectID := uuid.New()

	members := &mockProjectMemberRepo{members: map[uuid.UUID][]model.ProjectMemberWithUser{
		projectID: {
			{ProjectMember: model.ProjectMember{UserID: memberID}, Email: "member@example.com", DisplayName: "Member"},
		},
	}}

	// No settings — defaults to false
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := &NotificationNewItemTask{
		members: members, settings: settings, sender: sender,
		baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	evt := model.NewItemEvent{
		WorkItemID: uuid.New(), ProjectKey: "TP", ProjectID: projectID,
		ItemNumber: 1, Title: "Test", CreatorID: creatorID, Type: "task",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (disabled), got %d", len(sender.sent))
	}
}

func TestNotificationNewItem_Execute_InvalidPayload(t *testing.T) {
	task := &NotificationNewItemTask{
		members: &mockProjectMemberRepo{}, settings: &mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		sender: &mockEmailSender{}, baseURL: "https://example.com", logger: zerolog.Nop(),
	}

	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}
