package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	JWTRefreshSecret string
	Port             string
	GRPCPort         string
	EncryptionKey    string
	CORSOrigin       string
}

func Load() *Config {
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "")
	if jwtRefreshSecret == "" {
		panic("JWT_REFRESH_SECRET environment variable is required")
	}

	encKey := getEnv("ENCRYPTION_KEY", "")
	if encKey == "" {
		panic("ENCRYPTION_KEY environment variable is required")
	}
	if len(encKey) < 32 {
		panic("ENCRYPTION_KEY must be at least 32 characters")
	}

	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=backup_saas port=5432 sslmode=disable"),
		JWTSecret:        jwtSecret,
		JWTRefreshSecret: jwtRefreshSecret,
		Port:             getEnv("PORT", "8080"),
		GRPCPort:         getEnv("GRPC_PORT", "9090"),
		EncryptionKey:    encKey,
		CORSOrigin:       getEnv("CORS_ORIGIN", "http://localhost:7540"),
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
