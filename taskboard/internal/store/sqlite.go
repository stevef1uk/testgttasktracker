package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"taskboard/internal/model"
)

// SQLiteStore implements persistence on top of modernc.org/sqlite (pure Go, no CGO).
type SQLiteStore struct {
	db *sql.DB
}

// Compile-time check that SQLiteStore satisfies Store.
var _ Store = (*SQLiteStore)(nil)

// taskColumns is the fixed column list used by every task SELECT.
const taskColumns = `id, project_id, title, description, status, priority, created_at, updated_at`

// NewSQLiteStore opens (or creates) the SQLite database at path, enables
// foreign key enforcement and a 5000ms busy timeout on every connection
// through the DSN, and applies SchemaSQLite idempotently.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	dsn := "file:" + path + "?" + q.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	// SQLite allows a single writer; a single connection avoids lock contention.
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}
	if _, err := db.ExecContext(ctx, SchemaSQLite); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply sqlite schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Ping verifies the database connection is usable.
func (s *SQLiteStore) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}
	return nil
}

// Close releases the database handle.
func (s *SQLiteStore) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close sqlite: %w", err)
	}
	return nil
}

// CreateProject inserts a project and returns the stored row.
func (s *SQLiteStore) CreateProject(ctx context.Context, name string) (model.Project, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO projects (name) VALUES (?)`, name)
	if err != nil {
		return model.Project{}, fmt.Errorf("insert project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Project{}, fmt.Errorf("project last insert id: %w", err)
	}
	return s.GetProject(ctx, id)
}

// GetProject returns the project with the given id or ErrNotFound.
func (s *SQLiteStore) GetProject(ctx context.Context, id int64) (model.Project, error) {
	var p model.Project
	var created string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM projects WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Project{}, ErrNotFound
	}
	if err != nil {
		return model.Project{}, fmt.Errorf("select project: %w", err)
	}
	p.CreatedAt, err = parseTimestamp(created)
	if err != nil {
		return model.Project{}, err
	}
	return p, nil
}

// ListProjects returns all projects ordered by ascending id.
func (s *SQLiteStore) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, created_at FROM projects ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := []model.Project{}
	for rows.Next() {
		var p model.Project
		var created string
		if err := rows.Scan(&p.ID, &p.Name, &created); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.CreatedAt, err = parseTimestamp(created)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

// DeleteProject removes a project; its tasks are removed by ON DELETE CASCADE.
func (s *SQLiteStore) DeleteProject(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return requireAffected(res)
}

// CreateTask inserts a task. Empty status or priority fall back to the defaults.
// An unknown projectID fails with a foreign key error.
func (s *SQLiteStore) CreateTask(ctx context.Context, projectID int64, title, description, status, priority string) (model.Task, error) {
	if status == "" {
		status = DefaultTaskStatus
	}
	if priority == "" {
		priority = DefaultTaskPriority
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (project_id, title, description, status, priority) VALUES (?, ?, ?, ?, ?)`,
		projectID, title, description, status, priority)
	if err != nil {
		return model.Task{}, fmt.Errorf("insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Task{}, fmt.Errorf("task last insert id: %w", err)
	}
	return s.GetTask(ctx, id)
}

// GetTask returns the task with the given id or ErrNotFound.
func (s *SQLiteStore) GetTask(ctx context.Context, id int64) (model.Task, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("select task: %w", err)
	}
	return t, nil
}

// ListTasks returns the tasks of a project matching filter, ordered by ascending id.
// Filter values are bound as parameters; only fixed column predicates are concatenated.
func (s *SQLiteStore) ListTasks(ctx context.Context, projectID int64, filter TaskFilter) ([]model.Task, error) {
	conds := []string{"project_id = ?"}
	args := []any{projectID}
	if filter.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		conds = append(conds, "priority = ?")
		args = append(args, filter.Priority)
	}
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE ` +
		strings.Join(conds, " AND ") + ` ORDER BY id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
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

// UpdateTask applies the non-nil fields of patch and preserves all others.
func (s *SQLiteStore) UpdateTask(ctx context.Context, id int64, patch TaskPatch) (model.Task, error) {
	cur, err := s.GetTask(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if patch.Title != nil {
		cur.Title = *patch.Title
	}
	if patch.Description != nil {
		cur.Description = *patch.Description
	}
	if patch.Status != nil {
		cur.Status = *patch.Status
	}
	if patch.Priority != nil {
		cur.Priority = *patch.Priority
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET title = ?, description = ?, status = ?, priority = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
		cur.Title, cur.Description, cur.Status, cur.Priority, id)
	if err != nil {
		return model.Task{}, fmt.Errorf("update task: %w", err)
	}
	if err := requireAffected(res); err != nil {
		return model.Task{}, err
	}
	return s.GetTask(ctx, id)
}

// DeleteTask removes a task or returns ErrNotFound.
func (s *SQLiteStore) DeleteTask(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return requireAffected(res)
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(sc rowScanner) (model.Task, error) {
	var t model.Task
	var created, updated string
	if err := sc.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &created, &updated); err != nil {
		return model.Task{}, err
	}
	var err error
	if t.CreatedAt, err = parseTimestamp(created); err != nil {
		return model.Task{}, err
	}
	if t.UpdatedAt, err = parseTimestamp(updated); err != nil {
		return model.Task{}, err
	}
	return t, nil
}

// parseTimestamp reads the strftime('%Y-%m-%dT%H:%M:%fZ') values written by the schema.
func parseTimestamp(raw string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04:05.000Z", time.RFC3339Nano} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse timestamp %q", raw)
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
