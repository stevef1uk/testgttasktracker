# Implementation plan

Active delivery phase: phase-3-browser (Playwright integration and browser QA).

Scope: only the four required_files below. Bead IDs come from `bd list --limit=0 --status=open,in_progress` in this session. No beads are created in this amendment.

## Rework amendment (review feedback)

1. **Verify path.** The phase verify is the native command only: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`. Compose files under taskboard/ (docker-compose.yml, Dockerfile.playwright) are outside this phase and are not verify paths (architecture.md line 143).
2. **Base URL.** taskboard/playwright.config.ts must default `use.baseURL` to `http://127.0.0.1:8080` (architecture.md line 57). The cached content with `http://app:8080` is superseded and must be rewritten. `BASE_URL` may override the default.
3. **Chromium blocker.** The previous run failed because Chromium was missing for Playwright (`browserType.launch: Executable doesn't exist`). This is an environment prerequisite, not a spec defect. The polecat must not install browsers on the host. The verify environment must supply a Chromium build for Playwright. If it cannot, report an environment blocker rather than closing ta-ud0.
4. **Server prerequisite.** The TaskBoard server must be running on 127.0.0.1:8080 before the Playwright verify runs. The server is started separately; playwright.config.ts defines no webServer block.

## Bead map

### ta-4r9: taskboard/package.json
- Scope: Node manifest for the browser test toolchain. Declares @playwright/test as a devDependency only. No build, bundle, or frontend scripts.
- Architecture: architecture.md "Planned file layout" (line 55) and phase-3-browser required_files.
- Interfaces: devDependency "@playwright/test" pinned to 1.62.1; no runtime dependencies.
- Depends on: none.
- Consumed by: taskboard/package-lock.json (ta-1jk), taskboard/playwright.config.ts (ta-aug), taskboard/playwright/taskboard.spec.ts (ta-ud0).
- Acceptance:
  - taskboard/package.json parses as valid JSON.
  - @playwright/test appears only under devDependencies, pinned to the version in the lockfile.
  - No build or bundle scripts. No Compose-related scripts or dependencies.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-1jk: taskboard/package-lock.json
- Scope: npm-generated lockfile next to taskboard/package.json, produced with `npm install --ignore-scripts`. Never handwritten.
- Architecture: architecture.md "Planned file layout" and phase-3-browser required_files.
- Interfaces: lockfile entry for @playwright/test at the same version as taskboard/package.json.
- Depends on: taskboard/package.json (ta-4r9).
- Consumed by: Node dependency installation before test review (architecture.md line 120).
- Acceptance:
  - taskboard/package-lock.json exists, was produced by npm, and has a lockfileVersion.
  - The resolved @playwright/test version matches taskboard/package.json.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-aug: taskboard/playwright.config.ts
- Scope: single minimal Playwright config. testDir is ./playwright. `use.baseURL` defaults to http://127.0.0.1:8080 and may be overridden by BASE_URL. No webServer block. No Compose hostnames such as app:8080.
- Architecture: architecture.md line 57 and SPEC Phase 3.
- Interfaces: default export of a Playwright config object; use.baseURL; testDir.
- Depends on: taskboard/package.json (ta-4r9) for @playwright/test types.
- Consumed by: taskboard/playwright/taskboard.spec.ts (ta-ud0), via --config in Verify.
- Acceptance:
  - Exports a default config.
  - use.baseURL resolves to http://127.0.0.1:8080 when BASE_URL is unset.
  - testDir points at the playwright/ directory containing taskboard.spec.ts.
  - One config only; no wrapper configs; no Compose-specific settings.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

### ta-ud0: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E specs against the running server at 127.0.0.1:8080. Covers the initial shell, creating a project and a task through browser controls, Move forward from To Do to Doing to Done, deleting a task, and the high-priority filter with reset (architecture.md lines 58 and 135).
- Architecture: architecture.md lines 58 and 135; SPEC "Playwright E2E" and the Phase 3 gate.
- Interfaces: test file under testDir; uses baseURL from playwright.config.ts; no imports from Go code.
- Depends on: taskboard/playwright.config.ts (ta-aug) and a TaskBoard server reachable on 127.0.0.1:8080.
- Consumed by: phase-3-browser QA gate.
- Acceptance:
  - Each test performs real clicks, fills, or selections and asserts DOM or API state after the action.
  - Tests cover: initial shell render; create project; create task appears in To Do; Move forward To Do to Doing to Done; delete removes the task; high-priority filter shows only high tasks and reset restores all.
  - No trivial tests (no expect(true), empty bodies, or import-only checks).
  - Each test uses a unique project name and cleans up what it creates.
- Environment prerequisites (not implemented by the polecat): Chromium installed for Playwright in the verify environment, and the TaskBoard server running on 127.0.0.1:8080 before Verify. The polecat must not install browsers on the host.
- Verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

## Ordering

1. ta-4r9 (taskboard/package.json)
2. ta-1jk (taskboard/package-lock.json)
3. ta-aug (taskboard/playwright.config.ts)
4. ta-ud0 (taskboard/playwright/taskboard.spec.ts)

## Future considerations

- None for this phase.
