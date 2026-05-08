package companion

import (
	"fmt"
	"strings"
	"time"
)

// BuildEveningPingPrompt creates the user prompt for evening ping.
func BuildEveningPingPrompt(uc *UserContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Имя: %s\n", uc.DisplayName))
	sb.WriteString(fmt.Sprintf("Текущий streak: %d\n", uc.CurrentStreak))
	sb.WriteString(fmt.Sprintf("Активных целей: %d\n", uc.ActiveGoalsCount))
	sb.WriteString(fmt.Sprintf("Пруфов сегодня: %d\n", uc.ProofsToday))
	sb.WriteString(fmt.Sprintf("День недели: %s\n", uc.DayOfWeek))
	if len(uc.ActiveGoalTitles) > 0 {
		sb.WriteString(fmt.Sprintf("Цели: %s\n", strings.Join(uc.ActiveGoalTitles, ", ")))
	}
	sb.WriteString("\nНапиши один вечерний вопрос для рефлексии.")
	return sb.String()
}

// BuildWeeklyRecapPrompt creates the user prompt for weekly recap.
func BuildWeeklyRecapPrompt(wc *WeeklyContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Имя: %s\n", wc.DisplayName))
	sb.WriteString(fmt.Sprintf("Период: %s — %s\n", wc.PeriodFrom.Format("2006-01-02"), wc.PeriodTo.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("Всего пруфов: %d (одобрено buddy: %d)\n", wc.TotalProofs, wc.BuddyApprovedCount))
	sb.WriteString(fmt.Sprintf("Текущий streak: %d\n", wc.CurrentStreak))

	sb.WriteString("\nЦели:\n")
	for _, g := range wc.Goals {
		if g.ProofCount == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s (%d пруфов)\n", g.Title, g.ProofCount))
		for _, p := range g.TopProofs {
			sb.WriteString(fmt.Sprintf("  • %s\n", p))
		}
	}

	sb.WriteString("\nНапиши краткий итог недели: главное достижение, один инсайт и логичный следующий шаг.")
	return sb.String()
}

// EveningPingSystemPrompt is the system prompt for evening ping generation.
const EveningPingSystemPrompt = `Ты вечерний компаньон по продуктивности. Задача: написать ОДИН рефлексивный вопрос пользователю на основе его контекста.

Правила:
- Длина: 8–25 слов
- Должен заканчиваться знаком вопроса (?)
- Без похвалы («молодец», «отлично»), без стыда («плохо»), без контроля («ты должен»)
- Тон: любопытный, нейтральный, поддерживающий
- Не упоминать приватные данные других людей
- Не использовать markdown, только текст
- Язык: русский`

// WeeklyRecapSystemPrompt is the system prompt for weekly recap generation.
const WeeklyRecapSystemPrompt = `Ты компаньон по росту. Задача: написать краткий итог недели на основе пруфов пользователя.

Структура (3–5 пунктов, каждый — одно предложение):
1. Главное достижение недели
2. Один инсайт или наблюдение
3. Следующий логичный шаг

Правила:
- Конкретика: ссылайся на цели и пруфы
- Тон: профессиональный, но живой
- Без оценок «слабый/сильный», без сравнений с другими
- Не упоминать приватные данные других людей
- Формат: plain text, русский язык`

// EveningPingTemplates is a pool of fallback templates for evening ping.
// Selected based on streak and day of week.
var EveningPingTemplates = map[string][]string{
	"low_streak": {
		"Какой один маленький шаг ты можешь сделать сегодня по своей цели?",
		"Что мешает тебе двигаться вперед, и как это можно обойти?",
		"Какую микропривычку можно заложить сегодня, чтобы не терять инерцию?",
	},
	"mid_streak": {
		"Какой навык ты прокачал на этой неделе больше всего?",
		"Что из сегодняшнего дня стоит зафиксировать как пруф?",
		"Как ты поддерживаешь свою серию активных дней?",
	},
	"high_streak": {
		"Что нового ты узнал о себе, поддерживая такую длинную серию?",
		"Какой следующий уровень по твоей цели ты видишь сейчас?",
		"Какую часть процесса ты бы хотел автоматизировать или упростить?",
	},
}

// WeeklyRecapTemplate is the fallback template for weekly recap.
func WeeklyRecapTemplate(wc *WeeklyContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Итог недели %s — %s\n\n", wc.PeriodFrom.Format("02.01"), wc.PeriodTo.Format("02.01")))

	if wc.TotalProofs == 0 {
		sb.WriteString("На этой неделе пруфов не было. Попробуй сделать один маленький шаг завтра — это лучше, чем ждать идеального момента.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("• Всего пруфов: %d (из них %d одобрено)\n", wc.TotalProofs, wc.BuddyApprovedCount))
	if wc.CurrentStreak > 0 {
		sb.WriteString(fmt.Sprintf("• Текущая серия: %d дней\n", wc.CurrentStreak))
	}

	if len(wc.Goals) > 0 {
		var topGoal string
		maxProofs := 0
		for _, g := range wc.Goals {
			if g.ProofCount > maxProofs {
				maxProofs = g.ProofCount
				topGoal = g.Title
			}
		}
		if topGoal != "" {
			sb.WriteString(fmt.Sprintf("• Больше всего пруфов по цели «%s»\n", topGoal))
		}
	}

	sb.WriteString("\nСледующий шаг: выбери одну цель и сделай к ней один конкретный шаг на следующей неделе.\n")
	return sb.String()
}

// SelectEveningPingTemplate picks a fallback template based on streak.
func SelectEveningPingTemplate(streak int) string {
	key := "low_streak"
	if streak >= 7 {
		key = "mid_streak"
	}
	if streak >= 30 {
		key = "high_streak"
	}
	templates := EveningPingTemplates[key]
	// Deterministic selection based on day of week to avoid repetition.
	dayOfWeek := int(time.Now().Weekday())
	if len(templates) > 0 {
		return templates[dayOfWeek%len(templates)]
	}
	return "Какой один шаг ты сделаешь сегодня по своей цели?"
}

// --- Lead Weekly Brief ---

// LeadBriefSystemPrompt is the system prompt for lead weekly brief.
const LeadBriefSystemPrompt = `Ты помогаешь руководителю команды. Задача: написать краткий бриф недели на основе агрегатных данных команды.

Структура (30–80 слов):
1. Общая сводка: пруфы, approve, reject
2. Один риск или наблюдение
3. Рекомендация

Правила:
- Только факты и агрегаты, без raw daily-log
- Без оценок «слабый/сильный/плохой работник»
- Без сравнений личностей
- Если рисков нет — явно напиши «Без рисков»
- Формат: plain text, русский язык`

// BuildLeadBriefPrompt creates the user prompt for lead brief.
func BuildLeadBriefPrompt(lc *LeadBriefContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Руководитель: %s\n", lc.DisplayName))
	sb.WriteString(fmt.Sprintf("Команда: %s\n", lc.TeamName))
	sb.WriteString(fmt.Sprintf("Период: %s — %s\n", lc.PeriodFrom.Format("2006-01-02"), lc.PeriodTo.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("Пруфов отправлено: %d\n", lc.TotalProofsSubmitted))
	sb.WriteString(fmt.Sprintf("Одобрено: %d\n", lc.TotalProofsApproved))
	sb.WriteString(fmt.Sprintf("Отклонено: %d\n", lc.TotalProofsRejected))
	sb.WriteString(fmt.Sprintf("Среднее время approve: %s\n", lc.ApprovalLatencyAvg))

	sb.WriteString("\nУчастники:\n")
	for _, m := range lc.Members {
		sb.WriteString(fmt.Sprintf("- %s: %d пруфов, streak %d, статус: %s\n", m.DisplayName, m.ProofCount, m.Streak, m.Status))
	}

	sb.WriteString("\nНапиши краткий бриф недели для руководителя.")
	return sb.String()
}

// LeadBriefTemplate is the fallback template for lead weekly brief.
func LeadBriefTemplate(lc *LeadBriefContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Бриф команды «%s» — %s\n\n", lc.TeamName, lc.PeriodTo.Format("02.01.2006")))

	if lc.TotalProofsSubmitted == 0 {
		sb.WriteString("На этой неделе в команде не было пруфов. Проведите stand-up или проверьте блокеры.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("• Пруфов: %d (одобрено %d, отклонено %d)\n", lc.TotalProofsSubmitted, lc.TotalProofsApproved, lc.TotalProofsRejected))
	sb.WriteString(fmt.Sprintf("• Среднее время approve: %s\n", lc.ApprovalLatencyAvg))

	// Find risk members
	var atRisk []string
	var stalled []string
	for _, m := range lc.Members {
		if m.Status == "at_risk" {
			atRisk = append(atRisk, m.DisplayName)
		} else if m.Status == "stalled" {
			stalled = append(stalled, m.DisplayName)
		}
	}

	if len(stalled) > 0 {
		sb.WriteString(fmt.Sprintf("• Без пруфов: %s\n", strings.Join(stalled, ", ")))
	}
	if len(atRisk) > 0 {
		sb.WriteString(fmt.Sprintf("• Слабый streak: %s\n", strings.Join(atRisk, ", ")))
	}
	if len(stalled) == 0 && len(atRisk) == 0 {
		sb.WriteString("• Без рисков\n")
	}

	sb.WriteString("\nРекомендация: проверьте очередь approve и дайте фидбек членам команды.\n")
	return sb.String()
}

// --- Streak Reminder ---

// StreakReminderTemplate is the fallback for streak reminder (no LLM needed).
func StreakReminderTemplate(sc *StreakContext) string {
	if sc.HoursLeft <= 0 {
		return fmt.Sprintf("%s, серия «%s» (%d дней) сейчас под угрозой. Сделай пруф сегодня, чтобы сохранить streak.",
			sc.DisplayName, sc.GoalTitle, sc.CurrentStreak)
	}
	return fmt.Sprintf("%s, у тебя осталось %d часов, чтобы сохранить серию «%s» (%d дней). Сделай пруф сегодня.",
		sc.DisplayName, sc.HoursLeft, sc.GoalTitle, sc.CurrentStreak)
}

// --- Proof Draft ---

// ProofDraftSystemPrompt is the system prompt for proof draft assembly.
const ProofDraftSystemPrompt = `Ты помогаешь пользователю собрать пруф из его ежедневных заметок.

Задача: на основе заметок daily log составить черновик пруфа — краткое описание того, что было сделано по цели.

Правила:
- Длина: 20–80 слов
- Конкретика: ссылайся на факты из заметок
- Тон: нейтральный, деловой
- Не добавляй выдуманных деталей — только то, что есть в заметках
- Если заметок мало или они не по теме цели — честно скажи, что данных недостаточно
- Формат: plain text, русский язык`

// BuildProofDraftPrompt creates the user prompt for proof draft.
func BuildProofDraftPrompt(pdc *ProofDraftContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Пользователь: %s\n", pdc.DisplayName))
	sb.WriteString(fmt.Sprintf("Команда: %s\n", pdc.TeamName))
	sb.WriteString(fmt.Sprintf("Цель: %s\n", pdc.GoalTitle))

	sb.WriteString("\nЗаметки из daily log (последние 7 дней):\n")
	for _, n := range pdc.Notes {
		dateStr := n.LogDate.Format("2006-01-02")
		artifactMark := ""
		if n.HasArtifact {
			artifactMark = " [есть артефакт]"
		}
		sb.WriteString(fmt.Sprintf("- %s: %s%s\n", dateStr, n.TextContent, artifactMark))
	}

	sb.WriteString("\nСоставь черновик пруфа по этой цели.")
	return sb.String()
}

// ProofDraftTemplate is the fallback template for proof draft.
func ProofDraftTemplate(pdc *ProofDraftContext) string {
	if len(pdc.Notes) == 0 {
		return fmt.Sprintf("По цели «%s» не найдено заметок за последние 7 дней. Добавь заметки в daily log — и я соберу пруф.", pdc.GoalTitle)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Черновик пруфа по цели «%s»:\n\n", pdc.GoalTitle))
	for _, n := range pdc.Notes {
		dateStr := n.LogDate.Format("02.01")
		sb.WriteString(fmt.Sprintf("• %s: %s\n", dateStr, n.TextContent))
	}
	sb.WriteString("\nПроверь и отправь как пруф, если всё верно.")
	return sb.String()
}

// --- Buddy Stalled ---

// BuddyStalledSystemPrompt is the system prompt for buddy stalled alert.
const BuddyStalledSystemPrompt = `Ты напоминаешь пользователю, что его buddy ждёт ответа на пруф.

Задача: написать короткое, уважительное напоминание, что нужно дать фидбек.

Правила:
- Длина: 15–40 слов
- Тон: дружелюбный, без давления
- Не упоминать приватные данные других людей
- Не использовать оценки «слабый/плохой/хороший»
- Формат: plain text, русский язык`

// BuildBuddyStalledPrompt creates the user prompt for buddy stalled alert.
func BuildBuddyStalledPrompt(bsc *BuddyStalledContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Buddy: %s\n", bsc.BuddyName))
	sb.WriteString(fmt.Sprintf("Пользователь с пруфом: %s\n", bsc.DisplayName))
	sb.WriteString(fmt.Sprintf("Цель: %s\n", bsc.GoalTitle))
	sb.WriteString(fmt.Sprintf("Пруф отправлен: %s\n", bsc.SubmittedAt.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("Ожидание: %d часов\n", bsc.HoursStalled))
	sb.WriteString("\nНапиши напоминание buddy, что пора дать фидбек.")
	return sb.String()
}

// BuddyStalledTemplate is the fallback template for buddy stalled alert.
func BuddyStalledTemplate(bsc *BuddyStalledContext) string {
	return fmt.Sprintf("Привет, %s! %s отправил пруф по цели «%s» %d часов назад. Дай фидбек, когда будет минутка — человек ждёт.",
		bsc.BuddyName, bsc.DisplayName, bsc.GoalTitle, bsc.HoursStalled)
}
