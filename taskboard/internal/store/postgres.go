package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"taskboard/internal/model"
)

// pgForeignKeyViolation is the SQLSTATE code for a foreign key violation.
const pgForeignKeyViolation = "23503"

// postgresDDL is idempotent schema creation, executed one statement at a time.
var postgresDDL = []string{
	`CREATE TABLE IF NOT EXISTS projects (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS tasks (
		id BIGSERIAL PRIMARY KEY,
		project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		priority TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id)`,
}

type postgresStore struct {
	db *sql.DB
}

// OpenPostgres opens a connection pool to the given DSN and ensures the schema exists.
func OpenPostgres(ctx context.Context, dsn string) (Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	// Serialize schema creation across concurrent starts: the advisory xact lock
	// is held until commit, so the idempotent DDL runs under one transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("begin schema tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext('taskboard_schema'))"); err != nil {
		_ = tx.Rollback()
		_ = db.Close()
		return nil, fmt.Errorf("acquire schema lock: %w", err)
	}
	for _, stmt := range postgresDDL {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			_ = db.Close()
			return nil, fmt.Errorf("create postgres schema: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("commit postgres schema: %w", err)
	}
	return &postgresStore{db: db}, nil
}

func isPgForeignKeyError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation
}

func (s *postgresStore) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (s *postgresStore) Close() error {
	return s.db.Close()
}

func (s *postgresStore) ListProjects(ctx context.Context) ([]model.Project, error) {
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

func (s *postgresStore) CreateProject(ctx context.Context, name string) (model.Project, error) {
	now := utcNow()
	var id int64
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO projects (name, created_at) VALUES ($1, $2) RETURNING id", name, now).Scan(&id)
	if err != nil {
		return model.Project{}, fmt.Errorf("create project: %w", err)
	}
	return model.Project{ID: id, Name: name, CreatedAt: now}, nil
}

func (s *postgresStore) DeleteProject(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id = $1", id)
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

func (s *postgresStore) ListTasks(ctx context.Context, projectID int64, status, priority string) ([]model.Task, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM projects WHERE id = $1", projectID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check project: %w", err)
	}

	query := "SELECT id, project_id, title, description, status, priority, created_at, updated_at FROM tasks WHERE project_id = $1"
	args := []any{projectID}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if priority != "" {
		args = append(args, priority)
		query += fmt.Sprintf(" AND priority = $%d", len(args))
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

func (s *postgresStore) getTask(ctx context.Context, id int64) (model.Task, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, project_id, title, description, status, priority, created_at, updated_at FROM tasks WHERE id = $1", id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func (s *postgresStore) CreateTask(ctx context.Context, in NewTask) (model.Task, error) {
	now := utcNow()
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO tasks (project_id, title, description, status, priority, created_at, updated_at)
		 VALUES ($1, $2, $3, 'todo', $4, $5, $6) RETURNING id`,
		in.ProjectID, in.Title, in.Description, in.Priority, now, now).Scan(&id)
	if isPgForeignKeyError(err) {
		return model.Task{}, ErrForeignKey
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("create task: %w", err)
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

func (s *postgresStore) UpdateTask(ctx context.Context, id int64, patch model.TaskPatch) (model.Task, error) {
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
		"UPDATE tasks SET title = $1, description = $2, status = $3, priority = $4, updated_at = $5 WHERE id = $6",
		t.Title, t.Description, t.Status, t.Priority, now, id)
	if err != nil {
		return model.Task{}, fmt.Errorf("update task: %w", err)
	}
	t.UpdatedAt = now
	return t, nil
}

func (s *postgresStore) DeleteTask(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = $1", id)
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
