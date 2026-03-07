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

		lang := getUserLanguage(ctx, t.settings, m.UserID)

		subject := i18n.T(lang, "email.new_item.subject",
			"projectKey", evt.ProjectKey,
			"type", evt.Type,
			"displayId", fmt.Sprintf("%s-%d", evt.ProjectKey, evt.ItemNumber),
			"title", evt.Title)

		body := newItemEmailHTML(lang, evt.ProjectKey, evt.ItemNumber, evt.Title, evt.Type, itemURL)

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

func newItemEmailHTML(lang, projectKey string, itemNumber int, title, itemType, itemURL string) string {
	intro := i18n.T(lang, "email.new_item.intro", "type", itemType)
	content := fmt.Sprintf("<p>%s</p>\n  %s", intro, itemCard(projectKey, itemNumber, title, ""))
	return emailHTML(lang, "email.new_item.cta", itemURL, "email.new_item.footer", content)
}
