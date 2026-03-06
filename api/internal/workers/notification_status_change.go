package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// NotificationStatusChangeTask sends an email to the assignee when a work item's status changes.
type NotificationStatusChangeTask struct {
	users    userRepository
	settings userSettingRepository
	sender   emailSender
	baseURL  string
	logger   zerolog.Logger
}

// NewNotificationStatusChangeTask creates the task.
func NewNotificationStatusChangeTask(
	users userRepository,
	settings userSettingRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationStatusChangeTask {
	return &NotificationStatusChangeTask{
		users:    users,
		settings: settings,
		sender:   sender,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationStatusChangeTask) Name() string {
	return "notification.status_change"
}

// Execute processes a status change event.
func (t *NotificationStatusChangeTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.StatusChangeEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid status change event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Int("item_number", evt.ItemNumber).
		Str("category", evt.Category).
		Logger()

	if !t.isEnabled(ctx, evt.AssigneeID, evt.ProjectID, evt.Category) {
		l.Debug().Msg("status change notification disabled for user")
		return nil
	}

	assignee, err := t.users.GetByID(ctx, evt.AssigneeID)
	if err != nil {
		return fmt.Errorf("loading assignee: %w", err)
	}

	actor, err := t.users.GetByID(ctx, evt.ActorID)
	if err != nil {
		return fmt.Errorf("loading actor: %w", err)
	}

	subject := fmt.Sprintf("[%s] #%d status changed to %s: %s",
		evt.ProjectKey, evt.ItemNumber, evt.NewStatus, evt.Title)

	itemURL := fmt.Sprintf("%s/projects/%s/items/%d",
		t.baseURL, evt.ProjectKey, evt.ItemNumber)

	body := statusChangeEmailHTML(actor.DisplayName, evt.ProjectKey, evt.ItemNumber, evt.Title, evt.OldStatus, evt.NewStatus, itemURL)

	if err := t.sender.Send(ctx, assignee.Email, subject, body); err != nil {
		return fmt.Errorf("sending status change email: %w", err)
	}

	l.Info().Str("to", assignee.Email).Msg("status change notification sent")
	return nil
}

func (t *NotificationStatusChangeTask) isEnabled(ctx context.Context, userID, projectID uuid.UUID, category string) bool {
	setting, err := t.settings.Get(ctx, userID, &projectID, "notifications")
	if err != nil {
		return false // default is false (opt-in)
	}

	var prefs model.NotificationPreferences
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		return false
	}

	switch category {
	case model.CategoryInProgress:
		return prefs.StatusChangesIntermediate
	case model.CategoryDone, model.CategoryCancelled:
		return prefs.StatusChangesFinal
	default:
		return false
	}
}

func statusChangeEmailHTML(actorName, projectKey string, itemNumber int, title, oldStatus, newStatus, itemURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  <p><strong>%s</strong> changed the status of a work item assigned to you:</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s-%d</p>
    <p style="margin: 0 0 12px 0; font-size: 18px; font-weight: 600;">%s</p>
    <p style="margin: 0; font-size: 14px; color: #475569;">Status: <strong>%s</strong> → <strong>%s</strong></p>
  </div>
  <p>
    <a href="%s" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 500;">View Work Item</a>
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">You received this email because you have status change notifications enabled for assigned items. You can change your notification preferences in your Taskwondo settings.</p>
</body>
</html>`, actorName, projectKey, itemNumber, title, oldStatus, newStatus, itemURL)
}
