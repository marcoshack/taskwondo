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

// NotificationOncallOverrideCreatedTask sends emails when an on-call override is created.
type NotificationOncallOverrideCreatedTask struct {
	users   userRepository
	sender  emailSender
	baseURL string
	logger  zerolog.Logger
}

// NewNotificationOncallOverrideCreatedTask creates the task.
func NewNotificationOncallOverrideCreatedTask(
	users userRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationOncallOverrideCreatedTask {
	return &NotificationOncallOverrideCreatedTask{
		users:   users,
		sender:  sender,
		baseURL: baseURL,
		logger:  logger,
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
	oncallURL := oncallTabURL(t.baseURL, evt.ProjectKey, evt.TeamID)

	// Notify override user (the person taking over)
	overrideUser, err := t.users.GetByID(ctx, evt.OverrideUserID)
	if err != nil {
		return fmt.Errorf("loading override user: %w", err)
	}

	subject := i18n.T(lang, "email.oncall.override.created.subject", "projectKey", evt.ProjectKey, "teamName", evt.TeamName)
	body := oncallOverrideCreatedEmailHTML(lang, evt.TeamName, evt.ProjectKey, evt.ProjectName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"), oncallURL)

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

		coveredSubject := i18n.T(lang, "email.oncall.override.covered.subject", "projectKey", evt.ProjectKey, "teamName", evt.TeamName)
		coveredBody := oncallOverrideCoveredEmailHTML(lang, evt.TeamName, evt.ProjectKey, evt.ProjectName, overrideUser.DisplayName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"), oncallURL)

		if err := t.sender.Send(ctx, scheduledUser.Email, coveredSubject, coveredBody); err != nil {
			return fmt.Errorf("sending override covered email: %w", err)
		}
		t.logger.Info().Str("to", scheduledUser.Email).Msg("oncall override covered notification sent")
	}

	return nil
}

// NotificationOncallOverrideCancelledTask sends emails when an on-call override is cancelled.
type NotificationOncallOverrideCancelledTask struct {
	users   userRepository
	sender  emailSender
	baseURL string
	logger  zerolog.Logger
}

// NewNotificationOncallOverrideCancelledTask creates the task.
func NewNotificationOncallOverrideCancelledTask(
	users userRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationOncallOverrideCancelledTask {
	return &NotificationOncallOverrideCancelledTask{
		users:   users,
		sender:  sender,
		baseURL: baseURL,
		logger:  logger,
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

	oncallURL := oncallTabURL(t.baseURL, evt.ProjectKey, evt.TeamID)
	subject := i18n.T(lang, "email.oncall.override.cancelled.subject", "projectKey", evt.ProjectKey, "teamName", evt.TeamName)
	body := oncallOverrideCancelledEmailHTML(lang, evt.TeamName, evt.ProjectKey, evt.ProjectName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"), oncallURL)

	if err := t.sender.Send(ctx, overrideUser.Email, subject, body); err != nil {
		return fmt.Errorf("sending override cancelled email: %w", err)
	}
	t.logger.Info().Str("to", overrideUser.Email).Msg("oncall override cancelled notification sent")

	return nil
}

func oncallOverrideCreatedEmailHTML(lang, teamName, projectKey, projectName, startAt, endAt, oncallURL string) string {
	content := fmt.Sprintf(`<p>You have been assigned an on-call override for <strong>%s</strong> in project %s <strong>%s</strong>.</p>
  <p>Your override period: <strong>%s</strong> to <strong>%s</strong>.</p>
  <p>Please make sure you are available to respond to any incoming issues during this period.</p>`, teamName, projectKeyBadge(projectKey), projectName, startAt, endAt)
	return emailHTML(lang, "email.oncall.cta", oncallURL, "email.oncall.override.footer", content)
}

func oncallOverrideCoveredEmailHTML(lang, teamName, projectKey, projectName, coveringUser, startAt, endAt, oncallURL string) string {
	content := fmt.Sprintf(`<p>Your on-call shift for <strong>%s</strong> in project %s <strong>%s</strong> has been covered by <strong>%s</strong>.</p>
  <p>Override period: <strong>%s</strong> to <strong>%s</strong>.</p>`, teamName, projectKeyBadge(projectKey), projectName, coveringUser, startAt, endAt)
	return emailHTML(lang, "email.oncall.cta", oncallURL, "email.oncall.override.footer", content)
}

func oncallOverrideCancelledEmailHTML(lang, teamName, projectKey, projectName, startAt, endAt, oncallURL string) string {
	content := fmt.Sprintf(`<p>An on-call override for <strong>%s</strong> in project %s <strong>%s</strong> has been cancelled.</p>
  <p>The override was scheduled for: <strong>%s</strong> to <strong>%s</strong>.</p>
  <p>The regular on-call rotation schedule will apply for this period.</p>`, teamName, projectKeyBadge(projectKey), projectName, startAt, endAt)
	return emailHTML(lang, "email.oncall.cta", oncallURL, "email.oncall.override.footer", content)
}

// projectKeyBadge renders a small inline HTML badge showing the project key,
// matching the visual style used for project keys in the frontend.
func projectKeyBadge(projectKey string) string {
	if projectKey == "" {
		return ""
	}
	return fmt.Sprintf(`<span style="display: inline-block; padding: 2px 8px; background-color: #e0e7ff; color: #3730a3; border-radius: 4px; font-size: 12px; font-weight: 600; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: 0.02em;">%s</span>`, projectKey)
}

// oncallTabURL builds the deep link to the on-call tab of a team's detail page.
func oncallTabURL(baseURL, projectKey string, teamID uuid.UUID) string {
	return fmt.Sprintf("%s/d/projects/%s/teams/%s?tab=oncall", baseURL, projectKey, teamID)
}
