// Package mailer sends transactional email over a self-hosted SMTP server
// (no third-party email service). It is a no-op when disabled, so callers can
// always call Send without guarding.
package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Config holds SMTP settings.
type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string
	StartTLS bool
}

// Mailer sends email via SMTP.
type Mailer struct {
	cfg Config
	log logger.Logger
}

// New builds a Mailer.
func New(cfg Config, log logger.Logger) *Mailer {
	return &Mailer{cfg: cfg, log: log.Named("mailer")}
}

// Enabled reports whether email delivery is configured.
func (m *Mailer) Enabled() bool { return m.cfg.Enabled && m.cfg.Host != "" }

// Send delivers an HTML email. Returns nil (logging) when disabled.
func (m *Mailer) Send(to, subject, htmlBody string) error {
	if !m.Enabled() {
		m.log.Debug("mail disabled; skipping send", "to", to, "subject", subject)
		return nil
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	msg := m.build(to, subject, htmlBody)

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	if !m.cfg.StartTLS {
		// Plain SMTP (e.g. internal relay) — smtp.SendMail STARTTLSes if offered.
		if err := smtp.SendMail(addr, auth, m.senderAddr(), []string{to}, []byte(msg)); err != nil {
			m.log.Error("smtp send failed", "to", to, "err", err)
			return err
		}
		return nil
	}
	return m.sendStartTLS(addr, auth, to, msg)
}

// sendStartTLS performs an explicit STARTTLS handshake before sending.
func (m *Mailer) sendStartTLS(addr string, auth smtp.Auth, to, msg string) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(m.senderAddr()); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func (m *Mailer) build(to, subject, htmlBody string) string {
	var b strings.Builder
	b.WriteString("From: " + m.cfg.From + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	b.WriteString(htmlBody)
	return b.String()
}

// senderAddr extracts the bare address from a "Name <addr>" From header.
func (m *Mailer) senderAddr() string {
	if i := strings.LastIndex(m.cfg.From, "<"); i >= 0 {
		return strings.TrimSuffix(m.cfg.From[i+1:], ">")
	}
	return m.cfg.From
}
