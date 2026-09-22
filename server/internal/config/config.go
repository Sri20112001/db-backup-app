package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ENV struct {
	JWTSecretKey     string
	JWTRefreshSecret string
	Port             string
	GRPCPort         string
	EncryptionKey    string
	CORSOrigin       string

	// Database fields
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     int
	DBSSLMode  string
	DBTimeZone string

	// Optional full DSN override (takes precedence in DSN())
	DatabaseURL string
}

var AppEnv *ENV

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, using system environment variables")
	}

	// Parse DB_PORT from string to int
	portInt, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		portInt = 5432
	}

	AppEnv = &ENV{
		JWTSecretKey:     getEnv("JWT_SECRET_KEY", getEnv("JWT_SECRET", "default-secret")),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "default-refresh-secret"),
		Port:             getEnv("PORT", getEnv("Port", "3500")),
		GRPCPort:         getEnv("GRPC_PORT", "9090"),
		EncryptionKey:    getEnv("ENCRYPTION_KEY", ""),
		CORSOrigin:       getEnv("CORS_ORIGIN", "http://localhost:5173"),

		DBHost:      getEnv("DB_HOST", "localhost"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", "homeops"),
		DBPort:      portInt,
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		DBTimeZone:  getEnv("DB_TIMEZONE", "UTC"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

// DSN returns the Postgres DSN for GORM.
// If DATABASE_URL is set, it is used as-is; otherwise it is built
// from the individual DB_* fields.
func (e *ENV) DSN() string {
	if e.DatabaseURL != "" {
		return e.DatabaseURL
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		e.DBHost, e.DBUser, e.DBPassword, e.DBName, e.DBPort, e.DBSSLMode, e.DBTimeZone,
	)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
