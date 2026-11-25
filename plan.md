# Implementation plan

Active phase: phase-3-browser (Playwright integration and browser QA).
Scope of this phase: the four `required_files` only. Backend and frontend paths belong to other phases and have no beads here.

## Bead map

### ta-o19: taskboard/package.json
- Scope: Node manifest for the Playwright E2E harness under `taskboard/`. Declares the `test` script and the `@playwright/test` dev dependency pinned to 1.62.1.
- Architecture: `taskboard/playwright/taskboard.spec.ts` provides Playwright browser E2E scenarios; the harness is driven from `taskboard/`.
- Interfaces: none (manifest only).
- Depends on: <none>
- Consumed by: taskboard/package-lock.json, taskboard/playwright.config.ts, taskboard/playwright/taskboard.spec.ts
- Acceptance:
  - `package.json` parses as JSON and lists `@playwright/test` at exactly 1.62.1 under devDependencies.
  - A `test` script exists and runs Playwright on the chromium project.
  - No runtime dependencies are added beyond what architecture requires.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile matching `taskboard/package.json`, so the install step reproduces the Playwright install exactly.
- Architecture: pins the same `@playwright/test` 1.62.1 version declared in package.json.
- Interfaces: none (lockfile only).
- Depends on: taskboard/package.json (ta-o19)
- Consumed by: the Playwright verify step, which needs `taskboard/node_modules/.bin/playwright` installed from this lockfile.
- Acceptance:
  - `lockfileVersion` is 3 and the root package name and devDependencies match package.json.
  - The install under `taskboard/` installs `node_modules/.bin/playwright` without modifying the lockfile.
  - The lockfile is not hand-edited to add packages that package.json does not declare.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts --list`

### ta-ff9: taskboard/playwright.config.ts
- Scope: Playwright configuration for the browser suite. Test directory is `./playwright`, per-test timeout 10000 ms, one chromium project.
- Architecture: default `baseURL` is `http://127.0.0.1:8080`, overridable through `BASE_URL`; the chromium project uses Desktop Chrome with the HTTPS-upgrade features disabled and `--ignore-certificate-errors` for local runs.
- Interfaces: default export from `defineConfig` with `testDir`, `timeout`, `use.baseURL`, and `projects` (name `chromium`).
- Depends on: taskboard/package.json (ta-o19) for `@playwright/test`
- Consumed by: taskboard/playwright/taskboard.spec.ts (through the Playwright runner)
- Acceptance:
  - `BASE_URL` overrides `use.baseURL`; without it the base URL is `http://127.0.0.1:8080`.
  - `forbidOnly` is set when `CI` is present; `workers` is 1 under CI.
  - Reporters are `html` (open: never) and `list`.
  - Exactly one project named `chromium`.
- Verify: `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts --list`

### ta-l7z: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E scenarios for the TaskBoard UI, run against the live app on the configured base URL.
- Architecture: covers the SPEC user flows: create and delete projects; create and edit tasks; move a task through todo, doing, and done; filter by status (all, todo, doing, done) and by priority (all, low, medium, high); delete a task.
- Interfaces: Playwright `test` blocks only; no exported symbols.
- Depends on: taskboard/playwright.config.ts (ta-ff9), taskboard/package.json (ta-o19)
- Consumed by: the `test` script in taskboard/package.json and the Playwright verify step.
- Acceptance:
  - One test per SPEC flow listed above; each test asserts the visible UI state after the action, not only that the request was sent.
  - Status filter and priority filter each have a test that checks which task titles remain visible.
  - Tests create their own projects and tasks and clean them up, so they do not depend on execution order.
  - The spec runs green with `./node_modules/.bin/playwright test --config playwright.config.ts playwright` against a running app.
- Verify: `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

## Dependency order
1. ta-o19 (package.json)
2. ta-qu7 (package-lock.json)
3. ta-ff9 (playwright.config.ts)
4. ta-l7z (taskboard.spec.ts)

## Future considerations
- Go unit and integration tests for the backend are owned by earlier phases and are not part of this plan.
