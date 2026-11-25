package store

import (
	"context"
	"errors"

	"taskboard/internal/model"
)

// SchemaSQLite creates the TaskBoard tables for the SQLite driver.
// Foreign keys are enforced per connection via PRAGMA in the SQLite store.
const SchemaSQLite = `
CREATE TABLE IF NOT EXISTS projects (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE TABLE IF NOT EXISTS tasks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
`

// ErrNotFound is returned when a project or task row does not exist.
var ErrNotFound = errors.New("not found")

// Default values applied when a task is created without explicit status or priority.
const (
	DefaultTaskStatus   = "todo"
	DefaultTaskPriority = "medium"
)

// TaskPatch carries optional updates; nil fields are preserved unchanged.
type TaskPatch struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *string
}

// TaskFilter narrows ListTasks; empty fields match every value.
type TaskFilter struct {
	Status   string
	Priority string
}

// Store is the persistence contract shared by the SQLite and PostgreSQL drivers.
// Every method takes a context as its first parameter.
type Store interface {
	// Ping verifies the underlying database connection is usable.
	Ping(ctx context.Context) error
	// Close releases database resources.
	Close() error

	// CreateProject inserts a project and returns the stored row.
	CreateProject(ctx context.Context, name string) (model.Project, error)
	// GetProject returns the project with the given id or ErrNotFound.
	GetProject(ctx context.Context, id int64) (model.Project, error)
	// ListProjects returns all projects ordered by ascending id.
	ListProjects(ctx context.Context) ([]model.Project, error)
	// DeleteProject removes a project and its tasks or returns ErrNotFound.
	DeleteProject(ctx context.Context, id int64) error

	// CreateTask inserts a task; empty status or priority use the defaults.
	CreateTask(ctx context.Context, projectID int64, title, description, status, priority string) (model.Task, error)
	// GetTask returns the task with the given id or ErrNotFound.
	GetTask(ctx context.Context, id int64) (model.Task, error)
	// ListTasks returns the tasks of a project matching filter, ordered by ascending id.
	ListTasks(ctx context.Context, projectID int64, filter TaskFilter) ([]model.Task, error)
	// UpdateTask applies the non-nil fields of patch and returns the stored row.
	UpdateTask(ctx context.Context, id int64, patch TaskPatch) (model.Task, error)
	// DeleteTask removes a task or returns ErrNotFound.
	DeleteTask(ctx context.Context, id int64) error
}
