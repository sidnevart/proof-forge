package email

import (
	"context"
	"fmt"
	"html"
	"net/smtp"
	"strings"

	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
)

type SMTPSender struct {
	cfg platformconfig.SMTPConfig
}

func NewSMTPSender(cfg platformconfig.SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) SendBuddyInvite(_ context.Context, p BuddyInviteParams) error {
	msg := s.buildBuddyInviteMessage(p)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	return smtp.SendMail(addr, auth, s.cfg.From, []string{p.To}, msg)
}

func (s *SMTPSender) buildBuddyInviteMessage(p BuddyInviteParams) []byte {
	subject := fmt.Sprintf("Вас пригласили присоединиться к цели — «%s»", p.GoalTitle)
	boundary := "proof-forge-invite-boundary"
	textBody := buildBuddyInviteText(p)
	htmlBody := buildBuddyInviteHTML(p)

	return []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n--%s\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n--%s--",
		s.cfg.From,
		p.To,
		subject,
		boundary,
		boundary,
		textBody,
		boundary,
		htmlBody,
		boundary,
	))
}

func buildBuddyInviteText(p BuddyInviteParams) string {
	return strings.Join([]string{
		fmt.Sprintf("%s приглашает вас присоединиться к цели «%s».", p.OwnerName, p.GoalTitle),
		"",
		"Что от вас ожидается:",
		"- принять приглашение;",
		"- смотреть подтверждения движения по цели;",
		"- подтверждать результат или возвращать материалы на доработку.",
		"",
		"Открыть приглашение:",
		p.InviteURL,
		"",
		"Ссылка действует 7 дней.",
	}, "\r\n")
}

func buildBuddyInviteHTML(p BuddyInviteParams) string {
	ownerName := html.EscapeString(p.OwnerName)
	goalTitle := html.EscapeString(p.GoalTitle)
	inviteURL := html.EscapeString(p.InviteURL)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
  <body style="margin:0;padding:24px;background:#0a1219;color:#edf2f8;font-family:Manrope,'IBM Plex Sans',sans-serif;">
    <div style="max-width:640px;margin:0 auto;border:1px solid rgba(186,202,221,0.18);border-radius:28px;background:#131a22;padding:32px;">
      <div style="display:inline-block;padding:8px 14px;border-radius:999px;border:1px solid rgba(186,202,221,0.18);color:#99a7b7;font-size:12px;font-weight:700;letter-spacing:0.08em;">
        ПРИГЛАШЕНИЕ К ЦЕЛИ
      </div>
      <h1 style="margin:20px 0 16px;font-size:34px;line-height:1.05;font-weight:800;">%s приглашает вас присоединиться к цели «%s»</h1>
      <p style="margin:0 0 24px;color:#99a7b7;font-size:16px;line-height:1.6;">
        Вы будете видеть подтверждения движения по цели и принимать решение: подтверждать результат, возвращать материалы на доработку или отклонять недостаточное подтверждение.
      </p>

      <div style="padding:20px;border-radius:20px;border:1px solid rgba(77,178,255,0.22);background:linear-gradient(180deg, rgba(77,178,255,0.12), rgba(255,255,255,0.03));margin-bottom:24px;">
        <div style="color:#708094;font-size:12px;font-weight:700;letter-spacing:0.08em;margin-bottom:10px;">ЦЕЛЬ</div>
        <div style="font-size:22px;font-weight:700;line-height:1.3;">%s</div>
      </div>

      <div style="padding:20px;border-radius:20px;border:1px solid rgba(186,202,221,0.18);background:rgba(255,255,255,0.03);margin-bottom:24px;">
        <div style="color:#708094;font-size:12px;font-weight:700;letter-spacing:0.08em;margin-bottom:12px;">ЧТО НУЖНО СДЕЛАТЬ</div>
        <ul style="margin:0;padding-left:18px;color:#99a7b7;line-height:1.6;">
          <li>Принять приглашение.</li>
          <li>Проверять подтверждения движения по цели.</li>
          <li>Выносить решение по результату.</li>
        </ul>
      </div>

      <a href="%s" style="display:inline-block;padding:14px 22px;border-radius:999px;background:linear-gradient(135deg,#4db2ff,#87d5ff);color:#081018;text-decoration:none;font-weight:800;">
        Открыть приглашение
      </a>

      <p style="margin:20px 0 0;color:#708094;font-size:13px;line-height:1.6;">
        Ссылка действует 7 дней. Если кнопка не работает, откройте ссылку вручную: %s
      </p>
    </div>
  </body>
</html>`, ownerName, goalTitle, goalTitle, inviteURL, inviteURL)
}
