package main

import (
	"testing"

	"taskboard/internal/config"
)

func TestOpenStoreRejectsUnknownDriver(t *testing.T) {
	_, err := openStore(config.Config{DBDriver: "mysql"})
	if err == nil {
		t.Fatal("openStore with unknown driver: want error, got nil")
	}
}

func TestOpenStoreSQLiteCreatesStore(t *testing.T) {
	s, err := openStore(config.Config{DBDriver: "sqlite", DBPath: t.TempDir() + "/tb.db"})
	if err != nil {
		t.Fatalf("openStore sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Ping(t.Context()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
