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
