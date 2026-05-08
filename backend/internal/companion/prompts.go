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
