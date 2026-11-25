package store_test

import (
	"context"
	"os"
	"testing"

	"taskboard/internal/store"
)

// testPostgresDSN returns the DSN for a dedicated disposable PostgreSQL test
// database, or skips. Set TASKBOARD_TEST_POSTGRES_URL to run these tests.
func testPostgresDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TASKBOARD_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("TASKBOARD_TEST_POSTGRES_URL not set; skipping PostgreSQL integration test")
	}
	return dsn
}

func openPostgresForTest(t *testing.T) *store.PostgresStore {
	t.Helper()
	s, err := store.NewPostgresStore(testPostgresDSN(t))
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestPostgresIntegrationPing(t *testing.T) {
	s := openPostgresForTest(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestPostgresIntegrationSchemaIdempotent(t *testing.T) {
	dsn := testPostgresDSN(t)
	for i := 0; i < 2; i++ {
		s, err := store.NewPostgresStore(dsn)
		if err != nil {
			t.Fatalf("open #%d: %v", i+1, err)
		}
		if err := s.Close(); err != nil {
			t.Fatalf("close #%d: %v", i+1, err)
		}
	}
}

func TestPostgresIntegrationProjectAndTaskRoundTrip(t *testing.T) {
	s := openPostgresForTest(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "pg-integration-"+t.Name())
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	t.Cleanup(func() { _ = s.DeleteProject(ctx, p.ID) })

	task, err := s.CreateTask(ctx, p.ID, "title", "", "", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.Status != store.DefaultTaskStatus || task.Priority != store.DefaultTaskPriority {
		t.Fatalf("defaults: got status=%q priority=%q", task.Status, task.Priority)
	}

	newTitle := "changed"
	updated, err := s.UpdateTask(ctx, task.ID, store.TaskPatch{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Title != newTitle || updated.Status != store.DefaultTaskStatus {
		t.Fatalf("patch: got title=%q status=%q", updated.Title, updated.Status)
	}

	tasks, err := s.ListTasks(ctx, p.ID, store.TaskFilter{Status: store.DefaultTaskStatus})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("ListTasks filter: got %d tasks", len(tasks))
	}
}

func TestPostgresIntegrationDeleteProjectCascades(t *testing.T) {
	s := openPostgresForTest(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "pg-cascade-"+t.Name())
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := s.CreateTask(ctx, p.ID, "t", "", "", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := s.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := s.GetTask(ctx, task.ID); err != store.ErrNotFound {
		t.Fatalf("GetTask after cascade: got %v, want ErrNotFound", err)
	}
}

func TestPostgresIntegrationRejectsUnknownProject(t *testing.T) {
	s := openPostgresForTest(t)
	if _, err := s.CreateTask(context.Background(), 1<<40, "t", "", "", ""); err == nil {
		t.Fatal("CreateTask with unknown project returned no error; want foreign key failure")
	}
}
