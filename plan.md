# Implementation plan

Active delivery phase: `phase-3-browser` (Playwright integration and browser QA).
Only the four `required_files` below are in scope. Backend and frontend paths belong to earlier phases and are not planned here.

## Bead map

### ta-o19: taskboard/package.json
- Scope: npm manifest for the E2E toolchain. Declares Playwright as a dev dependency and defines the npm scripts used to run browser tests. No server code.
- Architecture: architecture.md lists `taskboard/package.json` as "E2E test dependencies and npm scripts" (phase-3 required_files).
- Interfaces: `scripts` entry that runs `playwright test --config playwright.config.ts playwright`; devDependency `@playwright/test`.
- Depends on: `<none>`
- Consumed by: ta-qu7 (`taskboard/package-lock.json` locks this manifest), ta-ff9 (`taskboard/playwright.config.ts` is loaded by the Playwright runner that this manifest installs), ta-xmb (spec runs under this toolchain).
- Acceptance:
  - File is valid JSON and declares `@playwright/test` under devDependencies.
  - Provides a test script that invokes the Playwright runner with `--config playwright.config.ts`.
  - No server or Go dependencies are added here.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile that pins the full npm dependency tree for `taskboard/package.json`.
- Architecture: architecture.md lists `taskboard/package-lock.json` as "Locked npm dependency tree".
- Interfaces: none (generated artifact).
- Depends on: ta-o19 (`taskboard/package.json`).
- Consumed by: ta-ff9, ta-xmb (both run after `npm ci --ignore-scripts` installs from this lockfile).
- Acceptance:
  - Lockfile version matches the npm version used to generate it and lists `@playwright/test` with a resolved version.
  - Lockfile is consistent with `taskboard/package.json` (`npm ci --ignore-scripts` succeeds without modification).
  - Not hand-edited; regenerate with npm if package.json changes.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-ff9: taskboard/playwright.config.ts
- Scope: Playwright runner configuration (test directory, base URL, browser projects, reporter).
- Architecture: architecture.md lists `taskboard/playwright.config.ts` as "Playwright test configuration", and the Playwright E2E command is `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`.
- Interfaces: default export of a Playwright config object; `testDir` points at the `playwright/` directory.
- Depends on: ta-o19 (`taskboard/package.json` provides `@playwright/test`).
- Consumed by: ta-xmb (spec discovery and base URL come from this config).
- Acceptance:
  - `testDir` resolves to `playwright/` so `taskboard.spec.ts` is discovered.
  - Config exports a valid Playwright configuration that loads without error under the command above.
  - Base URL and server startup, if configured, point at the taskboard app and do not hard-code unrelated ports.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-xmb: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E tests for the taskboard UI, driven by Playwright against the running app.
- Architecture: architecture.md lists `taskboard/playwright/taskboard.spec.ts` as "E2E browser automation tests for UI" and names it under Playwright E2E tests.
- Interfaces: Playwright `test(...)` cases only; no exported symbols.
- Depends on: ta-ff9 (config and test directory), ta-o19 (test runner dependency), ta-qu7 (locked toolchain).
- Consumed by: phase-3 QA (`qa_verify_command` runs this spec).
- Acceptance:
  - Covers the SPEC browser flows for the taskboard UI: page loads, creating a task through the form, and the task appears on the board.
  - Each case asserts a visible result from SPEC, not only that a page loaded.
  - Uses real UI controls (locators on buttons, inputs, and forms) and verifies at least one user action changes rendered state.
  - No trivial always-pass tests (no bare `expect(true)`, no empty test bodies).
  - Test names map to SPEC behavior.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

## Dependency order

1. ta-o19 (`taskboard/package.json`)
2. ta-qu7 (`taskboard/package-lock.json`)
3. ta-ff9 (`taskboard/playwright.config.ts`)
4. ta-xmb (`taskboard/playwright/taskboard.spec.ts`)

## Phase context

- Phase verification: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`
- Beads ta-o19, ta-qu7, and ta-ff9 are already closed. If any of their files are missing or stubbed, planning sync reopens them. Do not create duplicates.
