# Implementation plan

Active delivery phase: phase-3-browser (Playwright integration and browser QA).

Scope: only the four required_files below. Bead IDs come from bd list --limit=0 in this session. No beads are created in this amendment; the existing beads are reused.

Amendment (rework): architecture.md line 143 states that taskboard/docker-compose.yml, Compose-based QA commands, and extra files are stale and not adopted. The Compose file currently on disk is not part of this phase and must not be used as a verify path. The phase verify is the native command below. The Playwright baseURL must default to http://127.0.0.1:8080 (architecture.md line 57), not http://app:8080.

## Bead map

### ta-4r9: taskboard/package.json
- Scope: Node manifest for the browser test toolchain. Declares @playwright/test as a devDependency only. No build, bundle, or frontend scripts.
- Architecture: architecture.md "Planned file layout" (Node manifest, line 55) and phase-3-browser required_files (lines 197-199).
- Interfaces: devDependency "@playwright/test" pinned to 1.62.1; no runtime dependencies.
- Depends on: none (first Node bead).
- Consumed by: taskboard/package-lock.json (ta-1jk), taskboard/playwright.config.ts (ta-aug), taskboard/playwright/taskboard.spec.ts (ta-ud0).
- Acceptance:
  - taskboard/package.json exists and parses as valid JSON.
  - @playwright/test appears only under devDependencies, pinned to the version recorded in the lockfile.
  - No build or bundle scripts are declared (SPEC Phase 3).
  - No Compose-related scripts or dependencies are declared.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-1jk: taskboard/package-lock.json
- Scope: npm-generated lockfile committed next to taskboard/package.json. Generated with npm install --ignore-scripts during implementation, never handwritten.
- Architecture: architecture.md "Planned file layout" (Node lockfile) and phase-3-browser required_files.
- Interfaces: lockfile entry for @playwright/test resolving to the same version as taskboard/package.json.
- Depends on: taskboard/package.json (ta-4r9).
- Consumed by: Node dependency installation before test review (architecture.md line 120).
- Acceptance:
  - taskboard/package-lock.json exists, was produced by npm, and contains lockfileVersion.
  - The resolved @playwright/test version matches taskboard/package.json.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-aug: taskboard/playwright.config.ts
- Scope: minimal Playwright config. testDir is ./playwright. use.baseURL must default to http://127.0.0.1:8080, the host-accessible TaskBoard server (architecture.md line 57; architecture.md line 120). It must NOT default to http://app:8080 or any Compose service hostname. BASE_URL may override it. No webServer block: the server is started separately before the run.
- Architecture: architecture.md line 57 and SPEC Phase 3.
- Interfaces: default export of a Playwright config object; use.baseURL; testDir.
- Depends on: taskboard/package.json (ta-4r9) for @playwright/test types.
- Consumed by: taskboard/playwright/taskboard.spec.ts (ta-ud0), via the --config flag in Verify.
- Acceptance:
  - taskboard/playwright.config.ts exports a default config.
  - use.baseURL resolves to http://127.0.0.1:8080 when BASE_URL is unset.
  - testDir resolves to the playwright/ directory containing taskboard.spec.ts.
  - Only one Playwright config exists; no wrapper configs and no Compose-specific settings.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-ud0: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E specs against the running server at 127.0.0.1:8080. Covers the initial shell, creating a project and a task through browser controls, Move forward from To Do to Doing to Done, deleting a task, and the high-priority filter with reset (architecture.md lines 58 and 135).
- Architecture: architecture.md lines 58 and 135; SPEC "Playwright E2E" section and the Phase 3 gate.
- Interfaces: test file under testDir; uses baseURL from playwright.config.ts; no imports from Go code.
- Depends on: taskboard/playwright.config.ts (ta-aug) and a TaskBoard server reachable on 127.0.0.1:8080.
- Consumed by: phase-3-browser QA gate.
- Acceptance:
  - taskboard/playwright/taskboard.spec.ts exists.
  - Each test performs real clicks, fills, or selections and asserts DOM or API state after the action.
  - Tests cover: initial shell render; create project; create task appears in To Do; Move forward To Do to Doing to Done; delete removes the task; high-priority filter shows only high tasks and reset restores all.
  - No trivial tests (no expect(true), empty bodies, or import-only checks).
  - Each test uses a unique project name and cleans up what it creates.
- Environment prerequisites (not implemented by the polecat): a Chromium build installed for Playwright in the environment that runs the verify command, and the TaskBoard server running on 127.0.0.1:8080 before Verify. The polecat must not install browsers on the host.
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
- The Compose setup under taskboard/ (docker-compose.yml, Dockerfile.playwright) is outside this phase and is not a verify path.

## Future considerations

- None for this phase.
