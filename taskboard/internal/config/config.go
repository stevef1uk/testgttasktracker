package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Config holds process settings loaded from environment variables.
type Config struct {
	Driver      string
	SQLitePath  string
	DatabaseURL string
	Host        string
	Port        int
}

// Load reads settings through getenv, applying defaults and validating values.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Driver:     "sqlite",
		SQLitePath: "taskboard.db",
		Host:       "127.0.0.1",
		Port:       8080,
	}

	if v := strings.TrimSpace(getenv("TASKBOARD_DB_DRIVER")); v != "" {
		cfg.Driver = v
	}
	if v := getenv("TASKBOARD_DB_PATH"); v != "" {
		cfg.SQLitePath = v
	}
	if v := strings.TrimSpace(getenv("TASKBOARD_DATABASE_URL")); v != "" {
		cfg.DatabaseURL = v
	}
	if v := strings.TrimSpace(getenv("TASKBOARD_HOST")); v != "" {
		cfg.Host = v
	}
	if v := strings.TrimSpace(getenv("TASKBOARD_PORT")); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("TASKBOARD_PORT must be an integer: %w", err)
		}
		cfg.Port = port
	}

	switch cfg.Driver {
	case "sqlite":
	case "postgres":
		if cfg.DatabaseURL == "" {
			return Config{}, fmt.Errorf("TASKBOARD_DATABASE_URL is required when TASKBOARD_DB_DRIVER is postgres")
		}
		if _, err := url.Parse(cfg.DatabaseURL); err != nil {
			return Config{}, fmt.Errorf("TASKBOARD_DATABASE_URL is not a valid URL")
		}
	default:
		return Config{}, fmt.Errorf("TASKBOARD_DB_DRIVER must be sqlite or postgres, got %q", cfg.Driver)
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("TASKBOARD_PORT must be between 1 and 65535, got %d", cfg.Port)
	}

	return cfg, nil
}

// Redacted returns a log-safe description of the configuration. Passwords in the
// database URL are replaced with a placeholder.
func (c Config) Redacted() string {
	dsn := "(none)"
	if c.Driver == "postgres" && c.DatabaseURL != "" {
		if u, err := url.Parse(c.DatabaseURL); err == nil {
			dsn = u.Redacted()
		} else {
			dsn = "(unparseable)"
		}
	} else if c.Driver == "sqlite" {
		dsn = c.SQLitePath
	}
	return fmt.Sprintf("driver=%s host=%s port=%d database=%s", c.Driver, c.Host, c.Port, dsn)
}
