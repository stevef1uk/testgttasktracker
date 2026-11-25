package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskboard/internal/web"
)

// renderIndex serves GET / through the real router and returns status, content type, and body.
func renderIndex(t *testing.T) (int, string, string) {
	t.Helper()
	app := web.NewApp(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)
	return rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()
}

func TestUIIndexStatusAndContentType(t *testing.T) {
	code, ctype, _ := renderIndex(t)
	if code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", code, http.StatusOK)
	}
	if !strings.HasPrefix(ctype, "text/html") {
		t.Fatalf("GET / Content-Type = %q, want text/html", ctype)
	}
}

func TestUIIndexTitleAndHeading(t *testing.T) {
	_, _, body := renderIndex(t)
	for _, want := range []string{
		"<title>TaskBoard</title>",
		"<h1>TaskBoard</h1>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
}

func TestUIIndexProjectFormElements(t *testing.T) {
	_, _, body := renderIndex(t)
	for _, want := range []string{
		`id="projects-panel"`,
		`<h2>Projects</h2>`,
		`id="project-form"`,
		`id="project-name"`,
		`id="project-create"`,
		`id="project-list"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing project element %q", want)
		}
	}
}

func TestUIIndexTaskFormElements(t *testing.T) {
	_, _, body := renderIndex(t)
	for _, want := range []string{
		`id="tasks-panel"`,
		`<h2>Tasks</h2>`,
		`id="task-form"`,
		`id="task-title"`,
		`id="task-description"`,
		`id="task-status"`,
		`id="task-priority"`,
		`id="task-create"`,
		`id="task-list"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing task element %q", want)
		}
	}
}

func TestUIIndexStatusAndPriorityOptions(t *testing.T) {
	_, _, body := renderIndex(t)
	for _, want := range []string{
		`<option value="todo">todo</option>`,
		`<option value="doing">doing</option>`,
		`<option value="done">done</option>`,
		`<option value="low">low</option>`,
		`<option value="medium" selected>medium</option>`,
		`<option value="high">high</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing option %q", want)
		}
	}
}

func TestUIIndexFilterControls(t *testing.T) {
	_, _, body := renderIndex(t)
	for _, want := range []string{
		`id="status-filter"`,
		`id="filter-status"`,
		`id="filter-priority"`,
		`<option value="">all statuses</option>`,
		`<option value="">all priorities</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing filter element %q", want)
		}
	}
}

func TestUIIndexKanbanColumns(t *testing.T) {
	_, _, body := renderIndex(t)
	for _, want := range []string{
		`id="kanban-board"`,
		`id="list-todo"`,
		`id="list-doing"`,
		`id="list-done"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing kanban column element %q", want)
		}
	}
}

func TestUIIndexLinksAssets(t *testing.T) {
	_, _, body := renderIndex(t)
	for _, want := range []string{
		`href="/static/style.css"`,
		`src="/static/app.js"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing asset reference %q", want)
		}
	}
}
