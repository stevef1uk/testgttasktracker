# Test Review: phase-1-app (re-review of api_test.go finding, bead ta-is1)

## Summary
Phase verify passed: go build, go test ./... (all packages ok), and the server health probe succeeded (VERIFY_EXIT=0).
Targeted run of the repaired tests: all six PASS.
- TestReadyReports200WhenHealthy
- TestReadyReports503WhenPingFails
- TestDeleteProjectReturns204AndRemovesFromList
- TestListProjectsAscendingIDOrder
- TestIndexServesRenderedShell
- TestListProjectsStoreFailureIs500NotEmptyArray

## Per-requirement results
### ta-is1 finding coverage: PASS
- Test file: taskboard/internal/web/api_test.go
- Verify result: passed (go test ./internal/web ok; targeted tests PASS)

(1) GET api ready
- 200 with status ready: TestReadyReports200WhenHealthy checks the status code and that the body is exactly {"status":"ready"} (line 271).
- 503 with database unavailable: TestReadyReports503WhenPingFails uses pingFailStore and checks the status and exact error body (line 286).
- Note: the 200 case uses the in-memory fakeStore, not a SQLite file. SQLite ping and readiness are covered by the phase-1-core store tests.

(2) DELETE api projects by ID
- 204 with empty body: lines 303 and 306.
- Follow-up list no longer contains the project: line 319 in TestDeleteProjectReturns204AndRemovesFromList.

(3) GET api projects ascending ID order
- TestListProjectsAscendingIDOrder checks len == 3 and strictly ascending IDs (line 345).
- The fake store is insertion-ordered, so this test does not prove SQL ORDER BY. SQL ordering is covered by the phase-1-core SQLite store tests.

(4) GET root
- TestIndexServesRenderedShell checks status 200 (line 357), the text/html content type (line 360), and an html element in the body (line 363).

(5) Store failure on list returns 500, not an empty array
- TestListProjectsStoreFailureIs500NotEmptyArray checks status 500 (line 371) and rejects an empty-array body (line 374).

## Overall assessment
All five findings have real assertions that run and pass. No test-file repair is required for ta-is1. Other in-scope rows were not re-audited in this step.

- Evidence: {"file":"taskboard/internal/web/api_test.go","line":374,"excerpt":"if got := strings.TrimSpace(rr.Body.String()); got == \"[]\" {"}
