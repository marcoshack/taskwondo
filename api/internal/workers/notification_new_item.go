package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// projectMemberRepository is the minimal interface for listing project members.
type projectMemberRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectMemberWithUser, error)
}

// NotificationNewItemTask sends emails to project members when a new work item is created.
type NotificationNewItemTask struct {
	members  projectMemberRepository
	settings userSettingRepository
	sender   emailSender
	baseURL  string
	logger   zerolog.Logger
}

// NewNotificationNewItemTask creates the task.
func NewNotificationNewItemTask(
	members projectMemberRepository,
	settings userSettingRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationNewItemTask {
	return &NotificationNewItemTask{
		members:  members,
		settings: settings,
		sender:   sender,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationNewItemTask) Name() string {
	return "notification.new_item"
}

// Execute processes a new item event.
func (t *NotificationNewItemTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.NewItemEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid new item event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Int("item_number", evt.ItemNumber).
		Logger()

	members, err := t.members.ListByProject(ctx, evt.ProjectID)
	if err != nil {
		return fmt.Errorf("listing project members: %w", err)
	}

	itemURL := fmt.Sprintf("%s/projects/%s/items/%d", t.baseURL, evt.ProjectKey, evt.ItemNumber)

	for _, m := range members {
		// Skip the creator
		if m.UserID == evt.CreatorID {
			continue
		}

		if !t.isEnabled(ctx, m.UserID, evt.ProjectID) {
			continue
		}

		subject := fmt.Sprintf("[%s] New %s created: %s-%d %s",
			evt.ProjectKey, evt.Type, evt.ProjectKey, evt.ItemNumber, evt.Title)

		body := newItemEmailHTML(evt.ProjectKey, evt.ItemNumber, evt.Title, evt.Type, itemURL)

		if err := t.sender.Send(ctx, m.Email, subject, body); err != nil {
			l.Error().Err(err).Str("to", m.Email).Msg("failed to send new item notification")
			continue
		}

		l.Info().Str("to", m.Email).Msg("new item notification sent")
	}

	return nil
}

func (t *NotificationNewItemTask) isEnabled(ctx context.Context, userID, projectID uuid.UUID) bool {
	setting, err := t.settings.Get(ctx, userID, &projectID, "notifications")
	if err != nil {
		return false // default is false (opt-in)
	}

	var prefs model.NotificationPreferences
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		return false
	}

	return prefs.NewItemCreated
}

func newItemEmailHTML(projectKey string, itemNumber int, title, itemType, itemURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  <p>A new <strong>%s</strong> was created in your project:</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s-%d</p>
    <p style="margin: 0; font-size: 18px; font-weight: 600;">%s</p>
  </div>
  <p>
    <a href="%s" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 500;">View Work Item</a>
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">You received this email because you have new item notifications enabled for this project. You can change your notification preferences in your Taskwondo settings.</p>
</body>
</html>`, itemType, projectKey, itemNumber, title, itemURL)
}
