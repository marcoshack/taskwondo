package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/i18n"
	"github.com/marcoshack/taskwondo/internal/model"
)

// NotificationNamespaceInviteEmailTask sends an email invite to a user inviting
// them to join a namespace.
type NotificationNamespaceInviteEmailTask struct {
	sender emailSender
	urls   *URLBuilder
	logger zerolog.Logger
}

// NewNotificationNamespaceInviteEmailTask creates the task.
func NewNotificationNamespaceInviteEmailTask(
	sender emailSender,
	urls *URLBuilder,
	logger zerolog.Logger,
) *NotificationNamespaceInviteEmailTask {
	return &NotificationNamespaceInviteEmailTask{
		sender: sender,
		urls:   urls,
		logger: logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationNamespaceInviteEmailTask) Name() string {
	return "notification.namespace_invite_email"
}

// Execute processes a namespace invite email event.
func (t *NotificationNamespaceInviteEmailTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.NamespaceInviteEmailEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid namespace invite email event payload")
		return nil
	}

	l := t.logger.With().
		Str("namespace_slug", evt.NamespaceSlug).
		Str("invitee_email", evt.InviteeEmail).
		Logger()

	lang := "en"

	subject := i18n.T(lang, "email.namespace_invite.subject",
		"namespaceName", evt.NamespaceDisplayName)

	inviteURL := t.urls.Invite(evt.InviteCode)

	body := namespaceInviteEmailHTML(lang, evt.InviterName, evt.NamespaceDisplayName, evt.NamespaceSlug, evt.Role, inviteURL)

	if err := t.sender.Send(ctx, evt.InviteeEmail, subject, body); err != nil {
		return fmt.Errorf("sending namespace invite email: %w", err)
	}

	l.Info().Msg("namespace invite email sent")
	return nil
}

func namespaceInviteEmailHTML(lang, inviterName, namespaceName, namespaceSlug, role, inviteURL string) string {
	intro := i18n.T(lang, "email.namespace_invite.intro",
		"inviterName", inviterName,
		"namespaceName", namespaceName,
		"role", role)
	content := fmt.Sprintf(`<p>%s</p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s</p>
    <p style="margin: 0; font-size: 18px; font-weight: 600;">%s</p>
  </div>`, intro, namespaceSlug, namespaceName)
	return emailHTML(lang, "email.namespace_invite.cta", inviteURL, "email.namespace_invite.footer", content)
}
