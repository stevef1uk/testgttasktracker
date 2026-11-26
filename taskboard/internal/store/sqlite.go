package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"taskboard/internal/model"
)

// sqliteDDL is idempotent schema creation, executed one statement at a time.
var sqliteDDL = []string{
	`CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		priority TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id)`,
}

type sqliteStore struct {
	db *sql.DB
}

type scanner interface {
	Scan(dest ...any) error
}

// OpenSQLite opens (or creates) the SQLite database at path and ensures the schema exists.
// Foreign keys and the busy timeout are set through the DSN so every pooled connection gets them.
func OpenSQLite(ctx context.Context, path string) (Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	for _, stmt := range sqliteDDL {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("create sqlite schema: %w", err)
		}
	}
	return &sqliteStore{db: db}, nil
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func isForeignKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

func scanTask(s scanner) (model.Task, error) {
	var t model.Task
	if err := s.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return model.Task{}, fmt.Errorf("scan task: %w", err)
	}
	return t, nil
}

func (s *sqliteStore) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}
	return nil
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}

func (s *sqliteStore) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, created_at FROM projects ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := []model.Project{}
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

func (s *sqliteStore) CreateProject(ctx context.Context, name string) (model.Project, error) {
	now := utcNow()
	res, err := s.db.ExecContext(ctx, "INSERT INTO projects (name, created_at) VALUES (?, ?)", name, now)
	if err != nil {
		return model.Project{}, fmt.Errorf("create project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Project{}, fmt.Errorf("project id: %w", err)
	}
	return model.Project{ID: id, Name: name, CreatedAt: now}, nil
}

func (s *sqliteStore) DeleteProject(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete project rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteStore) ListTasks(ctx context.Context, projectID int64, status, priority string) ([]model.Task, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM projects WHERE id = ?", projectID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check project: %w", err)
	}

	query := "SELECT id, project_id, title, description, status, priority, created_at, updated_at FROM tasks WHERE project_id = ?"
	args := []any{projectID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if priority != "" {
		query += " AND priority = ?"
		args = append(args, priority)
	}
	query += " ORDER BY id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (s *sqliteStore) getTask(ctx context.Context, id int64) (model.Task, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, project_id, title, description, status, priority, created_at, updated_at FROM tasks WHERE id = ?", id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func (s *sqliteStore) CreateTask(ctx context.Context, in NewTask) (model.Task, error) {
	now := utcNow()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (project_id, title, description, status, priority, created_at, updated_at)
		 VALUES (?, ?, ?, 'todo', ?, ?, ?)`,
		in.ProjectID, in.Title, in.Description, in.Priority, now, now)
	if isForeignKeyError(err) {
		return model.Task{}, ErrForeignKey
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Task{}, fmt.Errorf("task id: %w", err)
	}
	return model.Task{
		ID:          id,
		ProjectID:   in.ProjectID,
		Title:       in.Title,
		Description: in.Description,
		Status:      "todo",
		Priority:    in.Priority,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *sqliteStore) UpdateTask(ctx context.Context, id int64, patch model.TaskPatch) (model.Task, error) {
	t, err := s.getTask(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if patch.Title != nil {
		t.Title = *patch.Title
	}
	if patch.Description != nil {
		t.Description = *patch.Description
	}
	if patch.Status != nil {
		t.Status = *patch.Status
	}
	if patch.Priority != nil {
		t.Priority = *patch.Priority
	}
	now := utcNow()
	_, err = s.db.ExecContext(ctx,
		"UPDATE tasks SET title = ?, description = ?, status = ?, priority = ?, updated_at = ? WHERE id = ?",
		t.Title, t.Description, t.Status, t.Priority, now, id)
	if err != nil {
		return model.Task{}, fmt.Errorf("update task: %w", err)
	}
	t.UpdatedAt = now
	return t, nil
}

func (s *sqliteStore) DeleteTask(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete task rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
