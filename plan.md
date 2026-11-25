# Implementation plan: phase-3-browser (Playwright integration and browser QA)

## Scope

Active phase `phase-3-browser` depends on `phase-2-kanban`. It owns exactly four first-delivery files:
`taskboard/package.json`, `taskboard/package-lock.json`, `taskboard/playwright.config.ts`, `taskboard/playwright/taskboard.spec.ts`.
Phase 1 and 2 files are read-only for this phase. Playwright drives the already-built UI served by the running Go app on `127.0.0.1:8080`. Deploy and release phases are out of scope.

Phase verification (exact, from architecture `qa_verify_command`):
`cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

The Go server must already be running before the browser step. Dependency installation is not part of this plan's verify lines.

## Bead map

### ta-o19: taskboard/package.json
- Scope: npm manifest declaring `@playwright/test` as the only devDependency, plus a `test` script that runs the Playwright suite. No frontend build scripts, bundler, or framework.
- Architecture: phase-3 required file; owns E2E test dependencies and npm scripts (architecture, Module / bead ownership table).
- Interfaces: none (manifest only).
- Depends on: `<none>`.
- Consumed by: `taskboard/package-lock.json`, `taskboard/playwright.config.ts`, `taskboard/playwright/taskboard.spec.ts`.
- Acceptance:
  - `devDependencies` contains `@playwright/test`; `dependencies` is empty or absent.
  - No build step, bundler, or framework is introduced (SPEC §2, §7).
  - Package name is `playwright-tests` and version is `1.0.0`, matching the cached content.
- Verify: phase gate `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`.

### ta-qu7: taskboard/package-lock.json
- Scope: committed lockfile matching `taskboard/package.json`. Lockfile version 3.
- Architecture: required lockfile (SPEC §3: go.sum and package-lock.json must be committed).
- Interfaces: none.
- Depends on: `taskboard/package.json`.
- Consumed by: Playwright runs in `taskboard/playwright.config.ts` and `taskboard/playwright/taskboard.spec.ts`.
- Acceptance:
  - Lockfile pins `@playwright/test` at the version in `taskboard/package.json`.
  - Root package entry name and version match `taskboard/package.json`.
  - No `node_modules` or generated reports are committed.
- Verify: covered by the phase gate above.

### ta-ff9: taskboard/playwright.config.ts
- Scope: Playwright configuration with `testDir` set to `./playwright`, base URL `http://127.0.0.1:8080` (overridable by `BASE_URL`), and a single Chromium project using the Desktop Chrome device profile.
- Architecture: phase-3 required file; config for `playwright/taskboard.spec.ts` (architecture, phase-3-browser).
- Interfaces: default export of `defineConfig({...})` from `@playwright/test`.
- Depends on: `taskboard/package.json`.
- Consumed by: `taskboard/playwright/taskboard.spec.ts` (through the runner).
- Acceptance:
  - `testDir` resolves to `playwright/` so the spec file is discovered by the verify command.
  - Base URL targets the host-accessible server; no second application container is started (SPEC §8).
  - Retries stay at 0 so failures are not hidden; config does not use `test.skip` or `test.fixme` to pass.
  - Reporter writes no artifacts into the committed tree (no `playwright-report` checked in).
- Verify: covered by the phase gate above.

### ta-l7z: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E tests driven by real clicks, typing, and select changes against the running app.
- Architecture: phase-3 required file; E2E / integration testing section lists these flows.
- Interfaces: uses `test`, `expect` from `@playwright/test`; no exported symbols.
- Depends on: `taskboard/playwright.config.ts`, `taskboard/package.json`, and the running Go app served from phase-2 files.
- Consumed by: phase verify command.
- Acceptance (SPEC §8 Playwright E2E, each asserting DOM or `/api` state):
  1. Initial page shows the `TaskBoard` heading and the three columns `To Do`, `Doing`, `Done`.
  2. Create a uniquely named project and a task through the UI; the card appears in `To Do`. Click `Move forward` to reach `Doing`, then `Done`; the forward control is hidden or disabled at `done`. Delete the task and assert it is gone.
  3. Create two tasks with different priorities, set the priority filter to `high`, assert only the high-priority card renders, reset to `all`, and assert both cards return.
  4. At least one test performs real input, click, and select actions and asserts the resulting DOM or `/api` state change. Static-only checks are insufficient.
  5. Each test uses a unique project name and deletes the projects it created, so it does not depend on prior runs.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`.

## Dependencies and ordering

1. `ta-o19` (package.json) before `ta-qu7` (lockfile generated from the manifest).
2. `ta-o19` and `ta-ff9` before `ta-l7z` (the spec imports `@playwright/test` and relies on the config).
3. All four before the phase verify command runs.

## Bead status note

`ta-o19` and `ta-qu7` are closed and their files exist, so they are not reopened. `ta-ff9` is in progress. `ta-l7z` is the only open implement bead. No duplicates were created.

## Future considerations

- Phase 4 Dockerfile and k3s manifests are not planned here.
- PostgreSQL E2E coverage is not in this phase; `TASKBOARD_TEST_POSTGRES_URL` is handled by Go tests.
