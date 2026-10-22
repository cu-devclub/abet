package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds every setting of the service. All values come from environment variables.
type Config struct {
	Port       string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	LogLevel   string

	InternalAPIToken    string
	SuperAdminEmails    []string
	AllowedEmailDomains []string

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	UploadTmpDir   string

	MaxUploadMB int
	MaxCoverMB  int
	MaxImportMB int

	BuildTimeoutMinutes      int
	ReconcileIntervalSeconds int

	RabbitMQHost        string
	RabbitMQPort        string
	RabbitMQUser        string
	RabbitMQPass        string
	RabbitMQVHost       string
	RabbitMQQueue       string
	RabbitMQStatusQueue string

	InstructorPackageDir string

	EnableSwagger bool
	EnablePprof   bool

	AppEnv       string
	CookieSecure bool
	// DevActAsRequested is the raw DEV_ACT_AS_ENABLED value; use DevActAsEnabled to know whether the feature is really on.
	DevActAsRequested bool
}

// DevActAsEnabled reports whether the dev-only "act as user" feature is on: it needs DEV_ACT_AS_ENABLED=true
// AND APP_ENV=development (D28). Anywhere else it stays off.
func (c *Config) DevActAsEnabled() bool { return c.DevActAsRequested && c.AppEnv == "development" }

// Load reads configuration from the environment, applying defaults.
func Load() *Config {
	return &Config{
		Port:       getEnv("PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "abet"),
		DBPassword: os.Getenv("DB_PASSWORD"), // never trimmed or defaulted: used verbatim
		DBName:     getEnv("DB_NAME", "main_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		LogLevel:   strings.ToLower(getEnv("LOG_LEVEL", "info")),

		InternalAPIToken:    os.Getenv("INTERNAL_API_TOKEN"),
		SuperAdminEmails:    splitList(os.Getenv("SUPER_ADMIN_EMAILS")),
		AllowedEmailDomains: splitList(getEnv("ALLOWED_EMAIL_DOMAINS", "chula.ac.th")),

		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinIOAccessKey: os.Getenv("MINIO_ACCESS_KEY"), // credentials are never trimmed or defaulted: used verbatim
		MinIOSecretKey: os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "abet"),
		MinIOUseSSL:    getEnvBool("MINIO_USE_SSL", false),
		UploadTmpDir:   getEnv("UPLOAD_TMP_DIR", os.TempDir()),

		MaxUploadMB: getEnvInt("MAX_UPLOAD_MB", 300),
		MaxCoverMB:  getEnvInt("MAX_COVER_MB", 20),
		MaxImportMB: getEnvInt("MAX_IMPORT_MB", 5),

		BuildTimeoutMinutes:      getEnvInt("BUILD_TIMEOUT_MINUTES", 60),
		ReconcileIntervalSeconds: getEnvInt("RECONCILE_INTERVAL_SECONDS", 60),

		RabbitMQHost:        getEnv("RABBITMQ_HOST", "rabbitmq"),
		RabbitMQPort:        getEnv("RABBITMQ_PORT", "5672"),
		RabbitMQUser:        getEnv("RABBITMQ_USER", "guest"),
		RabbitMQPass:        getEnv("RABBITMQ_PASS", "guest"),
		RabbitMQVHost:       getEnv("RABBITMQ_VHOST", "/"),
		RabbitMQQueue:       getEnv("RABBITMQ_QUEUE", "course_build_queue"),
		RabbitMQStatusQueue: getEnv("RABBITMQ_STATUS_QUEUE", "course_status_queue"),

		InstructorPackageDir: getEnv("INSTRUCTOR_PACKAGE_DIR", "/app/assets/instructor-package"),

		EnableSwagger: getEnvBool("ENABLE_SWAGGER", false),
		EnablePprof:   getEnvBool("ENABLE_PPROF", false),

		AppEnv:            strings.ToLower(getEnv("APP_ENV", "")),
		CookieSecure:      getEnvBool("COOKIE_SECURE", false),
		DevActAsRequested: getEnvBool("DEV_ACT_AS_ENABLED", false),
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// splitList splits a comma-separated list, trimming and lower-casing every item and dropping empties.
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}
