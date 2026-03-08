package workers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// languageKey returns a settings map key for a user's language preference (nil projectID).
func languageKey(userID uuid.UUID) string {
	return userID.String() + "language"
}

// mockUserRepo is a test double for userRepository.
type mockUserRepo struct {
	users map[uuid.UUID]*model.User
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

// mockProjectRepo is a test double for projectRepository.
type mockProjectRepo struct {
	projects map[uuid.UUID]*model.Project
}

func (m *mockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return p, nil
}

func (m *mockProjectRepo) ResolveNamespaces(_ context.Context, _ []string) (map[string]model.ProjectNamespaceInfo, error) {
	return nil, nil
}

// mockUserSettingRepo is a test double for userSettingRepository.
type mockUserSettingRepo struct {
	settings map[string]*model.UserSetting // key: userID+projectID
}

func (m *mockUserSettingRepo) Get(_ context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) (*model.UserSetting, error) {
	k := userID.String()
	if projectID != nil {
		k += projectID.String()
	}
	k += key
	s, ok := m.settings[k]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func settingKey(userID, projectID uuid.UUID, key string) string {
	return userID.String() + projectID.String() + key
}

// mockEmailSender records sent emails for assertion.
type mockEmailSender struct {
	sent []sentEmail
}

type sentEmail struct {
	to, subject, body string
}

func (m *mockEmailSender) Send(_ context.Context, to, subject, body string) error {
	m.sent = append(m.sent, sentEmail{to: to, subject: subject, body: body})
	return nil
}

func TestNotificationAssignment_Name(t *testing.T) {
	task := &NotificationAssignmentTask{}
	if task.Name() != "notification.assignment" {
		t.Fatalf("expected notification.assignment, got %s", task.Name())
	}
}

func TestNotificationAssignment_Execute(t *testing.T) {
	assigneeID := uuid.New()
	assignerID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		assigneeID: {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Alice"},
		assignerID: {ID: assignerID, Email: "assigner@example.com", DisplayName: "Bob"},
	}}
	projects := &mockProjectRepo{projects: map[uuid.UUID]*model.Project{
		projectID: {ID: projectID, Name: "Test Project", Key: "TP"},
	}}
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := newTestAssignmentTask(users, projects, settings, sender)

	evt := model.AssignmentEvent{
		WorkItemID: uuid.New(),
		ProjectKey: "TP",
		ItemNumber: 42,
		Title:      "Fix the bug",
		AssigneeID: assigneeID,
		AssignerID: assignerID,
		ProjectID:  projectID,
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
	if sender.sent[0].subject != "[TP] Work item #42 assigned to you: Fix the bug" {
		t.Errorf("unexpected subject: %s", sender.sent[0].subject)
	}
}

func TestNotificationAssignment_DisabledPreference(t *testing.T) {
	assigneeID := uuid.New()
	assignerID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		assigneeID: {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Alice"},
	}}
	projects := &mockProjectRepo{projects: map[uuid.UUID]*model.Project{
		projectID: {ID: projectID, Name: "Test Project", Key: "TP"},
	}}

	// Explicitly disable assignment notifications
	prefs := model.NotificationPreferences{AssignedToMe: false}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(assigneeID, projectID, "notifications"): {
			UserID:    assigneeID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestAssignmentTask(users, projects, settings, sender)

	evt := model.AssignmentEvent{
		WorkItemID: uuid.New(),
		ProjectKey: "TP",
		ItemNumber: 1,
		Title:      "Some task",
		AssigneeID: assigneeID,
		AssignerID: assignerID,
		ProjectID:  projectID,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails when disabled, got %d", len(sender.sent))
	}
}

func TestNotificationAssignment_LanguagePreference(t *testing.T) {
	assigneeID := uuid.New()
	assignerID := uuid.New()
	projectID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		assigneeID: {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Alice"},
		assignerID: {ID: assignerID, Email: "assigner@example.com", DisplayName: "Bob"},
	}}
	projects := &mockProjectRepo{projects: map[uuid.UUID]*model.Project{
		projectID: {ID: projectID, Name: "Test Project", Key: "TP"},
	}}

	// Set language to Portuguese
	langJSON, _ := json.Marshal("pt")
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		languageKey(assigneeID): {
			UserID: assigneeID,
			Key:    "language",
			Value:  langJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestAssignmentTask(users, projects, settings, sender)

	evt := model.AssignmentEvent{
		WorkItemID: uuid.New(),
		ProjectKey: "TP",
		ItemNumber: 42,
		Title:      "Fix the bug",
		AssigneeID: assigneeID,
		AssignerID: assignerID,
		ProjectID:  projectID,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
	// Subject should be in Portuguese
	if !strings.Contains(sender.sent[0].subject, "atribuído a você") {
		t.Errorf("expected Portuguese subject, got %s", sender.sent[0].subject)
	}
	// Body should contain Portuguese CTA
	if !strings.Contains(sender.sent[0].body, "Ver Item de Trabalho") {
		t.Errorf("expected Portuguese CTA in body, got %s", sender.sent[0].body)
	}
}

func TestNotificationAssignment_InvalidPayload(t *testing.T) {
	task := newTestAssignmentTask(
		&mockUserRepo{users: map[uuid.UUID]*model.User{}},
		&mockProjectRepo{projects: map[uuid.UUID]*model.Project{}},
		&mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		&mockEmailSender{},
	)

	// Invalid JSON — should not return error (no retry)
	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}

// newTestAssignmentTask creates a task with a mock sender instead of *email.Sender.
// We use the emailSenderInterface to allow testing without SMTP.
func newTestAssignmentTask(users userRepository, projects projectRepository, settings userSettingRepository, sender emailSender) *NotificationAssignmentTask {
	return &NotificationAssignmentTask{
		users:    users,
		projects: projects,
		settings: settings,
		sender:   sender,
		baseURL:  "https://example.com",
		logger:   zerolog.Nop(),
	}
}
