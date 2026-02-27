package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// userRepository is the minimal interface for looking up users.
type userRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// projectRepository is the minimal interface for looking up projects.
type projectRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

// userSettingRepository is the minimal interface for reading user settings.
type userSettingRepository interface {
	Get(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) (*model.UserSetting, error)
}

// emailSender sends emails.
type emailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// NotificationAssignmentTask sends an email to the assignee when a work item is assigned.
type NotificationAssignmentTask struct {
	users    userRepository
	projects projectRepository
	settings userSettingRepository
	sender   emailSender
	baseURL  string
	logger   zerolog.Logger
}

// NewNotificationAssignmentTask creates the task.
func NewNotificationAssignmentTask(
	users userRepository,
	projects projectRepository,
	settings userSettingRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationAssignmentTask {
	return &NotificationAssignmentTask{
		users:    users,
		projects: projects,
		settings: settings,
		sender:   sender,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationAssignmentTask) Name() string {
	return "notification.assignment"
}

// Execute processes an assignment event.
func (t *NotificationAssignmentTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.AssignmentEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		// Bad payload — no point retrying
		t.logger.Error().Err(err).Msg("invalid assignment event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Int("item_number", evt.ItemNumber).
		Str("assignee_id", evt.AssigneeID.String()).
		Logger()

	// Check assignee's notification preferences
	if !t.isEnabled(ctx, evt.AssigneeID, evt.ProjectID) {
		l.Debug().Msg("assignment notification disabled for user")
		return nil
	}

	// Load assignee
	assignee, err := t.users.GetByID(ctx, evt.AssigneeID)
	if err != nil {
		return fmt.Errorf("loading assignee: %w", err)
	}

	// Load assigner for display name
	assigner, err := t.users.GetByID(ctx, evt.AssignerID)
	if err != nil {
		return fmt.Errorf("loading assigner: %w", err)
	}

	// Load project for display name
	project, err := t.projects.GetByID(ctx, evt.ProjectID)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	subject := fmt.Sprintf("[%s] Work item #%d assigned to you: %s",
		evt.ProjectKey, evt.ItemNumber, evt.Title)

	itemURL := fmt.Sprintf("%s/projects/%s/items/%d",
		t.baseURL, evt.ProjectKey, evt.ItemNumber)

	body := assignmentEmailHTML(project.Name, assigner.DisplayName, evt.Title, evt.ProjectKey, evt.ItemNumber, itemURL)

	if err := t.sender.Send(ctx, assignee.Email, subject, body); err != nil {
		return fmt.Errorf("sending assignment email: %w", err)
	}

	l.Info().Str("to", assignee.Email).Msg("assignment notification sent")
	return nil
}

// isEnabled checks whether the assignee has assignment notifications enabled.
func (t *NotificationAssignmentTask) isEnabled(ctx context.Context, userID, projectID uuid.UUID) bool {
	setting, err := t.settings.Get(ctx, userID, &projectID, "notifications")
	if err != nil {
		// No setting found — use default (assigned_to_me = true)
		return true
	}

	var prefs model.NotificationPreferences
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		t.logger.Warn().Err(err).Msg("invalid notification preferences, using defaults")
		return true
	}

	return prefs.AssignedToMe
}

func assignmentEmailHTML(projectName, assignerName, title, projectKey string, itemNumber int, itemURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  <p><strong>%s</strong> assigned a work item to you in <strong>%s</strong>:</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s-%d</p>
    <p style="margin: 0; font-size: 18px; font-weight: 600;">%s</p>
  </div>
  <p>
    <a href="%s" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 500;">View Work Item</a>
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">You received this email because you have assignment notifications enabled for this project. You can change your notification preferences in your Taskwondo settings.</p>
</body>
</html>`,
		assignerName, projectName, projectKey, itemNumber, title, itemURL)
}
