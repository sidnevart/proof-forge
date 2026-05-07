package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	DB       DBConfig
	Log      LogConfig
	Invite   InviteConfig
	Session  SessionConfig
	Worker   WorkerConfig
	Storage  StorageConfig
	AI       AIConfig
	Telegram TelegramConfig
	SMTP     SMTPConfig
}

type AppConfig struct {
	Name      string
	Env       string
	WebOrigin string
}

type HTTPConfig struct {
	Host              string
	Port              int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func (c HTTPConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type DBConfig struct {
	URL                string
	MaxConns           int32
	MinConns           int32
	MaxConnLifetime    time.Duration
	MaxConnIdleTime    time.Duration
	HealthcheckTimeout time.Duration
	RunMigrations      bool
}

type LogConfig struct {
	Level  string
	Format string
}

type InviteConfig struct {
	TTL time.Duration
}

type SessionConfig struct {
	CookieName        string
	RefreshCookieName string
	// TTL is the lifetime of the access cookie. Kept short (~15m) so a stolen
	// access cookie has a small blast radius. The frontend silently calls
	// /v1/auth/refresh when it gets a 401, swapping access cookie for a fresh
	// one. Legacy long-lived sessions still validate against the same
	// user_sessions table — they were issued with TTL=30d and continue to
	// authenticate until they expire.
	TTL time.Duration
	// RefreshTTL is the lifetime of the refresh cookie (rotation horizon).
	// Refresh tokens live in the refresh_tokens table and rotate on every
	// /v1/auth/refresh call.
	RefreshTTL time.Duration
	// CookieDomain pins the cookie's Domain attribute in production so the
	// browser scopes the cookie to the apex domain instead of the response
	// host (which can drift between Next.js and the API). Empty in dev.
	CookieDomain string
}

type WorkerConfig struct {
	RecapSweepInterval time.Duration
}

type StorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
	Enabled         bool
}

type AIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Enabled bool
}

type TelegramConfig struct {
	BotToken      string
	BotUsername   string
	WebhookBase   string
	WebhookSecret string
	Enabled       bool
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Enabled  bool
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Name:      getEnv("APP_NAME", "proofforge"),
			Env:       getEnv("APP_ENV", "development"),
			WebOrigin: getEnv("WEB_ORIGIN", "http://localhost:3000"),
		},
		HTTP: HTTPConfig{
			Host:              getEnv("APP_HOST", "0.0.0.0"),
			Port:              mustInt("APP_PORT", 8080),
			ReadHeaderTimeout: mustDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       mustDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      mustDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:       mustDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout:   mustDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		DB: DBConfig{
			URL:                strings.TrimSpace(os.Getenv("DATABASE_URL")),
			MaxConns:           int32(mustInt("DB_MAX_CONNS", 10)),
			MinConns:           int32(mustInt("DB_MIN_CONNS", 1)),
			MaxConnLifetime:    mustDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime:    mustDuration("DB_MAX_CONN_IDLE_TIME", 15*time.Minute),
			HealthcheckTimeout: mustDuration("DB_HEALTHCHECK_TIMEOUT", 3*time.Second),
			RunMigrations:      mustBool("DB_RUN_MIGRATIONS", true),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Invite: InviteConfig{
			TTL: mustDuration("INVITE_TTL", 7*24*time.Hour),
		},
		Session: SessionConfig{
			CookieName:        getEnv("SESSION_COOKIE_NAME", "pf_session"),
			RefreshCookieName: getEnv("REFRESH_COOKIE_NAME", "pf_refresh"),
			// Default access TTL: 15 minutes. The frontend refreshes
			// transparently on 401, so the user never sees this expire unless
			// the refresh token is also gone or revoked.
			TTL: mustDuration("SESSION_TTL", 15*time.Minute),
			// Default refresh TTL: 30 days. After this the user is genuinely
			// asked to log in again.
			RefreshTTL:   mustDuration("REFRESH_TTL", 30*24*time.Hour),
			CookieDomain: getEnv("COOKIE_DOMAIN", ""),
		},
		Worker: WorkerConfig{
			RecapSweepInterval: mustDuration("WORKER_RECAP_SWEEP_INTERVAL", time.Minute),
		},
		Storage: StorageConfig{
			Endpoint:        getEnv("S3_ENDPOINT", ""),
			Region:          getEnv("S3_REGION", "us-east-1"),
			Bucket:          getEnv("S3_BUCKET", ""),
			AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
			UsePathStyle:    mustBool("S3_USE_PATH_STYLE", false),
			Enabled:         getEnv("S3_BUCKET", "") != "",
		},
		AI: AIConfig{
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
			APIKey:  getEnv("OPENAI_API_KEY", ""),
			Model:   getEnv("OPENAI_MODEL", "gpt-4o-mini"),
			Enabled: getEnv("OPENAI_API_KEY", "") != "",
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     mustInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@proof-forge.local"),
			Enabled:  getEnv("SMTP_HOST", "") != "",
		},
		Telegram: TelegramConfig{
			BotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
			BotUsername:   getEnv("TELEGRAM_BOT_USERNAME", ""),
			WebhookBase:   getEnv("TELEGRAM_WEBHOOK_BASE_URL", ""),
			WebhookSecret: getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
			Enabled:       getEnv("TELEGRAM_BOT_TOKEN", "") != "" && getEnv("TELEGRAM_WEBHOOK_BASE_URL", "") != "",
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if c.App.Name == "" {
		errs = append(errs, errors.New("APP_NAME is required"))
	}
	if c.App.WebOrigin == "" {
		errs = append(errs, errors.New("WEB_ORIGIN is required"))
	}
	if c.DB.URL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		errs = append(errs, fmt.Errorf("APP_PORT must be between 1 and 65535, got %d", c.HTTP.Port))
	}
	if c.DB.MaxConns < c.DB.MinConns {
		errs = append(errs, errors.New("DB_MAX_CONNS must be >= DB_MIN_CONNS"))
	}
	if c.Worker.RecapSweepInterval <= 0 {
		errs = append(errs, errors.New("WORKER_RECAP_SWEEP_INTERVAL must be positive"))
	}
	if c.Invite.TTL <= 0 {
		errs = append(errs, errors.New("INVITE_TTL must be positive"))
	}
	if c.Session.CookieName == "" {
		errs = append(errs, errors.New("SESSION_COOKIE_NAME is required"))
	}
	if c.Session.TTL <= 0 {
		errs = append(errs, errors.New("SESSION_TTL must be positive"))
	}
	if c.Session.RefreshTTL <= 0 {
		errs = append(errs, errors.New("REFRESH_TTL must be positive"))
	}
	if c.Session.RefreshTTL <= c.Session.TTL {
		// A refresh token shorter than the access token defeats the whole point
		// of the rotation flow: the frontend would refresh and immediately get
		// rejected on the next request.
		errs = append(errs, errors.New("REFRESH_TTL must be greater than SESSION_TTL"))
	}
	if c.Session.RefreshCookieName == "" {
		errs = append(errs, errors.New("REFRESH_COOKIE_NAME is required"))
	}

	if len(errs) == 0 {
		return nil
	}

	var b strings.Builder
	for i, err := range errs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(err.Error())
	}
	return errors.New(b.String())
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func mustInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("invalid int for %s: %v", key, err))
	}
	return parsed
}

func mustBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("invalid bool for %s: %v", key, err))
	}
	return parsed
}

func mustDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Sprintf("invalid duration for %s: %v", key, err))
	}
	return parsed
}
