package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/i18n"
	"github.com/marcoshack/taskwondo/internal/model"
)

// NotificationOncallOverrideCreatedTask sends emails when an on-call override is created.
type NotificationOncallOverrideCreatedTask struct {
	users  userRepository
	sender emailSender
	logger zerolog.Logger
}

// NewNotificationOncallOverrideCreatedTask creates the task.
func NewNotificationOncallOverrideCreatedTask(
	users userRepository,
	sender emailSender,
	logger zerolog.Logger,
) *NotificationOncallOverrideCreatedTask {
	return &NotificationOncallOverrideCreatedTask{
		users:  users,
		sender: sender,
		logger: logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationOncallOverrideCreatedTask) Name() string {
	return "oncall.override.created"
}

// Execute processes an on-call override created event.
func (t *NotificationOncallOverrideCreatedTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.OncallOverrideCreatedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid oncall override created event payload")
		return nil
	}

	lang := "en"

	// Notify override user (the person taking over)
	overrideUser, err := t.users.GetByID(ctx, evt.OverrideUserID)
	if err != nil {
		return fmt.Errorf("loading override user: %w", err)
	}

	subject := i18n.T(lang, "email.oncall.override.created.subject", "teamName", evt.TeamName)
	body := oncallOverrideCreatedEmailHTML(evt.TeamName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"))

	if err := t.sender.Send(ctx, overrideUser.Email, subject, body); err != nil {
		return fmt.Errorf("sending override created email: %w", err)
	}
	t.logger.Info().Str("to", overrideUser.Email).Msg("oncall override created notification sent")

	// Notify scheduled user (the person being covered) if different
	if evt.ScheduledUser != evt.OverrideUserID {
		scheduledUser, err := t.users.GetByID(ctx, evt.ScheduledUser)
		if err != nil {
			return fmt.Errorf("loading scheduled user: %w", err)
		}

		coveredSubject := i18n.T(lang, "email.oncall.override.covered.subject", "teamName", evt.TeamName)
		coveredBody := oncallOverrideCoveredEmailHTML(evt.TeamName, overrideUser.DisplayName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"))

		if err := t.sender.Send(ctx, scheduledUser.Email, coveredSubject, coveredBody); err != nil {
			return fmt.Errorf("sending override covered email: %w", err)
		}
		t.logger.Info().Str("to", scheduledUser.Email).Msg("oncall override covered notification sent")
	}

	return nil
}

// NotificationOncallOverrideCancelledTask sends emails when an on-call override is cancelled.
type NotificationOncallOverrideCancelledTask struct {
	users  userRepository
	sender emailSender
	logger zerolog.Logger
}

// NewNotificationOncallOverrideCancelledTask creates the task.
func NewNotificationOncallOverrideCancelledTask(
	users userRepository,
	sender emailSender,
	logger zerolog.Logger,
) *NotificationOncallOverrideCancelledTask {
	return &NotificationOncallOverrideCancelledTask{
		users:  users,
		sender: sender,
		logger: logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationOncallOverrideCancelledTask) Name() string {
	return "oncall.override.cancelled"
}

// Execute processes an on-call override cancelled event.
func (t *NotificationOncallOverrideCancelledTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.OncallOverrideCancelledEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		t.logger.Error().Err(err).Msg("invalid oncall override cancelled event payload")
		return nil
	}

	lang := "en"

	// Notify override user that their override was cancelled
	overrideUser, err := t.users.GetByID(ctx, evt.OverrideUserID)
	if err != nil {
		return fmt.Errorf("loading override user: %w", err)
	}

	subject := i18n.T(lang, "email.oncall.override.cancelled.subject", "teamName", evt.TeamName)
	body := oncallOverrideCancelledEmailHTML(evt.TeamName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"))

	if err := t.sender.Send(ctx, overrideUser.Email, subject, body); err != nil {
		return fmt.Errorf("sending override cancelled email: %w", err)
	}
	t.logger.Info().Str("to", overrideUser.Email).Msg("oncall override cancelled notification sent")

	return nil
}

func oncallOverrideCreatedEmailHTML(teamName, startAt, endAt string) string {
	content := fmt.Sprintf(`<p>You have been assigned an on-call override for <strong>%s</strong>.</p>
  <p>Your override period: <strong>%s</strong> to <strong>%s</strong>.</p>
  <p>Please make sure you are available to respond to any incoming issues during this period.</p>`, teamName, startAt, endAt)
	return wrapOncallEmail(content)
}

func oncallOverrideCoveredEmailHTML(teamName, coveringUser, startAt, endAt string) string {
	content := fmt.Sprintf(`<p>Your on-call shift for <strong>%s</strong> has been covered by <strong>%s</strong>.</p>
  <p>Override period: <strong>%s</strong> to <strong>%s</strong>.</p>`, teamName, coveringUser, startAt, endAt)
	return wrapOncallEmail(content)
}

func oncallOverrideCancelledEmailHTML(teamName, startAt, endAt string) string {
	content := fmt.Sprintf(`<p>An on-call override for <strong>%s</strong> has been cancelled.</p>
  <p>The override was scheduled for: <strong>%s</strong> to <strong>%s</strong>.</p>
  <p>The regular on-call rotation schedule will apply for this period.</p>`, teamName, startAt, endAt)
	return wrapOncallEmail(content)
}

func wrapOncallEmail(content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  %s
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">This is an automated on-call override notification from Taskwondo.</p>
</body>
</html>`, content)
}
