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

	lang := getUserLanguage(ctx, t.settings, evt.UserID)

	subject := i18n.T(lang, "email.member_added.subject",
		"projectName", evt.ProjectName)

	projectURL := fmt.Sprintf("%s/d/projects/%s", t.baseURL, evt.ProjectKey)

	body := memberAddedEmailHTML(lang, addedBy.DisplayName, evt.ProjectName, evt.ProjectKey, evt.Role, projectURL)

	if err := t.sender.Send(ctx, user.Email, subject, body); err != nil {
		return fmt.Errorf("sending member added email: %w", err)
	}

	l.Info().Str("to", user.Email).Msg("member added notification sent")
	return nil
}

func (t *NotificationMemberAddedTask) isEnabled(ctx context.Context, userID, _ uuid.UUID) bool {
	setting, err := t.settings.Get(ctx, userID, nil, "global_notifications")
	if err != nil {
		// No global setting yet — default is false (opt-in)
		return false
	}

	var prefs model.GlobalNotificationPreferences
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		return false
	}

	return prefs.AddedToProject
}

func memberAddedEmailHTML(lang, addedByName, projectName, projectKey, role, projectURL string) string {
	intro := i18n.T(lang, "email.member_added.intro",
		"addedByName", addedByName,
		"projectName", projectName,
		"role", role)
	content := fmt.Sprintf(`<p>%s</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s</p>
    <p style="margin: 0; font-size: 18px; font-weight: 600;">%s</p>
  </div>`, intro, projectKey, projectName)
	return emailHTML(lang, "email.member_added.cta", projectURL, "email.member_added.footer", content)
}
