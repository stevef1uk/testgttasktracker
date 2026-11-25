# Implementation plan — phase-4-deploy (Container build and k3s deployment artifacts)

Active phase: `phase-4-deploy`. Depends on `phase-3-browser`. Scope is limited to the five first-delivery files below. Later phase `phase-5-release` gets its own beads and owns no first-delivery files.

Phase verification: `cd taskboard && go test ./... && docker build -t taskboard:local .`

Dependency rule for this phase: the edges below form an acyclic graph. Build-context exclusions and the Kubernetes Secret template have no repo-file prerequisites. The Deployment depends on the Secret template. The Service depends on the Deployment. The Dockerfile depends on the Go module, entrypoint, and embedded web assets from earlier phases. It does not depend on `.dockerignore` as a code edge, although Docker reads `.dockerignore` as build context.

Toolchain rule (fix from failed implementation review): `taskboard/go.mod` declares `go 1.26.1`. The Dockerfile builder image must be a Go image whose version is at least the `go` directive in `go.mod` (for example `golang:1.26-alpine`, pinned by tag or digest). `golang:1.22-alpine` and `golang:1.25-alpine` both fail `go mod download` with `GOTOOLCHAIN=local`. Do not lower `go.mod` to match the image. SPEC §2 requires Go >= 1.25, so a 1.26 builder satisfies SPEC.

## Bead map

### ta-uns: taskboard/Dockerfile
- Scope: Multi-stage production container image for the TaskBoard Go binary, built from build context `taskboard/`.
- Architecture: Docker multi-stage build recipe (architecture.md, Docker & Deployment). Builder stage runs `CGO_ENABLED=0 GOOS=linux go build -o /app/taskboard ./cmd/taskboard`. Final stage is alpine:3.20 with CA certificates, a non-root numeric UID, and `/app/taskboard`.
- Builder image: a Go version at least the `go` directive in `taskboard/go.mod` (currently 1.26.1). Use `golang:1.26-alpine` or newer and pin it. Architecture.md's `golang:1.25-alpine` is superseded for this bead because it cannot build the module; record the change in the Dockerfile comment.
- Interfaces: `CMD ["/app/taskboard"]`; `EXPOSE 8080`; `ENV TASKBOARD_HOST=0.0.0.0`.
- Depends on: taskboard/go.mod, taskboard/go.sum, taskboard/cmd/taskboard/main.go, taskboard/internal/web/app.go, taskboard/internal/web/api.go, taskboard/web/assets.go, taskboard/web/templates/index.html, taskboard/web/static/app.js, taskboard/web/static/style.css
- Consumed by: <none>
- Acceptance:
  - `go mod download` in the builder stage succeeds with `GOTOOLCHAIN=local`; no "go.mod requires go >= ..." error.
  - `docker build -t taskboard:local .` run from `taskboard/` exits 0.
  - Build context is `taskboard/`; the final image has no Go toolchain or Node runtime.
  - Final image runs as a non-root numeric UID.
  - Embedded HTML, JS, and CSS are inside the binary (no runtime file copy needed).
  - No database files, `node_modules`, or secrets are copied into the image.
- Verify: `cd taskboard && go test ./... && docker build -t taskboard:local .`, then a container smoke test of `/api/health`, `/`, and `/static/app.js`.

### ta-2ok: taskboard/.dockerignore
- Scope: Build-context exclusions for the Docker build rooted at `taskboard/`.
- Architecture: `.dockerignore` excludes databases, secrets, `.git`, node_modules, test reports, and build outputs (architecture.md, Docker & Deployment).
- Interfaces: None (exclusion patterns only).
- Depends on: <none>
- Consumed by: <none> (Docker reads it as build context; there is no repo-file import edge)
- Acceptance:
  - Excludes `*.db`, `taskboard.db`, `.git`, `node_modules`, `playwright-report`, `test-results`, build outputs, and `.env` or secret files.
  - Does not exclude `cmd/`, `internal/`, `web/`, `go.mod`, or `go.sum`, which the build needs.
  - `docker build` still sends the Go sources and the embedded web assets.
- Verify: `cd taskboard && docker build -t taskboard:local .` succeeds with the exclusions in place.

### ta-0x5: taskboard/deploy/k8s/secret.example.yaml
- Scope: Example Kubernetes Secret template carrying the PostgreSQL connection URL under key `database-url`.
- Architecture: Secret example with obvious placeholder values and instructions not to apply it unchanged or commit real credentials (architecture.md, Kubernetes Secret Example).
- Interfaces: Secret key `database-url`, consumed by the Deployment env reference.
- Depends on: <none>
- Consumed by: taskboard/deploy/k8s/deployment.yaml
- Acceptance:
  - Placeholder URL only; no live password or real host credentials.
  - Key name is exactly `database-url`.
  - Comments instruct operators to create the real Secret out of band.
- Verify: YAML structure check; the Secret key name matches the env reference in deployment.yaml.

### ta-om9: taskboard/deploy/k8s/deployment.yaml
- Scope: Single-replica Kubernetes Deployment for TaskBoard.
- Architecture: Targets port 8080, liveness `GET /api/health`, readiness `GET /api/ready`, and injects `TASKBOARD_DB_DRIVER` and `TASKBOARD_DATABASE_URL` from Secret key `database-url` (architecture.md, Kubernetes Deployment).
- Interfaces: Container port 8080 named `http`; env `TASKBOARD_HOST=0.0.0.0`; Secret reference `database-url`.
- Depends on: taskboard/deploy/k8s/secret.example.yaml
- Consumed by: taskboard/deploy/k8s/service.yaml
- Acceptance:
  - `replicas: 1` by default; the image name is replaceable and not tied to a registry.
  - Non-root `securityContext`; no privileged container, host networking, or hostPath volume.
  - Resource requests and limits set.
  - No PostgreSQL Deployment, StatefulSet, or TaskBoard PVC.
- Verify: YAML structure check; probe paths; port 8080 named `http`; Secret key wiring (`database-url`).

### ta-cp8: taskboard/deploy/k8s/service.yaml
- Scope: Internal ClusterIP Service for TaskBoard.
- Architecture: `ClusterIP` Service targeting port 8080; ingress is out of scope (architecture.md, Kubernetes Service).
- Interfaces: Service port 8080 to targetPort `http` (8080); the selector matches Deployment pod labels.
- Depends on: taskboard/deploy/k8s/deployment.yaml
- Consumed by: <none>
- Acceptance:
  - `type: ClusterIP`.
  - Selector labels exactly match the Deployment template labels.
  - Port and targetPort both resolve to 8080.
- Verify: YAML structure check; selector and port match against deployment.yaml.

## Future considerations
- Ingress and TLS provisioning are out of scope per SPEC §13.
- Restart persistence and the PostgreSQL integration gate are deferred to phase-5-release.
