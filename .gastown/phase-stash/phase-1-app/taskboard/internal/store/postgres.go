package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"taskboard/internal/model"
)

// postgresSchemaLockKey serializes schema creation across simultaneous starts.
const postgresSchemaLockKey int64 = 746211904231

// SchemaPostgres creates the TaskBoard tables for the PostgreSQL driver.
// It is applied inside a transaction guarded by pg_advisory_xact_lock.
const SchemaPostgres = `
CREATE TABLE IF NOT EXISTS projects (
	id BIGSERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS tasks (
	id BIGSERIAL PRIMARY KEY,
	project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
`

// pgTaskColumns is the fixed column list used by every task SELECT and RETURNING clause.
const pgTaskColumns = `id, project_id, title, description, status, priority, created_at, updated_at`

// PostgresStore implements persistence on top of PostgreSQL via the pgx stdlib driver.
type PostgresStore struct {
	db *sql.DB
}

// Compile-time check that PostgresStore satisfies Store.
var _ Store = (*PostgresStore)(nil)

// NewPostgresStore opens a PostgreSQL connection pool from dsn, verifies connectivity,
// and applies SchemaPostgres under a transactional advisory lock.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := applyPostgresSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

// applyPostgresSchema creates tables inside a transaction holding an advisory lock,
// so concurrent starts cannot race on CREATE TABLE. Re-running is idempotent.
func applyPostgresSchema(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, postgresSchemaLockKey); err != nil {
		return fmt.Errorf("acquire schema lock: %w", err)
	}
	if _, err := tx.ExecContext(ctx, SchemaPostgres); err != nil {
		return fmt.Errorf("apply postgres schema: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit postgres schema: %w", err)
	}
	return nil
}

// Ping verifies the database connection is usable.
func (s *PostgresStore) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

// Close releases the connection pool.
func (s *PostgresStore) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close postgres: %w", err)
	}
	return nil
}

// CreateProject inserts a project and returns the stored row.
func (s *PostgresStore) CreateProject(ctx context.Context, name string) (model.Project, error) {
	row := s.db.QueryRowContext(ctx,
		`INSERT INTO projects (name) VALUES ($1) RETURNING id, name, created_at`, name)
	p, err := scanPGProject(row)
	if err != nil {
		return model.Project{}, fmt.Errorf("insert project: %w", err)
	}
	return p, nil
}

// GetProject returns the project with the given id or ErrNotFound.
func (s *PostgresStore) GetProject(ctx context.Context, id int64) (model.Project, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM projects WHERE id = $1`, id)
	p, err := scanPGProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Project{}, ErrNotFound
	}
	if err != nil {
		return model.Project{}, fmt.Errorf("select project: %w", err)
	}
	return p, nil
}

// ListProjects returns all projects ordered by ascending id.
func (s *PostgresStore) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, created_at FROM projects ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := []model.Project{}
	for rows.Next() {
		p, err := scanPGProject(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

// DeleteProject removes a project; its tasks are removed by ON DELETE CASCADE.
func (s *PostgresStore) DeleteProject(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return requireAffected(res)
}

// CreateTask inserts a task. Empty status or priority fall back to the defaults.
func (s *PostgresStore) CreateTask(ctx context.Context, projectID int64, title, description, status, priority string) (model.Task, error) {
	if status == "" {
		status = DefaultTaskStatus
	}
	if priority == "" {
		priority = DefaultTaskPriority
	}
	row := s.db.QueryRowContext(ctx,
		`INSERT INTO tasks (project_id, title, description, status, priority)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+pgTaskColumns,
		projectID, title, description, status, priority)
	t, err := scanPGTask(row)
	if err != nil {
		return model.Task{}, fmt.Errorf("insert task: %w", err)
	}
	return t, nil
}

// GetTask returns the task with the given id or ErrNotFound.
func (s *PostgresStore) GetTask(ctx context.Context, id int64) (model.Task, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+pgTaskColumns+` FROM tasks WHERE id = $1`, id)
	t, err := scanPGTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("select task: %w", err)
	}
	return t, nil
}

// ListTasks returns the tasks of a project matching filter, ordered by ascending id.
// Filter values are bound as numbered parameters; only fixed column predicates are concatenated.
func (s *PostgresStore) ListTasks(ctx context.Context, projectID int64, filter TaskFilter) ([]model.Task, error) {
	conds := []string{"project_id = $1"}
	args := []any{projectID}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.Priority != "" {
		args = append(args, filter.Priority)
		conds = append(conds, fmt.Sprintf("priority = $%d", len(args)))
	}
	query := `SELECT ` + pgTaskColumns + ` FROM tasks WHERE ` +
		strings.Join(conds, " AND ") + ` ORDER BY id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		t, err := scanPGTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

// UpdateTask applies the non-nil fields of patch in a single atomic statement;
// nil fields keep their stored values via COALESCE.
func (s *PostgresStore) UpdateTask(ctx context.Context, id int64, patch TaskPatch) (model.Task, error) {
	row := s.db.QueryRowContext(ctx,
		`UPDATE tasks SET
			title = COALESCE($1::text, title),
			description = COALESCE($2::text, description),
			status = COALESCE($3::text, status),
			priority = COALESCE($4::text, priority),
			updated_at = now()
		 WHERE id = $5 RETURNING `+pgTaskColumns,
		optionalText(patch.Title),
		optionalText(patch.Description),
		optionalText(patch.Status),
		optionalText(patch.Priority),
		id)
	t, err := scanPGTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("update task: %w", err)
	}
	return t, nil
}

// DeleteTask removes a task or returns ErrNotFound.
func (s *PostgresStore) DeleteTask(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return requireAffected(res)
}

// optionalText converts a nullable patch field into a driver value; nil becomes SQL NULL.
func optionalText(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

// scanPGProject reads one project row and normalizes timestamps to UTC.
func scanPGProject(sc rowScanner) (model.Project, error) {
	var p model.Project
	var created time.Time
	if err := sc.Scan(&p.ID, &p.Name, &created); err != nil {
		return model.Project{}, err
	}
	p.CreatedAt = created.UTC()
	return p, nil
}

// scanPGTask reads one task row and normalizes timestamps to UTC.
func scanPGTask(sc rowScanner) (model.Task, error) {
	var t model.Task
	var created, updated time.Time
	if err := sc.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &created, &updated); err != nil {
		return model.Task{}, err
	}
	t.CreatedAt = created.UTC()
	t.UpdatedAt = updated.UTC()
	return t, nil
}
