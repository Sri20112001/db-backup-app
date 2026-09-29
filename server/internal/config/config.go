package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	JWTRefreshSecret string
	Port             string
	GRPCPort         string
	EncryptionKey    string
	CORSOrigin       string
	// SMTP is optional: empty SMTPHost disables outgoing alert email.
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	// Token lifetimes. Existing sessions keep their issued expiry; new
	// logins pick these up. E.g. 30-day refresh: REFRESH_TOKEN_TTL_DAYS=30.
	AccessTokenTTLMinutes int
	RefreshTokenTTLDays   int
	// TLS is opt-in: set both to serve HTTPS. Empty = plain HTTP (fine on
	// loopback/LAN behind a reverse proxy, not for agents over the internet).
	TLSCertFile string
	TLSKeyFile  string
}

func Load() *Config {
	// Dev convenience: pick up `server/.env` for plain `go run` / binaries
	// started without a process manager. Missing file is fine (air, docker,
	// and systemd-style envs don't need it). Real environment variables
	// always take precedence — godotenv never overrides those.
	_ = godotenv.Load()

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}
	if jwtSecret == "change-me-in-production-32chars!!" {
		panic("JWT_SECRET is set to the default placeholder — set a real secret before deploying")
	}
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "")
	if jwtRefreshSecret == "" {
		panic("JWT_REFRESH_SECRET environment variable is required")
	}
	if jwtRefreshSecret == "change-me-refresh-32chars-secret" {
		panic("JWT_REFRESH_SECRET is set to the default placeholder — set a real secret before deploying")
	}

	encKey := getEnv("ENCRYPTION_KEY", "")
	if encKey == "" {
		panic("ENCRYPTION_KEY environment variable is required")
	}
	if encKey == "change-me-32-byte-encryption-key" {
		panic("ENCRYPTION_KEY is set to the default placeholder — set a real key before deploying")
	}
	if len(encKey) < 32 {
		panic("ENCRYPTION_KEY must be at least 32 characters")
	}

	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=db_backup port=5432 sslmode=disable"),
		JWTSecret:        jwtSecret,
		JWTRefreshSecret: jwtRefreshSecret,
		Port:             getEnv("PORT", "7541"),
		GRPCPort:         getEnv("GRPC_PORT", "9090"),
		EncryptionKey:    encKey,
		CORSOrigin:       getEnv("CORS_ORIGIN", "http://localhost:7540"),
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         getEnvInt("SMTP_PORT", 587),
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPassword:     getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:         getEnv("SMTP_FROM", ""),
		AccessTokenTTLMinutes: getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 60),
		RefreshTokenTTLDays:   getEnvInt("REFRESH_TOKEN_TTL_DAYS", 7),
		TLSCertFile: getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  getEnv("TLS_KEY_FILE", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
