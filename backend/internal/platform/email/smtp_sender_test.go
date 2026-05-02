package email

import (
	"strings"
	"testing"

	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
)

func TestBuildBuddyInviteMessage_UsesMultipartRussianCopy(t *testing.T) {
	sender := NewSMTPSender(platformconfig.SMTPConfig{From: "noreply@example.com"})

	msg := sender.buildBuddyInviteMessage(BuddyInviteParams{
		To:        "partner@example.com",
		OwnerName: "Артём",
		GoalTitle: "Запустить новый лендинг",
		InviteURL: "https://example.com/invites/abc",
	})

	message := string(msg)

	if !strings.Contains(message, "Subject: Вас пригласили присоединиться к цели") {
		t.Fatalf("expected russian subject, got %q", message)
	}
	if !strings.Contains(message, "Content-Type: multipart/alternative;") {
		t.Fatalf("expected multipart message, got %q", message)
	}
	if !strings.Contains(message, "text/plain; charset=UTF-8") {
		t.Fatalf("expected plain text part, got %q", message)
	}
	if !strings.Contains(message, "text/html; charset=UTF-8") {
		t.Fatalf("expected html part, got %q", message)
	}
	if !strings.Contains(message, "Артём приглашает вас присоединиться к цели «Запустить новый лендинг»") {
		t.Fatalf("expected russian invite copy, got %q", message)
	}
	if !strings.Contains(message, "https://example.com/invites/abc") {
		t.Fatalf("expected invite url, got %q", message)
	}
	if strings.Contains(message, "accountability") || strings.Contains(message, "buddy") {
		t.Fatalf("expected no mixed english terminology, got %q", message)
	}
}
