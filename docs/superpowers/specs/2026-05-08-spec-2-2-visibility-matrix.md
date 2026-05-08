# Спек 2.2 — Матрица видимости

**DNA:** `docs/superpowers/specs/2026-05-08-product-dna-design.md`  
**Группа:** 2 · Роли и приватность  
**Зависимости:** Спек 2.1 (authz), Спек 1.1, 1.2  
**Сложность:** L  
**Effort для Sonnet 4.6:** medium

---

## Бизнес-контекст

ProofForge публично показывает вклад, приватно показывает риски. Каждый тип данных имеет свою видимость в зависимости от роли наблюдателя.

**Принцип DNA П5:** публично вклад, приватно риски.  
**Принцип DNA:** AI работает на пользователя, не на руководителя.

Это не просто набор IF-условий — это centralized visibility service, который каждый обработчик вызывает перед возвратом данных.

---

## Матрица видимости

| Данные | Сам пользователь | Buddy | Участник круга | Teamspace Lead | Community Leader | Workspace Owner | Platform Admin |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| Цель (title, mode) | ✅ | ✅ | ✅* | ✅ | ✅** | Агрег. | ✅ |
| Описание цели | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Proof (финальный) | ✅ | ✅ | ✅* | ✅ | ✅** | Агрег. | ✅ |
| Черновик proof | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Streak | ✅ | ✅ | Агрег. | Агрег. | Агрег. | Агрег. | ✅ |
| Место в рейтинге | ✅ (своё) | ❌ | Позиция в круге | ❌ | ❌ | ❌ | ✅ |
| Блокеры (личные) | ✅ | ✅ | ❌ | Агрег. | Агрег. | ❌ | ❌ |
| Советы buddy | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Итог сезона/досье | ✅ | По согласию | ❌ | По согласию | По согласию | ❌ | ❌ |
| Риск отвалиться | ✅ | ✅ | ❌ | Агрег.*** | Агрег.*** | ❌ | ✅ |
| История ИПР | ✅ | ❌ | ❌ | По согласию | ❌ | ❌ | ❌ |
| AI-рекомендации | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Черновик цели | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Интерес к инициативам | ✅ | ❌ | ❌ | Агрег. | Агрег. | Агрег. | ✅ |

*только если пользователь сделал proof видимым для круга (настройка на уровне proof)  
**только топ-пруфы недели, выбранные вручную или по лайкам  
***только агрегированно "N человек нуждаются во внимании", без имён

---

## Что реализовать

### Backend
1. Пакет `backend/internal/visibility/` — visibility service
2. Тип `VisibilityContext` — текущий наблюдатель + его роли
3. Методы фильтрации для каждого типа данных
4. Применить в существующих handlers для goals и check-ins

### Не реализовывать
- UI настроек приватности (будет в profile settings)
- Согласие на sharing (отдельная фича)

---

## Go: Visibility Service

```go
// backend/internal/visibility/visibility.go
package visibility

import (
    "context"
    "github.com/google/uuid"
)

// Роль наблюдателя относительно конкретного пользователя
type ObserverRole string

const (
    ObserverRoleSelf              ObserverRole = "self"
    ObserverRoleBuddy             ObserverRole = "buddy"
    ObserverRoleCircleMember      ObserverRole = "circle_member"
    ObserverRoleTeamspaceLead     ObserverRole = "teamspace_lead"
    ObserverRoleCommunityLeader   ObserverRole = "community_leader"
    ObserverRoleWorkspaceOwner    ObserverRole = "workspace_owner"
    ObserverRolePlatformAdmin     ObserverRole = "platform_admin"
    ObserverRoleStranger          ObserverRole = "stranger"
)

type VisibilityContext struct {
    ObserverID   uuid.UUID
    SubjectID    uuid.UUID    // чьи данные смотрят
    ObserverRole ObserverRole
}

type Service struct {
    authz *authz.Authorizer
    db    Database
}

// Определить роль наблюдателя относительно субъекта
func (s *Service) ResolveRole(ctx context.Context, observerID, subjectID uuid.UUID) (ObserverRole, error)

// Фильтры данных:

type GoalView struct {
    ID          uuid.UUID
    Title       string
    MovementMode string
    Description *string     // nil если не видно
    Status      string
}

func (s *Service) FilterGoal(ctx context.Context, goal *goals.Goal, vc VisibilityContext) GoalView

type ProofView struct {
    ID          uuid.UUID
    Content     *string     // nil для черновиков
    Status      string
    CreatedAt   time.Time
    IsVisible   bool
}

func (s *Service) FilterProof(ctx context.Context, proof *checkins.CheckIn, vc VisibilityContext) ProofView

type UserStatsView struct {
    Streak          *int     // nil если не видно, число если видно
    StreakAggregated *string  // "активен" без числа
    RiskLevel       *string  // nil или "attention_needed" (агрегированно)
}

func (s *Service) FilterUserStats(ctx context.Context, stats *stats.UserStats, vc VisibilityContext) UserStatsView
```

### ResolveRole — логика определения роли наблюдателя

```go
func (s *Service) ResolveRole(ctx context.Context, observerID, subjectID uuid.UUID) (ObserverRole, error) {
    if observerID == subjectID {
        return ObserverRoleSelf, nil
    }
    
    // Проверить platform admin
    if isAdmin, _ := s.authz.IsPlatformAdmin(ctx, observerID); isAdmin {
        return ObserverRolePlatformAdmin, nil
    }
    
    // Проверить buddy (активный пакт между observer и subject)
    if isBuddy, _ := s.db.HasActivePact(ctx, observerID, subjectID); isBuddy {
        return ObserverRoleBuddy, nil
    }
    
    // Проверить общий circle
    if sameCircle, _ := s.db.ShareCircle(ctx, observerID, subjectID); sameCircle {
        return ObserverRoleCircleMember, nil
    }
    
    // Проверить teamspace lead (observer = lead в teamspace где subject = member)
    if isLead, _ := s.db.IsTeamspaceLead(ctx, observerID, subjectID); isLead {
        return ObserverRoleTeamspaceLead, nil
    }
    
    // Проверить community leader
    if isLeader, _ := s.db.IsCommunityLeader(ctx, observerID, subjectID); isLeader {
        return ObserverRoleCommunityLeader, nil
    }
    
    // Проверить workspace owner
    if isOwner, _ := s.db.IsWorkspaceOwner(ctx, observerID, subjectID); isOwner {
        return ObserverRoleWorkspaceOwner, nil
    }
    
    return ObserverRoleStranger, nil
}
```

---

## Применение в handlers

**Существующий `GET /v1/goals/:id`** — обновить:

```go
func (h *Handler) GetGoal(w http.ResponseWriter, r *http.Request) {
    goalID := chi.URLParam(r, "id")
    observerID := authctx.UserID(r.Context())
    
    goal, err := h.goalSvc.GetByID(r.Context(), goalID)
    if err != nil { /* ... */ }
    
    // Определить роль и отфильтровать данные
    vc, err := h.visibility.ResolveRole(r.Context(), observerID, goal.UserID)
    filtered := h.visibility.FilterGoal(r.Context(), goal, vc)
    
    render.JSON(w, r, map[string]any{"data": filtered})
}
```

---

## Пользовательские настройки видимости

Добавить в profile settings (будущий спек) возможность:
- Сделать proof видимым для круга (default: только buddy)
- Поделиться итоговым досье с руководителем (explicit consent)

В БД хранить как настройку на уровне check-in:
```sql
-- Добавить к check_ins (отдельная миграция в profile settings спеке):
-- visibility TEXT NOT NULL DEFAULT 'buddy' CHECK (visibility IN ('self', 'buddy', 'circle', 'space'))
```

В текущем спеке — реализовать только логику чтения существующего значения.

---

## Acceptance Criteria

- [ ] `ResolveRole` корректно определяет роль в 7 случаях (unit-тесты для каждого)
- [ ] `GET /v1/goals/:id` от buddy возвращает title но НЕ возвращает description
- [ ] `GET /v1/goals/:id` от teamspace_lead возвращает title но НЕ возвращает description
- [ ] Streak виден buddy как число, teamspace lead — только как агрегат "активен/неактивен"
- [ ] Риск отвалиться: самому пользователю виден его уровень, lead видит только "N человек нуждаются во внимании" без имён
- [ ] Все существующие endpoints для goals и check-ins применяют visibility service
- [ ] Platform admin видит все данные (обходит фильтры)

---

## Что нельзя делать

- Не делать visibility logic inline в каждом handler — только через centralized service
- Не показывать teamspace lead имена людей "в риске" — только агрегат
- Не применять visibility к platform_admin (они видят всё)
- Не кешировать resolved role (может меняться в реальном времени)
