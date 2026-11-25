package config

import (
	"strings"
	"testing"
)

// clearTaskboardEnv unsets every TASKBOARD_* variable for the duration of the test.
// envOr treats empty values as unset, so setting to "" yields the defaults.
func clearTaskboardEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TASKBOARD_DB_DRIVER",
		"TASKBOARD_DB_PATH",
		"TASKBOARD_DATABASE_URL",
		"TASKBOARD_HOST",
		"TASKBOARD_PORT",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	clearTaskboardEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}
	if cfg.DBDriver != "sqlite" {
		t.Errorf("DBDriver = %q, want %q", cfg.DBDriver, "sqlite")
	}
	if cfg.DBPath != "taskboard.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "taskboard.db")
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Host, "127.0.0.1")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	clearTaskboardEnv(t)
	t.Setenv("TASKBOARD_DB_DRIVER", "sqlite")
	t.Setenv("TASKBOARD_DB_PATH", "/tmp/custom.db")
	t.Setenv("TASKBOARD_HOST", "0.0.0.0")
	t.Setenv("TASKBOARD_PORT", "9090")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}
	if cfg.DBDriver != "sqlite" || cfg.DBPath != "/tmp/custom.db" || cfg.Host != "0.0.0.0" || cfg.Port != 9090 {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigPortBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		want    int
	}{
		{name: "zero rejected", raw: "0", wantErr: true},
		{name: "lower bound accepted", raw: "1", want: 1},
		{name: "upper bound accepted", raw: "65535", want: 65535},
		{name: "above upper bound rejected", raw: "65536", wantErr: true},
		{name: "negative rejected", raw: "-1", wantErr: true},
		{name: "non-numeric rejected", raw: "http", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTaskboardEnv(t)
			t.Setenv("TASKBOARD_PORT", tt.raw)

			cfg, err := LoadConfig()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("LoadConfig() with port %q: expected error, got config %+v", tt.raw, cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig() with port %q: unexpected error: %v", tt.raw, err)
			}
			if cfg.Port != tt.want {
				t.Errorf("Port = %d, want %d", cfg.Port, tt.want)
			}
		})
	}
}

func TestLoadConfigRejectsUnknownDriver(t *testing.T) {
	clearTaskboardEnv(t)
	t.Setenv("TASKBOARD_DB_DRIVER", "mysql")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() with driver mysql: expected error, got nil")
	}
}

func TestLoadConfigRejectsPostgresWithoutURL(t *testing.T) {
	clearTaskboardEnv(t)
	t.Setenv("TASKBOARD_DB_DRIVER", "postgres")
	t.Setenv("TASKBOARD_DATABASE_URL", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() with postgres and empty URL: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "TASKBOARD_DATABASE_URL") {
		t.Errorf("error %q should name TASKBOARD_DATABASE_URL", err.Error())
	}
}

func TestLoadConfigAcceptsPostgresWithURL(t *testing.T) {
	clearTaskboardEnv(t)
	const dsn = "postgres://user:secret@localhost:5432/taskboard"
	t.Setenv("TASKBOARD_DB_DRIVER", "postgres")
	t.Setenv("TASKBOARD_DATABASE_URL", dsn)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}
	if cfg.DBDriver != "postgres" || cfg.DatabaseURL != dsn {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigErrorsDoNotEchoURL(t *testing.T) {
	clearTaskboardEnv(t)
	const dsn = "postgres://user:topsecret@host/db"
	t.Setenv("TASKBOARD_DB_DRIVER", "mysql")
	t.Setenv("TASKBOARD_DATABASE_URL", dsn)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
	if strings.Contains(err.Error(), "topsecret") {
		t.Errorf("error %q must not echo the database URL", err.Error())
	}
}
