package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/i18n"
	"github.com/marcoshack/taskwondo/internal/model"
)

// NotificationInviteEmailTask sends an invite email to a user who does not yet have an account.
type NotificationInviteEmailTask struct {
	sender  emailSender
	baseURL string
	logger  zerolog.Logger
}

// NewNotificationInviteEmailTask creates the task.
func NewNotificationInviteEmailTask(
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationInviteEmailTask {
	return &NotificationInviteEmailTask{
		sender:  sender,
		baseURL: baseURL,
		logger:  logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationInviteEmailTask) Name() string {
	return "notification.invite_email"
}

// Execute processes an invite email event.
func (t *NotificationInviteEmailTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.InviteEmailEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid invite email event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Str("invitee_email", evt.InviteeEmail).
		Logger()

	lang := "en" // No user account yet, default to English

	subject := i18n.T(lang, "email.invite.subject",
		"projectName", evt.ProjectName)

	inviteURL := fmt.Sprintf("%s/invite/%s", t.baseURL, evt.InviteCode)

	body := inviteEmailHTML(lang, evt.InviterName, evt.ProjectName, evt.ProjectKey, evt.Role, inviteURL)

	if err := t.sender.Send(ctx, evt.InviteeEmail, subject, body); err != nil {
		return fmt.Errorf("sending invite email: %w", err)
	}

	l.Info().Msg("invite email sent")
	return nil
}

func inviteEmailHTML(lang, inviterName, projectName, projectKey, role, inviteURL string) string {
	intro := i18n.T(lang, "email.invite.intro",
		"inviterName", inviterName,
		"projectName", projectName,
		"role", role)
	content := fmt.Sprintf(`<p>%s</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s</p>
    <p style="margin: 0; font-size: 18px; font-weight: 600;">%s</p>
  </div>`, intro, projectKey, projectName)
	return emailHTML(lang, "email.invite.cta", inviteURL, "email.invite.footer", content)
}
