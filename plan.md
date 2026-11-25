# Implementation plan (phase-4-deploy)

Dependency order (earliest first): ta-0x5, ta-2ok, ta-uns, ta-om9, ta-cp8.

## Bead map

### ta-0x5: taskboard/deploy/k8s/secret.example.yaml
- Scope: Example-only Secret carrying DATABASE_URL, never applied as-is.
- Architecture: architecture.md, Kubernetes secret example (phase-4-deploy).
- Interfaces: Secret `taskboard-db`, key `database-url`, placeholder values only (CHANGE_ME_*).
- Depends on: <none>.
- Consumed by: taskboard/deploy/k8s/deployment.yaml (ta-om9) via Secret name and key.
- Acceptance: file header warns it must not be applied unchanged and must not hold real credentials.
- Acceptance: contains only placeholder credentials; no real host, user or password.
- Verify: kubectl apply --dry-run=client -f taskboard/deploy/k8s/secret.example.yaml (YAML must parse).

### ta-2ok: taskboard/.dockerignore
- Scope: Build context exclusions for Docker.
- Architecture: architecture.md, build context exclusions.
- Interfaces: none (exclusion file only).
- Depends on: <none>.
- Consumed by: taskboard/Dockerfile (ta-uns).
- Acceptance: excludes .git, node_modules, test reports, build outputs, databases, secrets (.env, *.pem, *.key) and deploy/ manifests.
- Acceptance: does not exclude go.mod, go.sum, cmd/, internal/ or web/ sources needed by the build.
- Verify: cd taskboard && docker build -t taskboard:local . (context must still contain go sources).

### ta-uns: taskboard/Dockerfile
- Scope: Multi-stage Docker build recipe for the taskboard binary.
- Architecture: architecture.md, Docker multi-stage build recipe (taskboard/Dockerfile).
- Interfaces: builder compiles `CGO_ENABLED=0 GOOS=linux go build -o /app/taskboard ./cmd/taskboard`; final stage alpine:3.20 running as a non-root numeric UID; `EXPOSE 8080`; `TASKBOARD_HOST=0.0.0.0`; CA certificates installed in the final image.
- Depends on: taskboard/go.mod, taskboard/go.sum, taskboard/cmd/taskboard/main.go, taskboard/.dockerignore (ta-2ok).
- Consumed by: taskboard/deploy/k8s/deployment.yaml (ta-om9) via image `taskboard:local`.
- Acceptance: builder image is a golang tag whose version is >= the go directive in go.mod (go.mod declares go 1.26.1, so use golang:1.26-alpine; architecture's golang:1.25-alpine is too old and must not be used).
- Acceptance: `go mod download` succeeds in the builder with GOTOOLCHAIN=local.
- Acceptance: final image contains no Go toolchain or Node runtime; web assets are embedded so no separate web directory is needed at runtime.
- Acceptance: non-root numeric UID in the final stage.
- Verify: cd taskboard && go test ./... && docker build -t taskboard:local .

### ta-om9: taskboard/deploy/k8s/deployment.yaml
- Scope: Kubernetes Deployment for the taskboard container.
- Architecture: architecture.md, Kubernetes deployment manifests.
- Interfaces: Deployment `taskboard`, label `app: taskboard`, container port `http` 8080, env DATABASE_URL from Secret `taskboard-db` key `database-url`, readiness and liveness HTTP probes on `/`.
- Depends on: taskboard/Dockerfile (ta-uns) for image `taskboard:local`; taskboard/deploy/k8s/secret.example.yaml (ta-0x5) for the Secret name and key.
- Consumed by: taskboard/deploy/k8s/service.yaml (ta-cp8) via label selector `app: taskboard`.
- Acceptance: replicas 1; imagePullPolicy IfNotPresent; probes target port name `http`.
- Acceptance: no credentials embedded in the manifest.
- Verify: kubectl apply --dry-run=client -f taskboard/deploy/k8s/deployment.yaml (YAML must parse).

### ta-cp8: taskboard/deploy/k8s/service.yaml
- Scope: Kubernetes ClusterIP Service for the taskboard Deployment.
- Architecture: architecture.md, Kubernetes Service configuration.
- Interfaces: Service `taskboard`, type ClusterIP, selector `app: taskboard`, port 80 to targetPort `http`.
- Depends on: taskboard/deploy/k8s/deployment.yaml (ta-om9) for the selector and port name.
- Consumed by: none.
- Acceptance: selector matches Deployment labels; no secret values present.
- Verify: kubectl apply --dry-run=client -f taskboard/deploy/k8s/service.yaml.

## Future considerations
- Architecture lists golang:1.25-alpine but go.mod requires go 1.26.1; architecture.md should be updated to golang:1.26-alpine in a later review.
