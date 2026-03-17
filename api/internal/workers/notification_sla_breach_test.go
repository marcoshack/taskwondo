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

// --- Mock implementations for SLA breach notification dependencies ---

type mockBreachEscalationRepo struct {
	lists map[uuid.UUID]*model.EscalationList
}

func (m *mockBreachEscalationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.EscalationList, error) {
	if l, ok := m.lists[id]; ok {
		return l, nil
	}
	return nil, model.ErrNotFound
}

type mockBreachNotifRecordRepo struct {
	recorded []recordedNotif
}

type recordedNotif struct {
	workItemID   uuid.UUID
	statusName   string
	level        int
	thresholdPct int
}

func (m *mockBreachNotifRecordRepo) RecordSent(_ context.Context, workItemID uuid.UUID, statusName string, level, thresholdPct int) error {
	m.recorded = append(m.recorded, recordedNotif{workItemID, statusName, level, thresholdPct})
	return nil
}

// --- Tests ---

func TestNotificationSLABreach_Name(t *testing.T) {
	task := &NotificationSLABreachTask{}
	if task.Name() != "notification.sla_breach" {
		t.Fatalf("expected notification.sla_breach, got %s", task.Name())
	}
}

func TestNotificationSLABreach_Execute_SendsEmail(t *testing.T) {
	escListID := uuid.New()
	escLevelID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	workItemID := uuid.New()

	escRepo := &mockBreachEscalationRepo{lists: map[uuid.UUID]*model.EscalationList{
		escListID: {
			ID:        escListID,
			ProjectID: projectID,
			Levels: []model.EscalationLevel{
				{
					ID:           escLevelID,
					ListID:       escListID,
					ThresholdPct: 75,
					Position:     0,
					Users: []model.EscalationLevelUser{
						{UserID: userID, DisplayName: "Manager", Email: "mgr@example.com"},
					},
				},
			},
		},
	}}

	notifRepo := &mockBreachNotifRecordRepo{}
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := NewNotificationSLABreachTask(
		escRepo, notifRepo, settings, sender, "https://example.com", zerolog.Nop(),
	)

	evt := model.SLABreachEvent{
		WorkItemID:       workItemID,
		ProjectID:        projectID,
		ProjectKey:       "TP",
		ItemNumber:       42,
		Title:            "Fix the bug",
		StatusName:       "Open",
		SLAPercentage:    85,
		TargetSeconds:    3600,
		ElapsedSeconds:   3060,
		EscalationLevel:  1,
		EscalationListID: escListID,
		ThresholdPct:     75,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}

	if sender.sent[0].to != "mgr@example.com" {
		t.Errorf("expected to=mgr@example.com, got %s", sender.sent[0].to)
	}

	if !strings.Contains(sender.sent[0].subject, "SLA Warning") {
		t.Errorf("expected subject to contain 'SLA Warning', got %s", sender.sent[0].subject)
	}
	if !strings.Contains(sender.sent[0].subject, "TP") {
		t.Errorf("expected subject to contain project key 'TP', got %s", sender.sent[0].subject)
	}
	if !strings.Contains(sender.sent[0].subject, "#42") {
		t.Errorf("expected subject to contain '#42', got %s", sender.sent[0].subject)
	}
	if !strings.Contains(sender.sent[0].subject, "Level 1") {
		t.Errorf("expected subject to contain 'Level 1', got %s", sender.sent[0].subject)
	}

	// Check email body contains key info
	body := sender.sent[0].body
	if !strings.Contains(body, "Fix the bug") {
		t.Errorf("expected body to contain title")
	}
	if !strings.Contains(body, "85%") {
		t.Errorf("expected body to contain percentage")
	}
	if !strings.Contains(body, "TP-42") {
		t.Errorf("expected body to contain display ID")
	}
	if !strings.Contains(body, "https://example.com/d/projects/TP/items/42") {
		t.Errorf("expected body to contain item URL")
	}

	// Verify notification was recorded
	if len(notifRepo.recorded) != 1 {
		t.Fatalf("expected 1 recorded notification, got %d", len(notifRepo.recorded))
	}
	if notifRepo.recorded[0].workItemID != workItemID {
		t.Errorf("expected recorded work item ID %s, got %s", workItemID, notifRepo.recorded[0].workItemID)
	}
	if notifRepo.recorded[0].level != 1 {
		t.Errorf("expected recorded level 1, got %d", notifRepo.recorded[0].level)
	}
}

func TestNotificationSLABreach_Execute_SkipsDisabledPreference(t *testing.T) {
	escListID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	escRepo := &mockBreachEscalationRepo{lists: map[uuid.UUID]*model.EscalationList{
		escListID: {
			ID:        escListID,
			ProjectID: projectID,
			Levels: []model.EscalationLevel{
				{
					ThresholdPct: 75,
					Users:        []model.EscalationLevelUser{{UserID: userID, Email: "mgr@example.com"}},
				},
			},
		},
	}}

	// Explicitly disable SLA breach notifications
	prefs := model.NotificationPreferences{SLABreach: false}
	prefsJSON, _ := json.Marshal(prefs)
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{
		settingKey(userID, projectID, "notifications"): {
			UserID:    userID,
			ProjectID: &projectID,
			Key:       "notifications",
			Value:     prefsJSON,
		},
	}}

	notifRepo := &mockBreachNotifRecordRepo{}
	sender := &mockEmailSender{}

	task := NewNotificationSLABreachTask(
		escRepo, notifRepo, settings, sender, "https://example.com", zerolog.Nop(),
	)

	evt := model.SLABreachEvent{
		WorkItemID:       uuid.New(),
		ProjectID:        projectID,
		ProjectKey:       "TP",
		ItemNumber:       1,
		Title:            "Test",
		StatusName:       "Open",
		SLAPercentage:    80,
		TargetSeconds:    3600,
		ElapsedSeconds:   2880,
		EscalationLevel:  1,
		EscalationListID: escListID,
		ThresholdPct:     75,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails when preference disabled, got %d", len(sender.sent))
	}

	// Should not record since no emails were sent
	if len(notifRepo.recorded) != 0 {
		t.Fatalf("expected 0 recorded notifications, got %d", len(notifRepo.recorded))
	}
}

func TestNotificationSLABreach_Execute_SendsToMultipleRecipients(t *testing.T) {
	escListID := uuid.New()
	user1ID := uuid.New()
	user2ID := uuid.New()
	projectID := uuid.New()

	escRepo := &mockBreachEscalationRepo{lists: map[uuid.UUID]*model.EscalationList{
		escListID: {
			ID:        escListID,
			ProjectID: projectID,
			Levels: []model.EscalationLevel{
				{
					ThresholdPct: 75,
					Users: []model.EscalationLevelUser{
						{UserID: user1ID, Email: "user1@example.com"},
						{UserID: user2ID, Email: "user2@example.com"},
					},
				},
			},
		},
	}}

	notifRepo := &mockBreachNotifRecordRepo{}
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := NewNotificationSLABreachTask(
		escRepo, notifRepo, settings, sender, "https://example.com", zerolog.Nop(),
	)

	evt := model.SLABreachEvent{
		WorkItemID:       uuid.New(),
		ProjectID:        projectID,
		ProjectKey:       "TP",
		ItemNumber:       1,
		Title:            "Test",
		StatusName:       "Open",
		SLAPercentage:    80,
		TargetSeconds:    3600,
		ElapsedSeconds:   2880,
		EscalationLevel:  1,
		EscalationListID: escListID,
		ThresholdPct:     75,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(sender.sent))
	}

	// Verify notification was recorded (once, not per-recipient)
	if len(notifRepo.recorded) != 1 {
		t.Fatalf("expected 1 recorded notification, got %d", len(notifRepo.recorded))
	}
}

func TestNotificationSLABreach_Execute_InvalidPayload(t *testing.T) {
	task := NewNotificationSLABreachTask(
		&mockBreachEscalationRepo{lists: map[uuid.UUID]*model.EscalationList{}},
		&mockBreachNotifRecordRepo{},
		&mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		&mockEmailSender{},
		"https://example.com",
		zerolog.Nop(),
	)

	// Invalid JSON — should not return error (no retry)
	if err := task.Execute(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil error for bad payload, got %v", err)
	}
}

func TestNotificationSLABreach_Execute_EscalationLevelOutOfRange(t *testing.T) {
	escListID := uuid.New()
	projectID := uuid.New()

	escRepo := &mockBreachEscalationRepo{lists: map[uuid.UUID]*model.EscalationList{
		escListID: {
			ID:        escListID,
			ProjectID: projectID,
			Levels: []model.EscalationLevel{
				{ThresholdPct: 75, Users: []model.EscalationLevelUser{{UserID: uuid.New(), Email: "mgr@example.com"}}},
			},
		},
	}}

	sender := &mockEmailSender{}

	task := NewNotificationSLABreachTask(
		escRepo,
		&mockBreachNotifRecordRepo{},
		&mockUserSettingRepo{settings: map[string]*model.UserSetting{}},
		sender,
		"https://example.com",
		zerolog.Nop(),
	)

	// Level 5 but only 1 level exists
	evt := model.SLABreachEvent{
		WorkItemID:       uuid.New(),
		ProjectID:        projectID,
		ProjectKey:       "TP",
		ItemNumber:       1,
		Title:            "Test",
		StatusName:       "Open",
		SLAPercentage:    80,
		TargetSeconds:    3600,
		ElapsedSeconds:   2880,
		EscalationLevel:  5,
		EscalationListID: escListID,
		ThresholdPct:     75,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected 0 emails (level out of range), got %d", len(sender.sent))
	}
}

func TestNotificationSLABreach_Execute_DefaultSLABreachEnabled(t *testing.T) {
	// When no settings exist at all, SLA breach should default to enabled
	escListID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	escRepo := &mockBreachEscalationRepo{lists: map[uuid.UUID]*model.EscalationList{
		escListID: {
			ID:        escListID,
			ProjectID: projectID,
			Levels: []model.EscalationLevel{
				{
					ThresholdPct: 75,
					Users:        []model.EscalationLevelUser{{UserID: userID, Email: "mgr@example.com"}},
				},
			},
		},
	}}

	// No settings at all — should default to enabled
	settings := &mockUserSettingRepo{settings: map[string]*model.UserSetting{}}
	sender := &mockEmailSender{}

	task := NewNotificationSLABreachTask(
		escRepo,
		&mockBreachNotifRecordRepo{},
		settings,
		sender,
		"https://example.com",
		zerolog.Nop(),
	)

	evt := model.SLABreachEvent{
		WorkItemID:       uuid.New(),
		ProjectID:        projectID,
		ProjectKey:       "TP",
		ItemNumber:       1,
		Title:            "Test",
		StatusName:       "Open",
		SLAPercentage:    80,
		TargetSeconds:    3600,
		ElapsedSeconds:   2880,
		EscalationLevel:  1,
		EscalationListID: escListID,
		ThresholdPct:     75,
	}
	payload, _ := json.Marshal(evt)

	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email (default enabled), got %d", len(sender.sent))
	}
}

func TestSLABreachEmailHTML(t *testing.T) {
	evt := model.SLABreachEvent{
		ProjectKey:      "TP",
		ItemNumber:      42,
		Title:           "Fix urgent bug",
		StatusName:      "In Progress",
		SLAPercentage:   85,
		TargetSeconds:   7200,
		ElapsedSeconds:  6120,
		EscalationLevel: 1,
	}

	html := slaBreachEmailHTML("en", evt, "https://example.com/d/projects/TP/items/42")

	// Check key content is present
	if !strings.Contains(html, "TP-42") {
		t.Error("expected display ID TP-42 in HTML")
	}
	if !strings.Contains(html, "Fix urgent bug") {
		t.Error("expected title in HTML")
	}
	if !strings.Contains(html, "85%") {
		t.Error("expected percentage in HTML")
	}
	if !strings.Contains(html, "In Progress") {
		t.Error("expected status name in HTML")
	}
	if !strings.Contains(html, "https://example.com/d/projects/TP/items/42") {
		t.Error("expected item URL in HTML")
	}
	if !strings.Contains(html, "View Work Item") {
		t.Error("expected CTA text in HTML")
	}
	// Progress bar should use warning color (amber) for 85%
	if !strings.Contains(html, "#f59e0b") {
		t.Error("expected warning color in progress bar")
	}
}

func TestSLABreachEmailHTML_BreachedColor(t *testing.T) {
	evt := model.SLABreachEvent{
		ProjectKey:      "TP",
		ItemNumber:      1,
		Title:           "Overdue item",
		SLAPercentage:   120,
		TargetSeconds:   3600,
		ElapsedSeconds:  4320,
		EscalationLevel: 2,
	}

	html := slaBreachEmailHTML("en", evt, "https://example.com/d/projects/TP/items/1")

	// Progress bar should use red color for >= 100%
	if !strings.Contains(html, "#ef4444") {
		t.Error("expected breached (red) color in progress bar")
	}
}
