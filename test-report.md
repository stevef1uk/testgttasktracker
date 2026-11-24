# Test Review: phase-1-core

## Summary
Phase verify command: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`
Result: green (all modules verified; ok config, ok model, ok store; EXIT=0).

## Per-requirement results

### ta-93p (config) - PASS
- Test file: taskboard/internal/config/config_test.go
- Coverage: TestLoadConfigDefaults; TestLoadConfigPortBoundaries (0, 1, 65535, 65536, -1, non-numeric); TestLoadConfigRejectsPostgresWithoutURL; TestLoadConfigErrorsDoNotEchoURL.
- Evidence: {"file":"taskboard/internal/config/config_test.go","line":73,"excerpt":"{name: \"above upper bound rejected\", raw: \"65536\", wantErr: true},"}
- Evidence: {"file":"taskboard/internal/config/config.go","line":35,"excerpt":"return Config{}, errors.New(\"TASKBOARD_DATABASE_URL: required when TASKBOARD_DB_DRIVER=postgres\")"}

### ta-i6d (model) - PASS
- Test file: taskboard/internal/model/model_test.go
- Coverage: TestValidateProjectName and TestValidateTaskTitle (80/81 and 120/121 code-point boundaries, multibyte, padding); TestValidateDescription (1000/1001); TestValidateStatus; TestValidatePriority; trim tests.
- Evidence: {"file":"taskboard/internal/model/model_test.go","line":20,"excerpt":"{\"81 multibyte code points rejected\", strings.Repeat(\"é\", 81), true},"}
- Evidence: {"file":"taskboard/internal/model/model.go","line":13,"excerpt":"MaxTaskTitleLen   = 120"}

### ta-zmd (store/sqlite) - PASS
- Test file: taskboard/internal/store/sqlite_test.go
- Coverage: TestSQLiteReinitIsIdempotent; TestSQLiteProjectsListInAscendingIDOrder; TestSQLiteTaskDefaultsAndUTCTimestamps; TestSQLiteListTasksFilters; TestSQLiteUpdateTaskPreservesOmittedFields; TestSQLiteDeleteProjectCascadesToTasks; TestSQLiteForeignKeyRejectsUnknownProject; TestSQLiteDataPersistsAcrossReopen; TestSQLiteParameterizesHostileInput.
- Evidence: {"file":"taskboard/internal/store/sqlite_test.go","line":282,"excerpt":"if _, err := s.CreateTask(context.Background(), 9999, \"orphan\", \"\", \"\", \"\"); err == nil {"}
- Evidence: {"file":"taskboard/internal/store/sqlite.go","line":220,"excerpt":"func (s *SQLiteStore) ListTasks(ctx context.Context, projectID int64, filter TaskFilter) ([]Task, error) {"}

## Deferred (out of scope for phase-1-core)
- phase-1-app, phase-2-kanban, phase-3-browser rows: bead IDs pending; not due in this phase.

## Overall assessment
All phase-1-core plan rows are covered by real asserting tests, and the phase verify command is green. The earlier CRUD, filter, and validation findings are resolved in sqlite.go, store.go, model.go, and config.go.
