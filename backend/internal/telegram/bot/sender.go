package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Sender is the interface for sending Telegram messages.
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, kb InlineKeyboardMarkup) (int, error)
	EditMessageText(ctx context.Context, chatID int64, messageID int, text string) error
	AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string) error
}

// Client is a minimal HTTP Bot API client.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

// New returns a Bot API client for the given bot token.
func New(token string) *Client {
	return &Client{
		token:   token,
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", token),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	return c.call(ctx, "sendMessage", payload, nil)
}

func (c *Client) SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, kb InlineKeyboardMarkup) (int, error) {
	payload := map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": kb,
	}
	var msg Message
	if err := c.call(ctx, "sendMessage", payload, &msg); err != nil {
		return 0, err
	}
	return msg.MessageID, nil
}

func (c *Client) EditMessageText(ctx context.Context, chatID int64, messageID int, text string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
	}
	return c.call(ctx, "editMessageText", payload, nil)
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string) error {
	payload := map[string]any{
		"callback_query_id": callbackQueryID,
		"text":              text,
	}
	return c.call(ctx, "answerCallbackQuery", payload, nil)
}

func (c *Client) call(ctx context.Context, method string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("bot: marshal %s: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("bot: build request %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("bot: %s: %w", method, err)
	}
	defer resp.Body.Close()

	if out != nil {
		var envelope apiResponse[json.RawMessage]
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			return fmt.Errorf("bot: decode %s: %w", method, err)
		}
		if !envelope.OK {
			return fmt.Errorf("bot: %s error: %s", method, envelope.Desc)
		}
		return json.Unmarshal(envelope.Result, out)
	}

	var envelope apiResponse[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("bot: decode %s: %w", method, err)
	}
	if !envelope.OK {
		return fmt.Errorf("bot: %s error: %s", method, envelope.Desc)
	}
	return nil
}

// NoopSender is a no-op sender for testing/development.
type NoopSender struct{}

func (NoopSender) SendMessage(_ context.Context, _ int64, _ string) error { return nil }
func (NoopSender) SendMessageWithKeyboard(_ context.Context, _ int64, _ string, _ InlineKeyboardMarkup) (int, error) {
	return 0, nil
}
func (NoopSender) EditMessageText(_ context.Context, _ int64, _ int, _ string) error { return nil }
func (NoopSender) AnswerCallbackQuery(_ context.Context, _, _ string) error           { return nil }
