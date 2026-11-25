package web

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"taskboard/internal/store"
	embedweb "taskboard/web"
)

// App wires the HTTP router to the persistence layer.
type App struct {
	store store.Store
	mux   *http.ServeMux
}

// NewApp builds an App backed by s and registers the page routes.
func NewApp(s store.Store) *App {
	a := &App{store: s, mux: http.NewServeMux()}
	a.mux.HandleFunc("GET /{$}", a.handleIndex)
	if staticFS, err := fs.Sub(embedweb.FS, "static"); err == nil {
		a.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
	}
	a.registerAPI()
	return a
}

// Handler returns the root http.Handler for the application.
func (a *App) Handler() http.Handler {
	return a.mux
}

// handleIndex serves the embedded single-page template.
func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(embedweb.FS, "templates/index.html")
	if err != nil {
		http.Error(w, "template unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// writeJSON encodes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
