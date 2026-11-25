package store_test

import (
	"testing"

	"taskboard/internal/store"
)

func TestNewPostgresStoreRejectsUnreachableDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{name: "closed port", dsn: "postgres://tb:tb@127.0.0.1:1/taskboard?sslmode=disable&connect_timeout=1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := store.NewPostgresStore(tc.dsn)
			if err == nil {
				_ = s.Close()
				t.Fatalf("NewPostgresStore(%q) returned no error; want connection failure", tc.dsn)
			}
		})
	}
}
