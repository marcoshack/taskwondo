package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// NotificationCommentOnAssignedTask sends an email to the assignee when someone comments on their item.
type NotificationCommentOnAssignedTask struct {
	users    userRepository
	settings userSettingRepository
	sender   emailSender
	baseURL  string
	logger   zerolog.Logger
}

// NewNotificationCommentOnAssignedTask creates the task.
func NewNotificationCommentOnAssignedTask(
	users userRepository,
	settings userSettingRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationCommentOnAssignedTask {
	return &NotificationCommentOnAssignedTask{
		users:    users,
		settings: settings,
		sender:   sender,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationCommentOnAssignedTask) Name() string {
	return "notification.comment_assigned"
}

// Execute processes a comment-on-assigned event.
func (t *NotificationCommentOnAssignedTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.CommentOnAssignedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid comment on assigned event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Int("item_number", evt.ItemNumber).
		Str("assignee_id", evt.AssigneeID.String()).
		Logger()

	if !t.isEnabled(ctx, evt.AssigneeID, evt.ProjectID) {
		l.Debug().Msg("comment on assigned notification disabled for user")
		return nil
	}

	assignee, err := t.users.GetByID(ctx, evt.AssigneeID)
	if err != nil {
		return fmt.Errorf("loading assignee: %w", err)
	}

	commenter, err := t.users.GetByID(ctx, evt.CommenterID)
	if err != nil {
		return fmt.Errorf("loading commenter: %w", err)
	}

	subject := fmt.Sprintf("[%s] New comment on #%d: %s",
		evt.ProjectKey, evt.ItemNumber, evt.Title)

	itemURL := fmt.Sprintf("%s/projects/%s/items/%d",
		t.baseURL, evt.ProjectKey, evt.ItemNumber)

	body := commentOnAssignedEmailHTML(commenter.DisplayName, evt.ProjectKey, evt.ItemNumber, evt.Title, evt.Preview, itemURL)

	if err := t.sender.Send(ctx, assignee.Email, subject, body); err != nil {
		return fmt.Errorf("sending comment on assigned email: %w", err)
	}

	l.Info().Str("to", assignee.Email).Msg("comment on assigned notification sent")
	return nil
}

func (t *NotificationCommentOnAssignedTask) isEnabled(ctx context.Context, userID, projectID uuid.UUID) bool {
	setting, err := t.settings.Get(ctx, userID, &projectID, "notifications")
	if err != nil {
		return false // default is false (opt-in)
	}

	var prefs model.NotificationPreferences
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		return false
	}

	return prefs.CommentsOnAssigned
}

func commentOnAssignedEmailHTML(commenterName, projectKey string, itemNumber int, title, preview, itemURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  <p><strong>%s</strong> commented on a work item assigned to you:</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s-%d</p>
    <p style="margin: 0 0 12px 0; font-size: 18px; font-weight: 600;">%s</p>
    <p style="margin: 0; font-size: 14px; color: #475569; font-style: italic;">"%s"</p>
  </div>
  <p>
    <a href="%s" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 500;">View Work Item</a>
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">You received this email because you have comment notifications enabled for assigned items. You can change your notification preferences in your Taskwondo settings.</p>
</body>
</html>`, commenterName, projectKey, itemNumber, title, preview, itemURL)
}
