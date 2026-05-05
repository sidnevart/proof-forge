package telegram

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles DB operations for telegram linking.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreatePendingToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO telegram_link_pending (token, user_id, expires_at) VALUES ($1, $2, $3)
		 ON CONFLICT (token) DO NOTHING`,
		token, userID, expiresAt,
	)
	return err
}

// RedeemPendingToken returns the pending token and deletes it. Returns ErrTokenNotFound if missing or expired.
func (r *Repository) RedeemPendingToken(ctx context.Context, token string) (PendingToken, error) {
	var pt PendingToken
	err := r.pool.QueryRow(ctx,
		`DELETE FROM telegram_link_pending WHERE token = $1 AND expires_at > now() RETURNING token, user_id, expires_at`,
		token,
	).Scan(&pt.Token, &pt.UserID, &pt.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PendingToken{}, ErrTokenNotFound
		}
		return PendingToken{}, err
	}
	return pt, nil
}

func (r *Repository) CreateTelegramLink(ctx context.Context, userID, chatID int64, username string) (TelegramLink, error) {
	var link TelegramLink
	err := r.pool.QueryRow(ctx,
		`INSERT INTO telegram_links (user_id, telegram_chat_id, telegram_username)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		   SET telegram_chat_id = EXCLUDED.telegram_chat_id,
		       telegram_username = EXCLUDED.telegram_username,
		       linked_at = NOW(),
		       status = 'active'
		 RETURNING id, user_id, telegram_chat_id, telegram_username, linked_at, status`,
		userID, chatID, username,
	).Scan(&link.ID, &link.UserID, &link.TelegramChatID, &link.TelegramUsername, &link.LinkedAt, &link.Status)
	if err != nil {
		return TelegramLink{}, err
	}
	return link, nil
}

func (r *Repository) GetTelegramLinkByUserID(ctx context.Context, userID int64) (TelegramLink, error) {
	var link TelegramLink
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, telegram_chat_id, COALESCE(telegram_username,''), linked_at, status
		 FROM telegram_links WHERE user_id = $1 AND status = 'active'`,
		userID,
	).Scan(&link.ID, &link.UserID, &link.TelegramChatID, &link.TelegramUsername, &link.LinkedAt, &link.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TelegramLink{}, ErrLinkNotFound
		}
		return TelegramLink{}, err
	}
	return link, nil
}

func (r *Repository) GetTelegramLinkByChatID(ctx context.Context, chatID int64) (TelegramLink, error) {
	var link TelegramLink
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, telegram_chat_id, COALESCE(telegram_username,''), linked_at, status
		 FROM telegram_links WHERE telegram_chat_id = $1 AND status = 'active'`,
		chatID,
	).Scan(&link.ID, &link.UserID, &link.TelegramChatID, &link.TelegramUsername, &link.LinkedAt, &link.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TelegramLink{}, ErrLinkNotFound
		}
		return TelegramLink{}, err
	}
	return link, nil
}

func (r *Repository) GetLinkedUsersByChatIDs(ctx context.Context, userIDs []int64) ([]TelegramLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, telegram_chat_id, COALESCE(telegram_username,''), linked_at, status
		 FROM telegram_links WHERE user_id = ANY($1) AND status = 'active'`,
		userIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []TelegramLink
	for rows.Next() {
		var l TelegramLink
		if err := rows.Scan(&l.ID, &l.UserID, &l.TelegramChatID, &l.TelegramUsername, &l.LinkedAt, &l.Status); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}
