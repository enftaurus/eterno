package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                      string
	Env                       string
	DatabaseURL               string
	DBMaxOpenConns            int
	DBMaxIdleConns            int
	DBConnMaxLifetime         time.Duration
	JWTSecret                 string
	JWTExpirationHours        int
	CORSAllowedOrigins        []string
	WorkerConcurrency         int
	WorkerPollIntervalSeconds int
	SyncRetryMaxAttempts      int
	SyncRetryBaseBackoffSec   int
	LeetCodeGraphQLEndpoint   string
	MockExternalProviders     bool
}

func Load() (*Config, error) {
	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")
	dbURL := getEnv("DATABASE_URL", "postgres://eterno:eterno_secret@127.0.0.1:5434/eterno?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "eterno-super-secret-jwt-key-2026-production-ready")

	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	connLifetimeMin := getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)
	jwtExpHours := getEnvInt("JWT_EXPIRATION_HOURS", 72)
	workerConcurrency := getEnvInt("WORKER_CONCURRENCY", 5)
	workerPollSec := getEnvInt("WORKER_POLL_INTERVAL_SECONDS", 5)
	retryMax := getEnvInt("SYNC_RETRY_MAX_ATTEMPTS", 5)
	retryBackoffSec := getEnvInt("SYNC_RETRY_BASE_BACKOFF_SECONDS", 2)

	corsOriginsStr := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173")
	corsOrigins := strings.Split(corsOriginsStr, ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	mockProviders := getEnvBool("MOCK_EXTERNAL_PROVIDERS", true)
	lcEndpoint := getEnv("LEETCODE_GRAPHQL_ENDPOINT", "https://leetcode.com/graphql")

	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must not be empty")
	}

	return &Config{
		Port:                      port,
		Env:                       env,
		DatabaseURL:               dbURL,
		DBMaxOpenConns:            maxOpen,
		DBMaxIdleConns:            maxIdle,
		DBConnMaxLifetime:         time.Duration(connLifetimeMin) * time.Minute,
		JWTSecret:                 jwtSecret,
		JWTExpirationHours:        jwtExpHours,
		CORSAllowedOrigins:        corsOrigins,
		WorkerConcurrency:         workerConcurrency,
		WorkerPollIntervalSeconds: workerPollSec,
		SyncRetryMaxAttempts:      retryMax,
		SyncRetryBaseBackoffSec:   retryBackoffSec,
		LeetCodeGraphQLEndpoint:   lcEndpoint,
		MockExternalProviders:     mockProviders,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		lower := strings.ToLower(val)
		return lower == "true" || lower == "1" || lower == "yes"
	}
	return defaultVal
}
