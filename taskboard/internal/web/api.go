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
	mux.HandleFunc("/api/tasks/", a.handleTaskByPath)
	return mux
}

type projectCreateRequest struct {
	Name string `json:"name"`
}

type taskCreateRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Priority    *string `json:"priority"`
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
	case len(parts) == 2 && parts[1] == "tasks":
		a.handleProjectTasks(w, r, id)
	default:
		writeError(w, store.ErrNotFound)
	}
}

// handleProjectTasks serves GET (filtered list) and POST (create) on /api/projects/{id}/tasks.
func (a *API) handleProjectTasks(w http.ResponseWriter, r *http.Request, projectID int64) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		priority := r.URL.Query().Get("priority")
		if status == "all" {
			status = ""
		}
		if priority == "all" {
			priority = ""
		}
		if status != "" {
			if err := model.ValidateStatus(status); err != nil {
				writeError(w, err)
				return
			}
		}
		if priority != "" {
			if err := model.ValidatePriority(priority); err != nil {
				writeError(w, err)
				return
			}
		}
		tasks, err := a.store.ListTasks(r.Context(), projectID, status, priority)
		if err != nil {
			writeError(w, err)
			return
		}
		if tasks == nil {
			tasks = []model.Task{}
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var req taskCreateRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		title, err := model.ValidateTaskTitle(req.Title)
		if err != nil {
			writeError(w, err)
			return
		}
		description := ""
		if req.Description != nil {
			description, err = model.ValidateDescription(*req.Description)
			if err != nil {
				writeError(w, err)
				return
			}
		}
		priority := "medium"
		if req.Priority != nil {
			if err := model.ValidatePriority(*req.Priority); err != nil {
				writeError(w, err)
				return
			}
			priority = *req.Priority
		}
		t, err := a.store.CreateTask(r.Context(), store.NewTask{
			ProjectID:   projectID,
			Title:       title,
			Description: description,
			Priority:    priority,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, t)
	default:
		methodNotAllowed(w)
	}
}

// handleTaskByPath serves PATCH and DELETE on /api/tasks/{id}.
func (a *API) handleTaskByPath(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/tasks/"), "/"), "/")
	if len(parts) != 1 {
		writeError(w, store.ErrNotFound)
		return
	}
	id, err := parseID(parts[0])
	if err != nil {
		writeError(w, err)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		a.handleTaskPatch(w, r, id)
	case http.MethodDelete:
		if err := a.store.DeleteTask(r.Context(), id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

// handleTaskPatch applies a nonempty subset of title, description, status, and
// priority. Unknown properties are ignored; explicit nulls are rejected.
func (a *API) handleTaskPatch(w http.ResponseWriter, r *http.Request, id int64) {
	var raw map[string]json.RawMessage
	if err := decodeJSON(r, &raw); err != nil {
		writeError(w, err)
		return
	}
	var patch model.TaskPatch
	for key, val := range raw {
		switch key {
		case "title", "description", "status", "priority":
		default:
			continue
		}
		s, err := decodePatchString(key, val)
		if err != nil {
			writeError(w, err)
			return
		}
		switch key {
		case "title":
			v, err := model.ValidateTaskTitle(s)
			if err != nil {
				writeError(w, err)
				return
			}
			patch.Title = &v
		case "description":
			v, err := model.ValidateDescription(s)
			if err != nil {
				writeError(w, err)
				return
			}
			patch.Description = &v
		case "status":
			if err := model.ValidateStatus(s); err != nil {
				writeError(w, err)
				return
			}
			patch.Status = &s
		case "priority":
			if err := model.ValidatePriority(s); err != nil {
				writeError(w, err)
				return
			}
			patch.Priority = &s
		}
	}
	if patch.IsEmpty() {
		writeError(w, fmt.Errorf("%w: patch must change at least one field", model.ErrValidation))
		return
	}
	t, err := a.store.UpdateTask(r.Context(), id, patch)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// decodePatchString rejects explicit null and non-string values for a patched field.
func decodePatchString(key string, val json.RawMessage) (string, error) {
	if strings.TrimSpace(string(val)) == "null" {
		return "", fmt.Errorf("%w: %s cannot be null", model.ErrValidation, key)
	}
	var s string
	if err := json.Unmarshal(val, &s); err != nil {
		return "", fmt.Errorf("%w: %s must be a string", model.ErrValidation, key)
	}
	return s, nil
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
