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

// mockWatcherRepo is a test double for watcherRepository.
type mockWatcherRepo struct {
	watchers map[uuid.UUID][]model.WorkItemWatcherWithUser
}

func (m *mockWatcherRepo) ListByWorkItem(_ context.Context, workItemID uuid.UUID) ([]model.WorkItemWatcherWithUser, error) {
	return m.watchers[workItemID], nil
}

func TestNotificationWatcher_Name(t *testing.T) {
	task := &NotificationWatcherTask{}
	if task.Name() != "notification.watcher" {
		t.Fatalf("expected notification.watcher, got %s", task.Name())
	}
}

func TestNotificationWatcher_Execute_SendsToWatchers(t *testing.T) {
	actorID := uuid.New()
	watcherID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcherID},
				DisplayName:     "Watcher",
				Email:           "watcher@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID: {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
	}}

	// Enable watcher notifications for the watcher
	prefs := model.NotificationPreferences{AnyUpdateOnWatched: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(watcherID, projectID, "notifications"): {
			UserID:    watcherID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 42,
		Title:      "Fix the bug",
		ActorID:    actorID,
		EventType:  "field_change",
		FieldName:  "status",
		OldValue:   "open",
		NewValue:   "in_progress",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
	if sender.sent[0].to != "watcher@example.com" {
		t.Errorf("expected to=watcher@example.com, got %s", sender.sent[0].to)
	}
	if sender.sent[0].subject != "[TP] #42 updated: Fix the bug" {
		t.Errorf("unexpected subject: %s", sender.sent[0].subject)
	}
}

func TestNotificationWatcher_Execute_SkipsActor(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	// Actor is also a watcher
	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: actorID},
				DisplayName:     "Actor",
				Email:           "actor@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID: {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
	}}

	prefs := model.NotificationPreferences{AnyUpdateOnWatched: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(actorID, projectID, "notifications"): {
			UserID:    actorID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 1,
		Title:      "Some task",
		ActorID:    actorID,
		EventType:  "field_change",
		FieldName:  "status",
		OldValue:   "open",
		NewValue:   "closed",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (actor is watcher), got %d", len(sender.sent))
	}
}

func TestNotificationWatcher_Execute_SkipsWhenPreferenceDisabled(t *testing.T) {
	actorID := uuid.New()
	watcherID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcherID},
				DisplayName:     "Watcher",
				Email:           "watcher@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID: {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
	}}

	// No settings at all — defaults to false for AnyUpdateOnWatched
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 1,
		Title:      "Some task",
		ActorID:    actorID,
		EventType:  "field_change",
		FieldName:  "priority",
		OldValue:   "low",
		NewValue:   "high",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (preference disabled), got %d", len(sender.sent))
	}
}

func TestNotificationWatcher_Execute_SendsCommentNotification(t *testing.T) {
	actorID := uuid.New()
	watcherID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcherID},
				DisplayName:     "Watcher",
				Email:           "watcher@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID: {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
	}}

	prefs := model.NotificationPreferences{CommentsOnWatched: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(watcherID, projectID, "notifications"): {
			UserID:    watcherID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 1,
		Title:      "Some task",
		ActorID:    actorID,
		EventType:  "comment_added",
		Summary:    "New comment added",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email for comment notification, got %d", len(sender.sent))
	}
	if sender.sent[0].to != "watcher@example.com" {
		t.Errorf("expected to=watcher@example.com, got %s", sender.sent[0].to)
	}
}

func TestNotificationWatcher_Execute_SkipsCommentWhenDisabled(t *testing.T) {
	actorID := uuid.New()
	watcherID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcherID},
				DisplayName:     "Watcher",
				Email:           "watcher@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID: {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
	}}

	// AnyUpdateOnWatched enabled but CommentsOnWatched disabled
	prefs := model.NotificationPreferences{AnyUpdateOnWatched: true, CommentsOnWatched: false}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(watcherID, projectID, "notifications"): {
			UserID:    watcherID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 1,
		Title:      "Some task",
		ActorID:    actorID,
		EventType:  "comment_added",
		Summary:    "New comment added",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (comment pref disabled), got %d", len(sender.sent))
	}
}

func TestNotificationWatcher_Execute_NoWatchers(t *testing.T) {
	actorID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID: {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
	}}
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  uuid.New(),
		ItemNumber: 1,
		Title:      "Some task",
		ActorID:    actorID,
		EventType:  "field_change",
		FieldName:  "status",
		OldValue:   "open",
		NewValue:   "closed",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (no watchers), got %d", len(sender.sent))
	}
}

func TestNotificationWatcher_Execute_MultipleWatchers(t *testing.T) {
	actorID := uuid.New()
	watcher1ID := uuid.New()
	watcher2ID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcher1ID},
				DisplayName:     "Watcher1",
				Email:           "watcher1@example.com",
			},
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcher2ID},
				DisplayName:     "Watcher2",
				Email:           "watcher2@example.com",
			},
			{
				// Actor is also a watcher — should be skipped
				WorkItemWatcher: model.WorkItemWatcher{UserID: actorID},
				DisplayName:     "Actor",
				Email:           "actor@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID: {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
	}}

	prefs := model.NotificationPreferences{AnyUpdateOnWatched: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(watcher1ID, projectID, "notifications"): {
			UserID:    watcher1ID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
		settingKey(watcher2ID, projectID, "notifications"): {
			UserID:    watcher2ID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
		settingKey(actorID, projectID, "notifications"): {
			UserID:    actorID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 10,
		Title:      "Multi-watcher task",
		ActorID:    actorID,
		EventType:  "field_change",
		FieldName:  "status",
		OldValue:   "open",
		NewValue:   "closed",
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 emails (actor excluded), got %d", len(sender.sent))
	}
}

func TestNotificationWatcher_Execute_InvalidPayload(t *testing.T) {
	task := newTestWatcherTask(
		&mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{}},
		&mockUserRepo{users: map[uuid.UUID]*model.User{}},
		&mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		&mockEmailSender{},
	)

	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}

func TestNotificationWatcher_Execute_ResolvesAssigneeUUID(t *testing.T) {
	actorID := uuid.New()
	assigneeID := uuid.New()
	watcherID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcherID},
				DisplayName:     "Watcher",
				Email:           "watcher@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID:    {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
		assigneeID: {ID: assigneeID, Email: "assignee@example.com", DisplayName: "Jane Doe"},
	}}

	prefs := model.NotificationPreferences{AnyUpdateOnWatched: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(watcherID, projectID, "notifications"): {
			UserID:    watcherID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 42,
		Title:      "Test task",
		ActorID:    actorID,
		EventType:  "field_change",
		FieldName:  "assignee",
		OldValue:   "",
		NewValue:   assigneeID.String(),
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
	// The email body should contain the display name, not the UUID
	body := sender.sent[0].body
	if strings.Contains(body, assigneeID.String()) {
		t.Errorf("email body should not contain assignee UUID %s", assigneeID.String())
	}
	if !strings.Contains(body, "Jane Doe") {
		t.Errorf("email body should contain assignee display name 'Jane Doe', got: %s", body)
	}
}

func TestNotificationWatcher_Execute_ResolvesAssigneeReassignment(t *testing.T) {
	actorID := uuid.New()
	oldAssigneeID := uuid.New()
	newAssigneeID := uuid.New()
	watcherID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	watcherRepo := &mockWatcherRepo{watchers: map[uuid.UUID][]model.WorkItemWatcherWithUser{
		workItemID: {
			{
				WorkItemWatcher: model.WorkItemWatcher{UserID: watcherID},
				DisplayName:     "Watcher",
				Email:           "watcher@example.com",
			},
		},
	}}
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		actorID:       {ID: actorID, Email: "actor@example.com", DisplayName: "Actor"},
		oldAssigneeID: {ID: oldAssigneeID, Email: "old@example.com", DisplayName: "Old Person"},
		newAssigneeID: {ID: newAssigneeID, Email: "new@example.com", DisplayName: "New Person"},
	}}

	prefs := model.NotificationPreferences{AnyUpdateOnWatched: true}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(watcherID, projectID, "notifications"): {
			UserID:    watcherID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}
	sender := &mockEmailSender{}

	task := newTestWatcherTask(watcherRepo, users, settings, sender)

	evt := model.WatcherEvent{
		WorkItemID: workItemID,
		ProjectKey: "TP",
		ProjectID:  projectID,
		ItemNumber: 42,
		Title:      "Test task",
		ActorID:    actorID,
		EventType:  "field_change",
		FieldName:  "assignee",
		OldValue:   oldAssigneeID.String(),
		NewValue:   newAssigneeID.String(),
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
	body := sender.sent[0].body
	if !strings.Contains(body, "Old Person") {
		t.Errorf("email body should contain old assignee name 'Old Person'")
	}
	if !strings.Contains(body, "New Person") {
		t.Errorf("email body should contain new assignee name 'New Person'")
	}
}

func newTestWatcherTask(watchers watcherRepository, users userRepository, settings userSettingRepository, sender emailSender) *NotificationWatcherTask {
	return &NotificationWatcherTask{
		watchers: watchers,
		users:    users,
		settings: settings,
		sender:   sender,
		baseURL:  "https://example.com",
		logger:   zerolog.Nop(),
	}
}
