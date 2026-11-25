package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config holds runtime settings read from environment variables.
type Config struct {
	DBDriver    string
	DBPath      string
	DatabaseURL string
	Host        string
	Port        int
}

// LoadConfig reads TASKBOARD_* environment variables and validates them.
func LoadConfig() (Config, error) {
	cfg := Config{
		DBDriver:    envOr("TASKBOARD_DB_DRIVER", "sqlite"),
		DBPath:      envOr("TASKBOARD_DB_PATH", "taskboard.db"),
		DatabaseURL: os.Getenv("TASKBOARD_DATABASE_URL"),
		Host:        envOr("TASKBOARD_HOST", "127.0.0.1"),
		Port:        8080,
	}

	if cfg.DBDriver != "sqlite" && cfg.DBDriver != "postgres" {
		return Config{}, fmt.Errorf("TASKBOARD_DB_DRIVER: unsupported driver %q (want sqlite or postgres)", cfg.DBDriver)
	}

	if cfg.DBDriver == "postgres" && cfg.DatabaseURL == "" {
		// Do not echo any URL value in this error.
		return Config{}, errors.New("TASKBOARD_DATABASE_URL: required when TASKBOARD_DB_DRIVER=postgres")
	}

	if raw := os.Getenv("TASKBOARD_PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("TASKBOARD_PORT: invalid port %q", raw)
		}
		cfg.Port = port
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
