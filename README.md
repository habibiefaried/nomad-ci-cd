![Go](https://github.com/habibiefaried/nomad-ci-cd/workflows/Go/badge.svg)

# nomad-ci-cd

A single-binary CI/CD tool that **builds Docker images** and **deploys to HashiCorp Nomad** from any CI pipeline (GitLab CI, GitHub Actions, etc.). All configuration is driven by environment variables — no config files needed.

## Requirements

| Dependency | Version |
|---|---|
| **Go** | 1.26+ |
| **Nomad** | v2.x (tested against v2.0.3) |
| **Docker** | any recent version (only needed if `DOCKERFILE` is set) |

## Architecture

```
┌──────────────┐     ┌──────────────────┐     ┌───────────────┐
│  CI Pipeline │ ──► │  nomad-ci-cd     │ ──► │ Docker Hub    │
│ (GitLab CI)  │     │  (this binary)   │     │ (push image)  │
└──────────────┘     │                  │     └───────────────┘
                     │  main.go         │
                     │  ├─ helper/      │     ┌───────────────┐
                     │  │  ├─ main.go   │ ──► │ Nomad v2.x    │
                     │  │  └─ docker.go │     │ Cluster       │
                     │  └─ nomad/       │     │ (submit job)  │
                     │     └─ main.go   │     └───────────────┘
                     └──────────────────┘
```

## Package structure

```
.
├── main.go              # Entry point — orchestrates the pipeline
├── helper/
│   ├── main.go          # Shell command runner
│   └── docker.go        # Docker build → login → push logic
├── nomad/
│   ├── main.go          # Nomad job HCL generation & API submission
│   └── main_test.go     # Unit + integration tests
├── go.mod               # Go 1.26, Nomad v2.x API client
└── .github/workflows/   # GitHub Actions CI (GoReleaser)
```

## How it works

### Phase 1 — Debug info
Prints the CI environment (`env`) and the runner's public IP (via `curl https://api.ipify.org`). Useful for debugging network issues in CI.

### Phase 2 — Docker build & push (`helper/docker.go`)
Skipped if `DOCKERFILE` or `IMAGE_URL` is not set. When enabled:
1. **Build**: `docker build -f $DOCKERFILE -t $IMAGE_URL .`
2. **Login**: `docker login --username $DOCKER_LOGIN_USERNAME --password-stdin`
3. **Push**: `docker push $IMAGE_URL`

### Phase 3 — Nomad deployment (`nomad/main.go`)
Skipped if `NOMAD_ADDRESS` is not set. When enabled:
1. **Generate** a Nomad job HCL from environment variables (job name, replicas, ports, CPU/memory, Traefik tags, DNS, constraints, env-file templates)
2. **Parse** the HCL via the Nomad v2.x API client (`client.Jobs().ParseHCL()`)
3. **Register** the job in the Nomad cluster (`client.Jobs().Register()`)

The generated HCL defines a `service` job with:
- A single `app` task group with configurable count
- A `docker` driver task using the image from Phase 2
- A `service` block with Traefik reverse-proxy tags for routing, TLS, middleware
- Optional: placement constraints, custom DNS servers, env-file templating

## Environment variables

### Required for Docker
| Variable | Description |
|---|---|
| `DOCKERFILE` | Path to the Dockerfile (e.g., `Dockerfile`, `build/Dockerfile.prod`) |
| `IMAGE_URL` | Docker image URL with tag (e.g., `registry.example.com/myapp:v1.2.3`) |
| `DOCKER_LOGIN_USERNAME` | Docker Hub username |
| `DOCKER_LOGIN_PASSWORD` | Docker Hub password or access token |

### Required for Nomad
| Variable | Description |
|---|---|
| `NOMAD_ADDRESS` | Nomad server address (e.g., `http://nomad.example.com:4646`). If unset, skip deployment |
| `NOMAD_CUSTOM_NAME` | Base name for the Nomad job |
| `DEPLOY_ENVIRONMENT` | Environment name appended to job name (`staging`, `prod`, etc.) |
| `NUM_REPLICA` | Number of container replicas |
| `PORT_NAME` | Port label — must be unique across the Nomad cluster |
| `TARGET_PORT` | Port the container listens on |
| `IMAGE_URL` | Docker image to deploy (same as Docker phase) |
| `JOB_CPU` | CPU allocation in MHz |
| `JOB_MEMORY` | Memory allocation in MB |
| `APP_HOST` | Domain name for Traefik routing. Use `#` to separate multiple domains |

### Optional — Traefik routing
| Variable | Description |
|---|---|
| `APP_PREFIX_REGEX` | URL path prefix to route (e.g., `/api`). Enables PathPrefix rule and stripprefix middleware |
| `TRAEFIK_PASSWORD` | Apache htpasswd-compatible credentials for basic auth protection |

### Optional — Advanced
| Variable | Description |
|---|---|
| `ENV_SOURCE` | Path to a `.env`-style file to template into the container. Defaults to `.env` |
| `CONTAINER_DNS_SERVER` | Custom DNS server for the container |
| `CONS_ATTR` | Nomad constraint attribute (e.g., `node.class`) |
| `CONS_OP` | Constraint operator (e.g., `==`, `!=`, `regexp`) |
| `CONS_VALUE` | Constraint value. All three `CONS_*` vars must be set for the constraint to apply |

## Traefik integration

The generated job includes Traefik service tags for automatic reverse-proxy configuration:

- **HTTP router** — routes traffic based on `Host()` rule (and optional `PathPrefix()`)
- **HTTPS redirect middleware** — redirects HTTP → HTTPS
- **HTTPS router** — TLS termination with Let's Encrypt certificate resolver (`myresolver`)
- **StripPrefix middleware** — strips path prefix before forwarding (when `APP_PREFIX_REGEX` is set)
- **Basic auth middleware** — password-protects the service (when `TRAEFIK_PASSWORD` is set)
- **Middleware chain** — wires middlewares together via `middlewares=<port>@consulcatalog`

All Traefik configuration happens through Consul Catalog tags — no separate Traefik config files needed.

## Example GitLab CI

```yaml
# .gitlab-ci.yml
variables:
  DOCKERFILE: Dockerfile
  IMAGE_URL: registry.gitlab.com/$CI_PROJECT_PATH:$CI_COMMIT_SHORT_SHA
  DOCKER_LOGIN_USERNAME: $CI_REGISTRY_USER
  DOCKER_LOGIN_PASSWORD: $CI_REGISTRY_PASSWORD
  NOMAD_ADDRESS: http://nomad.internal:4646
  NOMAD_CUSTOM_NAME: my-api
  DEPLOY_ENVIRONMENT: staging
  NUM_REPLICA: "2"
  PORT_NAME: http
  TARGET_PORT: "8080"
  JOB_CPU: "500"
  JOB_MEMORY: "256"
  APP_HOST: api.example.com
  APP_PREFIX_REGEX: /api

deploy:
  stage: deploy
  image: golang:1.26
  script:
    - go run .
```

## Development

```bash
# Build
go build ./...

# Run tests (unit + HCL parsing)
go test -v -run "TestConstraintGenerator|TestGenerateDNSServer|TestHostGenerator|TestTagGenerator|TestJobGeneration|TestHCLParsing" ./nomad/

# Run integration test against local Nomad
go test -v -run TestSubmitJob_LocalNomad ./nomad/

# Run all tests
go test -v ./...

# Static analysis
go vet ./...
```

The integration test requires a local Nomad running in dev mode:
```bash
nomad agent -dev
```

## Release

Uses [GoReleaser](https://goreleaser.com/) — triggered by pushing a `v*` tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Builds a static Linux amd64/386 binary. Config in `goreleaser.yml`.

## License

MIT
