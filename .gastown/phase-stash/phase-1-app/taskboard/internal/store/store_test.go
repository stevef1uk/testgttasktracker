package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"taskboard/internal/model"
	"taskboard/internal/store"
)

// newContractStore opens a fresh SQLite-backed Store in a temp directory.
// It is used only to exercise the shared Store contract through the interface.
func newContractStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// openAt opens (or reopens) a SQLite store at path and registers cleanup.
func openAt(t *testing.T, path string) store.Store {
	t.Helper()
	s, err := store.NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestStoreContractMissingRowsReturnErrNotFound(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)
	title := "x"

	tests := []struct {
		name string
		call func() error
	}{
		{name: "GetProject", call: func() error { _, err := s.GetProject(ctx, 999); return err }},
		{name: "GetTask", call: func() error { _, err := s.GetTask(ctx, 999); return err }},
		{name: "UpdateTask", call: func() error {
			_, err := s.UpdateTask(ctx, 999, store.TaskPatch{Title: &title})
			return err
		}},
		{name: "DeleteProject", call: func() error { return s.DeleteProject(ctx, 999) }},
		{name: "DeleteTask", call: func() error { return s.DeleteTask(ctx, 999) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("%s on missing row: got %v, want ErrNotFound", tc.name, err)
			}
		})
	}
}

func TestStoreContractListReturnsNonNilSlices(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if projects == nil {
		t.Fatal("ListProjects on empty store returned nil; want empty slice")
	}

	p, err := s.CreateProject(ctx, "contract-empty-tasks")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	tasks, err := s.ListTasks(ctx, p.ID, store.TaskFilter{})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if tasks == nil {
		t.Fatal("ListTasks on empty project returned nil; want empty slice")
	}
}

// Fresh schema on a new file yields no rows; repeated opens keep existing rows (idempotent init).
func TestSQLiteReinitIsIdempotentAndKeepsRows(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "reinit.db")

	s1 := openAt(t, path)
	projects, err := s1.ListProjects(ctx)
	if err != nil || len(projects) != 0 {
		t.Fatalf("fresh schema: projects=%d err=%v, want 0 rows", len(projects), err)
	}
	p, err := s1.CreateProject(ctx, "keep-me")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if _, err := s1.CreateTask(ctx, p.ID, "kept task", "", "", ""); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	for i := 0; i < 2; i++ {
		s2 := openAt(t, path)
		got, err := s2.GetProject(ctx, p.ID)
		if err != nil || got.Name != "keep-me" {
			t.Fatalf("reopen %d: GetProject = %+v, %v; want keep-me", i, got, err)
		}
		tasks, err := s2.ListTasks(ctx, p.ID, store.TaskFilter{})
		if err != nil || len(tasks) != 1 {
			t.Fatalf("reopen %d: tasks=%d err=%v, want 1", i, len(tasks), err)
		}
	}
}

// Projects and tasks list in ascending ID order.
func TestSQLiteListsInAscendingIDOrder(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)

	var projIDs []int64
	for _, name := range []string{"alpha", "beta", "gamma"} {
		p, err := s.CreateProject(ctx, name)
		if err != nil {
			t.Fatalf("CreateProject %s: %v", name, err)
		}
		projIDs = append(projIDs, p.ID)
	}
	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("ListProjects len = %d, want 3", len(projects))
	}
	for i, p := range projects {
		if p.ID != projIDs[i] {
			t.Fatalf("ListProjects[%d].ID = %d, want %d", i, p.ID, projIDs[i])
		}
		if i > 0 && projects[i-1].ID >= p.ID {
			t.Fatalf("ListProjects not ascending: %d then %d", projects[i-1].ID, p.ID)
		}
	}

	var taskIDs []int64
	for _, title := range []string{"t1", "t2", "t3"} {
		task, err := s.CreateTask(ctx, projIDs[0], title, "", "", "")
		if err != nil {
			t.Fatalf("CreateTask %s: %v", title, err)
		}
		taskIDs = append(taskIDs, task.ID)
	}
	tasks, err := s.ListTasks(ctx, projIDs[0], store.TaskFilter{})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("ListTasks len = %d, want 3", len(tasks))
	}
	for i, task := range tasks {
		if task.ID != taskIDs[i] {
			t.Fatalf("ListTasks[%d].ID = %d, want %d", i, task.ID, taskIDs[i])
		}
	}
}

// Empty status/priority use defaults; timestamps are populated.
func TestSQLiteTaskDefaultsAndTimestamps(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)
	p, err := s.CreateProject(ctx, "defaults")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := s.CreateTask(ctx, p.ID, "defaulted", "", "", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.Status != store.DefaultTaskStatus {
		t.Errorf("Status = %q, want %q", task.Status, store.DefaultTaskStatus)
	}
	if task.Priority != store.DefaultTaskPriority {
		t.Errorf("Priority = %q, want %q", task.Priority, store.DefaultTaskPriority)
	}
	if task.ProjectID != p.ID {
		t.Errorf("ProjectID = %d, want %d", task.ProjectID, p.ID)
	}
	if task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() {
		t.Errorf("timestamps not set: created=%v updated=%v", task.CreatedAt, task.UpdatedAt)
	}
}

// Status, priority, and combined filters apply AND semantics.
func TestSQLiteListTasksFilters(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)
	p, err := s.CreateProject(ctx, "filters")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	fixtures := []struct {
		title, status, priority string
	}{
		{"a", "todo", "low"},
		{"b", "doing", "high"},
		{"c", "doing", "low"},
		{"d", "done", "high"},
	}
	for _, f := range fixtures {
		if _, err := s.CreateTask(ctx, p.ID, f.title, "", f.status, f.priority); err != nil {
			t.Fatalf("CreateTask %s: %v", f.title, err)
		}
	}

	tests := []struct {
		name   string
		filter store.TaskFilter
		want   []string
	}{
		{name: "no filter", filter: store.TaskFilter{}, want: []string{"a", "b", "c", "d"}},
		{name: "status only", filter: store.TaskFilter{Status: "doing"}, want: []string{"b", "c"}},
		{name: "priority only", filter: store.TaskFilter{Priority: "high"}, want: []string{"b", "d"}},
		{name: "status AND priority", filter: store.TaskFilter{Status: "doing", Priority: "low"}, want: []string{"c"}},
		{name: "no match", filter: store.TaskFilter{Status: "done", Priority: "low"}, want: []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.ListTasks(ctx, p.ID, tc.filter)
			if err != nil {
				t.Fatalf("ListTasks: %v", err)
			}
			if got == nil {
				t.Fatal("ListTasks returned nil; want non-nil slice")
			}
			var titles []string
			for _, task := range got {
				titles = append(titles, task.Title)
			}
			if strings.Join(titles, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("titles = %v, want %v", titles, tc.want)
			}
		})
	}
}

// A partial PATCH changes only the non-nil fields.
func TestSQLiteUpdateTaskPreservesOmittedFields(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)
	p, err := s.CreateProject(ctx, "patch")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	orig, err := s.CreateTask(ctx, p.ID, "original", "desc", "todo", "medium")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	newStatus := "doing"
	updated, err := s.UpdateTask(ctx, orig.ID, store.TaskPatch{Status: &newStatus})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Status != "doing" {
		t.Errorf("Status = %q, want doing", updated.Status)
	}
	if updated.Title != "original" || updated.Description != "desc" || updated.Priority != "medium" {
		t.Errorf("omitted fields changed: %+v", updated)
	}

	got, err := s.GetTask(ctx, orig.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != "doing" || got.Title != "original" || got.Description != "desc" || got.Priority != "medium" {
		t.Errorf("persisted task = %+v, want status doing with other fields preserved", got)
	}
}

// Deleting a project removes its tasks.
func TestSQLiteDeleteProjectCascadesToTasks(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)
	p, err := s.CreateProject(ctx, "doomed")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := s.CreateTask(ctx, p.ID, "child", "", "", "")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := s.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := s.GetTask(ctx, task.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetTask after cascade: got %v, want ErrNotFound", err)
	}
	if _, err := s.GetProject(ctx, p.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetProject after delete: got %v, want ErrNotFound", err)
	}
}

// Foreign keys are enforced: a task cannot reference an unknown project.
func TestSQLiteForeignKeyRejectsUnknownProject(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)
	if _, err := s.CreateTask(ctx, 424242, "orphan", "", "", ""); err == nil {
		t.Fatal("CreateTask with unknown project_id succeeded; want foreign key error")
	}
}

// Data written before close is readable after reopening the same file.
func TestSQLiteDataPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "persist.db")

	s1, err := store.NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	p, err := s1.CreateProject(ctx, "persisted")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := s1.CreateTask(ctx, p.ID, "remember", "", "doing", "high")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2 := openAt(t, path)
	gotP, err := s2.GetProject(ctx, p.ID)
	if err != nil || gotP.Name != "persisted" {
		t.Fatalf("GetProject after reopen = %+v, %v", gotP, err)
	}
	gotT, err := s2.GetTask(ctx, task.ID)
	if err != nil || gotT.Title != "remember" || gotT.Status != "doing" || gotT.Priority != "high" {
		t.Fatalf("GetTask after reopen = %+v, %v", gotT, err)
	}
}

// Names are stored and returned literally, including SQL-injection-style text and Unicode.
func TestSQLiteStoresNamesLiterally(t *testing.T) {
	ctx := context.Background()
	s := newContractStore(t)
	names := []string{
		"Robert'); DROP TABLE projects;--",
		"naïve café 日本語",
	}
	for _, name := range names {
		p, err := s.CreateProject(ctx, name)
		if err != nil {
			t.Fatalf("CreateProject(%q): %v", name, err)
		}
		got, err := s.GetProject(ctx, p.ID)
		if err != nil || got.Name != name {
			t.Fatalf("GetProject name = %q, %v; want %q", got.Name, err, name)
		}
	}
	projects, err := s.ListProjects(ctx)
	if err != nil || len(projects) != len(names) {
		t.Fatalf("ListProjects after injection-style names: len=%d err=%v", len(projects), err)
	}
}

// Length limits are counted in Unicode code points, not bytes.
func TestModelLengthBoundariesUseCodePoints(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "80 multibyte code points ok", value: strings.Repeat("é", model.MaxProjectNameLen), wantErr: false},
		{name: "81 multibyte code points rejected", value: strings.Repeat("é", model.MaxProjectNameLen+1), wantErr: true},
		{name: "blank rejected", value: "   ", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := model.ValidateProjectName(tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateProjectName err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
