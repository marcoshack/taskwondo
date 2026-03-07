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

	lang := getUserLanguage(ctx, t.settings, evt.AssigneeID)

	subject := i18n.T(lang, "email.comment_assigned.subject",
		"projectKey", evt.ProjectKey,
		"itemNumber", fmt.Sprintf("%d", evt.ItemNumber),
		"title", evt.Title)

	itemURL := fmt.Sprintf("%s/projects/%s/items/%d",
		t.baseURL, evt.ProjectKey, evt.ItemNumber)

	body := commentOnAssignedEmailHTML(lang, commenter.DisplayName, evt.ProjectKey, evt.ItemNumber, evt.Title, evt.Preview, itemURL)

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

func commentOnAssignedEmailHTML(lang, commenterName, projectKey string, itemNumber int, title, preview, itemURL string) string {
	intro := i18n.T(lang, "email.comment_assigned.intro", "commenterName", commenterName)
	previewHTML := fmt.Sprintf(`<span style="font-style: italic;">"%s"</span>`, preview)
	content := fmt.Sprintf("<p>%s</p>\n  %s", intro, itemCard(projectKey, itemNumber, title, previewHTML))
	return emailHTML(lang, "email.comment_assigned.cta", itemURL, "email.comment_assigned.footer", content)
}
