# Implementation plan

Active delivery phase: phase-3-browser (Playwright integration and browser QA). Only the four required_files below are in scope. Bead IDs come from bd list --limit=0 in this session; no beads are created for this amendment.

## Bead map

### ta-4r9: taskboard/package.json
- Scope: Node manifest for the browser test toolchain. Declares @playwright/test as a devDependency only. No build, bundle, or frontend scripts.
- Architecture: architecture.md "Planned file layout" (Node manifest) and phase-3-browser required_files.
- Depends on: none (first Node bead).
- Consumed by: taskboard/package-lock.json, taskboard/playwright.config.ts, taskboard/playwright/taskboard.spec.ts.
- Acceptance:
  - taskboard/package.json exists and parses as valid JSON.
  - @playwright/test appears only under devDependencies, pinned to the version in the lockfile.
  - No build or bundle scripts are declared (SPEC Phase 3).
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-1jk: taskboard/package-lock.json
- Scope: npm-generated lockfile committed next to taskboard/package.json. Generated with npm install --ignore-scripts during implementation, never handwritten.
- Architecture: architecture.md "Planned file layout" (Node lockfile) and phase-3-browser required_files.
- Depends on: taskboard/package.json (ta-4r9).
- Consumed by: Node dependency installation before test review.
- Acceptance:
  - taskboard/package-lock.json exists, was produced by npm, and contains lockfileVersion.
  - The resolved @playwright/test version matches taskboard/package.json.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-aug: taskboard/playwright.config.ts
- Scope: minimal Playwright config. testDir is ./playwright. use.baseURL must default to http://127.0.0.1:8080, the host-accessible TaskBoard server from SPEC section 8. It must NOT default to http://app:8080, which is an unreachable container hostname. BASE_URL may override it. No webServer block: the server is already running.
- Architecture: architecture.md line 57 and SPEC Phase 3 (preserve a canonical generated config if one exists; otherwise create a minimal compatible one).
- Depends on: taskboard/package.json (ta-4r9) for @playwright/test types.
- Consumed by: taskboard/playwright/taskboard.spec.ts through the --config flag in Verify.
- Acceptance:
  - taskboard/playwright.config.ts exports a default config.
  - use.baseURL resolves to http://127.0.0.1:8080 when BASE_URL is unset.
  - testDir resolves to the playwright/ directory containing taskboard.spec.ts.
  - Only one Playwright config exists; no wrapper configs.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-ud0: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E specs against the running server at 127.0.0.1:8080. Covers the initial shell, creating a project and a task through browser controls, Move forward from To Do to Doing to Done, deleting a task, and the high-priority filter with reset (architecture.md lines 58 and 135).
- Architecture: architecture.md lines 58 and 135; SPEC "Playwright E2E" section and Phase 3 gate.
- Depends on: taskboard/playwright.config.ts (ta-aug) and a TaskBoard server reachable on 127.0.0.1:8080.
- Consumed by: phase-3-browser QA gate.
- Acceptance:
  - taskboard/playwright/taskboard.spec.ts exists.
  - Each test performs real clicks, fills, or selections and asserts DOM or API state after the action.
  - Tests cover: initial shell render; create project; create task appears in To Do; Move forward To Do to Doing to Done; delete removes the task; high-priority filter shows only high tasks and reset restores all.
  - No trivial tests (no expect(true), empty bodies, or import-only checks).
  - Each test uses a unique project name and cleans up what it creates.
- Environment prerequisites (not implemented by the polecat): a Chromium build installed for Playwright (npx playwright install chromium, run by the environment owner or the configured Playwright image), and the TaskBoard server running on 127.0.0.1:8080 before Verify.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

## Ordering

1. ta-4r9 (taskboard/package.json)
2. ta-1jk (taskboard/package-lock.json)
3. ta-aug (taskboard/playwright.config.ts)
4. ta-ud0 (taskboard/playwright/taskboard.spec.ts)

## Phase context

- Phase verification: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright
- Node dependencies are installed before test review, never during it.
- Node modules, Playwright reports, and test-results are never committed.

## Future considerations

- None for this phase.
