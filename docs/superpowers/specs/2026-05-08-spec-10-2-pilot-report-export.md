# Спек 10.2 — Pilot Report Export

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 10 · Аналитика пилота  
**Зависимости:** Спек 10.1 (analytics events), 3.5 (admin metrics), 6.4 (admin panel)  
**Сложность:** S  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

После пилота нужно показать результаты командам и руководству: сколько вошли, сдали первый пруф, активны каждую неделю, завершили цели. Данные в CSV или JSON — для дальнейшего анализа в Excel/Google Sheets.

**Доступ:** только `is_platform_admin = true`.

---

## Маршрут

```
GET /v1/admin/pilot-report?workspace_id=X&format=csv|json
GET /v1/admin/pilot-report?workspace_id=X&format=csv&from=2025-01-01&to=2025-03-31
```

---

## Backend

### Структура

```go
// backend/internal/admin/pilot_report.go

type PilotReportMetrics struct {
    // Период
    PeriodFrom string `json:"period_from"`
    PeriodTo   string `json:"period_to"`

    // Воронка
    TotalInvited       int     `json:"total_invited"`
    TotalSignedUp      int     `json:"total_signed_up"`
    SignupRate         float64 `json:"signup_rate_pct"`

    FirstProofRate     float64 `json:"first_proof_rate_pct"`
    UsersWithFirstProof int    `json:"users_with_first_proof"`

    // Активность
    WeeklyActiveUsers  []WeeklyActive `json:"weekly_active_users"` // 8 недель
    AvgWeeklyProofs    float64        `json:"avg_weekly_proofs"`
    TotalProofs        int            `json:"total_proofs"`
    BuddyApprovedCount int            `json:"buddy_approved_count"`

    // Завершение
    GoalsCompleted     int     `json:"goals_completed"`
    CompletionRate     float64 `json:"completion_rate_pct"`

    // AI использование
    AIContractsCreated int `json:"ai_contracts_created"`
    DossiersGenerated  int `json:"dossiers_generated"`
    BuddyMatchesUsed   int `json:"buddy_matches_used"`
}

type WeeklyActive struct {
    WeekStart    string `json:"week_start"`    // "2025-01-06"
    ActiveUsers  int    `json:"active_users"`
    ProofsCount  int    `json:"proofs_count"`
}
```

### SQL запросы

```go
func (r *AdminRepo) GetPilotReport(ctx context.Context, workspaceID string, from, to time.Time) (*PilotReportMetrics, error) {
    var m PilotReportMetrics
    m.PeriodFrom = from.Format("2006-01-02")
    m.PeriodTo = to.Format("2006-01-02")

    // Signed up в периоде
    r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM users u
        JOIN workspace_memberships wm ON wm.user_id = u.id
        WHERE wm.workspace_id = $1 AND u.created_at BETWEEN $2 AND $3
    `, workspaceID, from, to).Scan(&m.TotalSignedUp)

    // First proof rate из analytics_events
    r.db.QueryRowContext(ctx, `
        SELECT COUNT(DISTINCT user_id) FROM analytics_events
        WHERE workspace_id = $1 AND event_type = 'first_proof_submitted'
          AND created_at BETWEEN $2 AND $3
    `, workspaceID, from, to).Scan(&m.UsersWithFirstProof)

    if m.TotalSignedUp > 0 {
        m.FirstProofRate = float64(m.UsersWithFirstProof) / float64(m.TotalSignedUp) * 100
    }

    // Weekly active — 8 недель
    rows, _ := r.db.QueryContext(ctx, `
        SELECT 
            DATE_TRUNC('week', ci.created_at) AS week_start,
            COUNT(DISTINCT ci.user_id) AS active_users,
            COUNT(*) AS proofs_count
        FROM check_ins ci
        JOIN goals g ON g.id = ci.goal_id
        JOIN workspace_memberships wm ON wm.user_id = ci.user_id AND wm.workspace_id = $1
        WHERE ci.created_at BETWEEN $2 AND $3
        GROUP BY week_start
        ORDER BY week_start
    `, workspaceID, from, to)
    defer rows.Close()
    for rows.Next() {
        var wa WeeklyActive
        var weekStart time.Time
        rows.Scan(&weekStart, &wa.ActiveUsers, &wa.ProofsCount)
        wa.WeekStart = weekStart.Format("2006-01-02")
        m.WeeklyActiveUsers = append(m.WeeklyActiveUsers, wa)
        m.TotalProofs += wa.ProofsCount
    }

    // Goals completed
    r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM goals g
        JOIN workspace_memberships wm ON wm.user_id = g.user_id AND wm.workspace_id = $1
        WHERE g.status = 'completed' AND g.updated_at BETWEEN $2 AND $3
    `, workspaceID, from, to).Scan(&m.GoalsCompleted)

    // AI usage
    r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM analytics_events
        WHERE workspace_id = $1 AND event_type = 'ai_suggestion_used'
          AND created_at BETWEEN $2 AND $3
    `, workspaceID, from, to).Scan(&m.AIContractsCreated)

    r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM analytics_events
        WHERE workspace_id = $1 AND event_type = 'dossier_generated'
          AND created_at BETWEEN $2 AND $3
    `, workspaceID, from, to).Scan(&m.DossiersGenerated)

    return &m, nil
}
```

### Handler

```go
func (h *AdminHandler) PilotReport(w http.ResponseWriter, r *http.Request) {
    workspaceID := r.URL.Query().Get("workspace_id")
    format := r.URL.Query().Get("format")
    if format == "" { format = "json" }
    if format != "json" && format != "csv" {
        http.Error(w, "format must be json or csv", 400)
        return
    }

    from, to, err := parseDateRange(r.URL.Query())
    if err != nil {
        // Default: последние 90 дней
        to = time.Now()
        from = to.AddDate(0, -3, 0)
    }

    if to.Sub(from) > 365*24*time.Hour {
        http.Error(w, "max period is 365 days", 400)
        return
    }

    report, err := h.adminRepo.GetPilotReport(r.Context(), workspaceID, from, to)
    if err != nil {
        http.Error(w, "internal error", 500)
        return
    }

    switch format {
    case "csv":
        w.Header().Set("Content-Type", "text/csv")
        w.Header().Set("Content-Disposition", 
            fmt.Sprintf(`attachment; filename="pilot-report-%s.csv"`, workspaceID[:8]))
        writeCSV(w, report)
    case "json":
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(report)
    }
}

func writeCSV(w io.Writer, report *PilotReportMetrics) {
    writer := csv.NewWriter(w)
    defer writer.Flush()

    // Summary
    writer.Write([]string{"Метрика", "Значение"})
    writer.Write([]string{"Период от", report.PeriodFrom})
    writer.Write([]string{"Период до", report.PeriodTo})
    writer.Write([]string{"Всего зарегистрировались", fmt.Sprintf("%d", report.TotalSignedUp)})
    writer.Write([]string{"Сдали первый пруф", fmt.Sprintf("%d", report.UsersWithFirstProof)})
    writer.Write([]string{"First proof rate %", fmt.Sprintf("%.1f", report.FirstProofRate)})
    writer.Write([]string{"Всего пруфов", fmt.Sprintf("%d", report.TotalProofs)})
    writer.Write([]string{"Завершили цели", fmt.Sprintf("%d", report.GoalsCompleted)})
    writer.Write([]string{"AI suggestions использовано", fmt.Sprintf("%d", report.AIContractsCreated)})
    writer.Write([]string{"Дошье сгенерировано", fmt.Sprintf("%d", report.DossiersGenerated)})
    writer.Write([]string{"", ""})

    // Weekly breakdown
    writer.Write([]string{"Неделя", "Активных пользователей", "Пруфов"})
    for _, w := range report.WeeklyActiveUsers {
        writer.Write([]string{w.WeekStart, fmt.Sprintf("%d", w.ActiveUsers), fmt.Sprintf("%d", w.ProofsCount)})
    }
}
```

---

## Frontend: кнопка в Admin Panel

Добавляется в `workspace-table.tsx` (спек 6.4) — в меню действий (···) каждого workspace:

```tsx
// В dropdown меню workspace строки:
<DropdownItem onClick={() => downloadReport(workspace.id, 'csv')}>
  Скачать отчёт (CSV)
</DropdownItem>
<DropdownItem onClick={() => downloadReport(workspace.id, 'json')}>
  Скачать отчёт (JSON)
</DropdownItem>
```

```tsx
function downloadReport(workspaceId: string, format: 'csv' | 'json') {
  const url = `/api/v1/admin/pilot-report?workspace_id=${workspaceId}&format=${format}`;
  const a = document.createElement('a');
  a.href = url;
  a.download = `pilot-report-${workspaceId.slice(0, 8)}.${format}`;
  a.click();
}
```

### Страница /admin/workspaces/:id/report (опционально, если нужен UI)

Отображает сводку прямо в браузере для платформ-админа без скачивания:

```tsx
export function WorkspaceReportPage({ params }) {
  const { data } = useSWR(
    `/api/v1/admin/pilot-report?workspace_id=${params.id}&format=json`,
    fetcher
  );

  if (!data) return <PageSkeleton />;

  return (
    <div className={styles.page}>
      <div className={styles.title}>Отчёт пилота</div>
      <div className={styles.period}>{data.period_from} — {data.period_to}</div>
      
      <div className={styles.grid}>
        <MetricCard label="Зарегистрировались" value={data.total_signed_up} />
        <MetricCard label="First proof rate" value={`${data.first_proof_rate_pct.toFixed(1)}%`} />
        <MetricCard label="Всего пруфов" value={data.total_proofs} />
        <MetricCard label="Завершили цели" value={data.goals_completed} />
      </div>

      <WeeklyChart weeks={data.weekly_active_users} />

      <div className={styles.exportBtns}>
        <button onClick={() => downloadReport(params.id, 'csv')} className={styles.exportBtn}>
          Скачать CSV
        </button>
        <button onClick={() => downloadReport(params.id, 'json')} className={styles.exportBtn}>
          Скачать JSON
        </button>
      </div>
    </div>
  );
}
```

---

## Acceptance Criteria

- [ ] `GET /v1/admin/pilot-report?workspace_id=X` возвращает JSON с корректными метриками
- [ ] `format=csv` возвращает валидный CSV с header `Content-Disposition: attachment`
- [ ] Period по умолчанию: последние 90 дней
- [ ] Период >365 дней → 400 с сообщением
- [ ] Не-platform_admin → 403
- [ ] `first_proof_rate_pct` вычисляется из analytics_events (не из check_ins напрямую)
- [ ] `weekly_active_users` массив содержит записи за все недели в периоде (включая пустые с 0)
- [ ] CSV кодировка UTF-8, корректно открывается в Excel
- [ ] Кнопка "Скачать отчёт (CSV)" появляется в menu dropdown каждого workspace в admin panel
- [ ] Дата в filename — первые 8 символов workspace_id (не UUID полностью)

---

## Что нельзя делать

- Не включать в отчёт PII: имена, email, тексты пруфов
- Не делать endpoint публичным или доступным без platform_admin
- Не генерировать PDF — только CSV и JSON (PDF потребует сложных зависимостей)
- Не добавлять real-time обновление отчёта — всегда snapshot по запросу
