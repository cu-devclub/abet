package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// DBConfig holds the Postgres connection parameters (kept separate so the password never goes through a URL).
type DBConfig struct {
	Host     string
	Port     uint16
	User     string
	Password string
	Name     string
	SSLMode  string
}

type Config struct {
	Port               string
	DB                 DBConfig
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	AllowedDomains     []string
	SessionSecret      string
	SessionTTL         time.Duration
	CookieSecure       bool
	PublicBaseURL      string
	MainServiceURL     string
	MainServiceTimeout time.Duration
	InternalAPIToken   string
	LogLevel           slog.Level
	EnablePprof        bool
	EnableSwagger      bool
}

const minSessionSecretLen = 32

// Load reads configuration from environment variables; it fails fast on invalid values.
func Load() (*Config, error) {
	publicBaseURL := strings.TrimRight(getEnv("PUBLIC_BASE_URL", "http://localhost:8000"), "/")

	ttlHours, err := strconv.Atoi(getEnv("SESSION_TTL_HOURS", "12"))
	if err != nil || ttlHours <= 0 {
		return nil, fmt.Errorf("SESSION_TTL_HOURS must be a positive integer")
	}

	secret := os.Getenv("SESSION_SECRET")
	if len(secret) < minSessionSecretLen {
		return nil, fmt.Errorf("SESSION_SECRET must be at least %d characters", minSessionSecretLen)
	}

	domains := splitList(getEnv("ALLOWED_EMAIL_DOMAINS", "chula.ac.th"))
	if len(domains) == 0 {
		return nil, fmt.Errorf("ALLOWED_EMAIL_DOMAINS must contain at least one domain")
	}

	dbPort, err := strconv.ParseUint(getEnv("DB_PORT", "5432"), 10, 16)
	if err != nil || dbPort == 0 {
		return nil, fmt.Errorf("DB_PORT must be a valid port number")
	}
	sslMode := getEnv("DB_SSLMODE", "disable")
	switch sslMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return nil, fmt.Errorf("DB_SSLMODE %q is not supported", sslMode)
	}
	db := DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     uint16(dbPort),
		User:     getEnv("DB_USER", "abet"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     getEnv("DB_NAME", "auth_db"),
		SSLMode:  sslMode,
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		DB:                 db,
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", publicBaseURL+"/public/api/v1/auth/google/callback"),
		AllowedDomains:     domains,
		SessionSecret:      secret,
		SessionTTL:         time.Duration(ttlHours) * time.Hour,
		CookieSecure:       getBool("COOKIE_SECURE", false),
		PublicBaseURL:      publicBaseURL,
		MainServiceURL:     strings.TrimRight(getEnv("MAIN_SERVICE_URL", "http://main-service:8080"), "/"),
		MainServiceTimeout: 5 * time.Second,
		InternalAPIToken:   os.Getenv("INTERNAL_API_TOKEN"),
		LogLevel:           parseLogLevel(getEnv("LOG_LEVEL", "info")),
		EnablePprof:        getBool("ENABLE_PPROF", false),
		EnableSwagger:      getBool("ENABLE_SWAGGER", false),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getBool(key string, defaultVal bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "":
		return defaultVal
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseLogLevel(raw string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(raw)))); err != nil {
		return slog.LevelInfo
	}
	return level
}
