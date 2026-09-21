package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/i18n"
	"github.com/marcoshack/taskwondo/internal/model"
)

// NotificationOncallRotationTask sends emails when an on-call rotation advances.
type NotificationOncallRotationTask struct {
	users    userRepository
	settings userSettingRepository
	sender   emailSender
	urls     *URLBuilder
	logger   zerolog.Logger
}

// NewNotificationOncallRotationTask creates the task.
func NewNotificationOncallRotationTask(
	users userRepository,
	settings userSettingRepository,
	sender emailSender,
	urls *URLBuilder,
	logger zerolog.Logger,
) *NotificationOncallRotationTask {
	return &NotificationOncallRotationTask{
		users:    users,
		settings: settings,
		sender:   sender,
		urls:     urls,
		logger:   logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationOncallRotationTask) Name() string {
	return "oncall.rotation.advanced"
}

// Execute processes an on-call rotation advanced event.
func (t *NotificationOncallRotationTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.OncallRotationAdvancedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid oncall rotation event payload")
		return nil
	}

	l := t.logger.With().
		Str("team_name", evt.TeamName).
		Str("old_user_id", evt.OldUserID.String()).
		Str("new_user_id", evt.NewUserID.String()).
		Logger()

	// Notify incoming on-call member
	newUser, err := t.users.GetByID(ctx, evt.NewUserID)
	if err != nil {
		return fmt.Errorf("loading new on-call user: %w", err)
	}

	oncallURL := t.urls.OncallTab(ctx, evt.ProjectID, evt.ProjectKey, evt.TeamID)

	lang := getUserLanguage(ctx, t.settings, evt.NewUserID)
	incomingSubject := i18n.T(lang, "email.oncall.incoming.subject", "projectKey", evt.ProjectKey, "teamName", evt.TeamName)
	incomingBody := oncallIncomingEmailHTML(lang, evt.TeamName, evt.ProjectKey, evt.ProjectName, oncallURL)

	if err := t.sender.Send(ctx, newUser.Email, incomingSubject, incomingBody); err != nil {
		return fmt.Errorf("sending incoming oncall email: %w", err)
	}
	l.Info().Str("to", newUser.Email).Msg("oncall incoming notification sent")

	// Notify outgoing on-call member (if different from incoming)
	if evt.OldUserID != evt.NewUserID {
		oldUser, err := t.users.GetByID(ctx, evt.OldUserID)
		if err != nil {
			return fmt.Errorf("loading old on-call user: %w", err)
		}

		outLang := getUserLanguage(ctx, t.settings, evt.OldUserID)
		outgoingSubject := i18n.T(outLang, "email.oncall.outgoing.subject", "projectKey", evt.ProjectKey, "teamName", evt.TeamName)
		outgoingBody := oncallOutgoingEmailHTML(outLang, evt.TeamName, evt.ProjectKey, evt.ProjectName, oncallURL)

		if err := t.sender.Send(ctx, oldUser.Email, outgoingSubject, outgoingBody); err != nil {
			return fmt.Errorf("sending outgoing oncall email: %w", err)
		}
		l.Info().Str("to", oldUser.Email).Msg("oncall outgoing notification sent")
	}

	return nil
}

func oncallIncomingEmailHTML(lang, teamName, projectKey, projectName, oncallURL string) string {
	intro := i18n.T(lang, "email.oncall.incoming.intro",
		"teamName", teamName,
		"projectBadge", projectKeyBadge(projectKey),
		"projectName", projectName)
	note := i18n.T(lang, "email.oncall.incoming.note")
	content := fmt.Sprintf("<p>%s</p>\n  <p>%s</p>", intro, note)
	return emailHTML(lang, "email.oncall.cta", oncallURL, "email.oncall.rotation.footer", content)
}

func oncallOutgoingEmailHTML(lang, teamName, projectKey, projectName, oncallURL string) string {
	intro := i18n.T(lang, "email.oncall.outgoing.intro",
		"teamName", teamName,
		"projectBadge", projectKeyBadge(projectKey),
		"projectName", projectName)
	note := i18n.T(lang, "email.oncall.outgoing.note")
	content := fmt.Sprintf("<p>%s</p>\n  <p>%s</p>", intro, note)
	return emailHTML(lang, "email.oncall.cta", oncallURL, "email.oncall.rotation.footer", content)
}
