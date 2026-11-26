package config

import (
	"strings"
	"testing"
)

func mapEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(mapEnv(map[string]string{}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Driver != "sqlite" {
		t.Errorf("Driver = %q, want sqlite", cfg.Driver)
	}
	if cfg.SQLitePath != "taskboard.db" {
		t.Errorf("SQLitePath = %q, want taskboard.db", cfg.SQLitePath)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
}

func TestLoadInvalidDriver(t *testing.T) {
	_, err := Load(mapEnv(map[string]string{"TASKBOARD_DB_DRIVER": "mysql"}))
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
}

func TestLoadPort(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"min valid", "1", false},
		{"max valid", "65535", false},
		{"zero", "0", true},
		{"too large", "65536", true},
		{"not a number", "abc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(mapEnv(map[string]string{"TASKBOARD_PORT": tt.value}))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadPostgresRequiresURL(t *testing.T) {
	_, err := Load(mapEnv(map[string]string{"TASKBOARD_DB_DRIVER": "postgres"}))
	if err == nil {
		t.Fatal("expected error for empty postgres URL")
	}
}

func TestLoadPostgresWithURL(t *testing.T) {
	cfg, err := Load(mapEnv(map[string]string{
		"TASKBOARD_DB_DRIVER":    "postgres",
		"TASKBOARD_DATABASE_URL": "postgres://user:secret@db.example:5432/taskboard?sslmode=require",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Driver != "postgres" {
		t.Errorf("Driver = %q, want postgres", cfg.Driver)
	}
}

func TestRedactedHidesPassword(t *testing.T) {
	cfg, err := Load(mapEnv(map[string]string{
		"TASKBOARD_DB_DRIVER":    "postgres",
		"TASKBOARD_DATABASE_URL": "postgres://user:topsecret@db.example:5432/taskboard?sslmode=require",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.Redacted()
	if strings.Contains(got, "topsecret") {
		t.Fatalf("Redacted() leaks password: %q", got)
	}
}
