package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                      string
	DBDriver                  string
	DBDSN                     string
	JWTSecret                 string
	JWTIssuer                 string
	JWTExpiry                 time.Duration
	LoginLimitPerMinute       int
	CertificateLimitPerMinute int
	SimulationLimitPerMinute  int
	ShutdownTimeout           time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:                      env("PORT", "8080"),
		DBDriver:                  env("DB_DRIVER", "postgres"),
		DBDSN:                     env("DB_DSN", "host=localhost user=pki password=pki dbname=pki_rollover port=5432 sslmode=disable"),
		JWTSecret:                 env("JWT_SECRET", "development-only-change-me"),
		JWTIssuer:                 "pki-certificate-rollover-impact",
		JWTExpiry:                 durationEnv("JWT_EXPIRY", 8*time.Hour),
		LoginLimitPerMinute:       intEnv("LOGIN_LIMIT_PER_MINUTE", 30),
		CertificateLimitPerMinute: intEnv("CERT_IMPORT_LIMIT_PER_MINUTE", 30),
		SimulationLimitPerMinute:  intEnv("SIMULATION_LIMIT_PER_MINUTE", 60),
		ShutdownTimeout:           durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	if cfg.Port == "" {
		return Config{}, fmt.Errorf("PORT must not be empty")
	}
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "sqlite" {
		return Config{}, fmt.Errorf("unsupported DB_DRIVER %q", cfg.DBDriver)
	}
	if cfg.DBDSN == "" {
		return Config{}, fmt.Errorf("DB_DSN must not be empty")
	}
	if len(cfg.JWTSecret) < 16 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 16 characters")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(env(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
