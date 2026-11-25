# Implementation plan

Active phase: phase-4-deploy (Container build and k3s deployment artifacts). Only the five bead-owned paths below are in scope. Phases 1 through 3 are context only and are not re-planned here.

Dependency order is acyclic: ta-uns -> ta-2ok -> ta-0x5 -> ta-om9 -> ta-cp8. The Secret example is a leaf input to the Deployment and does not depend on it.

## Bead map

### ta-uns: taskboard/Dockerfile
- Scope: multi-stage Dockerfile, build context `taskboard/`.
- Architecture: golang:1.25 builder stage running `CGO_ENABLED=0 GOOS=linux go build -o /app/taskboard ./cmd/taskboard`; final alpine:3.20 stage with CA certificates, a non-root numeric UID, the compiled binary `/app/taskboard`, and embedded templates and static assets; `ENV TASKBOARD_HOST=0.0.0.0`; `EXPOSE 8080`; `CMD ["/app/taskboard"]`.
- Interfaces: none (build recipe only).
- Depends on: <none> within phase-4-deploy. Build inputs from earlier phases (taskboard/go.mod, taskboard/go.sum, taskboard/cmd/taskboard/main.go, taskboard/web/assets.go, taskboard/web/templates/index.html, taskboard/web/static/app.js, taskboard/web/static/style.css) are read-only context.
- Consumed by: taskboard/deploy/k8s/deployment.yaml (image `taskboard:local`, port 8080).
- Acceptance: builder base image is golang:1.25 or newer, never 1.22; build target is `./cmd/taskboard`; output binary is named `taskboard`; final stage sets a numeric non-root `USER`; no comment claims the app runs on the host; final image has no Go toolchain or Node runtime.
- Verify: `cd taskboard && go test ./... && docker build -t taskboard:local .`

### ta-2ok: taskboard/.dockerignore
- Scope: build-context exclusions for `taskboard/`.
- Architecture: exclude databases, secrets, `.git`, `node_modules`, Playwright reports, test output, and build outputs.
- Interfaces: none.
- Depends on: <none>.
- Consumed by: taskboard/Dockerfile.
- Acceptance: excludes `.git`, `node_modules`, `*.db`, `.env*`, `*.pem`, `*.key`, `playwright-report`, `test-results`, `coverage.out`, and the deploy directory; does not exclude `web/`, `cmd/`, or `internal/`, since the build needs them.
- Verify: `cd taskboard && docker build -t taskboard:local .`

### ta-0x5: taskboard/deploy/k8s/secret.example.yaml
- Scope: example Secret only, with placeholder values.
- Architecture: Secret `taskboard-db`, key `database-url`, placeholder credentials, and a comment telling the operator not to apply it unchanged.
- Interfaces: none.
- Depends on: <none>.
- Consumed by: taskboard/deploy/k8s/deployment.yaml (Secret name and key); operators creating the real Secret out of band.
- Acceptance: contains only `CHANGE_ME` placeholders; no live password; key name is `database-url`; Secret name is `taskboard-db`.
- Verify: YAML parse of the file.

### ta-om9: taskboard/deploy/k8s/deployment.yaml
- Scope: single-replica Deployment on port 8080.
- Architecture: liveness probe `httpGet` path `/api/health`; readiness probe `httpGet` path `/api/ready`; container `securityContext` with `runAsNonRoot: true` and `allowPrivilegeEscalation: false`; resource requests and limits; `TASKBOARD_DB_DRIVER=postgres`; `TASKBOARD_DATABASE_URL` from Secret `taskboard-db` key `database-url`; no privileged container, host networking, or hostPath.
- Interfaces: none.
- Depends on: taskboard/Dockerfile (image and port), taskboard/deploy/k8s/secret.example.yaml (Secret name and key).
- Consumed by: taskboard/deploy/k8s/service.yaml (selector `app: taskboard`, port name `http`).
- Acceptance: probes target `/api/health` and `/api/ready`, not `/`; the port name `http` matches the Service targetPort; labels match the Service selector; the Secret reference matches the example Secret name and key.
- Verify: YAML parse plus selector and probe-path cross-check against the Service and the SPEC route table.

### ta-cp8: taskboard/deploy/k8s/service.yaml
- Scope: internal ClusterIP Service.
- Architecture: type ClusterIP; port 80 to targetPort `http` (8080); selector `app: taskboard`.
- Interfaces: none.
- Depends on: taskboard/deploy/k8s/deployment.yaml (labels and port name).
- Consumed by: none within the repository.
- Acceptance: selector matches the Deployment labels; no Ingress is defined.
- Verify: YAML parse and selector cross-check.

## Future considerations
- Ingress and TLS are out of scope per SPEC §9.
- Multi-architecture builds (linux/amd64 and linux/arm64) are a documentation item, not an acceptance criterion for this phase.
