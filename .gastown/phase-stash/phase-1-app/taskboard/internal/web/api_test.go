package web_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"taskboard/internal/store"
	"taskboard/internal/web"
)

func newTestApp(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return web.NewApp(s).Handler()
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestAppTaskCreateDefaults: new tasks default to status todo, priority medium,
// empty description; a client-supplied status is ignored.
func TestAppTaskCreateDefaults(t *testing.T) {
	h := newTestApp(t)
	if rec := do(t, h, http.MethodPost, "/api/projects", `{"name":"Defaults"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create project status = %d, want 201", rec.Code)
	}
	rec := do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"Write docs"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`"status":"todo"`, `"priority":"medium"`, `"description":""`, `"project_id":1`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %s: %s", want, body)
		}
	}
	rec = do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"Ignore status","status":"done"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"status":"done"`) || !strings.Contains(rec.Body.String(), `"status":"todo"`) {
		t.Errorf("client-supplied status not ignored: %s", rec.Body.String())
	}
}

// TestAppTaskPatchRejectsEmptyAndNullFields: PATCH needs at least one updatable
// field and must not send explicit nulls; errors are JSON with an error key.
func TestAppTaskPatchRejectsEmptyAndNullFields(t *testing.T) {
	bodies := []string{`{}`, `{"title":null}`, `{"description":null}`, `{"status":null}`, `{"priority":null}`}
	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			h := newTestApp(t)
			do(t, h, http.MethodPost, "/api/projects", `{"name":"Patch"}`)
			do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"Patch me"}`)
			rec := do(t, h, http.MethodPatch, "/api/tasks/1", body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), `"error"`) {
				t.Fatalf("want JSON error body, got %s", rec.Body.String())
			}
		})
	}
}

// TestAppTaskFiltersAreANDCombined: status and priority filters apply together.
func TestAppTaskFiltersAreANDCombined(t *testing.T) {
	h := newTestApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"Filters"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"A","priority":"high"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"B","priority":"low"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"C","priority":"high"}`)

	tests := []struct {
		name    string
		query   string
		want    []string
		notWant []string
	}{
		{name: "priority only", query: "priority=high", want: []string{`"title":"A"`, `"title":"C"`}, notWant: []string{`"title":"B"`}},
		{name: "status AND priority", query: "status=todo&priority=high", want: []string{`"title":"A"`, `"title":"C"`}, notWant: []string{`"title":"B"`}},
		{name: "status AND other priority", query: "status=todo&priority=low", want: []string{`"title":"B"`}, notWant: []string{`"title":"A"`, `"title":"C"`}},
		{name: "AND with no overlap is empty", query: "status=todo&priority=medium", want: nil, notWant: []string{`"title":"A"`, `"title":"B"`, `"title":"C"`}},
		{name: "status AND status mismatch is empty", query: "status=done&priority=high", want: nil, notWant: []string{`"title":"A"`, `"title":"C"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, "/api/projects/1/tasks?"+tc.query, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			body := strings.TrimSpace(rec.Body.String())
			if strings.HasPrefix(body, "null") {
				t.Fatalf("body = %s, want [] not null", body)
			}
			for _, w := range tc.want {
				if !strings.Contains(body, w) {
					t.Errorf("body missing %s: %s", w, body)
				}
			}
			for _, nw := range tc.notWant {
				if strings.Contains(body, nw) {
					t.Errorf("body unexpectedly contains %s: %s", nw, body)
				}
			}
			if len(tc.want) == 0 && body != "[]" {
				t.Errorf("body = %s, want []", body)
			}
		})
	}
}

// TestAppMissingProjectOrTaskReturns404 checks 404 for unknown project and task ids.
func TestAppMissingProjectOrTaskReturns404(t *testing.T) {
	h := newTestApp(t)
	tests := []struct {
		name, method, path, body string
	}{
		{name: "list tasks of missing project", method: http.MethodGet, path: "/api/projects/999999/tasks"},
		{name: "create task in missing project", method: http.MethodPost, path: "/api/projects/999999/tasks", body: `{"title":"x"}`},
		{name: "delete missing project", method: http.MethodDelete, path: "/api/projects/999999"},
		{name: "get missing task", method: http.MethodGet, path: "/api/tasks/999999"},
		{name: "patch missing task", method: http.MethodPatch, path: "/api/tasks/999999", body: `{"title":"x"}`},
		{name: "delete missing task", method: http.MethodDelete, path: "/api/tasks/999999"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestAppLengthLimitsRejected: oversized inputs are rejected with 400 and a JSON error.
func TestAppLengthLimitsRejected(t *testing.T) {
	h := newTestApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"Limits"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"Short"}`)

	huge5k := strings.Repeat("x", 5000)
	huge20k := strings.Repeat("x", 20000)
	tests := []struct {
		name, method, path, body string
	}{
		{name: "project name too long", method: http.MethodPost, path: "/api/projects", body: `{"name":"` + huge5k + `"}`},
		{name: "task title too long", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"` + huge5k + `"}`},
		{name: "task description too long", method: http.MethodPost, path: "/api/projects/1/tasks", body: `{"title":"ok","description":"` + huge20k + `"}`},
		{name: "patch title too long", method: http.MethodPatch, path: "/api/tasks/1", body: `{"title":"` + huge5k + `"}`},
		{name: "patch description too long", method: http.MethodPatch, path: "/api/tasks/1", body: `{"description":"` + huge20k + `"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), `"error"`) {
				t.Fatalf("want JSON error body, got %s", rec.Body.String())
			}
		})
	}
}

// TestAppMissingStaticFileReturns404: a static path with no embedded file returns 404.
func TestAppMissingStaticFileReturns404(t *testing.T) {
	h := newTestApp(t)
	if rec := do(t, h, http.MethodGet, "/static/does-not-exist.css", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing static file status = %d, want 404", rec.Code)
	}
}

func TestAPIProjectListStartsEmptyArray(t *testing.T) {
	h := newTestApp(t)
	rec := do(t, h, http.MethodGet, "/api/projects", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestAPIProjectCreateValidation(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{name: "valid", body: `{"name":"Alpha"}`, wantCode: http.StatusCreated},
		{name: "blank name", body: `{"name":"   "}`, wantCode: http.StatusBadRequest},
		{name: "malformed json", body: `{`, wantCode: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestApp(t)
			rec := do(t, h, http.MethodPost, "/api/projects", tc.body)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

func TestAPIMissingProjectReturnsNotFound(t *testing.T) {
	h := newTestApp(t)
	rec := do(t, h, http.MethodGet, "/api/projects/999/tasks", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAPIInvalidIDReturnsBadRequest(t *testing.T) {
	h := newTestApp(t)
	rec := do(t, h, http.MethodGet, "/api/projects/abc/tasks", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAPITaskCreateAndPatch(t *testing.T) {
	h := newTestApp(t)
	if rec := do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create project: status = %d", rec.Code)
	}

	rec := do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"T"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task: status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"todo"`) {
		t.Fatalf("default status missing: %s", rec.Body.String())
	}

	rec = do(t, h, http.MethodPatch, "/api/tasks/1", `{"status":"done"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch task: status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"done"`) || !strings.Contains(rec.Body.String(), `"title":"T"`) {
		t.Fatalf("patch did not preserve title: %s", rec.Body.String())
	}
}

func TestAPITaskRejectsInvalidEnum(t *testing.T) {
	h := newTestApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)

	// Client-supplied status on create is ignored; the task starts at the default.
	rec := do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"T","status":"bogus"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"todo"`) {
		t.Fatalf("client status not ignored on create: %s", rec.Body.String())
	}

	// Invalid enums are still rejected when they are patched.
	if rec := do(t, h, http.MethodPatch, "/api/tasks/1", `{"status":"bogus"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("patch invalid status: status = %d, want 400", rec.Code)
	}
	if rec := do(t, h, http.MethodPatch, "/api/tasks/1", `{"priority":"urgent"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("patch invalid priority: status = %d, want 400", rec.Code)
	}
}

func TestAPIDeleteProjectCascadesAndReturnsNoContent(t *testing.T) {
	h := newTestApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)
	do(t, h, http.MethodPost, "/api/projects/1/tasks", `{"title":"T"}`)

	if rec := do(t, h, http.MethodDelete, "/api/projects/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete project: status = %d", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/projects/1/tasks", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("task after cascade: status = %d, want 404", rec.Code)
	}
}

// SPEC §8 (1): exact liveness JSON from a healthy store.
func TestAPIHealthReturnsExactJSON(t *testing.T) {
	h := newTestApp(t)
	rec := do(t, h, http.MethodGet, "/api/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("body = %q, want {\"status\":\"ok\"}", got)
	}
}

// SPEC §8 (1): exact readiness JSON against a healthy SQLite store.
func TestAPIReadyReturnsExactJSON(t *testing.T) {
	h := newTestApp(t)
	rec := do(t, h, http.MethodGet, "/api/ready", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ready"}` {
		t.Fatalf("body = %q, want {\"status\":\"ready\"}", got)
	}
}

// Readiness must return 503 with a safe generic body when the store cannot ping.
func TestAPIReadyReturns503WhenPingFails(t *testing.T) {
	s, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "ready.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	h := web.NewApp(s).Handler()
	_ = s.Close()
	rec := do(t, h, http.MethodGet, "/api/ready", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"database unavailable"}` {
		t.Fatalf("body = %q, want {\"error\":\"database unavailable\"}", got)
	}
}

// SPEC §8 (2) and §5: blank, whitespace-only, and overlong (81 code points) names are 400.
func TestAPIProjectNameBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{name: "empty name", body: `{"name":""}`, wantCode: http.StatusBadRequest},
		{name: "whitespace-only name", body: `{"name":" \t  "}`, wantCode: http.StatusBadRequest},
		{name: "81 code points", body: `{"name":"` + strings.Repeat("a", 81) + `"}`, wantCode: http.StatusBadRequest},
		{name: "80 code points", body: `{"name":"` + strings.Repeat("a", 80) + `"}`, wantCode: http.StatusCreated},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestApp(t)
			rec := do(t, h, http.MethodPost, "/api/projects", tc.body)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

// SPEC §8 (2): project create/list/delete round trip changes list state.
func TestAPIProjectRoundTrip(t *testing.T) {
	h := newTestApp(t)
	if rec := do(t, h, http.MethodPost, "/api/projects", `{"name":"Gamma"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create: status = %d body=%s", rec.Code, rec.Body.String())
	}
	rec := do(t, h, http.MethodGet, "/api/projects", "")
	if !strings.Contains(rec.Body.String(), `"name":"Gamma"`) {
		t.Fatalf("list after create missing project: %s", rec.Body.String())
	}
	if rec := do(t, h, http.MethodDelete, "/api/projects/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d", rec.Code)
	}
	rec = do(t, h, http.MethodGet, "/api/projects", "")
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("list after delete = %q, want []", got)
	}
}

// SPEC §8 (2): projects list in ascending ID order.
func TestAPIProjectsListAscendingID(t *testing.T) {
	h := newTestApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"Alpha"}`)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"Beta"}`)
	body := do(t, h, http.MethodGet, "/api/projects", "").Body.String()
	first := strings.Index(body, `"id":1`)
	second := strings.Index(body, `"id":2`)
	if first < 0 || second < 0 || first > second {
		t.Fatalf("projects not in ascending ID order: %s", body)
	}
}

// SPEC §8 (4): 204 responses have empty bodies; missing project delete is 404; malformed ID is 400.
func TestAPIDeleteProjectStatusAndBodies(t *testing.T) {
	h := newTestApp(t)
	do(t, h, http.MethodPost, "/api/projects", `{"name":"P"}`)
	if rec := do(t, h, http.MethodDelete, "/api/projects/1", ""); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete: status = %d body-len = %d, want 204 and empty body", rec.Code, rec.Body.Len())
	}
	if rec := do(t, h, http.MethodDelete, "/api/projects/999", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete missing: status = %d, want 404", rec.Code)
	}
	if rec := do(t, h, http.MethodDelete, "/api/projects/abc", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("delete malformed id: status = %d, want 400", rec.Code)
	}
}

func TestAPIIndexServesHTML(t *testing.T) {
	h := newTestApp(t)
	rec := do(t, h, http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `id="task-list"`) {
		t.Fatal("index.html missing task-list element")
	}
}
