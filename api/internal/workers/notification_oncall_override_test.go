package workers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestNotificationOncallOverrideCreated_Execute(t *testing.T) {
	overrideUserID := uuid.New()
	scheduledUserID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		overrideUserID:  {ID: overrideUserID, Email: "override@example.com", DisplayName: "Olivia"},
		scheduledUserID: {ID: scheduledUserID, Email: "scheduled@example.com", DisplayName: "Sam"},
	}}
	sender := &mockEmailSender{}

	task := &NotificationOncallOverrideCreatedTask{
		users:  users,
		sender: sender,
		urls:   newTestURLBuilder(),
		logger: zerolog.Nop(),
	}

	teamID := uuid.New()
	evt := model.OncallOverrideCreatedEvent{
		OverrideID:     uuid.New(),
		RotationID:     uuid.New(),
		TeamID:         teamID,
		TeamName:       "Payments",
		ProjectID:      uuid.New(),
		ProjectKey:     "PAY",
		ProjectName:    "Payments Platform",
		OverrideUserID: overrideUserID,
		ScheduledUser:  scheduledUserID,
		StartAt:        time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC),
		EndAt:          time.Date(2026, 4, 10, 17, 0, 0, 0, time.UTC),
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(sender.sent))
	}

	// Override user email (index 0)
	overrideEmail := sender.sent[0]
	if overrideEmail.to != "override@example.com" {
		t.Errorf("expected override email to override@example.com, got %s", overrideEmail.to)
	}
	if !strings.Contains(overrideEmail.subject, "[PAY]") {
		t.Errorf("expected override subject to contain project key prefix, got %s", overrideEmail.subject)
	}
	if !strings.Contains(overrideEmail.body, "PAY") || !strings.Contains(overrideEmail.body, "Payments Platform") {
		t.Errorf("expected override body to contain project key and name, got %s", overrideEmail.body)
	}
	expectedURL := "https://example.com/d/projects/PAY/teams/" + teamID.String() + "?tab=oncall"
	if !strings.Contains(overrideEmail.body, expectedURL) {
		t.Errorf("expected override body to contain oncall tab URL %q, got %s", expectedURL, overrideEmail.body)
	}

	// Covered user email (index 1)
	coveredEmail := sender.sent[1]
	if coveredEmail.to != "scheduled@example.com" {
		t.Errorf("expected covered email to scheduled@example.com, got %s", coveredEmail.to)
	}
	if !strings.Contains(coveredEmail.subject, "[PAY]") {
		t.Errorf("expected covered subject to contain project key prefix, got %s", coveredEmail.subject)
	}
	if !strings.Contains(coveredEmail.body, "Payments Platform") {
		t.Errorf("expected covered body to contain project name, got %s", coveredEmail.body)
	}
	if !strings.Contains(coveredEmail.body, expectedURL) {
		t.Errorf("expected covered body to contain oncall tab URL %q, got %s", expectedURL, coveredEmail.body)
	}
}

func TestNotificationOncallOverrideCancelled_Execute(t *testing.T) {
	overrideUserID := uuid.New()

	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		overrideUserID: {ID: overrideUserID, Email: "override@example.com", DisplayName: "Olivia"},
	}}
	sender := &mockEmailSender{}

	task := &NotificationOncallOverrideCancelledTask{
		users:  users,
		sender: sender,
		urls:   newTestURLBuilder(),
		logger: zerolog.Nop(),
	}

	teamID := uuid.New()
	evt := model.OncallOverrideCancelledEvent{
		OverrideID:     uuid.New(),
		RotationID:     uuid.New(),
		TeamID:         teamID,
		TeamName:       "Payments",
		ProjectID:      uuid.New(),
		ProjectKey:     "PAY",
		ProjectName:    "Payments Platform",
		OverrideUserID: overrideUserID,
		ScheduledUser:  overrideUserID,
		StartAt:        time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC),
		EndAt:          time.Date(2026, 4, 10, 17, 0, 0, 0, time.UTC),
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}

	email := sender.sent[0]
	if !strings.Contains(email.subject, "[PAY]") {
		t.Errorf("expected cancelled subject to contain project key prefix, got %s", email.subject)
	}
	if !strings.Contains(email.body, "PAY") || !strings.Contains(email.body, "Payments Platform") {
		t.Errorf("expected cancelled body to contain project key and name, got %s", email.body)
	}
	expectedURL := "https://example.com/d/projects/PAY/teams/" + teamID.String() + "?tab=oncall"
	if !strings.Contains(email.body, expectedURL) {
		t.Errorf("expected cancelled body to contain oncall tab URL %q, got %s", expectedURL, email.body)
	}
}
