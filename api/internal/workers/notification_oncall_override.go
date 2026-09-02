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
	users    userRepository
	settings userSettingRepository
	sender   emailSender
	urls     *URLBuilder
	logger   zerolog.Logger
}

// NewNotificationOncallOverrideCreatedTask creates the task.
func NewNotificationOncallOverrideCreatedTask(
	users userRepository,
	settings userSettingRepository,
	sender emailSender,
	urls *URLBuilder,
	logger zerolog.Logger,
) *NotificationOncallOverrideCreatedTask {
	return &NotificationOncallOverrideCreatedTask{
		users:    users,
		settings: settings,
		sender:   sender,
		urls:     urls,
		logger:   logger,
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

	oncallURL := t.urls.OncallTab(ctx, evt.ProjectID, evt.ProjectKey, evt.TeamID)

	// Notify override user (the person taking over)
	overrideUser, err := t.users.GetByID(ctx, evt.OverrideUserID)
	if err != nil {
		return fmt.Errorf("loading override user: %w", err)
	}

	lang := getUserLanguage(ctx, t.settings, evt.OverrideUserID)
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

		coveredLang := getUserLanguage(ctx, t.settings, evt.ScheduledUser)
		coveredSubject := i18n.T(coveredLang, "email.oncall.override.covered.subject", "projectKey", evt.ProjectKey, "teamName", evt.TeamName)
		coveredBody := oncallOverrideCoveredEmailHTML(coveredLang, evt.TeamName, evt.ProjectKey, evt.ProjectName, overrideUser.DisplayName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"), oncallURL)

		if err := t.sender.Send(ctx, scheduledUser.Email, coveredSubject, coveredBody); err != nil {
			return fmt.Errorf("sending override covered email: %w", err)
		}
		t.logger.Info().Str("to", scheduledUser.Email).Msg("oncall override covered notification sent")
	}

	return nil
}

// NotificationOncallOverrideCancelledTask sends emails when an on-call override is cancelled.
type NotificationOncallOverrideCancelledTask struct {
	users    userRepository
	settings userSettingRepository
	sender   emailSender
	urls     *URLBuilder
	logger   zerolog.Logger
}

// NewNotificationOncallOverrideCancelledTask creates the task.
func NewNotificationOncallOverrideCancelledTask(
	users userRepository,
	settings userSettingRepository,
	sender emailSender,
	urls *URLBuilder,
	logger zerolog.Logger,
) *NotificationOncallOverrideCancelledTask {
	return &NotificationOncallOverrideCancelledTask{
		users:    users,
		settings: settings,
		sender:   sender,
		urls:     urls,
		logger:   logger,
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

	// Notify override user that their override was cancelled
	overrideUser, err := t.users.GetByID(ctx, evt.OverrideUserID)
	if err != nil {
		return fmt.Errorf("loading override user: %w", err)
	}

	oncallURL := t.urls.OncallTab(ctx, evt.ProjectID, evt.ProjectKey, evt.TeamID)
	lang := getUserLanguage(ctx, t.settings, evt.OverrideUserID)
	subject := i18n.T(lang, "email.oncall.override.cancelled.subject", "projectKey", evt.ProjectKey, "teamName", evt.TeamName)
	body := oncallOverrideCancelledEmailHTML(lang, evt.TeamName, evt.ProjectKey, evt.ProjectName, evt.StartAt.Format("2006-01-02 15:04 MST"), evt.EndAt.Format("2006-01-02 15:04 MST"), oncallURL)

	if err := t.sender.Send(ctx, overrideUser.Email, subject, body); err != nil {
		return fmt.Errorf("sending override cancelled email: %w", err)
	}
	t.logger.Info().Str("to", overrideUser.Email).Msg("oncall override cancelled notification sent")

	return nil
}

func oncallOverrideCreatedEmailHTML(lang, teamName, projectKey, projectName, startAt, endAt, oncallURL string) string {
	intro := i18n.T(lang, "email.oncall.override.created.intro",
		"teamName", teamName,
		"projectBadge", projectKeyBadge(projectKey),
		"projectName", projectName)
	period := i18n.T(lang, "email.oncall.override.created.period", "startAt", startAt, "endAt", endAt)
	note := i18n.T(lang, "email.oncall.override.created.note")
	content := fmt.Sprintf("<p>%s</p>\n  <p>%s</p>\n  <p>%s</p>", intro, period, note)
	return emailHTML(lang, "email.oncall.cta", oncallURL, "email.oncall.override.footer", content)
}

func oncallOverrideCoveredEmailHTML(lang, teamName, projectKey, projectName, coveringUser, startAt, endAt, oncallURL string) string {
	intro := i18n.T(lang, "email.oncall.override.covered.intro",
		"teamName", teamName,
		"projectBadge", projectKeyBadge(projectKey),
		"projectName", projectName,
		"coveringUser", coveringUser)
	period := i18n.T(lang, "email.oncall.override.covered.period", "startAt", startAt, "endAt", endAt)
	content := fmt.Sprintf("<p>%s</p>\n  <p>%s</p>", intro, period)
	return emailHTML(lang, "email.oncall.cta", oncallURL, "email.oncall.override.footer", content)
}

func oncallOverrideCancelledEmailHTML(lang, teamName, projectKey, projectName, startAt, endAt, oncallURL string) string {
	intro := i18n.T(lang, "email.oncall.override.cancelled.intro",
		"teamName", teamName,
		"projectBadge", projectKeyBadge(projectKey),
		"projectName", projectName)
	period := i18n.T(lang, "email.oncall.override.cancelled.period", "startAt", startAt, "endAt", endAt)
	note := i18n.T(lang, "email.oncall.override.cancelled.note")
	content := fmt.Sprintf("<p>%s</p>\n  <p>%s</p>\n  <p>%s</p>", intro, period, note)
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
