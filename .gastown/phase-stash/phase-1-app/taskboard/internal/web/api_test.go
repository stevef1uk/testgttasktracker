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
