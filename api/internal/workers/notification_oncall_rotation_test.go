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

func TestNotificationOncallRotation_Name(t *testing.T) {
	task := &NotificationOncallRotationTask{}
	if task.Name() != "oncall.rotation.advanced" {
		t.Fatalf("expected oncall.rotation.advanced, got %s", task.Name())
	}
}

func TestNotificationOncallRotation_Execute(t *testing.T) {
	oldUserID := uuid.New()
	newUserID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		oldUserID: {ID: oldUserID, Email: "old@example.com", DisplayName: "Alice"},
		newUserID: {ID: newUserID, Email: "new@example.com", DisplayName: "Bob"},
	}}
	sender := &mockEmailSender{}

	task := &NotificationOncallRotationTask{
		users:  users,
		sender: sender,
		logger: zerolog.Nop(),
	}

	evt := model.OncallRotationAdvancedEvent{
		RotationID: uuid.New(),
		TeamID:     uuid.New(),
		ProjectID:  uuid.New(),
		TeamName:   "Engineering",
		OldUserID:  oldUserID,
		NewUserID:  newUserID,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(sender.sent))
	}

	// Check incoming notification
	if sender.sent[0].to != "new@example.com" {
		t.Errorf("expected incoming email to new@example.com, got %s", sender.sent[0].to)
	}
	if !strings.Contains(sender.sent[0].subject, "Engineering") {
		t.Errorf("expected subject to contain team name, got %s", sender.sent[0].subject)
	}
	if !strings.Contains(sender.sent[0].body, "now on-call") {
		t.Errorf("expected body to contain 'now on-call', got %s", sender.sent[0].body)
	}

	// Check outgoing notification
	if sender.sent[1].to != "old@example.com" {
		t.Errorf("expected outgoing email to old@example.com, got %s", sender.sent[1].to)
	}
	if !strings.Contains(sender.sent[1].body, "has ended") {
		t.Errorf("expected body to contain 'has ended', got %s", sender.sent[1].body)
	}
}

func TestNotificationOncallRotation_SameUser(t *testing.T) {
	userID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		userID: {ID: userID, Email: "solo@example.com", DisplayName: "Solo"},
	}}
	sender := &mockEmailSender{}

	task := &NotificationOncallRotationTask{
		users:  users,
		sender: sender,
		logger: zerolog.Nop(),
	}

	evt := model.OncallRotationAdvancedEvent{
		RotationID: uuid.New(),
		TeamID:     uuid.New(),
		ProjectID:  uuid.New(),
		TeamName:   "Solo Team",
		OldUserID:  userID,
		NewUserID:  userID, // same user
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only incoming notification (no outgoing since same user)
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email for same user, got %d", len(sender.sent))
	}
	if sender.sent[0].to != "solo@example.com" {
		t.Errorf("expected email to solo@example.com, got %s", sender.sent[0].to)
	}
}

func TestNotificationOncallRotation_InvalidPayload(t *testing.T) {
	task := &NotificationOncallRotationTask{
		users:  &mockUserRepo{users: map[uuid.UUID]*model.User{}},
		sender: &mockEmailSender{},
		logger: zerolog.Nop(),
	}

	// Invalid JSON — should not return error (no retry)
	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}
