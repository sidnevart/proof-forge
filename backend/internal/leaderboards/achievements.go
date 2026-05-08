package leaderboards

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Achievement catalogue definition.
type achievementDef struct {
	ID          string
	Title       string
	Description string
	Icon        string
	Target      int // for streak/count achievements; 0 = no progress bar
}

var catalogue = []achievementDef{
	{"first_proof", "Первый пруф", "Сдал первое доказательство прогресса", "proof", 1},
	{"streak_4_weeks", "4 недели подряд", "Не пропустил ни одной недели", "streak", 4},
	{"streak_8_weeks", "8 недель подряд", "Сдавай пруфы каждую неделю", "streak", 8},
	{"streak_12_weeks", "12 недель подряд", "Сдавай пруфы каждую неделю", "streak", 12},
	{"helped_5_people", "Помог 5 участникам", "Дал ревью ≥5 разным участникам", "contribute", 5},
	{"became_buddy", "Стал buddy", "Помог кому-то не слиться", "buddy", 1},
	{"became_mentor", "Наставник", "Назначен потенциальным наставником", "mentor", 1},
	{"season_completed", "Сезон завершён", "Выполнил все proof contracts за сезон", "season", 1},
	{"first_public_artifact", "Первый артефакт", "Опубликовал проверенный артефакт", "artifact", 1},
	{"ipr_goal_linked", "ИПР цель", "Создал цель для индивидуального плана развития", "ipr", 1},
}

type AchievementsResponse struct {
	Unlocked     []UnlockedAchievement `json:"unlocked"`
	Locked       []LockedAchievement   `json:"locked"`
	RecentUnlock *UnlockedAchievement  `json:"recent_unlock"`
}

type UnlockedAchievement struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UnlockedAt  time.Time `json:"unlocked_at"`
	Icon        string    `json:"icon"`
}

type LockedAchievement struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Progress    int    `json:"progress"`
	Target      int    `json:"target"`
	Icon        string `json:"icon"`
}

func getAchievements(ctx context.Context, pool *pgxpool.Pool, userID int64) (*AchievementsResponse, error) {
	// Load unlocked achievements from DB.
	rows, err := pool.Query(ctx, `
		SELECT achievement_id, unlocked_at FROM user_achievements
		WHERE user_id = $1
		ORDER BY unlocked_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("achievements: load: %w", err)
	}
	defer rows.Close()

	unlockedMap := map[string]time.Time{}
	var latestUnlockedAt time.Time
	var latestID string
	for rows.Next() {
		var id string
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, err
		}
		unlockedMap[id] = at
		if at.After(latestUnlockedAt) {
			latestUnlockedAt = at
			latestID = id
		}
	}

	// Compute progress for locked achievements.
	progress, err := computeProgress(ctx, pool, userID)
	if err != nil {
		return nil, err
	}

	resp := &AchievementsResponse{}
	for _, def := range catalogue {
		if at, ok := unlockedMap[def.ID]; ok {
			resp.Unlocked = append(resp.Unlocked, UnlockedAchievement{
				ID:          def.ID,
				Title:       def.Title,
				Description: def.Description,
				UnlockedAt:  at,
				Icon:        def.Icon,
			})
		} else {
			p := progress[def.ID]
			resp.Locked = append(resp.Locked, LockedAchievement{
				ID:          def.ID,
				Title:       def.Title,
				Description: def.Description,
				Progress:    p,
				Target:      def.Target,
				Icon:        def.Icon,
			})
		}
	}
	if resp.Unlocked == nil {
		resp.Unlocked = []UnlockedAchievement{}
	}
	if resp.Locked == nil {
		resp.Locked = []LockedAchievement{}
	}

	// recent_unlock: last unlocked within 24 hours.
	if !latestUnlockedAt.IsZero() && time.Since(latestUnlockedAt) < 24*time.Hour {
		for _, u := range resp.Unlocked {
			if u.ID == latestID {
				cp := u
				resp.RecentUnlock = &cp
				break
			}
		}
	}

	return resp, nil
}

func computeProgress(ctx context.Context, pool *pgxpool.Pool, userID int64) (map[string]int, error) {
	p := map[string]int{}

	// Proof count.
	var totalProofs int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM check_ins WHERE owner_user_id = $1 AND status = 'approved'`, userID).Scan(&totalProofs)
	p["first_proof"] = totalProofs

	// Streak.
	streak, _, _ := computeStreakAndRecord(ctx, pool, userID)
	p["streak_4_weeks"] = streak
	p["streak_8_weeks"] = streak
	p["streak_12_weeks"] = streak

	// Reviews given to distinct users.
	var distinctReviewees int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ci.owner_user_id)
		FROM check_in_reviews cr
		JOIN check_ins ci ON ci.id = cr.check_in_id
		WHERE cr.reviewer_user_id = $1
	`, userID).Scan(&distinctReviewees)
	p["helped_5_people"] = distinctReviewees

	// Buddy: has any active pact as buddy.
	var buddyCount int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM pacts WHERE buddy_user_id = $1`, userID).Scan(&buddyCount)
	p["became_buddy"] = buddyCount

	// IPR goal.
	var iprGoals int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM goals
		WHERE owner_user_id = $1 AND movement_mode = 'regular_rhythm'
		  AND (category ILIKE '%ипр%' OR category ILIKE '%ipr%')
	`, userID).Scan(&iprGoals)
	p["ipr_goal_linked"] = iprGoals

	return p, nil
}

// CheckAndUnlockAchievements evaluates and stores newly unlocked achievements.
// Called after check-in approval and by daily cron.
func CheckAndUnlockAchievements(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	progress, err := computeProgress(ctx, pool, userID)
	if err != nil {
		return err
	}

	var totalProofs int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM check_ins WHERE owner_user_id = $1 AND status = 'approved'`, userID).Scan(&totalProofs)

	candidates := []string{}

	if totalProofs >= 1 {
		candidates = append(candidates, "first_proof")
	}

	streak, _, _ := computeStreakAndRecord(ctx, pool, userID)
	if streak >= 4 {
		candidates = append(candidates, "streak_4_weeks")
	}
	if streak >= 8 {
		candidates = append(candidates, "streak_8_weeks")
	}
	if streak >= 12 {
		candidates = append(candidates, "streak_12_weeks")
	}

	if progress["helped_5_people"] >= 5 {
		candidates = append(candidates, "helped_5_people")
	}
	if progress["became_buddy"] >= 1 {
		candidates = append(candidates, "became_buddy")
	}
	if progress["ipr_goal_linked"] >= 1 {
		candidates = append(candidates, "ipr_goal_linked")
	}

	// Check artifact proof.
	var artifactCount int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ci.id)
		FROM check_ins ci
		JOIN evidence_items ei ON ei.check_in_id = ci.id AND ei.kind = 'link'
		WHERE ci.owner_user_id = $1 AND ci.status = 'approved'
	`, userID).Scan(&artifactCount)
	if artifactCount >= 1 {
		candidates = append(candidates, "first_public_artifact")
	}

	// Season completed: all proof_contracts fulfilled for any challenge goal.
	var seasonDone int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT g.id)
		FROM goals g
		WHERE g.owner_user_id = $1 AND g.movement_mode = 'challenge'
		  AND NOT EXISTS (
			SELECT 1 FROM proof_contracts pc
			WHERE pc.goal_id = g.id AND pc.status NOT IN ('fulfilled','cancelled')
		  )
		  AND EXISTS (
			SELECT 1 FROM proof_contracts pc WHERE pc.goal_id = g.id AND pc.status = 'fulfilled'
		  )
	`, userID).Scan(&seasonDone)
	if err == nil && seasonDone >= 1 {
		candidates = append(candidates, "season_completed")
	}

	for _, id := range candidates {
		_, err := pool.Exec(ctx, `
			INSERT INTO user_achievements(user_id, achievement_id)
			VALUES($1, $2)
			ON CONFLICT(user_id, achievement_id) DO NOTHING
		`, userID, id)
		if err != nil {
			return fmt.Errorf("unlock achievement %s: %w", id, err)
		}
	}

	return nil
}

