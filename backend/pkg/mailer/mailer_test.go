package mailer

import (
	"strings"
	"testing"

	"sapphirebroking.com/sftp_service/pkg/logger"
)

func newTestMailer(cfg Config) *Mailer {
	return New(cfg, logger.NewNop())
}

func TestEnabled(t *testing.T) {
	if newTestMailer(Config{Enabled: false, Host: "x"}).Enabled() {
		t.Fatal("disabled should not be enabled")
	}
	if newTestMailer(Config{Enabled: true, Host: ""}).Enabled() {
		t.Fatal("no host should not be enabled")
	}
	if !newTestMailer(Config{Enabled: true, Host: "smtp"}).Enabled() {
		t.Fatal("enabled+host should be enabled")
	}
}

func TestSendDisabledIsNoop(t *testing.T) {
	if err := newTestMailer(Config{Enabled: false}).Send("a@b.com", "hi", "<p>hi</p>"); err != nil {
		t.Fatalf("disabled send should be nil, got %v", err)
	}
}

func TestSenderAddr(t *testing.T) {
	m := newTestMailer(Config{From: "Sapphire <no-reply@corp.com>"})
	if m.senderAddr() != "no-reply@corp.com" {
		t.Fatalf("got %q", m.senderAddr())
	}
	m2 := newTestMailer(Config{From: "plain@corp.com"})
	if m2.senderAddr() != "plain@corp.com" {
		t.Fatalf("got %q", m2.senderAddr())
	}
}

func TestBuildHeaders(t *testing.T) {
	m := newTestMailer(Config{From: "S <s@corp.com>"})
	msg := m.build("to@corp.com", "Subject Line", "<b>body</b>")
	for _, want := range []string{"From: S <s@corp.com>", "To: to@corp.com", "Subject: Subject Line", "text/html", "<b>body</b>"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q", want)
		}
	}
}
