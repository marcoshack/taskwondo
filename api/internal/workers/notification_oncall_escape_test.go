package workers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/marcoshack/taskwondo/internal/model"
)

// Team names, project names and display names are user-controlled; each test
// value carries markup that must reach the email as text, never as HTML.
const (
	evilTeam    = `<script>alert("team")</script>`
	evilProject = `<img src=x onerror=alert(1)>`
	evilUser    = `<a href="https://evil.example/phish">Click me</a>`
)

// escapedForm is what html.EscapeString makes of each value above.
var escapedForm = map[string]string{
	evilTeam:    `&lt;script&gt;alert(&#34;team&#34;)&lt;/script&gt;`,
	evilProject: `&lt;img src=x onerror=alert(1)&gt;`,
	evilUser:    `&lt;a href=&#34;https://evil.example/phish&#34;&gt;Click me&lt;/a&gt;`,
}

// assertEscaped fails if any raw value appears in body, or if its escaped form
// is missing (proving the value is still rendered, just as text).
func assertEscaped(t *testing.T, body string, raw ...string) {
	t.Helper()
	for _, r := range raw {
		if strings.Contains(body, r) {
			t.Errorf("email body contains unescaped %q:\n%s", r, body)
		}
	}
	for _, r := range raw {
		if esc := escapedForm[r]; !strings.Contains(body, esc) {
			t.Errorf("email body missing escaped %q:\n%s", esc, body)
		}
	}
	// The template's own markup must survive escaping.
	if !strings.Contains(body, "<strong>") || !strings.Contains(body, `<span style="display: inline-block;`) {
		t.Errorf("expected trusted template markup to remain HTML:\n%s", body)
	}
}

func TestNotificationOncallOverride_EscapesUserValues(t *testing.T) {
	overrideUserID := uuid.New()
	scheduledUserID := uuid.New()
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		overrideUserID:  {ID: overrideUserID, Email: "override@example.com", DisplayName: evilUser},
		scheduledUserID: {ID: scheduledUserID, Email: "scheduled@example.com", DisplayName: "Sam"},
	}}
	start := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	end := start.Add(8 * time.Hour)

	t.Run("created and covered", func(t *testing.T) {
		sender := &mockEmailSender{}
		task := &NotificationOncallOverrideCreatedTask{users: users, sender: sender, urls: newTestURLBuilder(), logger: zerolog.Nop()}
		payload, _ := json.Marshal(model.OncallOverrideCreatedEvent{
			TeamID: uuid.New(), TeamName: evilTeam,
			ProjectID: uuid.New(), ProjectKey: "PAY", ProjectName: evilProject,
			OverrideUserID: overrideUserID, ScheduledUser: scheduledUserID,
			StartAt: start, EndAt: end,
		})
		if err := task.Execute(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
		if len(sender.sent) != 2 {
			t.Fatalf("expected 2 emails, got %d", len(sender.sent))
		}
		assertEscaped(t, sender.sent[0].body, evilTeam, evilProject)
		// The covered email names the covering user.
		assertEscaped(t, sender.sent[1].body, evilTeam, evilProject, evilUser)
	})

	t.Run("cancelled", func(t *testing.T) {
		sender := &mockEmailSender{}
		task := &NotificationOncallOverrideCancelledTask{users: users, sender: sender, urls: newTestURLBuilder(), logger: zerolog.Nop()}
		payload, _ := json.Marshal(model.OncallOverrideCancelledEvent{
			TeamID: uuid.New(), TeamName: evilTeam,
			ProjectID: uuid.New(), ProjectKey: "PAY", ProjectName: evilProject,
			OverrideUserID: overrideUserID, ScheduledUser: scheduledUserID,
			StartAt: start, EndAt: end,
		})
		if err := task.Execute(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
		if len(sender.sent) != 1 {
			t.Fatalf("expected 1 email, got %d", len(sender.sent))
		}
		assertEscaped(t, sender.sent[0].body, evilTeam, evilProject)
	})
}

func TestNotificationOncallRotation_EscapesUserValues(t *testing.T) {
	oldUserID := uuid.New()
	newUserID := uuid.New()
	users := &mockUserRepo{users: map[uuid.UUID]*model.User{
		oldUserID: {ID: oldUserID, Email: "old@example.com", DisplayName: evilUser},
		newUserID: {ID: newUserID, Email: "new@example.com", DisplayName: evilUser},
	}}
	sender := &mockEmailSender{}
	task := &NotificationOncallRotationTask{users: users, sender: sender, urls: newTestURLBuilder(), logger: zerolog.Nop()}

	payload, _ := json.Marshal(model.OncallRotationAdvancedEvent{
		TeamID: uuid.New(), TeamName: evilTeam,
		ProjectID: uuid.New(), ProjectKey: "ENG", ProjectName: evilProject,
		OldUserID: oldUserID, NewUserID: newUserID,
	})
	if err := task.Execute(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(sender.sent))
	}
	// One email per template: incoming to the new on-call user, outgoing to the
	// old one. Index by recipient so a mix-up of templates fails.
	byTo := map[string]sentEmail{}
	for _, e := range sender.sent {
		byTo[e.to] = e
	}
	for to, phrase := range map[string]string{
		"new@example.com": "You are now on-call",
		"old@example.com": "has ended",
	} {
		e, ok := byTo[to]
		if !ok {
			t.Fatalf("expected an email to %s, got %+v", to, sender.sent)
		}
		if !strings.Contains(e.body, phrase) {
			t.Errorf("email to %s should use the template containing %q:\n%s", to, phrase, e.body)
		}
		assertEscaped(t, e.body, evilTeam, evilProject)
		// The rotation templates do not name users today; if one starts to,
		// the display name must still arrive escaped.
		if strings.Contains(e.body, evilUser) {
			t.Errorf("email to %s contains unescaped display name:\n%s", to, e.body)
		}
	}
}

func TestProjectKeyBadge_EscapesKey(t *testing.T) {
	got := projectKeyBadge(`<b>"PAY"</b>`)
	if strings.Contains(got, "<b>") {
		t.Fatalf("badge contains unescaped key: %s", got)
	}
	if !strings.Contains(got, `&lt;b&gt;&#34;PAY&#34;&lt;/b&gt;</span>`) {
		t.Fatalf("badge missing escaped key: %s", got)
	}
	if !strings.HasPrefix(got, `<span style=`) {
		t.Fatalf("badge markup itself must stay HTML: %s", got)
	}
	if projectKeyBadge("") != "" {
		t.Fatal("expected empty badge for empty key")
	}
}
