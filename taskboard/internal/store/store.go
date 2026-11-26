package store

import (
	"context"
	"errors"

	"taskboard/internal/model"
)

// ErrNotFound is returned when the requested project or task does not exist.
var ErrNotFound = errors.New("not found")

// ErrForeignKey is returned when a write references a missing parent row.
var ErrForeignKey = errors.New("foreign key constraint violated")

// NewTask carries the fields required to create a task.
type NewTask struct {
	ProjectID   int64
	Title       string
	Description string
	Priority    string
}

// Store is the repository interface implemented by each database dialect.
type Store interface {
	Ping(ctx context.Context) error
	ListProjects(ctx context.Context) ([]model.Project, error)
	CreateProject(ctx context.Context, name string) (model.Project, error)
	DeleteProject(ctx context.Context, id int64) error
	ListTasks(ctx context.Context, projectID int64, status, priority string) ([]model.Task, error)
	CreateTask(ctx context.Context, in NewTask) (model.Task, error)
	UpdateTask(ctx context.Context, id int64, patch model.TaskPatch) (model.Task, error)
	DeleteTask(ctx context.Context, id int64) error
	Close() error
}
