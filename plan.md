# Implementation plan (phase-4-deploy)

Scope: container build and k3s deployment artifacts only. Five files, one bead each. Backend, frontend, and test paths belong to other phases and get no beads here.

Verify for every bead: `cd taskboard && go test ./... && docker build -t taskboard:local .` (run from the rig working directory).

## Bead map

### ta-uns: taskboard/Dockerfile
- Scope: multi-stage container build for the taskboard Go service.
- Architecture: architecture.md, deployment section for taskboard/Dockerfile.
- Acceptance:
  - Builder stage compiles with CGO_ENABLED=0 and produces a static binary for taskboard/cmd/taskboard.
  - Runtime stage is minimal and includes CA certificates so outbound TLS to Postgres (sslmode=require) works.
  - Container runs as a non-root numeric UID; no USER root in the final stage.
  - EXPOSE 8080 matches the port the server listens on per SPEC.
  - No secrets, credentials, or database files are copied into the image.
- Verify: `cd taskboard && docker build -t taskboard:local .`

### ta-2ok: taskboard/.dockerignore
- Scope: keep build context small and free of secrets and local artifacts.
- Architecture: architecture.md, deployment section for taskboard/.dockerignore.
- Acceptance:
  - Excludes .git, node_modules, playwright-report, test-results, and blob-report.
  - Excludes local databases (for example *.db, *.sqlite) and .env files and *.pem / *.key files.
  - Excludes test output (coverage.out, *.test, *.out, *.log).
  - Does not exclude go.mod, go.sum, or the sources the Dockerfile builds.
- Verify: `cd taskboard && docker build -t taskboard:local .`

### ta-om9: taskboard/deploy/k8s/deployment.yaml
- Scope: single-replica k3s Deployment for the taskboard image.
- Architecture: architecture.md, deployment section for taskboard/deploy/k8s/deployment.yaml.
- Acceptance:
  - Container port 8080 is named http and matches the Service targetPort.
  - Liveness and readiness httpGet probes target the health and ready endpoints named in architecture.md (/api/health and /api/ready if SPEC defines them). Reconcile any probe path that differs from the cached "/" value.
  - securityContext sets runAsNonRoot: true and a non-zero runAsUser; allowPrivilegeEscalation is false.
  - DATABASE_URL is read via secretKeyRef from secret taskboard-db, key database-url. No literal credentials.
- Verify: `kubectl apply --dry-run=client -f taskboard/deploy/k8s/deployment.yaml`

### ta-cp8: taskboard/deploy/k8s/service.yaml
- Scope: ClusterIP Service fronting the Deployment.
- Architecture: architecture.md, deployment section for taskboard/deploy/k8s/service.yaml.
- Acceptance:
  - Selector app: taskboard matches the Deployment pod labels exactly.
  - Service port 80 maps to targetPort http, which resolves to containerPort 8080.
  - Type is ClusterIP; no NodePort or LoadBalancer.
- Verify: `kubectl apply --dry-run=client -f taskboard/deploy/k8s/service.yaml`

### ta-0x5: taskboard/deploy/k8s/secret.example.yaml
- Scope: documented example of the database Secret. Never applied as-is.
- Architecture: architecture.md, deployment section for taskboard/deploy/k8s/secret.example.yaml.
- Acceptance:
  - Secret name taskboard-db and key database-url match the Deployment's secretKeyRef.
  - Values are obvious placeholders (CHANGE_ME_USER, CHANGE_ME_PASSWORD, CHANGE_ME_HOST), not real credentials.
  - Header comment states the file must not be applied unchanged or committed with real values, and shows the kubectl create secret command.
- Verify: `kubectl apply --dry-run=client -f taskboard/deploy/k8s/secret.example.yaml`

## Future considerations
- Ingress and TLS termination are not in phase-4-deploy scope; add in a later phase if SPEC requires them.
