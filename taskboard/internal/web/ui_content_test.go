package web_test

import (
	"io/fs"
	"net/http"
	"strings"
	"testing"

	webassets "taskboard/web"
)

// hasUnsafeInnerHTML reports whether source assigns non-empty markup through
// innerHTML. Clearing a node with innerHTML = "" inserts no user text and is allowed.
func hasUnsafeInnerHTML(src string) bool {
	cleaned := strings.ReplaceAll(src, `.innerHTML = "";`, "")
	cleaned = strings.ReplaceAll(cleaned, `.innerHTML = ""`, "")
	return strings.Contains(cleaned, "innerHTML")
}

// TestUIIndexServesHTMLShell checks that GET / returns the rendered HTML shell
// and that it references the vanilla JavaScript client and stylesheet.
func TestUIIndexServesHTMLShell(t *testing.T) {
	h := newTaskTestApp(t)
	rec := taskTestRequest(t, h, http.MethodGet, "/", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("GET / content type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	for _, ref := range []string{"/static/app.js", "/static/style.css"} {
		if !strings.Contains(body, ref) {
			t.Errorf("index shell does not reference %s", ref)
		}
	}
}

// TestUIIndexHasNoUnsafeInnerHTML checks the shell never assigns markup through
// innerHTML, which the architecture forbids for user-provided text.
func TestUIIndexHasNoUnsafeInnerHTML(t *testing.T) {
	h := newTaskTestApp(t)
	rec := taskTestRequest(t, h, http.MethodGet, "/", "")
	if hasUnsafeInnerHTML(rec.Body.String()) {
		t.Errorf("index shell must not assign markup through innerHTML")
	}
}

// TestUIStaticAssetsServed checks that each static asset is served under
// /static/ with an explicit content type.
func TestUIStaticAssetsServed(t *testing.T) {
	cases := []struct {
		path        string
		contentType string
	}{
		{"/static/app.js", "javascript"},
		{"/static/style.css", "text/css"},
	}
	h := newTaskTestApp(t)
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := taskTestRequest(t, h, http.MethodGet, tc.path, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200", tc.path, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, tc.contentType) {
				t.Errorf("GET %s content type = %q, want it to contain %q", tc.path, ct, tc.contentType)
			}
			if rec.Body.Len() == 0 {
				t.Errorf("GET %s body is empty", tc.path)
			}
		})
	}
}

// TestUIEmbeddedStaticAssetsExist checks the static assets are embedded in the
// filesystem exported by the web package and are non-empty.
func TestUIEmbeddedStaticAssetsExist(t *testing.T) {
	for _, name := range []string{"static/app.js", "static/style.css"} {
		data, err := fs.ReadFile(webassets.FS, name)
		if err != nil {
			t.Errorf("read embedded %s: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("embedded %s is empty", name)
		}
	}
}

// TestUIAppJSNeverUsesUnsafeInnerHTML checks the client writes user text only
// via textContent or DOM APIs.
func TestUIAppJSNeverUsesUnsafeInnerHTML(t *testing.T) {
	data, err := fs.ReadFile(webassets.FS, "static/app.js")
	if err != nil {
		t.Fatalf("read embedded static/app.js: %v", err)
	}
	if hasUnsafeInnerHTML(string(data)) {
		t.Errorf("app.js must not assign markup through innerHTML")
	}
}
