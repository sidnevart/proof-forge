package ai

import "context"

// FakeAssistantProvider returns plausible canned responses for dev/test.
type FakeAssistantProvider struct{}

func NewFakeAssistantProvider() *FakeAssistantProvider { return &FakeAssistantProvider{} }

func (f *FakeAssistantProvider) GoalToProofs(_ context.Context, _ string) (GoalToProofsResult, error) {
	return GoalToProofsResult{
		Suggestions: []ProofSuggestion{
			{WhatToProve: "Прочитать главу 1 и написать конспект", HowToProve: "Ссылка на заметки в Notion", MovementMode: "single_proof", DueDays: 3},
			{WhatToProve: "Завершить первый модуль курса", HowToProve: "Скриншот сертификата или прогресс-бара", MovementMode: "regular_rhythm", DueDays: 7},
			{WhatToProve: "Применить навык в рабочей задаче", HowToProve: "PR или описание решённой задачи", MovementMode: "work_initiative", DueDays: 14},
		},
	}, nil
}

func (f *FakeAssistantProvider) NextStep(_ context.Context, _, _ string) (NextStepResult, error) {
	return NextStepResult{
		Step:      "Выдели 90 минут и сделай один конкретный шаг по цели",
		Rationale: "Маленький шаг лучше большого плана без действия",
		DueDays:   7,
	}, nil
}

func (f *FakeAssistantProvider) ProofCheck(_ context.Context, draft string) (ProofCheckResult, error) {
	hasArt := len(draft) > 50
	hasCon := len(draft) > 100
	hasNext := len(draft) > 150
	score := 40
	if hasArt {
		score += 20
	}
	if hasCon {
		score += 20
	}
	if hasNext {
		score += 20
	}
	return ProofCheckResult{
		HasArtifact:   hasArt,
		HasConclusion: hasCon,
		HasNextStep:   hasNext,
		Score:         score,
		Feedback:      "Добавь конкретный артефакт — ссылку или скриншот",
		Suggestion:    "Укажи что именно сделал и какой вывод сделал из этого",
	}, nil
}

func (f *FakeAssistantProvider) AntiProof(_ context.Context, _, stuckDescription string) (AntiProofResult, error) {
	return AntiProofResult{
		Summary:     "Попытка была предпринята, но столкнулась с препятствием",
		WhatBlocked: stuckDescription,
		NextAttempt: "Попробовать другой подход или запросить помощь",
	}, nil
}

func (f *FakeAssistantProvider) GrowthDossier(_ context.Context, inputs []DossierInput) (GrowthDossierResult, error) {
	goals := make([]DossierEntry, 0, len(inputs))
	for _, inp := range inputs {
		goals = append(goals, DossierEntry{
			GoalTitle:   inp.GoalTitle,
			ProofsCount: len(inp.ProofsTexts),
			Highlights:  []string{"Регулярное движение по цели", "Конкретные артефакты"},
			Skills:      []string{"Самоорганизация", "Рефлексия"},
			Conclusion:  "Цель активно прорабатывается",
		})
	}
	return GrowthDossierResult{
		PeriodLabel:    "Итоговый период",
		TotalProofs:    len(inputs) * 3,
		ActiveWeeks:    4,
		TopSkills:      []string{"Планирование", "Самодисциплина", "Рефлексия"},
		Goals:          goals,
		OverallSummary: "Активный период с конкретными результатами и регулярными пруфами",
		NextFocus:      "Углубиться в следующий уровень по приоритетной цели",
	}, nil
}
