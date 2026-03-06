package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// NotificationMemberAddedTask sends an email to a user when they are added to a project.
type NotificationMemberAddedTask struct {
	users    userRepository
	settings userSettingRepository
	sender   emailSender
	baseURL  string
	logger   zerolog.Logger
}

// NewNotificationMemberAddedTask creates the task.
func NewNotificationMemberAddedTask(
	users userRepository,
	settings userSettingRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationMemberAddedTask {
	return &NotificationMemberAddedTask{
		users:    users,
		settings: settings,
		sender:   sender,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationMemberAddedTask) Name() string {
	return "notification.member_added"
}

// Execute processes a member-added event.
func (t *NotificationMemberAddedTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.MemberAddedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid member added event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Str("user_id", evt.UserID.String()).
		Logger()

	if !t.isEnabled(ctx, evt.UserID, evt.ProjectID) {
		l.Debug().Msg("member added notification disabled for user")
		return nil
	}

	user, err := t.users.GetByID(ctx, evt.UserID)
	if err != nil {
		return fmt.Errorf("loading user: %w", err)
	}

	addedBy, err := t.users.GetByID(ctx, evt.AddedByID)
	if err != nil {
		return fmt.Errorf("loading added-by user: %w", err)
	}

	subject := fmt.Sprintf("You've been added to project %s", evt.ProjectName)

	projectURL := fmt.Sprintf("%s/projects/%s", t.baseURL, evt.ProjectKey)

	body := memberAddedEmailHTML(addedBy.DisplayName, evt.ProjectName, evt.ProjectKey, evt.Role, projectURL)

	if err := t.sender.Send(ctx, user.Email, subject, body); err != nil {
		return fmt.Errorf("sending member added email: %w", err)
	}

	l.Info().Str("to", user.Email).Msg("member added notification sent")
	return nil
}

func (t *NotificationMemberAddedTask) isEnabled(ctx context.Context, userID, projectID uuid.UUID) bool {
	setting, err := t.settings.Get(ctx, userID, &projectID, "notifications")
	if err != nil {
		// No project-level setting yet — default is false (opt-in)
		return false
	}

	var prefs model.NotificationPreferences
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		return false
	}

	return prefs.AddedToProject
}

func memberAddedEmailHTML(addedByName, projectName, projectKey, role, projectURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  <p><strong>%s</strong> added you to the project <strong>%s</strong> as a <strong>%s</strong>.</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s</p>
    <p style="margin: 0; font-size: 18px; font-weight: 600;">%s</p>
  </div>
  <p>
    <a href="%s" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 500;">View Project</a>
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">You received this email because you have project membership notifications enabled. You can change your notification preferences in your Taskwondo settings.</p>
</body>
</html>`, addedByName, projectName, role, projectKey, projectName, projectURL)
}
