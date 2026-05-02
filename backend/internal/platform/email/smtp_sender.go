package email

import (
	"context"
	"fmt"
	"net/smtp"

	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
)

type SMTPSender struct {
	cfg platformconfig.SMTPConfig
}

func NewSMTPSender(cfg platformconfig.SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) SendBuddyInvite(_ context.Context, p BuddyInviteParams) error {
	subject := fmt.Sprintf("Вас пригласили в accountability loop — «%s»", p.GoalTitle)
	body := fmt.Sprintf(
		"%s создал цель «%s» и выбрал вас как accountability buddy.\r\n\r\nЧтобы принять приглашение, перейдите по ссылке:\r\n%s\r\n\r\nСсылка действительна 7 дней.",
		p.OwnerName, p.GoalTitle, p.InviteURL,
	)

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.From, p.To, subject, body,
	))

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	return smtp.SendMail(addr, auth, s.cfg.From, []string{p.To}, msg)
}
