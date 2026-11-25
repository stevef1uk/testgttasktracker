# Implementation plan: phase-4-deploy (Container build and k3s deployment artifacts)

Active phase: `phase-4-deploy`. Depends on `phase-3-browser`. This plan covers only the five first-delivery files below. Phase `phase-5-release` owns no first-delivery files.

Phase verification: `cd taskboard && go test ./... && docker build -t taskboard:local .`

## Rework amendment (from failed implementation review)

The previous implementation left `taskboard/Dockerfile` as the stale template from the rig root. It still has `FROM golang:1.22-alpine`, builds `./cmd/server`, copies `/server`, and copies `web/`. That fails `go mod download` with `go.mod requires go >= 1.26.1 (running go 1.22.12; GOTOOLCHAIN=local)`.

The Dockerfile bead must fully replace that file. Do not patch it line by line. The replacement must:
- Use `FROM golang:1.26-alpine AS builder` (or a newer pinned 1.26+ tag or digest). Architecture's `golang:1.25-alpine` is superseded because `taskboard/go.mod` declares `go 1.26.1`.
- Build `./cmd/taskboard` into `/app/taskboard`, not `./cmd/server` and not `/server`.
- Use `CMD ["/app/taskboard"]` and `EXPOSE 8080`, with `ENV TASKBOARD_HOST=0.0.0.0`.
- Run as a non-root numeric UID in the alpine:3.20 runtime stage.

Do not lower `go.mod`. SPEC §2 requires Go >= 1.25, so a 1.26 builder satisfies it.

The cached `deployment.yaml` also needs amendment. Its probes use `/` and its env var is `DATABASE_URL`. Architecture requires probes on `/api/health` (liveness) and `/api/ready` (readiness), and env `TASKBOARD_DB_DRIVER` plus `TASKBOARD_DATABASE_URL` from Secret key `database-url`. The polecat must follow architecture, not the cached content.

## Bead map

### ta-uns: taskboard/Dockerfile
- Scope: Replace the entire `taskboard/Dockerfile` with a multi-stage production image for the TaskBoard binary. The build context is `taskboard/`.
- Architecture: Docker & Deployment section. Builder runs `CGO_ENABLED=0 GOOS=linux go build -o /app/taskboard ./cmd/taskboard`. Runtime is alpine:3.20 with ca-certificates, a non-root numeric UID, and `/app/taskboard`.
- Builder image: `golang:1.26-alpine` or newer, pinned. Keep the Go version at least the `go` directive in `taskboard/go.mod` (currently 1.26.1).
- Interfaces: `CMD ["/app/taskboard"]`; `EXPOSE 8080`; `ENV TASKBOARD_HOST=0.0.0.0`.
- Depends on: taskboard/go.mod, taskboard/go.sum, taskboard/cmd/taskboard/main.go, taskboard/internal/web/app.go, taskboard/internal/web/api.go, taskboard/web/assets.go, taskboard/web/templates/index.html, taskboard/web/static/app.js, taskboard/web/static/style.css
- Consumed by: <none>
- Acceptance:
  - The file contains no `golang:1.22`, no `./cmd/server`, and no `/server` path.
  - `go mod download` in the builder stage succeeds with `GOTOOLCHAIN=local`.
  - `docker build -t taskboard:local .` run from `taskboard/` exits 0.
  - The final image has no Go toolchain and no Node runtime.
  - The final image runs as a non-root numeric UID.
  - Embedded HTML, JS, and CSS are inside the binary, so no runtime file copy is needed.
  - No database files, `node_modules`, or secrets are copied into the image.
- Verify: `cd taskboard && go test ./... && docker build -t taskboard:local .`. If Docker is unavailable, report the build as not run.

### ta-2ok: taskboard/.dockerignore
- Scope: Build-context exclusions for the Docker build rooted at `taskboard/`.
- Architecture: Excludes databases, secrets, `.git`, node_modules, test reports, and build outputs.
- Interfaces: None (exclusion patterns only).
- Depends on: <none>
- Consumed by: <none> (Docker reads it as build context; there is no repo-file import edge)
- Acceptance:
  - Excludes `*.db`, `taskboard.db`, `.git`, `node_modules`, `playwright-report`, `test-results`, build outputs, and `.env` or secret files.
  - Does not exclude `cmd/`, `internal/`, `web/`, `go.mod`, or `go.sum`.
  - `docker build` still sends the Go sources and the embedded web assets.
- Verify: `cd taskboard && docker build -t taskboard:local .` succeeds with the exclusions in place.

### ta-0x5: taskboard/deploy/k8s/secret.example.yaml
- Scope: Example Kubernetes Secret carrying the PostgreSQL URL under key `database-url`.
- Architecture: Secret example with placeholder values and instructions not to apply it unchanged or commit real credentials (Kubernetes Secret Example).
- Interfaces: Secret key `database-url`, consumed by the Deployment env reference.
- Depends on: <none>
- Consumed by: taskboard/deploy/k8s/deployment.yaml
- Acceptance:
  - Placeholder URL only; no live password or real host credentials.
  - Key name is exactly `database-url`.
  - Comments instruct operators to create the real Secret out of band.
- Verify: YAML structure check; key name matches the env reference in deployment.yaml.

### ta-om9: taskboard/deploy/k8s/deployment.yaml
- Scope: Single-replica Kubernetes Deployment for TaskBoard.
- Architecture: Port 8080, liveness `GET /api/health`, readiness `GET /api/ready`, and `TASKBOARD_DB_DRIVER` plus `TASKBOARD_DATABASE_URL` injected from Secret key `database-url`.
- Interfaces: Container port 8080 named `http`; env `TASKBOARD_HOST=0.0.0.0`; `TASKBOARD_DB_DRIVER=postgres`; `TASKBOARD_DATABASE_URL` from Secret `database-url`.
- Depends on: taskboard/deploy/k8s/secret.example.yaml
- Consumed by: taskboard/deploy/k8s/service.yaml
- Acceptance:
  - `replicas: 1`; image name is replaceable and not tied to a registry.
  - Readiness probe uses `/api/ready` and liveness uses `/api/health`, not `/`.
  - Non-root `securityContext`; no privileged container, host networking, or hostPath volume.
  - Resource requests and limits are set.
  - No PostgreSQL Deployment, StatefulSet, or TaskBoard PVC.
- Verify: YAML structure check; probe paths; port 8080 named `http`; Secret key wiring (`database-url`).

### ta-cp8: taskboard/deploy/k8s/service.yaml
- Scope: Internal ClusterIP Service for TaskBoard.
- Architecture: `ClusterIP` Service targeting port 8080. Ingress is out of scope.
- Interfaces: Service port 80 to targetPort `http` (8080); selector matches Deployment pod labels.
- Depends on: taskboard/deploy/k8s/deployment.yaml
- Consumed by: <none>
- Acceptance:
  - `type: ClusterIP`.
  - Selector labels exactly match the Deployment template labels.
  - targetPort resolves to the container port named `http` (8080).
- Verify: YAML structure check; selector and port match against deployment.yaml.

## Future considerations
- Ingress and TLS provisioning are out of scope per SPEC §13.
- Restart persistence and the PostgreSQL integration gate are deferred to phase-5-release.
