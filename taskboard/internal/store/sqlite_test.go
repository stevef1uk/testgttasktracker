package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func openTestSQLite(t *testing.T) Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenSQLite(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSQLiteProjectRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestSQLite(t)

	p, err := s.CreateProject(ctx, "Demo")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID <= 0 || p.Name != "Demo" {
		t.Fatalf("unexpected project: %+v", p)
	}

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != p.ID {
		t.Fatalf("ListProjects = %+v, want one project %d", projects, p.ID)
	}

	if err := s.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if err := s.DeleteProject(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second DeleteProject error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteListProjectsEmptyIsNonNil(t *testing.T) {
	s := openTestSQLite(t)
	projects, err := s.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if projects == nil {
		t.Fatal("ListProjects returned nil slice; want empty slice")
	}
}

func TestSQLiteTaskForeignKeyEnforced(t *testing.T) {
	ctx := context.Background()
	s := openTestSQLite(t)

	_, err := s.CreateTask(ctx, NewTask{ProjectID: 999, Title: "orphan", Priority: "medium"})
	if !errors.Is(err, ErrForeignKey) {
		t.Fatalf("CreateTask with missing project error = %v, want ErrForeignKey", err)
	}
}

func TestSQLiteTaskLifecycleAndCascade(t *testing.T) {
	ctx := context.Background()
	s := openTestSQLite(t)

	p, err := s.CreateProject(ctx, "Demo")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := s.CreateTask(ctx, NewTask{ProjectID: p.ID, Title: "Build API", Description: "Details", Priority: "high"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.Status != "todo" || task.ProjectID != p.ID {
		t.Fatalf("unexpected task: %+v", task)
	}

	tasks, err := s.ListTasks(ctx, p.ID, "", "")
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("ListTasks len = %d, want 1", len(tasks))
	}

	if err := s.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if err := s.DeleteTask(ctx, task.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteTask after cascade error = %v, want ErrNotFound", err)
	}
}

func TestSQLitePingAndReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "reopen.db")

	s, err := OpenSQLite(ctx, path)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if _, err := s.CreateProject(ctx, "Persisted"); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := OpenSQLite(ctx, path)
	if err != nil {
		t.Fatalf("reopen OpenSQLite: %v", err)
	}
	defer s2.Close()
	projects, err := s2.ListProjects(ctx)
	if err != nil || len(projects) != 1 {
		t.Fatalf("after reopen ListProjects = %+v, err %v; want 1 project", projects, err)
	}
}
