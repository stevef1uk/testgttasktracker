package web_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"taskboard/internal/model"
	"taskboard/internal/store"
	"taskboard/internal/web"
)

// fakeStore is an in-memory store.Store used to exercise the HTTP layer.
type fakeStore struct {
	mu       sync.Mutex
	projects []model.Project
	tasks    []model.Task
	nextID   int64
}

func newFakeStore() *fakeStore { return &fakeStore{nextID: 1} }

func (f *fakeStore) Ping(ctx context.Context) error { return nil }
func (f *fakeStore) Close() error                   { return nil }

func (f *fakeStore) ListProjects(ctx context.Context) ([]model.Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []model.Project{}
	out = append(out, f.projects...)
	return out, nil
}

func (f *fakeStore) CreateProject(ctx context.Context, name string) (model.Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := model.Project{ID: f.nextID, Name: name, CreatedAt: "2024-01-01T00:00:00Z"}
	f.nextID++
	f.projects = append(f.projects, p)
	return p, nil
}

func (f *fakeStore) DeleteProject(ctx context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, p := range f.projects {
		if p.ID == id {
			f.projects = append(f.projects[:i], f.projects[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}

func (f *fakeStore) ListTasks(ctx context.Context, projectID int64, status, priority string) ([]model.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []model.Task{}
	for _, t := range f.tasks {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeStore) CreateTask(ctx context.Context, in store.NewTask) (model.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	found := false
	for _, p := range f.projects {
		if p.ID == in.ProjectID {
			found = true
		}
	}
	if !found {
		return model.Task{}, store.ErrForeignKey
	}
	t := model.Task{
		ID: f.nextID, ProjectID: in.ProjectID, Title: in.Title,
		Description: in.Description, Status: "todo", Priority: in.Priority,
		CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z",
	}
	f.nextID++
	f.tasks = append(f.tasks, t)
	return t, nil
}

func (f *fakeStore) UpdateTask(ctx context.Context, id int64, patch model.TaskPatch) (model.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.tasks {
		if f.tasks[i].ID == id {
			if patch.Title != nil {
				f.tasks[i].Title = *patch.Title
			}
			if patch.Description != nil {
				f.tasks[i].Description = *patch.Description
			}
			if patch.Status != nil {
				f.tasks[i].Status = *patch.Status
			}
			if patch.Priority != nil {
				f.tasks[i].Priority = *patch.Priority
			}
			return f.tasks[i], nil
		}
	}
	return model.Task{}, store.ErrNotFound
}

func (f *fakeStore) DeleteTask(ctx context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, t := range f.tasks {
		if t.ID == id {
			f.tasks = append(f.tasks[:i], f.tasks[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}

var _ store.Store = (*fakeStore)(nil)

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestHealth(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	rr := do(t, h, http.MethodGet, "/api/health", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestListProjectsEmptyIsArray(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	rr := do(t, h, http.MethodGet, "/api/projects", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "[]" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestCreateProject(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{name: "valid", body: `{"name":"Alpha"}`, wantCode: http.StatusCreated},
		{name: "blank name", body: `{"name":"   "}`, wantCode: http.StatusBadRequest},
		{name: "malformed json", body: `{"name":`, wantCode: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := web.NewAPI(newFakeStore())
			rr := do(t, h, http.MethodPost, "/api/projects", tc.body)
			if rr.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rr.Code, tc.wantCode, rr.Body.String())
			}
			if tc.wantCode == http.StatusCreated {
				var p model.Project
				if err := json.Unmarshal(rr.Body.Bytes(), &p); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if p.Name != "Alpha" || p.ID == 0 {
					t.Fatalf("project = %+v", p)
				}
			}
		})
	}
}

func TestCreateTaskValidation(t *testing.T) {
	fs := newFakeStore()
	_, _ = fs.CreateProject(context.Background(), "Alpha")
	h := web.NewAPI(fs)

	tests := []struct {
		name     string
		path     string
		body     string
		wantCode int
	}{
		{name: "valid", path: "/api/projects/1/tasks", body: `{"title":"Write docs","priority":"high"}`, wantCode: http.StatusCreated},
		{name: "blank title", path: "/api/projects/1/tasks", body: `{"title":"","priority":"low"}`, wantCode: http.StatusBadRequest},
		{name: "bad priority", path: "/api/projects/1/tasks", body: `{"title":"x","priority":"urgent"}`, wantCode: http.StatusBadRequest},
		{name: "missing project", path: "/api/projects/999/tasks", body: `{"title":"x","priority":"low"}`, wantCode: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := do(t, h, http.MethodPost, tc.path, tc.body)
			if rr.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rr.Code, tc.wantCode, rr.Body.String())
			}
		})
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	fs := newFakeStore()
	_, _ = fs.CreateProject(context.Background(), "Alpha")
	_, _ = fs.CreateTask(context.Background(), store.NewTask{ProjectID: 1, Title: "t", Priority: "low"})
	h := web.NewAPI(fs)

	rr := do(t, h, http.MethodPatch, "/api/tasks/2", `{"status":"done"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	var got model.Task
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != "done" {
		t.Fatalf("status field = %q, want done", got.Status)
	}

	rr = do(t, h, http.MethodPatch, "/api/tasks/2", `{"status":"archived"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid status: got %d, want 400", rr.Code)
	}

	rr = do(t, h, http.MethodPatch, "/api/tasks/777", `{"status":"done"}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing task: got %d, want 404", rr.Code)
	}
}

func TestDeleteNotFound(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	tests := []struct {
		name string
		path string
	}{
		{name: "project", path: "/api/projects/42"},
		{name: "task", path: "/api/tasks/42"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := do(t, h, http.MethodDelete, tc.path, "")
			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rr.Code)
			}
		})
	}
}

func TestMalformedIDReturns400(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	tests := []struct {
		name string
		path string
	}{
		{name: "project_not_number", path: "/api/projects/abc"},
		{name: "project_negative", path: "/api/projects/-1"},
		{name: "project_zero", path: "/api/projects/0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := do(t, h, http.MethodDelete, tc.path, "")
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 for malformed integer path %s", rr.Code, tc.path)
			}
		})
	}
}

func TestStoreErrorIsNotLeakedAsPanic(t *testing.T) {
	// A non-sentinel store failure must produce a 500, not a panic.
	h := web.NewAPI(failingStore{store.Store(newFakeStore())})
	rr := do(t, h, http.MethodGet, "/api/projects", "")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
}

// failingStore wraps a store and fails ListProjects with an unexpected error.
type failingStore struct {
	store.Store
}

func (failingStore) ListProjects(ctx context.Context) ([]model.Project, error) {
	return nil, errors.New("boom")
}
