package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"taskboard/internal/model"
	"taskboard/internal/store"
)

// storeTaskPatchStatus builds a TaskPatch that only changes the status field.
func storeTaskPatchStatus(status string) model.TaskPatch {
	return model.TaskPatch{Status: &status}
}

// openTestPostgres connects to the database named by TASKBOARD_TEST_POSTGRES_URL.
// The test is skipped when the variable is unset so the default suite runs without a server.
func openTestPostgres(t *testing.T) store.Store {
	t.Helper()
	dsn := os.Getenv("TASKBOARD_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("TASKBOARD_TEST_POSTGRES_URL not set; skipping Postgres integration test")
	}
	s, err := store.OpenPostgres(context.Background(), dsn)
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestPostgresProjectAndTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	s := openTestPostgres(t)

	if err := s.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	p, err := s.CreateProject(ctx, "integration-project")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID == 0 || p.Name != "integration-project" {
		t.Fatalf("project = %+v", p)
	}

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	found := false
	for _, got := range projects {
		if got.ID == p.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("created project %d not listed", p.ID)
	}

	task, err := s.CreateTask(ctx, store.NewTask{ProjectID: p.ID, Title: "t1", Priority: "high"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.Status != "todo" {
		t.Fatalf("new task status = %q, want todo", task.Status)
	}

	done := "done"
	updated, err := s.UpdateTask(ctx, task.ID, storeTaskPatchStatus(done))
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Status != "done" {
		t.Fatalf("updated status = %q, want done", updated.Status)
	}

	tasks, err := s.ListTasks(ctx, p.ID, "done", "")
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("filtered tasks = %+v", tasks)
	}

	if err := s.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if err := s.DeleteProject(ctx, p.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second DeleteProject err = %v, want ErrNotFound", err)
	}
}

func TestPostgresTaskForeignKey(t *testing.T) {
	ctx := context.Background()
	s := openTestPostgres(t)

	_, err := s.CreateTask(ctx, store.NewTask{ProjectID: 1 << 40, Title: "orphan", Priority: "low"})
	if !errors.Is(err, store.ErrForeignKey) {
		t.Fatalf("CreateTask err = %v, want ErrForeignKey", err)
	}
}
