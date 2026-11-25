package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"taskboard/internal/model"
	"taskboard/internal/store"
)

// projectJSON is the wire shape of a project.
type projectJSON struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// taskJSON is the wire shape of a task.
type taskJSON struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type projectInput struct {
	Name string `json:"name"`
}

type taskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

type taskPatchInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
}

// registerAPI attaches the REST routes for projects and tasks.
func (a *App) registerAPI() {
	a.mux.HandleFunc("GET /api/health", a.health)
	a.mux.HandleFunc("GET /api/ready", a.ready)
	a.mux.HandleFunc("GET /api/projects", a.listProjects)
	a.mux.HandleFunc("POST /api/projects", a.createProject)
	a.mux.HandleFunc("DELETE /api/projects/{id}", a.deleteProject)
	a.mux.HandleFunc("GET /api/projects/{id}/tasks", a.listTasks)
	a.mux.HandleFunc("POST /api/projects/{id}/tasks", a.createTask)
	a.mux.HandleFunc("GET /api/tasks/{id}", a.getTask)
	a.mux.HandleFunc("PATCH /api/tasks/{id}", a.patchTask)
	a.mux.HandleFunc("DELETE /api/tasks/{id}", a.deleteTask)
}

// health reports liveness without touching the database.
func (a *App) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready reports readiness by verifying the database connection.
func (a *App) ready(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// writeError sends a JSON error body with the given status.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// pathID parses the {id} path segment as a positive int64.
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

// storeError maps store sentinel errors to HTTP status codes.
func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

func toProjectJSON(p model.Project) projectJSON {
	return projectJSON{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt}
}

func toTaskJSON(t model.Task) taskJSON {
	return taskJSON{
		ID: t.ID, ProjectID: t.ProjectID, Title: t.Title, Description: t.Description,
		Status: t.Status, Priority: t.Priority, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func (a *App) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := a.store.ListProjects(r.Context())
	if err != nil {
		storeError(w, err)
		return
	}
	out := make([]projectJSON, 0, len(projects))
	for _, p := range projects {
		out = append(out, toProjectJSON(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) createProject(w http.ResponseWriter, r *http.Request) {
	var in projectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := model.NormalizeProjectName(in.Name)
	if err := model.ValidateProjectName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := a.store.CreateProject(r.Context(), name)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectJSON(p))
}

func (a *App) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.store.DeleteProject(r.Context(), id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) listTasks(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := store.TaskFilter{
		Status:   r.URL.Query().Get("status"),
		Priority: r.URL.Query().Get("priority"),
	}
	if filter.Status != "" {
		if err := model.ValidateStatus(filter.Status); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if filter.Priority != "" {
		if err := model.ValidatePriority(filter.Priority); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if _, err := a.store.GetProject(r.Context(), id); err != nil {
		storeError(w, err)
		return
	}
	tasks, err := a.store.ListTasks(r.Context(), id, filter)
	if err != nil {
		storeError(w, err)
		return
	}
	out := make([]taskJSON, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toTaskJSON(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) createTask(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in taskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	title := model.NormalizeTaskTitle(in.Title)
	if err := model.ValidateTaskTitle(title); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := model.ValidateDescription(in.Description); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Client-supplied status is ignored; new tasks always start as todo.
	// Priority defaults to medium when omitted and is validated when supplied.
	priority := in.Priority
	if priority == "" {
		priority = store.DefaultTaskPriority
	}
	if err := model.ValidatePriority(priority); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := a.store.GetProject(r.Context(), projectID); err != nil {
		storeError(w, err)
		return
	}
	t, err := a.store.CreateTask(r.Context(), projectID, title, in.Description, store.DefaultTaskStatus, priority)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTaskJSON(t))
}

// getTask returns a single task by id, or 404 when it does not exist.
func (a *App) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := a.store.GetTask(r.Context(), id)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskJSON(t))
}

func (a *App) patchTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	present := 0
	for _, field := range []string{"title", "description", "status", "priority"} {
		v, ok := raw[field]
		if !ok {
			continue
		}
		present++
		if string(v) == "null" {
			writeError(w, http.StatusBadRequest, field+" cannot be null")
			return
		}
	}
	if present == 0 {
		writeError(w, http.StatusBadRequest, "no updatable fields provided")
		return
	}
	var in taskPatchInput
	if err := json.Unmarshal(data, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	patch := store.TaskPatch{Description: in.Description, Status: in.Status, Priority: in.Priority}
	if in.Title != nil {
		title := model.NormalizeTaskTitle(*in.Title)
		if err := model.ValidateTaskTitle(title); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		patch.Title = &title
	}
	if in.Description != nil {
		if err := model.ValidateDescription(*in.Description); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if in.Status != nil {
		if err := model.ValidateStatus(*in.Status); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if in.Priority != nil {
		if err := model.ValidatePriority(*in.Priority); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	t, err := a.store.UpdateTask(r.Context(), id, patch)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskJSON(t))
}

func (a *App) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.store.DeleteTask(r.Context(), id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
