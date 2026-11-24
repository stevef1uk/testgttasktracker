# Implementation plan

Active phase: phase-3-browser (Playwright integration and browser QA).
Required files: taskboard/package.json, taskboard/package-lock.json, taskboard/playwright.config.ts, taskboard/playwright/taskboard.spec.ts.

## Bead map

### ta-o19: taskboard/package.json
- Scope: Playwright test manifest for the taskboard E2E suite. Declares the @playwright/test devDependency pinned to 1.62.1 and a test script that runs the chromium project.
- Architecture: phase-3-browser section of architecture.md (Configure Playwright dependencies).
- Interfaces: npm script `test`; devDependency `@playwright/test` at 1.62.1.
- Depends on: <none>
- Consumed by: ta-qu7 (lockfile resolves this manifest), ta-ev8 (config imports @playwright/test), ta-m0z (spec uses the test runner).
- Acceptance: file parses as JSON; devDependencies pin @playwright/test to 1.62.1; scripts.test runs the chromium project.
- Verify: from taskboard/, the Playwright CLI resolves at version 1.62.1.

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile matching taskboard/package.json, lockfileVersion 3.
- Architecture: phase-3-browser section of architecture.md.
- Interfaces: none (generated lock data).
- Depends on: ta-o19 (taskboard/package.json).
- Consumed by: ta-ev8, ta-m0z (installed packages).
- Acceptance: lockfile name and root devDependencies match package.json; installing from the lockfile does not change it.
- Verify: from taskboard/, the lockfile's root devDependencies equal those in package.json.

### ta-ev8: taskboard/playwright.config.ts
- Scope: Playwright configuration. testDir is ./playwright, baseURL comes from BASE_URL with a localhost:8080 fallback, and a single chromium project uses Desktop Chrome settings.
- Architecture: phase-3-browser section of architecture.md (Add test configuration).
- Interfaces: default export from defineConfig with projects [chromium].
- Depends on: ta-o19 (@playwright/test dependency), ta-qu7 (installed packages).
- Consumed by: ta-m0z (spec runs under this config).
- Acceptance: config loads under playwright test --list; exactly one project named chromium; baseURL targets the local taskboard server on port 8080.
- Verify: from taskboard/, run `go test ./...` and then `./node_modules/.bin/playwright test --config playwright.config.ts playwright`.

### ta-m0z: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E spec for the taskboard UI covering project creation, task movement between columns, priority filtering, and task deletion, as named in architecture.md phase-3-browser.
- Architecture: phase-3-browser section of architecture.md (Implement E2E browser test suite).
- Interfaces: test file using @playwright/test `test` and `expect`.
- Depends on: ta-ev8 (config), ta-qu7 (installed packages).
- Consumed by: plan_review and the QA browser pass.
- Acceptance: each test asserts a user-visible outcome from SPEC; the spec passes against the running server on baseURL.
- Verify: from taskboard/, run `./node_modules/.bin/playwright test --config playwright.config.ts playwright`.

## Future considerations
- Phase-4 deploy files (Dockerfile, deployment manifests) are out of scope for this phase and get their own beads when that phase is active.
