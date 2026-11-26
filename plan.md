# Implementation plan — phase-4-deploy

Active phase: phase-4-deploy (Container build and k3s deployment artifacts). Depends on phase-3-browser, which is complete. This plan covers only the five first-delivery files owned by this phase. Later phases (phase-5-release) own no first-delivery files.

## Bead map

### ta-2r6: taskboard/Dockerfile
- Scope: Multi-stage container build for the TaskBoard Go binary. Build context is taskboard/.
- Architecture: Docker & Deployment section. Builder stage uses golang:1.25-alpine with CGO_ENABLED=0 and GOOS=linux, building taskboard/cmd/taskboard into a static binary. Runtime stage uses alpine:3.20 with CA certificates and a non-root UID. Only the binary is copied to /app/taskboard. The image contains no Go toolchain, Node runtime, or source tree.
- Interfaces: Container entrypoint CMD ["/app/taskboard"]; EXPOSE 8080; ENV TASKBOARD_HOST=0.0.0.0. The image embeds templates and static assets through taskboard/web/assets.go (go:embed).
- Depends on: taskboard/cmd/taskboard/main.go, taskboard/web/assets.go, taskboard/go.mod, taskboard/go.sum (phase-1-core and phase-1-app).
- Consumed by: taskboard/deploy/k8s/deployment.yaml (image reference and containerPort 8080).
- Acceptance:
  - Builder sets CGO_ENABLED=0 and GOOS=linux; the binary is built from taskboard/cmd/taskboard.
  - Final stage is alpine:3.20 with ca-certificates installed.
  - Container runs as a non-root numeric UID; no root process.
  - EXPOSE 8080 and CMD ["/app/taskboard"] are present.
  - TASKBOARD_HOST defaults to 0.0.0.0 inside the image.
  - No database file, .git, node_modules, or secrets are copied into the image.
- Verify: cd taskboard && go test ./... && docker build -t taskboard:local .

### ta-1zs: taskboard/.dockerignore
- Scope: Exclude non-build artifacts from the Docker build context rooted at taskboard/.
- Architecture: Docker & Deployment section and the .dockerignore row of the Planned file layout. Build context is taskboard/.
- Interfaces: None (build-context filter only).
- Depends on: taskboard/Dockerfile (ta-2r6).
- Consumed by: taskboard/Dockerfile build step.
- Acceptance:
  - Excludes taskboard.db, *.db, and other SQLite database files.
  - Excludes .git, node_modules, playwright-report, test-results, and build outputs.
  - Excludes real secret manifests (any real Secret file, not secret.example.yaml).
  - Does not exclude go.mod, go.sum, cmd/, internal/, or web/ needed for the build.
- Verify: cd taskboard && docker build -t taskboard:local . (confirm build succeeds and image contains no database file).

### ta-w3w: taskboard/deploy/k8s/deployment.yaml
- Scope: Single-replica Kubernetes Deployment for TaskBoard using PostgreSQL configuration injected from a Secret.
- Architecture: Kubernetes manifests section. Single replica by default. Container port 8080. Non-root securityContext. Requests and limits set. Liveness probe GET /api/health. Readiness probe GET /api/ready. Environment TASKBOARD_DB_DRIVER=postgres and TASKBOARD_HOST=0.0.0.0. TASKBOARD_DATABASE_URL is read from Secret key database-url. Image name is clearly replaceable with no assumed registry.
- Interfaces: Pod labels consumed by the Service selector; container port name http on 8080.
- Depends on: taskboard/Dockerfile (ta-2r6) for the image; taskboard/deploy/k8s/secret.example.yaml (ta-31x) for the Secret key name database-url.
- Consumed by: taskboard/deploy/k8s/service.yaml (selector must match pod labels).
- Acceptance:
  - replicas: 1.
  - containerPort 8080; service targetPort matches.
  - runAsNonRoot: true and a non-zero runAsUser; allowPrivilegeEscalation false.
  - No privileged container, hostNetwork, or hostPath volume.
  - livenessProbe httpGet path /api/health port 8080; readinessProbe httpGet path /api/ready port 8080.
  - env TASKBOARD_DATABASE_URL uses valueFrom.secretKeyRef with key database-url.
  - env TASKBOARD_DB_DRIVER is postgres.
  - resources.requests and resources.limits are set.
  - No PostgreSQL Deployment, StatefulSet, or PVC is defined.
- Verify: validate YAML structure (e.g. python yaml or kubectl --dry-run=client if available; otherwise report not run) and confirm the probe paths, port, and secretKeyRef key match the spec.

### ta-bmz: taskboard/deploy/k8s/service.yaml
- Scope: Internal ClusterIP Service exposing the TaskBoard Deployment.
- Architecture: Kubernetes manifests section. Type ClusterIP. Targets port 8080. No Ingress.
- Interfaces: Service port 8080 mapping to targetPort 8080.
- Depends on: taskboard/deploy/k8s/deployment.yaml (ta-w3w) for pod labels.
- Consumed by: Cluster-internal clients and any operator-provided ingress configuration documented in README.
- Acceptance:
  - type: ClusterIP.
  - port 8080 and targetPort 8080 (or named port http).
  - selector labels match the Deployment pod template labels exactly.
  - No NodePort or LoadBalancer type.
  - No Ingress resource is defined.
- Verify: validate YAML structure and confirm selector labels equal the Deployment pod template labels.

### ta-31x: taskboard/deploy/k8s/secret.example.yaml
- Scope: Example Kubernetes Secret with obvious placeholders for the PostgreSQL connection URL. Example only.
- Architecture: Kubernetes manifests section and SPEC section 9. Placeholder values only; no live password. Key name database-url. Comments instruct the operator not to apply it unchanged or commit real credentials, and to prefer creating the Secret out of band.
- Interfaces: Secret key database-url consumed by TASKBOARD_DATABASE_URL in the Deployment.
- Depends on: none (standalone manifest; key name fixed by spec).
- Consumed by: taskboard/deploy/k8s/deployment.yaml (secretKeyRef name and key).
- Acceptance:
  - kind: Secret with a placeholder name (for example taskboard-db).
  - Contains a database-url key whose value is an obvious placeholder such as postgres://taskboard:CHANGE_ME@postgres.example.internal:5432/taskboard?sslmode=require.
  - Includes a comment that the file must not be applied unchanged or committed with real credentials.
  - Contains no real password, URL, or host from a live cluster.
- Verify: validate YAML structure and confirm the database-url key name matches the Deployment secretKeyRef.

## Phase gate

cd taskboard && go test ./... && docker build -t taskboard:local . Then run a local container smoke test for GET /api/health, GET /, and static files. If Docker or Kubernetes tooling is unavailable, record the check as not run and do not claim it passed. Never print or commit real credentials.

## Future considerations

- Ingress/TLS configuration is out of scope per SPEC and is documented in README only.
- Multi-architecture build (linux/amd64, linux/arm64) is a release-time concern and is not required by this phase's acceptance.
