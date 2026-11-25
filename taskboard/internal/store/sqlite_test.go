package store

import (
	"context"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore(%q) error = %v", path, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestNewSQLiteStorePing(t *testing.T) {
	s := openTestStore(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestNewSQLiteStoreCreatesSchema(t *testing.T) {
	s := openTestStore(t)
	for _, table := range []string{"projects", "tasks"} {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not created: %v", table, err)
		}
	}
}

func TestNewSQLiteStoreEnforcesForeignKeys(t *testing.T) {
	s := openTestStore(t)
	var fk int
	if err := s.db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys error = %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestNewSQLiteStoreBusyTimeout(t *testing.T) {
	s := openTestStore(t)
	var ms int
	if err := s.db.QueryRow("PRAGMA busy_timeout").Scan(&ms); err != nil {
		t.Fatalf("PRAGMA busy_timeout error = %v", err)
	}
	if ms != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", ms)
	}
}

func TestSQLiteFreshSchemaIsEmpty(t *testing.T) {
	s := openTestStore(t)
	projects, err := s.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("len(projects) = %d, want 0", len(projects))
	}
}

func TestSQLiteReinitIsIdempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "idem.db")

	s1, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("first NewSQLiteStore() error = %v", err)
	}
	if _, err := s1.CreateProject(ctx, "keep"); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	s2, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("second NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	projects, err := s2.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "keep" {
		t.Errorf("projects after reinit = %+v, want single project %q", projects, "keep")
	}
}

func TestSQLiteProjectsListInAscendingIDOrder(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	names := []string{"alpha", "beta", "gamma"}
	var ids []int64
	for _, n := range names {
		p, err := s.CreateProject(ctx, n)
		if err != nil {
			t.Fatalf("CreateProject(%q) error = %v", n, err)
		}
		ids = append(ids, p.ID)
	}

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != len(names) {
		t.Fatalf("len(projects) = %d, want %d", len(projects), len(names))
	}
	for i, p := range projects {
		if p.ID != ids[i] || p.Name != names[i] {
			t.Errorf("projects[%d] = (%d, %q), want (%d, %q)", i, p.ID, p.Name, ids[i], names[i])
		}
	}
}

func TestSQLiteTaskDefaultsAndUTCTimestamps(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "proj")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := s.CreateTask(ctx, p.ID, "title", "", "", "")
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if task.Status != DefaultTaskStatus {
		t.Errorf("Status = %q, want %q", task.Status, DefaultTaskStatus)
	}
	if task.Priority != DefaultTaskPriority {
		t.Errorf("Priority = %q, want %q", task.Priority, DefaultTaskPriority)
	}
	if task.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want a timestamp")
	}
	if task.CreatedAt.Location().String() != "UTC" {
		t.Errorf("CreatedAt location = %q, want UTC", task.CreatedAt.Location())
	}
	if p.CreatedAt.Location().String() != "UTC" {
		t.Errorf("project CreatedAt location = %q, want UTC", p.CreatedAt.Location())
	}
}

func TestSQLiteListTasksFilters(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "proj")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	fixtures := []struct{ title, status, priority string }{
		{"a", "todo", "high"},
		{"b", "done", "high"},
		{"c", "todo", "low"},
		{"d", "done", "low"},
	}
	for _, f := range fixtures {
		if _, err := s.CreateTask(ctx, p.ID, f.title, "", f.status, f.priority); err != nil {
			t.Fatalf("CreateTask(%q) error = %v", f.title, err)
		}
	}

	tests := []struct {
		name   string
		filter TaskFilter
		want   []string
	}{
		{"no filter", TaskFilter{}, []string{"a", "b", "c", "d"}},
		{"status todo", TaskFilter{Status: "todo"}, []string{"a", "c"}},
		{"priority high", TaskFilter{Priority: "high"}, []string{"a", "b"}},
		{"status and priority", TaskFilter{Status: "done", Priority: "low"}, []string{"d"}},
		{"no match", TaskFilter{Status: "in_progress"}, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.ListTasks(ctx, p.ID, tc.filter)
			if err != nil {
				t.Fatalf("ListTasks() error = %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d (%+v)", len(got), len(tc.want), got)
			}
			for i, task := range got {
				if task.Title != tc.want[i] {
					t.Errorf("got[%d].Title = %q, want %q", i, task.Title, tc.want[i])
				}
			}
		})
	}
}

func TestSQLiteUpdateTaskPreservesOmittedFields(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "proj")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := s.CreateTask(ctx, p.ID, "title", "desc", "todo", "low")
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	status := "done"
	updated, err := s.UpdateTask(ctx, task.ID, TaskPatch{Status: &status})
	if err != nil {
		t.Fatalf("UpdateTask(status) error = %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("Status = %q, want done", updated.Status)
	}
	if updated.Title != "title" || updated.Description != "desc" || updated.Priority != "low" {
		t.Errorf("omitted fields changed: %+v", updated)
	}

	newTitle, newDesc, newPriority := "new", "newdesc", "high"
	updated, err = s.UpdateTask(ctx, task.ID, TaskPatch{
		Title: &newTitle, Description: &newDesc, Status: &status, Priority: &newPriority,
	})
	if err != nil {
		t.Fatalf("UpdateTask(all) error = %v", err)
	}
	if updated.Title != "new" || updated.Description != "newdesc" || updated.Status != "done" || updated.Priority != "high" {
		t.Errorf("full patch not applied: %+v", updated)
	}
	if !updated.CreatedAt.Equal(task.CreatedAt) {
		t.Errorf("CreatedAt changed: %v -> %v", task.CreatedAt, updated.CreatedAt)
	}
}

func TestSQLiteUpdateMissingTaskReturnsNotFound(t *testing.T) {
	s := openTestStore(t)
	title := "x"
	if _, err := s.UpdateTask(context.Background(), 999, TaskPatch{Title: &title}); err != ErrNotFound {
		t.Errorf("UpdateTask(missing) error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteDeleteProjectCascadesToTasks(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "proj")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := s.CreateTask(ctx, p.ID, "t", "", "", "")
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := s.DeleteProject(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if _, err := s.GetTask(ctx, task.ID); err != ErrNotFound {
		t.Errorf("GetTask after cascade error = %v, want ErrNotFound", err)
	}
	if err := s.DeleteProject(ctx, p.ID); err != ErrNotFound {
		t.Errorf("DeleteProject(missing) error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteForeignKeyRejectsUnknownProject(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.CreateTask(context.Background(), 9999, "orphan", "", "", ""); err == nil {
		t.Error("CreateTask with unknown project_id succeeded, want foreign key error")
	}
}

func TestSQLiteDataPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "persist.db")

	s1, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	p, err := s1.CreateProject(ctx, "persisted")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	task, err := s1.CreateTask(ctx, p.ID, "task", "d", "in_progress", "high")
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	s2, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("reopen NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	gotP, err := s2.GetProject(ctx, p.ID)
	if err != nil || gotP.Name != "persisted" {
		t.Errorf("GetProject after reopen = %+v, %v", gotP, err)
	}
	gotT, err := s2.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask after reopen error = %v", err)
	}
	if gotT.Title != "task" || gotT.Status != "in_progress" || gotT.Priority != "high" {
		t.Errorf("GetTask after reopen = %+v", gotT)
	}
}

func TestSQLiteParameterizesHostileInput(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	hostile := "x'); DROP TABLE tasks; --"
	p, err := s.CreateProject(ctx, hostile)
	if err != nil {
		t.Fatalf("CreateProject(hostile) error = %v", err)
	}
	if p.Name != hostile {
		t.Errorf("Name = %q, want literal %q", p.Name, hostile)
	}
	if _, err := s.CreateTask(ctx, p.ID, hostile, hostile, "todo", "low"); err != nil {
		t.Fatalf("CreateTask(hostile) error = %v", err)
	}
	tasks, err := s.ListTasks(ctx, p.ID, TaskFilter{Status: "todo' OR '1'='1"})
	if err != nil {
		t.Fatalf("ListTasks(hostile filter) error = %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("hostile status filter matched %d rows, want 0", len(tasks))
	}
}
