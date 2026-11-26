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
	mux.HandleFunc("/api/projects", a.handleProjects)
	mux.HandleFunc("/api/projects/", a.handleProjectByPath)
	mux.HandleFunc("/api/tasks/", a.handleTaskByID)
	return mux
}

type projectCreateRequest struct {
	Name string `json:"name"`
}

type taskCreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

type taskPatchRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if err := a.store.Ping(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
		switch r.Method {
		case http.MethodGet:
			a.listTasks(w, r, id)
		case http.MethodPost:
			a.createTask(w, r, id)
		default:
			methodNotAllowed(w)
		}
	default:
		writeError(w, store.ErrNotFound)
	}
}

func (a *API) listTasks(w http.ResponseWriter, r *http.Request, projectID int64) {
	status := r.URL.Query().Get("status")
	if status != "" {
		if err := model.ValidateStatus(status); err != nil {
			writeError(w, err)
			return
		}
	}
	priority := r.URL.Query().Get("priority")
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
}

func (a *API) createTask(w http.ResponseWriter, r *http.Request, projectID int64) {
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
	description, err := model.ValidateDescription(req.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := model.ValidatePriority(req.Priority); err != nil {
		writeError(w, err)
		return
	}
	t, err := a.store.CreateTask(r.Context(), store.NewTask{
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Priority:    req.Priority,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (a *API) handleTaskByID(w http.ResponseWriter, r *http.Request) {
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
		a.updateTask(w, r, id)
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

func (a *API) updateTask(w http.ResponseWriter, r *http.Request, id int64) {
	var req taskPatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	var patch model.TaskPatch
	if req.Title != nil {
		title, err := model.ValidateTaskTitle(*req.Title)
		if err != nil {
			writeError(w, err)
			return
		}
		patch.Title = &title
	}
	if req.Description != nil {
		description, err := model.ValidateDescription(*req.Description)
		if err != nil {
			writeError(w, err)
			return
		}
		patch.Description = &description
	}
	if req.Status != nil {
		if err := model.ValidateStatus(*req.Status); err != nil {
			writeError(w, err)
			return
		}
		patch.Status = req.Status
	}
	if req.Priority != nil {
		if err := model.ValidatePriority(*req.Priority); err != nil {
			writeError(w, err)
			return
		}
		patch.Priority = req.Priority
	}
	if patch.IsEmpty() {
		writeError(w, fmt.Errorf("%w: no updatable fields supplied", model.ErrValidation))
		return
	}
	t, err := a.store.UpdateTask(r.Context(), id, patch)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id < 1 {
		return 0, store.ErrNotFound
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
