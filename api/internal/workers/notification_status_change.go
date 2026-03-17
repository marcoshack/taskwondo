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

	lang := getUserLanguage(ctx, t.settings, evt.AssigneeID)

	subject := i18n.T(lang, "email.status_change.subject",
		"projectKey", evt.ProjectKey,
		"itemNumber", fmt.Sprintf("%d", evt.ItemNumber),
		"newStatus", evt.NewStatus,
		"title", evt.Title)

	itemURL := fmt.Sprintf("%s/d/projects/%s/items/%d",
		t.baseURL, evt.ProjectKey, evt.ItemNumber)

	body := statusChangeEmailHTML(lang, actor.DisplayName, evt.ProjectKey, evt.ItemNumber, evt.Title, evt.OldStatus, evt.NewStatus, itemURL)

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

	prefs := model.DefaultNotificationPreferences()
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

func statusChangeEmailHTML(lang, actorName, projectKey string, itemNumber int, title, oldStatus, newStatus, itemURL string) string {
	intro := i18n.T(lang, "email.status_change.intro", "actorName", actorName)
	statusLine := i18n.T(lang, "email.status_change.status",
		"oldStatus", oldStatus,
		"newStatus", newStatus)
	content := fmt.Sprintf("<p>%s</p>\n  %s", intro, itemCard(projectKey, itemNumber, title, statusLine))
	return emailHTML(lang, "email.status_change.cta", itemURL, "email.status_change.footer", content)
}
