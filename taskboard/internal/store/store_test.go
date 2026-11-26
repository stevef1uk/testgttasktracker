package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"taskboard/internal/model"
	"taskboard/internal/store"
)

// openStore opens a fresh SQLite-backed Store in a temp directory.
func openStore(t *testing.T) store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store_test.db")
	s, err := store.OpenSQLite(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestStoreSentinelErrors(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	tests := []struct {
		name    string
		call    func() error
		wantErr error
	}{
		{
			name:    "delete missing project",
			call:    func() error { return s.DeleteProject(ctx, 9999) },
			wantErr: store.ErrNotFound,
		},
		{
			name:    "delete missing task",
			call:    func() error { return s.DeleteTask(ctx, 9999) },
			wantErr: store.ErrNotFound,
		},
		{
			name: "create task with missing project",
			call: func() error {
				_, err := s.CreateTask(ctx, store.NewTask{ProjectID: 9999, Title: "x", Priority: "low"})
				return err
			},
			wantErr: store.ErrForeignKey,
		},
		{
			name: "list tasks for missing project",
			call: func() error {
				_, err := s.ListTasks(ctx, 9999, "", "")
				return err
			},
			wantErr: store.ErrNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestStoreUpdateTaskPartialPatch(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	p, err := s.CreateProject(ctx, "patch-project")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := s.CreateTask(ctx, store.NewTask{ProjectID: p.ID, Title: "orig", Description: "keep", Priority: "low"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	newTitle := "renamed"
	got, err := s.UpdateTask(ctx, task.ID, model.TaskPatch{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if got.Title != "renamed" {
		t.Fatalf("title = %q, want renamed", got.Title)
	}
	if got.Description != "keep" || got.Status != "todo" || got.Priority != "low" {
		t.Fatalf("unpatched fields changed: %+v", got)
	}
}

func TestStoreListTasksFilters(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	p, err := s.CreateProject(ctx, "filter-project")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if _, err := s.CreateTask(ctx, store.NewTask{ProjectID: p.ID, Title: "a", Priority: "high"}); err != nil {
		t.Fatalf("CreateTask a: %v", err)
	}
	if _, err := s.CreateTask(ctx, store.NewTask{ProjectID: p.ID, Title: "b", Priority: "low"}); err != nil {
		t.Fatalf("CreateTask b: %v", err)
	}

	tests := []struct {
		name     string
		priority string
		want     int
	}{
		{name: "no filter", priority: "", want: 2},
		{name: "high only", priority: "high", want: 1},
		{name: "medium none", priority: "medium", want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tasks, err := s.ListTasks(ctx, p.ID, "", tc.priority)
			if err != nil {
				t.Fatalf("ListTasks: %v", err)
			}
			if len(tasks) != tc.want {
				t.Fatalf("len = %d, want %d", len(tasks), tc.want)
			}
		})
	}
}
