package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// watcherRepository is the minimal interface for listing watchers.
type watcherRepository interface {
	ListByWorkItem(ctx context.Context, workItemID uuid.UUID) ([]model.WorkItemWatcherWithUser, error)
}

// NotificationWatcherTask sends email notifications to watchers when a watched work item changes.
type NotificationWatcherTask struct {
	watchers watcherRepository
	users    userRepository
	settings userSettingRepository
	sender   emailSender
	baseURL  string
	logger   zerolog.Logger
}

// NewNotificationWatcherTask creates the task.
func NewNotificationWatcherTask(
	watchers watcherRepository,
	users userRepository,
	settings userSettingRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationWatcherTask {
	return &NotificationWatcherTask{
		watchers: watchers,
		users:    users,
		settings: settings,
		sender:   sender,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationWatcherTask) Name() string {
	return "notification.watcher"
}

// Execute processes a watcher notification event.
func (t *NotificationWatcherTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.WatcherEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid watcher event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Int("item_number", evt.ItemNumber).
		Str("event_type", evt.EventType).
		Logger()

	// List all watchers for this work item
	watchers, err := t.watchers.ListByWorkItem(ctx, evt.WorkItemID)
	if err != nil {
		return fmt.Errorf("listing watchers: %w", err)
	}

	if len(watchers) == 0 {
		l.Debug().Msg("no watchers for work item")
		return nil
	}

	// Load actor display name for the email
	actor, err := t.users.GetByID(ctx, evt.ActorID)
	if err != nil {
		return fmt.Errorf("loading actor: %w", err)
	}

	// Resolve user UUIDs to display names for user-reference fields (e.g. assignee)
	t.resolveUserFields(ctx, &evt)

	for _, w := range watchers {
		// Skip the actor — don't notify users about their own changes
		if w.UserID == evt.ActorID {
			continue
		}

		// Check watcher's notification preferences for this project
		if !t.isWatcherEnabled(ctx, w.UserID, evt.ProjectID, evt.EventType) {
			l.Debug().Str("user_id", w.UserID.String()).Msg("watcher notification disabled for user")
			continue
		}

		subject := fmt.Sprintf("[%s] #%d updated: %s",
			evt.ProjectKey, evt.ItemNumber, evt.Title)

		itemURL := fmt.Sprintf("%s/projects/%s/items/%d",
			t.baseURL, evt.ProjectKey, evt.ItemNumber)

		body := watcherEmailHTML(actor.DisplayName, evt, itemURL)

		if err := t.sender.Send(ctx, w.Email, subject, body); err != nil {
			l.Error().Err(err).Str("to", w.Email).Msg("failed to send watcher notification")
			continue
		}

		l.Info().Str("to", w.Email).Msg("watcher notification sent")
	}

	return nil
}

// isWatcherEnabled checks whether the user has watcher notifications enabled for the project.
func (t *NotificationWatcherTask) isWatcherEnabled(ctx context.Context, userID, projectID uuid.UUID, eventType string) bool {
	setting, err := t.settings.Get(ctx, userID, &projectID, "notifications")
	if err != nil {
		// No setting found — default is false (opt-in)
		return false
	}

	var prefs model.NotificationPreferences
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		t.logger.Warn().Err(err).Msg("invalid notification preferences, using defaults")
		return false
	}

	if eventType == "comment_added" {
		return prefs.CommentsOnWatched
	}
	return prefs.AnyUpdateOnWatched
}

// resolveUserFields replaces user UUIDs with display names in event field values.
func (t *NotificationWatcherTask) resolveUserFields(ctx context.Context, evt *model.WatcherEvent) {
	if evt.FieldName != "assignee" {
		return
	}
	if evt.OldValue != "" {
		if uid, err := uuid.Parse(evt.OldValue); err == nil {
			if u, err := t.users.GetByID(ctx, uid); err == nil {
				evt.OldValue = u.DisplayName
			}
		}
	}
	if evt.NewValue != "" {
		if uid, err := uuid.Parse(evt.NewValue); err == nil {
			if u, err := t.users.GetByID(ctx, uid); err == nil {
				evt.NewValue = u.DisplayName
			}
		}
	}
}

func watcherEmailHTML(actorName string, evt model.WatcherEvent, itemURL string) string {
	changeDetail := evt.Summary
	if changeDetail == "" && evt.FieldName != "" {
		if evt.OldValue != "" {
			changeDetail = fmt.Sprintf("%s changed from <strong>%s</strong> to <strong>%s</strong>",
				evt.FieldName, evt.OldValue, evt.NewValue)
		} else {
			changeDetail = fmt.Sprintf("%s set to <strong>%s</strong>",
				evt.FieldName, evt.NewValue)
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  <p><strong>%s</strong> updated a work item you are watching:</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s-%d</p>
    <p style="margin: 0 0 12px 0; font-size: 18px; font-weight: 600;">%s</p>
    <p style="margin: 0; font-size: 14px; color: #475569;">%s</p>
  </div>
  <p>
    <a href="%s" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 500;">View Work Item</a>
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">You received this email because you are watching this work item and have watcher notifications enabled. You can change your notification preferences in your Taskwondo settings.</p>
</body>
</html>`,
		actorName, evt.ProjectKey, evt.ItemNumber, evt.Title, changeDetail, itemURL)
}
