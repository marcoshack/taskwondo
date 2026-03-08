package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/i18n"
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

		lang := getUserLanguage(ctx, t.settings, w.UserID)

		subject := i18n.T(lang, "email.watcher.subject",
			"projectKey", evt.ProjectKey,
			"itemNumber", fmt.Sprintf("%d", evt.ItemNumber),
			"title", evt.Title)

		itemURL := fmt.Sprintf("%s/d/projects/%s/items/%d",
			t.baseURL, evt.ProjectKey, evt.ItemNumber)

		body := watcherEmailHTML(lang, actor.DisplayName, evt, itemURL)

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

func watcherEmailHTML(lang, actorName string, evt model.WatcherEvent, itemURL string) string {
	changeDetail := evt.Summary
	if changeDetail == "" && evt.FieldName != "" {
		if evt.OldValue != "" {
			changeDetail = i18n.T(lang, "email.watcher.field_changed",
				"fieldName", evt.FieldName,
				"oldValue", evt.OldValue,
				"newValue", evt.NewValue)
		} else {
			changeDetail = i18n.T(lang, "email.watcher.field_set",
				"fieldName", evt.FieldName,
				"newValue", evt.NewValue)
		}
	}

	intro := i18n.T(lang, "email.watcher.intro", "actorName", actorName)
	content := fmt.Sprintf("<p>%s</p>\n  %s", intro, itemCard(evt.ProjectKey, evt.ItemNumber, evt.Title, changeDetail))
	return emailHTML(lang, "email.watcher.cta", itemURL, "email.watcher.footer", content)
}
