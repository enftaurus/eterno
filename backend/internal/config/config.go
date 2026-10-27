package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultDevJWTSecret = "eterno-dev-only-secret-do-not-use-in-production"
	MinJWTSecretLength  = 32
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
	env := strings.ToLower(getEnv("ENV", "development"))
	isProd := env == "production"

	// 1. Port Validation
	portStr := getEnv("PORT", "8080")
	portNum, err := strconv.Atoi(portStr)
	if err != nil || portNum < 1 || portNum > 65535 {
		return nil, fmt.Errorf("invalid PORT %q: must be integer between 1 and 65535", portStr)
	}

	// 2. Database URL
	dbURL := getEnv("DATABASE_URL", "postgres://eterno:eterno_secret@127.0.0.1:5434/eterno?sslmode=disable")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL must not be empty")
	}

	// 3. Pool Settings
	maxOpen, err := getEnvIntStrict("DB_MAX_OPEN_CONNS", 25)
	if err != nil || maxOpen <= 0 {
		return nil, fmt.Errorf("DB_MAX_OPEN_CONNS must be a positive integer, got %v", maxOpen)
	}

	maxIdle, err := getEnvIntStrict("DB_MAX_IDLE_CONNS", 10)
	if err != nil || maxIdle <= 0 {
		return nil, fmt.Errorf("DB_MAX_IDLE_CONNS must be a positive integer, got %v", maxIdle)
	}
	if maxIdle > maxOpen {
		return nil, fmt.Errorf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)", maxIdle, maxOpen)
	}

	connLifetimeMin, err := getEnvIntStrict("DB_CONN_MAX_LIFETIME_MINUTES", 30)
	if err != nil || connLifetimeMin <= 0 {
		return nil, fmt.Errorf("DB_CONN_MAX_LIFETIME_MINUTES must be a positive integer, got %v", connLifetimeMin)
	}

	// 4. JWT Secret Validation
	jwtSecret := getEnv("JWT_SECRET", "")
	if isProd {
		if jwtSecret == "" {
			return nil, errors.New("JWT_SECRET is required in production environment")
		}
		if jwtSecret == DefaultDevJWTSecret {
			return nil, errors.New("cannot use default development JWT_SECRET in production")
		}
		if len(jwtSecret) < MinJWTSecretLength {
			return nil, fmt.Errorf("production JWT_SECRET must be at least %d characters (got %d)", MinJWTSecretLength, len(jwtSecret))
		}
	} else if jwtSecret == "" {
		jwtSecret = DefaultDevJWTSecret
	}

	jwtExpHours, err := getEnvIntStrict("JWT_EXPIRATION_HOURS", 72)
	if err != nil || jwtExpHours <= 0 {
		return nil, fmt.Errorf("JWT_EXPIRATION_HOURS must be positive, got %v", jwtExpHours)
	}

	// 5. Worker Settings
	workerConcurrency, err := getEnvIntStrict("WORKER_CONCURRENCY", 5)
	if err != nil || workerConcurrency <= 0 {
		return nil, fmt.Errorf("WORKER_CONCURRENCY must be positive, got %v", workerConcurrency)
	}

	workerPollSec, err := getEnvIntStrict("WORKER_POLL_INTERVAL_SECONDS", 5)
	if err != nil || workerPollSec <= 0 {
		return nil, fmt.Errorf("WORKER_POLL_INTERVAL_SECONDS must be positive, got %v", workerPollSec)
	}

	retryMax, err := getEnvIntStrict("SYNC_RETRY_MAX_ATTEMPTS", 5)
	if err != nil || retryMax < 0 {
		return nil, fmt.Errorf("SYNC_RETRY_MAX_ATTEMPTS must be non-negative, got %v", retryMax)
	}

	retryBackoffSec, err := getEnvIntStrict("SYNC_RETRY_BASE_BACKOFF_SECONDS", 2)
	if err != nil || retryBackoffSec <= 0 {
		return nil, fmt.Errorf("SYNC_RETRY_BASE_BACKOFF_SECONDS must be positive, got %v", retryBackoffSec)
	}

	// 6. CORS Origins Validation
	corsOriginsStr := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173")
	rawOrigins := strings.Split(corsOriginsStr, ",")
	corsOrigins := make([]string, 0, len(rawOrigins))
	for _, o := range rawOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed == "" {
			continue
		}
		if isProd && trimmed == "*" {
			return nil, errors.New("wildcard CORS origin '*' is strictly prohibited in production")
		}
		if _, err := url.Parse(trimmed); err != nil {
			return nil, fmt.Errorf("invalid CORS origin URL %q: %w", trimmed, err)
		}
		corsOrigins = append(corsOrigins, trimmed)
	}
	if len(corsOrigins) == 0 {
		return nil, errors.New("at least one CORS allowed origin must be configured")
	}

	// 7. Mock Providers Default & Production Prohibition
	// Defaults to REAL-PROVIDER MODE (false) per Section 2.1
	mockProviders, err := getEnvBoolStrict("MOCK_EXTERNAL_PROVIDERS", false)
	if err != nil {
		return nil, fmt.Errorf("invalid MOCK_EXTERNAL_PROVIDERS configuration: %w", err)
	}

	if isProd && mockProviders {
		return nil, errors.New("MOCK_EXTERNAL_PROVIDERS=true is strictly prohibited in production environment")
	}

	lcEndpoint := getEnv("LEETCODE_GRAPHQL_ENDPOINT", "https://leetcode.com/graphql")
	if _, err := url.Parse(lcEndpoint); err != nil {
		return nil, fmt.Errorf("invalid LEETCODE_GRAPHQL_ENDPOINT URL: %w", err)
	}

	return &Config{
		Port:                      portStr,
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

func getEnvIntStrict(key string, defaultVal int) (int, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return defaultVal, nil
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s=%q is not a valid integer", key, val)
	}
	return intVal, nil
}

func getEnvBoolStrict(key string, defaultVal bool) (bool, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return defaultVal, nil
	}
	lower := strings.ToLower(strings.TrimSpace(val))
	switch lower {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("environment variable %s=%q is not a valid boolean (expected true/false/1/0)", key, val)
	}
}
