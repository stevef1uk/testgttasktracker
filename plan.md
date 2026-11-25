# Implementation plan — phase-4-deploy (Container build and k3s deployment artifacts)

Active phase: `phase-4-deploy`. Depends on `phase-3-browser`. Scope is limited to the five first-delivery files below. Later phases (phase-5-release) get their own beads.

Phase verification: `cd taskboard && go test ./... && docker build -t taskboard:local .`

Dependency rule for this phase: edges below form an acyclic graph. Build-context exclusions and Kubernetes Secret templates have no repo-file prerequisites; the Deployment depends on the Secret template; the Service depends on the Deployment. The Dockerfile depends on the Go module, entrypoint, and embedded web assets from earlier phases, not on `.dockerignore` (Docker reads `.dockerignore` as build context, which is not a file-level code dependency).

## Bead map

### ta-0x5: taskboard/deploy/k8s/secret.example.yaml
- Scope: Example Kubernetes Secret template carrying the PostgreSQL connection URL under key `database-url`.
- Architecture: Secret example with obvious placeholder values and instructions not to apply it unchanged or commit real credentials (architecture.md, Kubernetes Secret Example).
- Interfaces: Secret name and key `database-url`, consumed by the Deployment env reference.
- Depends on: <none>
- Consumed by: taskboard/deploy/k8s/deployment.yaml
- Acceptance:
  - Placeholder URL only; no live password or real host credentials.
  - Key name is exactly `database-url`.
  - Comments instruct operators to create the real Secret out of band.
- Verify: YAML structure check; Secret key name matches the env reference in deployment.yaml.

### ta-2ok: taskboard/.dockerignore
- Scope: Build-context exclusions for the Docker build rooted at `taskboard/`.
- Architecture: `.dockerignore` excludes databases, secrets, `.git`, node_modules, test reports, and build outputs (architecture.md, Docker & Deployment).
- Interfaces: None (exclusion patterns only).
- Depends on: <none>
- Consumed by: <none> (Docker reads it as build context; no repo-file import edge)
- Acceptance:
  - Excludes `*.db`, `taskboard.db`, `.git`, `node_modules`, `playwright-report`, `test-results`, build outputs, and `.env` or secret files.
  - Does not exclude `cmd/`, `internal/`, or `web/` sources required by the build.
- Verify: `cd taskboard && docker build -t taskboard:local .` succeeds with exclusions in place.

### ta-uns: taskboard/Dockerfile
- Scope: Multi-stage production container image for the TaskBoard Go binary.
- Architecture: Docker multi-stage build recipe (architecture.md, Docker & Deployment). Builder stage uses golang:1.25-alpine and runs `CGO_ENABLED=0 GOOS=linux go build -o /app/taskboard ./cmd/taskboard`. Final stage uses alpine:3.20 with a non-root numeric UID, CA certificates, and `/app/taskboard`.
- Interfaces: `CMD ["/app/taskboard"]`; `EXPOSE 8080`; sets `TASKBOARD_HOST=0.0.0.0`.
- Depends on: taskboard/go.mod, taskboard/go.sum, taskboard/cmd/taskboard/main.go, taskboard/internal/web/app.go, taskboard/internal/web/api.go, taskboard/web/assets.go, taskboard/web/templates/index.html, taskboard/web/static/app.js, taskboard/web/static/style.css
- Consumed by: <none>
- Acceptance:
  - Build context is `taskboard/`; no Go toolchain or Node runtime in the final image.
  - Final image runs as a non-root numeric UID.
  - Embedded HTML, JS, and CSS are present in the binary (no runtime file copy needed).
  - No database files, node_modules, or secrets are copied into the image.
- Verify: `cd taskboard && docker build -t taskboard:local .`, then container smoke test of `/api/health`, `/`, and `/static/app.js`.

### ta-om9: taskboard/deploy/k8s/deployment.yaml
- Scope: Single-replica Kubernetes Deployment for TaskBoard.
- Architecture: Targets port 8080, liveness `GET /api/health`, readiness `GET /api/ready`, injects `TASKBOARD_DB_DRIVER` and `TASKBOARD_DATABASE_URL` from Secret key `database-url` (architecture.md, Kubernetes Deployment).
- Interfaces: Container port 8080; env `TASKBOARD_HOST=0.0.0.0`; Secret reference `database-url`.
- Depends on: taskboard/deploy/k8s/secret.example.yaml
- Consumed by: taskboard/deploy/k8s/service.yaml
- Acceptance:
  - `replicas: 1` by default; image name is replaceable and not tied to a specific registry.
  - Non-root `securityContext`; no privileged container, host networking, or hostPath volume.
  - Resource requests and limits set.
  - No PostgreSQL Deployment, StatefulSet, or TaskBoard PVC.
- Verify: YAML structure check; probe paths; port 8080; Secret key wiring (`database-url`).

### ta-cp8: taskboard/deploy/k8s/service.yaml
- Scope: Internal ClusterIP Service for TaskBoard.
- Architecture: `ClusterIP` Service targeting port 8080; ingress is out of scope (architecture.md, Kubernetes Service).
- Interfaces: Service port 8080 to targetPort 8080; selector matches Deployment pod labels.
- Depends on: taskboard/deploy/k8s/deployment.yaml
- Consumed by: <none>
- Acceptance:
  - `type: ClusterIP`.
  - Selector labels exactly match Deployment template labels.
  - Port and targetPort both 8080.
- Verify: YAML structure check; selector and port match against deployment.yaml.

## Future considerations
- Ingress and TLS provisioning are out of scope per SPEC section 13.
- Restart persistence and the PostgreSQL integration gate are deferred to phase-5-release.
