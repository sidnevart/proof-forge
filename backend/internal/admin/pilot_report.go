package admin

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PilotReportMetrics is the full pilot report payload.
type PilotReportMetrics struct {
	PeriodFrom string `json:"period_from"`
	PeriodTo   string `json:"period_to"`

	TotalSignedUp       int     `json:"total_signed_up"`
	UsersWithFirstProof int     `json:"users_with_first_proof"`
	FirstProofRatePct   float64 `json:"first_proof_rate_pct"`

	WeeklyActiveUsers []WeeklyActive `json:"weekly_active_users"`
	AvgWeeklyProofs   float64        `json:"avg_weekly_proofs"`
	TotalProofs       int            `json:"total_proofs"`
	BuddyApprovedCount int           `json:"buddy_approved_count"`

	GoalsCompleted    int     `json:"goals_completed"`
	CompletionRatePct float64 `json:"completion_rate_pct"`

	AIContractsCreated int `json:"ai_contracts_created"`
	BuddyMatchesUsed   int `json:"buddy_matches_used"`
}

// WeeklyActive is a per-week activity snapshot.
type WeeklyActive struct {
	WeekStart    string `json:"week_start"`
	ActiveUsers  int    `json:"active_users"`
	ProofsCount  int    `json:"proofs_count"`
}

func getPilotReport(ctx context.Context, pool *pgxpool.Pool, workspaceID int64, from, to time.Time) (*PilotReportMetrics, error) {
	c := ctx
	m := &PilotReportMetrics{
		PeriodFrom: from.Format("2006-01-02"),
		PeriodTo:   to.Format("2006-01-02"),
	}

	// Members who signed up (joined a team in this workspace) within the period.
	_ = pool.QueryRow(c, `
		SELECT COUNT(DISTINCT tm.user_id)
		FROM team_memberships tm
		JOIN teams t ON t.id = tm.team_id
		JOIN users u ON u.id = tm.user_id
		WHERE t.workspace_id = $1 AND u.created_at BETWEEN $2 AND $3
	`, workspaceID, from, to).Scan(&m.TotalSignedUp)

	// Users with first_proof event in this workspace's analytics.
	_ = pool.QueryRow(c, `
		SELECT COUNT(DISTINCT user_id)
		FROM analytics_events
		WHERE workspace_id = $1 AND event_name = 'first_proof_submitted'
		  AND ts BETWEEN $2 AND $3
	`, workspaceID, from, to).Scan(&m.UsersWithFirstProof)

	if m.TotalSignedUp > 0 {
		m.FirstProofRatePct = float64(m.UsersWithFirstProof) / float64(m.TotalSignedUp) * 100
	}

	// Weekly active users breakdown.
	rows, err := pool.Query(c, `
		SELECT
			TO_CHAR(DATE_TRUNC('week', ci.submitted_at), 'YYYY-MM-DD') AS week_start,
			COUNT(DISTINCT ci.owner_user_id) AS active_users,
			COUNT(*) AS proofs_count
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		JOIN teams t ON t.id = tm.team_id
		WHERE t.workspace_id = $1
		  AND ci.submitted_at BETWEEN $2 AND $3
		  AND ci.status IN ('submitted', 'approved')
		GROUP BY week_start
		ORDER BY week_start
	`, workspaceID, from, to)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var wa WeeklyActive
			if err := rows.Scan(&wa.WeekStart, &wa.ActiveUsers, &wa.ProofsCount); err == nil {
				m.WeeklyActiveUsers = append(m.WeeklyActiveUsers, wa)
				m.TotalProofs += wa.ProofsCount
			}
		}
	}
	if m.WeeklyActiveUsers == nil {
		m.WeeklyActiveUsers = []WeeklyActive{}
	}
	if len(m.WeeklyActiveUsers) > 0 {
		m.AvgWeeklyProofs = float64(m.TotalProofs) / float64(len(m.WeeklyActiveUsers))
	}

	// Buddy-approved proofs count.
	_ = pool.QueryRow(c, `
		SELECT COUNT(ci.id)
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		JOIN teams t ON t.id = tm.team_id
		WHERE t.workspace_id = $1
		  AND ci.status = 'approved'
		  AND ci.approved_at BETWEEN $2 AND $3
	`, workspaceID, from, to).Scan(&m.BuddyApprovedCount)

	// Goals completed.
	_ = pool.QueryRow(c, `
		SELECT COUNT(g.id)
		FROM goals g
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id
		JOIN teams t ON t.id = tm.team_id
		WHERE t.workspace_id = $1
		  AND g.status = 'completed'
		  AND g.updated_at BETWEEN $2 AND $3
	`, workspaceID, from, to).Scan(&m.GoalsCompleted)

	if m.TotalSignedUp > 0 {
		m.CompletionRatePct = float64(m.GoalsCompleted) / float64(m.TotalSignedUp) * 100
	}

	// AI usage from analytics_events.
	_ = pool.QueryRow(c, `
		SELECT COUNT(*) FROM analytics_events
		WHERE workspace_id = $1 AND event_name = 'ai_suggestion_used' AND ts BETWEEN $2 AND $3
	`, workspaceID, from, to).Scan(&m.AIContractsCreated)

	_ = pool.QueryRow(c, `
		SELECT COUNT(*) FROM analytics_events
		WHERE workspace_id = $1 AND event_name = 'buddy_matched' AND ts BETWEEN $2 AND $3
	`, workspaceID, from, to).Scan(&m.BuddyMatchesUsed)

	return m, nil
}

func (h *Handler) handlePilotReport(w http.ResponseWriter, r *http.Request) {
	wsIDStr := r.URL.Query().Get("workspace_id")
	if wsIDStr == "" {
		http.Error(w, "workspace_id required", http.StatusBadRequest)
		return
	}
	var workspaceID int64
	if _, err := fmt.Sscanf(wsIDStr, "%d", &workspaceID); err != nil {
		http.Error(w, "invalid workspace_id", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		http.Error(w, "format must be json or csv", http.StatusBadRequest)
		return
	}

	to := time.Now()
	from := to.AddDate(0, -3, 0)
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			to = t
		}
	}

	if to.Sub(from) > 365*24*time.Hour {
		http.Error(w, "max period is 365 days", http.StatusBadRequest)
		return
	}

	report, err := getPilotReport(r.Context(), h.pool, workspaceID, from, to)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="pilot-report-%d.csv"`, workspaceID))
		writePilotCSV(w, report)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": report})
	}
}

func writePilotCSV(w http.ResponseWriter, r *PilotReportMetrics) {
	wr := csv.NewWriter(w)
	defer wr.Flush()

	_ = wr.Write([]string{"Метрика", "Значение"})
	_ = wr.Write([]string{"Период от", r.PeriodFrom})
	_ = wr.Write([]string{"Период до", r.PeriodTo})
	_ = wr.Write([]string{"Всего зарегистрировались", fmt.Sprintf("%d", r.TotalSignedUp)})
	_ = wr.Write([]string{"Сдали первый пруф", fmt.Sprintf("%d", r.UsersWithFirstProof)})
	_ = wr.Write([]string{"First proof rate %", fmt.Sprintf("%.1f", r.FirstProofRatePct)})
	_ = wr.Write([]string{"Всего пруфов", fmt.Sprintf("%d", r.TotalProofs)})
	_ = wr.Write([]string{"Одобрено buddy", fmt.Sprintf("%d", r.BuddyApprovedCount)})
	_ = wr.Write([]string{"Завершили цели", fmt.Sprintf("%d", r.GoalsCompleted)})
	_ = wr.Write([]string{"Completion rate %", fmt.Sprintf("%.1f", r.CompletionRatePct)})
	_ = wr.Write([]string{"AI suggestions использовано", fmt.Sprintf("%d", r.AIContractsCreated)})
	_ = wr.Write([]string{"Buddy matches использовано", fmt.Sprintf("%d", r.BuddyMatchesUsed)})
	_ = wr.Write([]string{"", ""})
	_ = wr.Write([]string{"Неделя", "Активных пользователей", "Пруфов"})
	for _, wa := range r.WeeklyActiveUsers {
		_ = wr.Write([]string{wa.WeekStart, fmt.Sprintf("%d", wa.ActiveUsers), fmt.Sprintf("%d", wa.ProofsCount)})
	}
}
