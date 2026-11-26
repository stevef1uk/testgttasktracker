package web

import (
	"fmt"
	"io/fs"
	"net/http"

	"taskboard/internal/store"
	webassets "taskboard/web"
)

// NewApp returns the root HTTP handler: the embedded index page at "/" and
// the JSON API mounted under "/api/".
func NewApp(s store.Store) (http.Handler, error) {
	index, err := fs.ReadFile(webassets.Templates, "templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("read index template: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", NewAPI(s))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	})
	return mux, nil
}
