package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/i18n"
)

// getUserLanguage returns the user's preferred language, defaulting to "en".
func getUserLanguage(ctx context.Context, settings userSettingRepository, userID uuid.UUID) string {
	setting, err := settings.Get(ctx, userID, nil, "language")
	if err != nil {
		return "en"
	}
	var lang string
	if err := json.Unmarshal(setting.Value, &lang); err != nil {
		return "en"
	}
	if lang == "" {
		return "en"
	}
	return lang
}

// emailHTML wraps content in the standard email layout with translated CTA and footer.
func emailHTML(lang, ctaKey, ctaURL, footerKey string, contentHTML string) string {
	cta := i18n.T(lang, ctaKey)
	footer := i18n.T(lang, footerKey)

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a1a; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-bottom: 3px solid #2563eb; padding-bottom: 16px; margin-bottom: 24px;">
    <h2 style="margin: 0; color: #2563eb;">Taskwondo</h2>
  </div>
  %s
  <p>
    <a href="%s" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 500;">%s</a>
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;">
  <p style="font-size: 12px; color: #94a3b8;">%s</p>
</body>
</html>`, contentHTML, ctaURL, cta, footer)
}

// itemCard renders the standard work item card used in most notification emails.
func itemCard(projectKey string, itemNumber int, title, detail string) string {
	detailHTML := ""
	if detail != "" {
		detailHTML = fmt.Sprintf(`<p style="margin: 0; font-size: 14px; color: #475569;">%s</p>`, detail)
	}
	return fmt.Sprintf(`<div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
    <p style="margin: 0 0 8px 0; font-size: 14px; color: #64748b;">%s-%d</p>
    <p style="margin: 0 0 %s 0; font-size: 18px; font-weight: 600;">%s</p>
    %s
  </div>`, projectKey, itemNumber, func() string {
		if detail != "" {
			return "12px"
		}
		return "0"
	}(), title, detailHTML)
}
