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
├── infra-test/
│   ├── README.md         # Local Nomad ACL setup guide
│   ├── nomad-acl.hcl     # Nomad v2.x config with ACL enabled
│   └── deployer-policy.hcl # CI/CD deployer ACL policy
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
| `IMAGE_URL` | Docker image URL with tag (e.g., `registry.example.com/myapp:v1.2.3`). The registry host is auto-extracted for login unless `DOCKER_REGISTRY` is set. Docker Hub images (no host prefix) skip registry login. |
| `DOCKER_LOGIN_USERNAME` | Docker registry username |
| `DOCKER_LOGIN_PASSWORD` | Docker registry password or access token |

### Optional — Docker
| Variable | Description |
|---|---|
| `DOCKER_REGISTRY` | Override the registry URL for `docker login`. By default, the registry is extracted from `IMAGE_URL` (e.g., `registry.example.com/myapp:v1` → login to `registry.example.com`). Set this when auto-detection fails or when the login endpoint differs from the image prefix. |

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

### Optional — Registry auth for Nomad
| Variable | Description |
|---|---|
| `NOMAD_REGISTRY_USERNAME` | Username for Nomad to pull the Docker image from a private registry. Falls back to `DOCKER_LOGIN_USERNAME` if not set. |
| `NOMAD_REGISTRY_PASSWORD` | Password for Nomad to pull the Docker image from a private registry. Falls back to `DOCKER_LOGIN_PASSWORD` if not set. |

> When both username and password are available, an `auth {}` block is included in the Nomad job config so Nomad can authenticate to the private registry when pulling the image. If your registry is public (or Docker Hub), leave these unset — no `auth {}` block is generated.

### Optional — Advanced
| Variable | Description |
|---|---|
| `ENV_SOURCE` | Path to a `.env`-style file to template into the container. Defaults to `.env` |
| `CONTAINER_DNS_SERVER` | Custom DNS server for the container |
| `CONS_ATTR` | Nomad constraint attribute (e.g., `node.class`) |
| `CONS_OP` | Constraint operator (e.g., `==`, `!=`, `regexp`) |
| `CONS_VALUE` | Constraint value. All three `CONS_*` vars must be set for the constraint to apply |

### Optional — Authentication (Nomad v2.x)

All auth mechanisms are picked up automatically from the environment — no code changes needed.
The binary calls `nomad.DefaultConfig()` which reads these env vars:

| Variable | Auth type | Description |
|---|---|---|
| `NOMAD_TOKEN` | ACL token | Nomad ACL secret ID. Sent as `X-Nomad-Token` header on every request. Required when Nomad ACL is enabled. |
| `NOMAD_HTTP_AUTH` | HTTP Basic | `username:password` for reverse-proxy or HTTP basic auth in front of Nomad |
| `NOMAD_CLIENT_CERT` | mTLS | Path to client certificate PEM file (requires `NOMAD_CLIENT_KEY`) |
| `NOMAD_CLIENT_KEY` | mTLS | Path to client private key PEM file (requires `NOMAD_CLIENT_CERT`) |
| `NOMAD_CACERT` | TLS | Path to CA certificate for verifying the Nomad server |
| `NOMAD_CAPATH` | TLS | Directory of CA certificates |
| `NOMAD_TLS_SERVER_NAME` | TLS | Override the TLS server name (SNI) |
| `NOMAD_SKIP_VERIFY` | TLS | Set to `true` to skip TLS verification (**dev only**) |

**Self-signed certificates** are supported two ways:
- **Secure**: set `NOMAD_CACERT` to the CA that signed the server cert — the client verifies the server against it
- **Skip (dev)**: set `NOMAD_SKIP_VERIFY=true` — trusts any certificate (like `curl -k`)

#### Minimal ACL setup (for CI/CD)

```bash
# Create a policy
nomad acl policy apply deployer - <<'EOF'
namespace "default" { policy = "write" capabilities = ["submit-job","read-job","list-jobs"] }
namespace "*"        { policy = "read"  capabilities = ["list-jobs"] }
node                 { policy = "read" }
EOF

# Create a token
nomad acl token create -name="ci-cd" -policy=deployer -type=client
# → Set NOMAD_TOKEN=<Secret ID> in your CI variables
```

See [`infra-test/`](infra-test/) for a complete local Nomad v2.x + ACL test environment.

## Traefik integration

The generated job includes Traefik service tags for automatic reverse-proxy configuration:

- **HTTP router** — routes traffic based on `Host()` rule (and optional `PathPrefix()`)
- **HTTPS redirect middleware** — redirects HTTP → HTTPS
- **HTTPS router** — TLS termination with Let's Encrypt certificate resolver (`myresolver`)
- **StripPrefix middleware** — strips path prefix before forwarding (when `APP_PREFIX_REGEX` is set)
- **Basic auth middleware** — password-protects the service (when `TRAEFIK_PASSWORD` is set)
- **Middleware chain** — wires middlewares together via `middlewares=<port>@consulcatalog`

All Traefik configuration happens through Consul Catalog tags — no separate Traefik config files needed.

## Example GitLab CI / GitHub Actions

```yaml
# .gitlab-ci.yml
variables:
  DOCKERFILE: Dockerfile
  IMAGE_URL: registry.example.com/$CI_PROJECT_PATH:$CI_COMMIT_SHORT_SHA
  DOCKER_LOGIN_USERNAME: $CI_REGISTRY_USER
  DOCKER_LOGIN_PASSWORD: $CI_REGISTRY_PASSWORD
  # Auto-detected: registry = registry.example.com → docker login registry.example.com
  # NOMAD_REGISTRY_USERNAME and NOMAD_REGISTRY_PASSWORD inherit from DOCKER_LOGIN_*
  # → auth {} block generated so Nomad can pull from the private registry
  NOMAD_ADDRESS: https://nomad.internal:4646
  NOMAD_TOKEN: $NOMAD_CI_TOKEN        # ACL token (set in GitLab CI Variables)
  NOMAD_CACERT: $NOMAD_CA_PEM         # CA cert if using internal PKI
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
