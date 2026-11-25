package web_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"taskboard/internal/model"
	"taskboard/internal/store"
	"taskboard/internal/web"
)

// serveGET routes a GET request through the real App handler and returns the recorder.
func serveGET(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	app := web.NewApp(nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// --- Shell and static assets (architecture: app_test.go HTML and content types) ---

func TestAppIndexRendersHTMLShell(t *testing.T) {
	rec := serveGET(t, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("GET / Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Errorf("GET / body missing doctype")
	}
	if !strings.Contains(body, "<h1>TaskBoard</h1>") {
		t.Errorf("GET / body missing TaskBoard heading")
	}
}

func TestAppIndexDoesNotServeOtherPaths(t *testing.T) {
	rec := serveGET(t, "/no-such-page")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /no-such-page status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAppStaticStylesheetServed(t *testing.T) {
	rec := serveGET(t, "/static/style.css")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/style.css status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Fatalf("GET /static/style.css Content-Type = %q, want text/css", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("GET /static/style.css body is empty")
	}
}

func TestAppStaticScriptServed(t *testing.T) {
	rec := serveGET(t, "/static/app.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/app.js status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("GET /static/app.js Content-Type = %q, want a JavaScript type", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("GET /static/app.js body is empty")
	}
	if !strings.Contains(rec.Body.String(), "/api/projects") {
		t.Errorf("app.js body missing /api/projects calls")
	}
}

func TestAppStaticMissingFileReturnsNotFound(t *testing.T) {
	rec := serveGET(t, "/static/missing.js")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /static/missing.js status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// --- Task REST API (SPEC sections 4-6 and TEST_PLAN task rows) ---

// newSQLiteApp builds an App over a fresh SQLite database in t.TempDir().
func newSQLiteApp(t *testing.T) (http.Handler, *store.SQLiteStore) {
	t.Helper()
	s, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return web.NewApp(s).Handler(), s
}

// decodeObject parses a JSON object body and fails the test on malformed JSON.
func decodeObject(t *testing.T, body string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode object %q: %v", body, err)
	}
	return out
}

// taskIDs returns the ids of a JSON array of tasks in response order.
func taskIDs(t *testing.T, body string) []float64 {
	t.Helper()
	var rows []map[string]any
	if err := json.Unmarshal([]byte(body), &rows); err != nil {
		t.Fatalf("decode task list %q: %v", body, err)
	}
	ids := make([]float64, 0, len(rows))
	for _, r := range rows {
		id, _ := r["id"].(float64)
		ids = append(ids, id)
	}
	return ids
}

func TestTaskCreateDefaultsAndIgnoresClientFields(t *testing.T) {
	h, _ := newSQLiteApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)

	body := `{"title":"T","id":999,"status":"done","created_at":"2001-01-01T00:00:00Z","updated_at":"2001-01-01T00:00:00Z"}`
	rec := do(t, h, http.MethodPost, "/api/projects/1/tasks", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeObject(t, rec.Body.String())
	if got["status"] != "todo" {
		t.Errorf("status = %v, want todo (client status must be ignored)", got["status"])
	}
	if got["priority"] != "medium" {
		t.Errorf("priority = %v, want medium default", got["priority"])
	}
	if got["id"] != float64(1) {
		t.Errorf("id = %v, want server-assigned 1", got["id"])
	}
	if ca, _ := got["created_at"].(string); strings.HasPrefix(ca, "2001") {
		t.Errorf("created_at = %q, client timestamp must be ignored", ca)
	}
	if ua, _ := got["updated_at"].(string); strings.HasPrefix(ua, "2001") {
		t.Errorf("updated_at = %q, client timestamp must be ignored", ua)
	}
}

func TestTaskPatchUpdatesOnlyProvidedFields(t *testing.T) {
	h, _ := newSQLiteApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"T","description":"D","priority":"low"}`)

	rec := do(t, h, http.MethodPatch, "/api/tasks/1", `{"priority":"high"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	got := decodeObject(t, rec.Body.String())
	want := map[string]any{"title": "T", "description": "D", "status": "todo", "priority": "high"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
}

func TestTaskPatchRejectsEmptyAndNullFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "empty object", body: `{}`},
		{name: "null title", body: `{"title":null}`},
		{name: "null description", body: `{"description":null}`},
		{name: "null status", body: `{"status":null}`},
		{name: "null priority", body: `{"priority":null}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newSQLiteApp(t)
			do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)
			do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"T"}`)
			rec := do(t, h, http.MethodPatch, "/api/tasks/1", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTaskListFiltersAndOrder(t *testing.T) {
	h, _ := newSQLiteApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"a","status":"todo","priority":"low"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"b","priority":"high"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"c","priority":"high"}`)
	do(t, h, http.MethodPatch, "/api/tasks/2", `{"status":"doing"}`)

	cases := []struct {
		name  string
		query string
		want  []float64
	}{
		{name: "unfiltered ascending", query: "", want: []float64{1, 2, 3}},
		{name: "status todo", query: "?status=todo", want: []float64{1, 3}},
		{name: "priority high", query: "?priority=high", want: []float64{2, 3}},
		{name: "status and priority", query: "?status=todo&priority=high", want: []float64{3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, "/api/projects/1/tasks"+tc.query, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			got := taskIDs(t, rec.Body.String())
			if len(got) != len(tc.want) {
				t.Fatalf("ids = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("ids = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestTaskListInvalidFiltersReturnBadRequest(t *testing.T) {
	h, _ := newSQLiteApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)
	for _, q := range []string{"?status=bogus", "?priority=urgent"} {
		if rec := do(t, h, http.MethodGet, "/api/projects/1/tasks"+q, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("GET tasks%s status = %d, want 400", q, rec.Code)
		}
	}
}

func TestTaskMissingEntitiesReturnNotFound(t *testing.T) {
	h, _ := newSQLiteApp(t)
	cases := []struct {
		name, method, path, body string
	}{
		{name: "list tasks of missing project", method: http.MethodGet, path: "/api/projects/999/tasks"},
		{name: "create task in missing project", method: http.MethodPost, path: "/api/projects/999/tasks", body: `{"title":"T"}`},
		{name: "patch missing task", method: http.MethodPatch, path: "/api/tasks/999", body: `{"title":"x"}`},
		{name: "delete missing task", method: http.MethodDelete, path: "/api/tasks/999"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (body %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTaskTitleAndDescriptionLimits(t *testing.T) {
	h, _ := newSQLiteApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"T"}`)

	tooLongTitle := strings.Repeat("a", model.MaxTaskTitleLen+1)
	okTitle := strings.Repeat("a", model.MaxTaskTitleLen)
	tooLongDesc := strings.Repeat("a", model.MaxDescriptionLen+1)
	okDesc := strings.Repeat("a", model.MaxDescriptionLen)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "create 121-rune title rejected", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"` + tooLongTitle + `"}`, want: http.StatusBadRequest},
		{name: "create 1001-rune description rejected", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"T","description":"` + tooLongDesc + `"}`, want: http.StatusBadRequest},
		{name: "create blank title rejected", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"   "}`, want: http.StatusBadRequest},
		{name: "create 120-rune title accepted", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"` + okTitle + `"}`, want: http.StatusCreated},
		{name: "create 1000-rune description accepted", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"T","description":"` + okDesc + `"}`, want: http.StatusCreated},
		{name: "patch 121-rune title rejected", method: http.MethodPatch, path: "/api/tasks/1", body: `{"title":"` + tooLongTitle + `"}`, want: http.StatusBadRequest},
		{name: "patch 1001-rune description rejected", method: http.MethodPatch, path: "/api/tasks/1", body: `{"description":"` + tooLongDesc + `"}`, want: http.StatusBadRequest},
		{name: "patch blank title rejected", method: http.MethodPatch, path: "/api/tasks/1", body: `{"title":"  "}`, want: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

// failingStore wraps a real store and fails selected task operations with a
// driver-like error, so tests can prove the HTTP layer hides SQL text.
type failingStore struct {
	store.Store
	failGetTask    bool
	failListTasks  bool
	failCreateTask bool
	failUpdateTask bool
	failDeleteTask bool
}

var errDriver = errors.New("sql: database is locked: SELECT * FROM tasks WHERE id = ?")

func (f *failingStore) GetTask(ctx context.Context, id int64) (model.Task, error) {
	if f.failGetTask {
		return model.Task{}, errDriver
	}
	return f.Store.GetTask(ctx, id)
}

func (f *failingStore) ListTasks(ctx context.Context, projectID int64, filter store.TaskFilter) ([]model.Task, error) {
	if f.failListTasks {
		return nil, errDriver
	}
	return f.Store.ListTasks(ctx, projectID, filter)
}

func (f *failingStore) CreateTask(ctx context.Context, projectID int64, title, description, status, priority string) (model.Task, error) {
	if f.failCreateTask {
		return model.Task{}, errDriver
	}
	return f.Store.CreateTask(ctx, projectID, title, description, status, priority)
}

func (f *failingStore) UpdateTask(ctx context.Context, id int64, patch store.TaskPatch) (model.Task, error) {
	if f.failUpdateTask {
		return model.Task{}, errDriver
	}
	return f.Store.UpdateTask(ctx, id, patch)
}

func (f *failingStore) DeleteTask(ctx context.Context, id int64) error {
	if f.failDeleteTask {
		return errDriver
	}
	return f.Store.DeleteTask(ctx, id)
}

func TestTaskStorageFailuresReturnGenericInternalError(t *testing.T) {
	base, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "fail.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = base.Close() })
	seed := web.NewApp(base).Handler()
	do(t, seed, http.MethodPost, "/api/projects", `{"name":"P"}`)
	do(t, seed, http.MethodPost, "/api/projects/1/tasks", `{"title":"T"}`)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		fail   *failingStore
	}{
		{name: "GET task", method: http.MethodGet, path: "/api/tasks/1", fail: &failingStore{Store: base, failGetTask: true}},
		{name: "GET task list", method: http.MethodGet, path: "/api/projects/1/tasks", fail: &failingStore{Store: base, failListTasks: true}},
		{name: "POST task", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"T"}`, fail: &failingStore{Store: base, failCreateTask: true}},
		{name: "PATCH task", method: http.MethodPatch, path: "/api/tasks/1", body: `{"title":"U"}`, fail: &failingStore{Store: base, failUpdateTask: true}},
		{name: "DELETE task", method: http.MethodDelete, path: "/api/tasks/1", fail: &failingStore{Store: base, failDeleteTask: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := web.NewApp(tc.fail).Handler()
			rec := do(t, h, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500 (body %s)", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
			got := decodeObject(t, rec.Body.String())
			if got["error"] != "internal error" {
				t.Fatalf("error = %v, want generic \"internal error\"", got["error"])
			}
			lower := strings.ToLower(rec.Body.String())
			for _, leak := range []string{"sql", "select", "database", "locked", "driver"} {
				if strings.Contains(lower, leak) {
					t.Fatalf("response leaks %q: %s", leak, rec.Body.String())
				}
			}
		})
	}
}
