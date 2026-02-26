package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/crypto"
	"github.com/marcoshack/taskwondo/internal/model"
)

// SettingsReader loads system settings by key.
type SettingsReader interface {
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
}

// Sender sends emails using SMTP configuration from system settings.
type Sender struct {
	encryptor *crypto.Encryptor
	settings  SettingsReader
}

// NewSender creates a new email Sender.
func NewSender(encryptor *crypto.Encryptor, settings SettingsReader) *Sender {
	return &Sender{encryptor: encryptor, settings: settings}
}

// loadConfig reads and decrypts the SMTP configuration from system settings.
func (s *Sender) loadConfig(ctx context.Context) (*model.SMTPConfig, error) {
	setting, err := s.settings.Get(ctx, model.SettingSMTPConfig)
	if err != nil {
		return nil, fmt.Errorf("loading smtp config: %w", err)
	}

	var cfg model.SMTPConfig
	if err := json.Unmarshal(setting.Value, &cfg); err != nil {
		return nil, fmt.Errorf("parsing smtp config: %w", err)
	}

	if cfg.Password != "" {
		decrypted, err := s.encryptor.Decrypt(cfg.Password)
		if err != nil {
			return nil, fmt.Errorf("decrypting smtp password: %w", err)
		}
		cfg.Password = decrypted
	}

	return &cfg, nil
}

// Send sends an email using the stored SMTP configuration.
func (s *Sender) Send(ctx context.Context, to, subject, htmlBody string) error {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}

	if !cfg.Enabled {
		return fmt.Errorf("smtp is not enabled")
	}

	return sendMail(ctx, cfg, to, subject, htmlBody)
}

// SendTest sends a test email to the given address.
func (s *Sender) SendTest(ctx context.Context, to string) error {
	log.Ctx(ctx).Info().Str("to", to).Msg("sending test email")
	return s.Send(ctx, to, "Taskwondo Test Email", testEmailHTML(to))
}

func testEmailHTML(to string) string {
	return fmt.Sprintf(`<html><body>
<h2>Taskwondo SMTP Test</h2>
<p>This is a test email sent to <strong>%s</strong> to verify your SMTP configuration.</p>
<p>If you received this email, your SMTP settings are working correctly.</p>
</body></html>`, to)
}

func sendMail(ctx context.Context, cfg *model.SMTPConfig, to, subject, htmlBody string) error {
	addr := net.JoinHostPort(cfg.SMTPHost, fmt.Sprintf("%d", cfg.SMTPPort))
	messageID := fmt.Sprintf("<%s@%s>", uuid.New().String(), cfg.SMTPHost)

	msg := buildMessage(cfg.FromName, cfg.FromAddress, to, subject, htmlBody, messageID)

	l := log.Ctx(ctx).With().
		Str("smtp_host", cfg.SMTPHost).
		Int("smtp_port", cfg.SMTPPort).
		Str("encryption", cfg.Encryption).
		Str("from", cfg.FromAddress).
		Str("to", to).
		Str("message_id", messageID).
		Logger()

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	var err error
	switch cfg.Encryption {
	case model.SMTPEncryptionTLS:
		err = sendWithImplicitTLS(&l, addr, cfg.SMTPHost, auth, cfg.FromAddress, to, msg)
	case model.SMTPEncryptionSTARTTLS:
		err = sendWithSTARTTLS(&l, addr, cfg.SMTPHost, auth, cfg.FromAddress, to, msg)
	case model.SMTPEncryptionNone:
		// For plaintext SMTP, skip auth — Go's PlainAuth refuses to send
		// credentials over unencrypted non-localhost connections.
		err = smtp.SendMail(addr, nil, cfg.FromAddress, []string{to}, msg)
		if err == nil {
			l.Info().Msg("email accepted by server")
		}
	default:
		return fmt.Errorf("unsupported encryption type: %s", cfg.Encryption)
	}

	if err != nil {
		l.Error().Err(err).Msg("email sending failed")
	}
	return err
}

func buildMessage(fromName, fromAddr, to, subject, htmlBody, messageID string) []byte {
	var from string
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddr)
	} else {
		from = fromAddr
	}

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("Date: " + time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700") + "\r\n")
	b.WriteString("Message-ID: " + messageID + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return []byte(b.String())
}

func smtpSendEnvelope(l *zerolog.Logger, c *smtp.Client, auth smtp.Auth, from, to string, msg []byte) error {
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}
	l.Debug().Msg("smtp auth successful")

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	l.Debug().Msg("smtp envelope accepted")

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("server rejected message data: %w", err)
	}

	// Read the QUIT response which may include queue info
	quitErr := c.Quit()

	l.Info().Msg("email accepted by server")

	return quitErr
}

func sendWithSTARTTLS(l *zerolog.Logger, addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("connecting to SMTP server: %w", err)
	}
	defer c.Close()
	l.Debug().Msg("smtp connected")

	tlsConfig := &tls.Config{ServerName: host}
	if err := c.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS: %w", err)
	}
	l.Debug().Msg("smtp STARTTLS established")

	return smtpSendEnvelope(l, c, auth, from, to, msg)
}

func sendWithImplicitTLS(l *zerolog.Logger, addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial: %w", err)
	}
	l.Debug().Msg("smtp TLS connected")

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("creating SMTP client: %w", err)
	}
	defer c.Close()

	return smtpSendEnvelope(l, c, auth, from, to, msg)
}
