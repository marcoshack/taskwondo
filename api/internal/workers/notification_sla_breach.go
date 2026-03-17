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

// slaBreachEscalationRepository loads escalation list details for notification sending.
type slaBreachEscalationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.EscalationList, error)
}

// slaBreachNotificationRepository records sent notifications.
type slaBreachNotificationRepository interface {
	RecordSent(ctx context.Context, workItemID uuid.UUID, statusName string, level, thresholdPct int) error
}

// NotificationSLABreachTask sends email notifications for SLA breach events.
type NotificationSLABreachTask struct {
	escalations   slaBreachEscalationRepository
	notifications slaBreachNotificationRepository
	settings      userSettingRepository
	sender        emailSender
	baseURL       string
	logger        zerolog.Logger
}

// NewNotificationSLABreachTask creates the task.
func NewNotificationSLABreachTask(
	escalations slaBreachEscalationRepository,
	notifications slaBreachNotificationRepository,
	settings userSettingRepository,
	sender emailSender,
	baseURL string,
	logger zerolog.Logger,
) *NotificationSLABreachTask {
	return &NotificationSLABreachTask{
		escalations:   escalations,
		notifications: notifications,
		settings:      settings,
		sender:        sender,
		baseURL:       baseURL,
		logger:        logger,
	}
}

// Name returns the task name used as the NATS subject suffix.
func (t *NotificationSLABreachTask) Name() string {
	return "notification.sla_breach"
}

// Execute processes an SLA breach event.
func (t *NotificationSLABreachTask) Execute(ctx context.Context, payload []byte) error {
	var evt model.SLABreachEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		// Bad payload — no point retrying
		t.logger.Error().Err(err).Msg("invalid SLA breach event payload")
		return nil
	}

	l := t.logger.With().
		Str("project_key", evt.ProjectKey).
		Int("item_number", evt.ItemNumber).
		Int("escalation_level", evt.EscalationLevel).
		Int("threshold_pct", evt.ThresholdPct).
		Logger()

	// Load escalation list to get recipients for this level
	escList, err := t.escalations.GetByID(ctx, evt.EscalationListID)
	if err != nil {
		return fmt.Errorf("loading escalation list: %w", err)
	}

	// Find the matching level (by position index: level 1 = index 0)
	if evt.EscalationLevel < 1 || evt.EscalationLevel > len(escList.Levels) {
		l.Warn().
			Int("levels_count", len(escList.Levels)).
			Msg("escalation level out of range, skipping")
		return nil
	}
	level := escList.Levels[evt.EscalationLevel-1]

	if len(level.Users) == 0 {
		l.Debug().Msg("no recipients for escalation level")
		return nil
	}

	// Send email to each recipient who has SLA breach notifications enabled
	emailsSent := 0
	for _, user := range level.Users {
		if !t.isSLABreachEnabled(ctx, user.UserID, evt.ProjectID) {
			l.Debug().Str("user_id", user.UserID.String()).Msg("SLA breach notification disabled for user")
			continue
		}

		lang := getUserLanguage(ctx, t.settings, user.UserID)

		subject := i18n.T(lang, "email.sla_breach.subject",
			"projectKey", evt.ProjectKey,
			"itemNumber", fmt.Sprintf("%d", evt.ItemNumber),
			"title", evt.Title,
			"level", fmt.Sprintf("%d", evt.EscalationLevel))

		itemURL := fmt.Sprintf("%s/d/projects/%s/items/%d",
			t.baseURL, evt.ProjectKey, evt.ItemNumber)

		body := slaBreachEmailHTML(lang, evt, itemURL)

		if err := t.sender.Send(ctx, user.Email, subject, body); err != nil {
			l.Error().Err(err).Str("to", user.Email).Msg("failed to send SLA breach notification")
			continue
		}

		emailsSent++
		l.Info().Str("to", user.Email).Msg("SLA breach notification sent")
	}

	// Record that this notification was sent (for deduplication)
	if emailsSent > 0 {
		if err := t.notifications.RecordSent(ctx, evt.WorkItemID, evt.StatusName, evt.EscalationLevel, evt.ThresholdPct); err != nil {
			l.Error().Err(err).Msg("failed to record SLA notification sent")
			return fmt.Errorf("recording notification sent: %w", err)
		}
	}

	return nil
}

// isSLABreachEnabled checks whether the user has SLA breach notifications enabled
// for the given project. Default is true (enabled).
func (t *NotificationSLABreachTask) isSLABreachEnabled(ctx context.Context, userID, projectID uuid.UUID) bool {
	setting, err := t.settings.Get(ctx, userID, &projectID, "notifications")
	if err != nil {
		// No setting found — default to enabled for SLA breach
		return true
	}

	prefs := model.DefaultNotificationPreferences()
	if err := json.Unmarshal(setting.Value, &prefs); err != nil {
		t.logger.Warn().Err(err).Msg("invalid notification preferences, using defaults")
		return true
	}

	return prefs.SLABreach
}

// slaBreachEmailHTML generates the HTML email body for an SLA breach notification.
func slaBreachEmailHTML(lang string, evt model.SLABreachEvent, itemURL string) string {
	intro := i18n.T(lang, "email.sla_breach.intro")

	slaStatus := i18n.T(lang, "email.sla_breach.sla_status",
		"percentage", fmt.Sprintf("%d", evt.SLAPercentage))

	elapsed := i18n.T(lang, "email.sla_breach.elapsed",
		"elapsed", formatDuration(evt.ElapsedSeconds),
		"target", formatDuration(evt.TargetSeconds))

	levelInfo := i18n.T(lang, "email.sla_breach.level",
		"level", fmt.Sprintf("%d", evt.EscalationLevel))

	statusInfo := i18n.T(lang, "email.sla_breach.status",
		"statusName", evt.StatusName)

	// Build the SLA detail card
	detailHTML := fmt.Sprintf(`<p style="margin: 4px 0; font-size: 14px; color: #475569;">%s</p>
    <p style="margin: 4px 0; font-size: 14px; color: #475569;">%s</p>
    <p style="margin: 4px 0; font-size: 14px; color: #475569;">%s</p>
    <p style="margin: 4px 0; font-size: 14px; color: #475569;">%s</p>`,
		slaStatus, elapsed, statusInfo, levelInfo)

	// SLA percentage bar color
	barColor := "#f59e0b" // warning (orange/amber)
	if evt.SLAPercentage >= 100 {
		barColor = "#ef4444" // breached (red)
	}
	barWidth := evt.SLAPercentage
	if barWidth > 100 {
		barWidth = 100
	}

	progressBar := fmt.Sprintf(`<div style="background: #e2e8f0; border-radius: 4px; height: 8px; margin: 12px 0;">
    <div style="background: %s; border-radius: 4px; height: 8px; width: %d%%;"></div>
  </div>`, barColor, barWidth)

	card := fmt.Sprintf(`<div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s-%d</p>
    <p style="margin: 0 0 12px 0; font-size: 18px; font-weight: 600;">%s</p>
    %s
    %s
  </div>`, evt.ProjectKey, evt.ItemNumber, evt.Title, progressBar, detailHTML)

	content := fmt.Sprintf("<p>%s</p>\n  %s", intro, card)
	return emailHTML(lang, "email.sla_breach.cta", itemURL, "email.sla_breach.footer", content)
}
