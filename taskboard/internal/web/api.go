package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"taskboard/internal/model"
	"taskboard/internal/store"
)

// API serves the JSON REST interface under /api.
type API struct {
	store store.Store
}

// NewAPI returns an http.Handler exposing the taskboard JSON API.
func NewAPI(s store.Store) http.Handler {
	a := &API{store: s}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/ready", a.handleReady)
	mux.HandleFunc("/api/projects", a.handleProjects)
	mux.HandleFunc("/api/projects/", a.handleProjectByPath)
	return mux
}

type projectCreateRequest struct {
	Name string `json:"name"`
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReady reports 200 when the store answers a ping and 503 otherwise.
func (a *API) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if err := a.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects, err := a.store.ListProjects(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		if projects == nil {
			projects = []model.Project{}
		}
		writeJSON(w, http.StatusOK, projects)
	case http.MethodPost:
		var req projectCreateRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		name, err := model.ValidateProjectName(req.Name)
		if err != nil {
			writeError(w, err)
			return
		}
		p, err := a.store.CreateProject(r.Context(), name)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, p)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleProjectByPath(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/projects/"), "/"), "/")
	id, err := parseID(parts[0])
	if err != nil {
		writeError(w, err)
		return
	}
	switch {
	case len(parts) == 1:
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		if err := a.store.DeleteProject(r.Context(), id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, store.ErrNotFound)
	}
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("%w: invalid id %q: must be positive integer", model.ErrValidation, s)
	}
	return id, nil
}

func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("%w: invalid JSON body: %v", model.ErrValidation, err)
	}
	return nil
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, model.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, store.ErrNotFound), errors.Is(err, store.ErrForeignKey):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := statusFor(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "internal server error"
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
