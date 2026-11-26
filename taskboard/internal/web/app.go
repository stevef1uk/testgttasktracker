package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"path"

	"taskboard/internal/store"
)

// App is the root HTTP application: the index page at "/", static assets under
// "/static/", and the JSON API mounted under "/api/".
type App struct {
	handler http.Handler
}

// ServeHTTP dispatches requests to the application's route tree.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

// NewApp builds the application. The index page is parsed and rendered through
// html/template from templates/index.html; static assets are served from the
// static/ directory of the same filesystem with explicit content types.
func NewApp(s store.Store, assets fs.FS) (*App, error) {
	tmpl, err := template.ParseFS(assets, "templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("parse index template: %w", err)
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, fmt.Errorf("open static assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", NewAPI(s))
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w)
			return
		}
		name := path.Clean(r.URL.Path[len("/static/"):])
		if name == "." || name == "" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(static, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ct := mime.TypeByExtension(path.Ext(name))
		if ct == "" {
			ct = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, nil); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	})
	return &App{handler: mux}, nil
}
