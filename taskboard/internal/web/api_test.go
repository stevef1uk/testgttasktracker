package web_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"taskboard/internal/model"
	"taskboard/internal/store"
	"taskboard/internal/web"
	webassets "taskboard/web"
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

func TestDeleteNotFound(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	tests := []struct {
		name string
		path string
	}{
		{name: "project", path: "/api/projects/42"},
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

// pingFailStore wraps a store and fails Ping, simulating an unavailable database.
type pingFailStore struct {
	store.Store
}

func (pingFailStore) Ping(ctx context.Context) error {
	return errors.New("down")
}

func TestReadyReports200WhenHealthy(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	rr := do(t, h, http.MethodGet, "/api/ready", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["status"] != "ready" || len(got) != 1 {
		t.Fatalf("body = %v, want {\"status\":\"ready\"}", got)
	}
}

func TestReadyReports503WhenPingFails(t *testing.T) {
	h := web.NewAPI(pingFailStore{Store: newFakeStore()})
	rr := do(t, h, http.MethodGet, "/api/ready", "")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["error"] != "database unavailable" || len(got) != 1 {
		t.Fatalf("body = %v, want {\"error\":\"database unavailable\"}", got)
	}
}

func TestDeleteProjectReturns204AndRemovesFromList(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	created := do(t, h, http.MethodPost, "/api/projects", `{"name":"Gone"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", created.Code)
	}
	var p model.Project
	if err := json.Unmarshal(created.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode: %v", err)
	}

	rr := do(t, h, http.MethodDelete, fmt.Sprintf("/api/projects/%d", p.ID), "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("delete body = %q, want empty", rr.Body.String())
	}

	list := do(t, h, http.MethodGet, "/api/projects", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", list.Code)
	}
	var projects []model.Project
	if err := json.Unmarshal(list.Body.Bytes(), &projects); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	for _, got := range projects {
		if got.ID == p.ID {
			t.Fatalf("deleted project %d still listed", p.ID)
		}
	}
}

func TestListProjectsAscendingIDOrder(t *testing.T) {
	h := web.NewAPI(newFakeStore())
	for _, name := range []string{"One", "Two", "Three"} {
		rr := do(t, h, http.MethodPost, "/api/projects", fmt.Sprintf(`{"name":%q}`, name))
		if rr.Code != http.StatusCreated {
			t.Fatalf("create %s status = %d, want 201", name, rr.Code)
		}
	}
	rr := do(t, h, http.MethodGet, "/api/projects", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var projects []model.Project
	if err := json.Unmarshal(rr.Body.Bytes(), &projects); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("len = %d, want 3", len(projects))
	}
	for i := 1; i < len(projects); i++ {
		if projects[i-1].ID >= projects[i].ID {
			t.Fatalf("IDs not ascending: %d then %d", projects[i-1].ID, projects[i].ID)
		}
	}
}

func TestIndexServesRenderedShell(t *testing.T) {
	app, err := web.NewApp(newFakeStore(), webassets.FS)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	rr := do(t, app, http.MethodGet, "/", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content type = %q, want text/html", ct)
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "<html") {
		t.Fatalf("body does not contain rendered index shell: %.200s", rr.Body.String())
	}
}

func TestListProjectsStoreFailureIs500NotEmptyArray(t *testing.T) {
	h := web.NewAPI(failingStore{Store: newFakeStore()})
	rr := do(t, h, http.MethodGet, "/api/projects", "")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got == "[]" {
		t.Fatalf("store failure returned empty array body %q", got)
	}
}
