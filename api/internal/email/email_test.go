package email

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/model"
)

type mockSettingsReader struct {
	settings map[string]*model.SystemSetting
	err      error
}

func (m *mockSettingsReader) Get(_ context.Context, key string) (*model.SystemSetting, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.settings[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func newTestEncryptor(t *testing.T) *crypto.Encryptor {
	t.Helper()
	key, err := crypto.DeriveKey("test-jwt-secret-that-is-long-enough")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func TestLoadConfigDecryptsPassword(t *testing.T) {
	enc := newTestEncryptor(t)

	encrypted, err := enc.Encrypt("my-secret-pass")
	if err != nil {
		t.Fatal(err)
	}

	cfg := model.SMTPConfig{
		Enabled:     true,
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		Username:    "user",
		Password:    encrypted,
		Encryption:  "starttls",
		FromAddress: "test@example.com",
	}

	raw, _ := json.Marshal(cfg)
	reader := &mockSettingsReader{
		settings: map[string]*model.SystemSetting{
			model.SettingSMTPConfig: {Key: model.SettingSMTPConfig, Value: raw},
		},
	}

	sender := NewSender(enc, reader)
	loaded, err := sender.loadConfig(context.Background())
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if loaded.Password != "my-secret-pass" {
		t.Errorf("got password %q, want %q", loaded.Password, "my-secret-pass")
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	enc := newTestEncryptor(t)
	reader := &mockSettingsReader{settings: map[string]*model.SystemSetting{}}

	sender := NewSender(enc, reader)
	_, err := sender.loadConfig(context.Background())
	if err == nil {
		t.Error("expected error when smtp config not found")
	}
}

func TestSendReturnsErrorWhenDisabled(t *testing.T) {
	enc := newTestEncryptor(t)

	cfg := model.SMTPConfig{Enabled: false}
	raw, _ := json.Marshal(cfg)
	reader := &mockSettingsReader{
		settings: map[string]*model.SystemSetting{
			model.SettingSMTPConfig: {Key: model.SettingSMTPConfig, Value: raw},
		},
	}

	sender := NewSender(enc, reader)
	err := sender.Send(context.Background(), "to@test.com", "subj", "body")
	if err == nil {
		t.Error("expected error when SMTP is disabled")
	}
}

func TestBuildMessage(t *testing.T) {
	msg := buildMessage("Taskwondo", "noreply@example.com", "user@test.com", "Test Subject", "<p>Hello</p>", "<test-id@example.com>")
	s := string(msg)

	if !contains(s, "From: Taskwondo <noreply@example.com>") {
		t.Error("missing or incorrect From header")
	}
	if !contains(s, "To: user@test.com") {
		t.Error("missing To header")
	}
	if !contains(s, "Subject: Test Subject") {
		t.Error("missing Subject header")
	}
	if !contains(s, "Content-Type: text/html") {
		t.Error("missing Content-Type header")
	}
	if !contains(s, "<p>Hello</p>") {
		t.Error("missing body")
	}
}

func TestBuildMessageNoFromName(t *testing.T) {
	msg := buildMessage("", "noreply@example.com", "user@test.com", "Sub", "body", "<test-id@example.com>")
	s := string(msg)
	if !contains(s, "From: noreply@example.com\r\n") {
		t.Errorf("expected bare email in From, got: %s", s)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
