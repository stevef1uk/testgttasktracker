package web_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"taskboard/internal/model"
	"taskboard/internal/store"
	"taskboard/internal/web"
	webassets "taskboard/web"
)

// newTaskTestApp builds the real application over a SQLite store in a
// temporary directory, as the architecture requires for web tests.
func newTaskTestApp(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "taskboard.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	app, err := web.NewApp(s, webassets.FS)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}

func taskTestRequest(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func taskTestDecode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
}

func taskTestCreateProject(t *testing.T, h http.Handler, name string) int64 {
	t.Helper()
	rec := taskTestRequest(t, h, http.MethodPost, "/api/projects", `{"name":"`+name+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project: status %d body %q", rec.Code, rec.Body.String())
	}
	var p model.Project
	taskTestDecode(t, rec, &p)
	return p.ID
}

func taskTestCreateTask(t *testing.T, h http.Handler, projectID int64, body string) model.Task {
	t.Helper()
	rec := taskTestRequest(t, h, http.MethodPost, fmt.Sprintf("/api/projects/%d/tasks", projectID), body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task: status %d body %q", rec.Code, rec.Body.String())
	}
	var task model.Task
	taskTestDecode(t, rec, &task)
	return task
}

func taskTestAssertError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]string
	taskTestDecode(t, rec, &body)
	if body["error"] == "" {
		t.Fatalf("expected non-empty error message, got %q", rec.Body.String())
	}
}

func TestAppCreateTaskDefaults(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Defaults")

	task := taskTestCreateTask(t, h, pid, `{"title":"Build API"}`)

	if task.ProjectID != pid {
		t.Errorf("project_id = %d, want %d", task.ProjectID, pid)
	}
	if task.Title != "Build API" {
		t.Errorf("title = %q, want %q", task.Title, "Build API")
	}
	if task.Description != "" {
		t.Errorf("description = %q, want empty string", task.Description)
	}
	if task.Status != "todo" {
		t.Errorf("status = %q, want todo", task.Status)
	}
	if task.Priority != "medium" {
		t.Errorf("priority = %q, want medium", task.Priority)
	}
}

func TestAppCreateTaskIgnoresClientServerFields(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Server Owned")

	body := `{"title":"T","status":"done","priority":"high","id":999,"created_at":"1999-01-01T00:00:00Z","updated_at":"1999-01-01T00:00:00Z"}`
	task := taskTestCreateTask(t, h, pid, body)

	if task.Status != "todo" {
		t.Errorf("status = %q, want server-assigned todo", task.Status)
	}
	if task.ID == 999 {
		t.Errorf("client-supplied id was honored")
	}
	if strings.HasPrefix(task.CreatedAt, "1999") || strings.HasPrefix(task.UpdatedAt, "1999") {
		t.Errorf("client-supplied timestamps were honored: %q %q", task.CreatedAt, task.UpdatedAt)
	}
}

func TestAppTimestampsAreExplicitUTC(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "UTC")
	task := taskTestCreateTask(t, h, pid, `{"title":"Clock"}`)

	for name, ts := range map[string]string{"created_at": task.CreatedAt, "updated_at": task.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			t.Errorf("%s = %q is not RFC3339: %v", name, ts, err)
			continue
		}
		if !strings.HasSuffix(ts, "Z") && !strings.HasSuffix(ts, "+00:00") {
			t.Errorf("%s = %q lacks an explicit UTC timezone", name, ts)
		}
	}
}

func TestAppCreateTaskValidation(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Validation")

	cases := []struct {
		name string
		body string
	}{
		{"empty title", `{"title":""}`},
		{"whitespace title", `{"title":"   "}`},
		{"missing title", `{"description":"x"}`},
		{"unknown priority", `{"title":"T","priority":"urgent"}`},
		{"title wrong type", `{"title":5}`},
		{"title too long", `{"title":"` + strings.Repeat("a", 121) + `"}`},
		{"malformed json", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := taskTestRequest(t, h, http.MethodPost, fmt.Sprintf("/api/projects/%d/tasks", pid), tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body %q", rec.Code, rec.Body.String())
			}
			taskTestAssertError(t, rec)
		})
	}
}

func TestAppCreateTaskTitleBoundaryAccepted(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Boundary")

	task := taskTestCreateTask(t, h, pid, `{"title":"`+strings.Repeat("b", 120)+`"}`)
	if n := len([]rune(task.Title)); n != 120 {
		t.Errorf("title rune length = %d, want 120", n)
	}
}

func TestAppCreateTaskMissingProjectIs404(t *testing.T) {
	h := newTaskTestApp(t)
	rec := taskTestRequest(t, h, http.MethodPost, "/api/projects/9999/tasks", `{"title":"Orphan"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %q", rec.Code, rec.Body.String())
	}
	taskTestAssertError(t, rec)
}

func TestAppUpdateTaskFields(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Update")
	task := taskTestCreateTask(t, h, pid, `{"title":"Original","description":"keep","priority":"low"}`)

	rec := taskTestRequest(t, h, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", task.ID),
		`{"title":"Renamed","status":"doing","priority":"high"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %q", rec.Code, rec.Body.String())
	}
	var got model.Task
	taskTestDecode(t, rec, &got)

	if got.Title != "Renamed" {
		t.Errorf("title = %q, want Renamed", got.Title)
	}
	if got.Status != "doing" {
		t.Errorf("status = %q, want doing", got.Status)
	}
	if got.Priority != "high" {
		t.Errorf("priority = %q, want high", got.Priority)
	}
	if got.Description != "keep" {
		t.Errorf("description = %q, want unchanged %q", got.Description, "keep")
	}
	if got.ProjectID != pid {
		t.Errorf("project_id changed to %d", got.ProjectID)
	}
}

func TestAppUpdateTaskValidation(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "PatchValidation")
	task := taskTestCreateTask(t, h, pid, `{"title":"Target"}`)
	path := fmt.Sprintf("/api/tasks/%d", task.ID)

	cases := []struct {
		name string
		body string
	}{
		{"empty object", `{}`},
		{"null title", `{"title":null}`},
		{"unknown status", `{"status":"archived"}`},
		{"unknown priority", `{"priority":"urgent"}`},
		{"blank title", `{"title":"  "}`},
		{"malformed json", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := taskTestRequest(t, h, http.MethodPatch, path, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body %q", rec.Code, rec.Body.String())
			}
			taskTestAssertError(t, rec)
		})
	}
}

func TestAppUpdateMissingTaskIs404(t *testing.T) {
	h := newTaskTestApp(t)
	rec := taskTestRequest(t, h, http.MethodPatch, "/api/tasks/9999", `{"title":"x"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %q", rec.Code, rec.Body.String())
	}
	taskTestAssertError(t, rec)
}

func TestAppDeleteTaskRemovesIt(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Delete")
	task := taskTestCreateTask(t, h, pid, `{"title":"Doomed"}`)
	path := fmt.Sprintf("/api/tasks/%d", task.ID)

	rec := taskTestRequest(t, h, http.MethodDelete, path, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("204 body must be empty, got %q", rec.Body.String())
	}

	list := taskTestRequest(t, h, http.MethodGet, fmt.Sprintf("/api/projects/%d/tasks", pid), "")
	var tasks []model.Task
	taskTestDecode(t, list, &tasks)
	for _, got := range tasks {
		if got.ID == task.ID {
			t.Errorf("deleted task still listed")
		}
	}

	again := taskTestRequest(t, h, http.MethodDelete, path, "")
	if again.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", again.Code)
	}
	taskTestAssertError(t, again)
}

func TestAppListTasksEmptyIsArray(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Empty")

	rec := taskTestRequest(t, h, http.MethodGet, fmt.Sprintf("/api/projects/%d/tasks", pid), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("empty task list body = %q, want []", rec.Body.String())
	}
}

func TestAppListTasksStatusFilter(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Filters")
	todo := taskTestCreateTask(t, h, pid, `{"title":"Todo item"}`)
	doing := taskTestCreateTask(t, h, pid, `{"title":"Doing item"}`)
	taskTestRequest(t, h, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", doing.ID), `{"status":"doing"}`)

	rec := taskTestRequest(t, h, http.MethodGet, fmt.Sprintf("/api/projects/%d/tasks?status=doing", pid), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %q", rec.Code, rec.Body.String())
	}
	var tasks []model.Task
	taskTestDecode(t, rec, &tasks)
	for _, got := range tasks {
		if got.ID == todo.ID {
			t.Errorf("status=doing filter returned todo task %d", todo.ID)
		}
		if got.Status != "doing" {
			t.Errorf("task %d has status %q, want doing", got.ID, got.Status)
		}
	}
}

func TestAppListTasksInvalidFilterIs400(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "BadFilter")

	rec := taskTestRequest(t, h, http.MethodGet, fmt.Sprintf("/api/projects/%d/tasks?status=archived", pid), "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %q", rec.Code, rec.Body.String())
	}
	taskTestAssertError(t, rec)
}

func TestAppListTasksMissingProjectIs404(t *testing.T) {
	h := newTaskTestApp(t)
	rec := taskTestRequest(t, h, http.MethodGet, "/api/projects/9999/tasks", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %q", rec.Code, rec.Body.String())
	}
	taskTestAssertError(t, rec)
}

func TestAppDeleteProjectCascadesTasks(t *testing.T) {
	h := newTaskTestApp(t)
	pid := taskTestCreateProject(t, h, "Cascade")
	task := taskTestCreateTask(t, h, pid, `{"title":"Child"}`)

	if rec := taskTestRequest(t, h, http.MethodDelete, fmt.Sprintf("/api/projects/%d", pid), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete project status = %d, want 204", rec.Code)
	}
	rec := taskTestRequest(t, h, http.MethodPatch, fmt.Sprintf("/api/tasks/%d", task.ID), `{"title":"x"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("orphaned task patch status = %d, want 404", rec.Code)
	}
}
