# infra-test

Configuration files and instructions for running a local Nomad v2.x cluster
with ACL authentication enabled — used for integration testing `nomad-ci-cd`.

## Files

| File | Purpose |
|---|---|
| `nomad-acl.hcl` | Nomad v2.x agent config — single server + client, ACL + optional TLS |
| `deployer-policy.hcl` | ACL policy for CI/CD pipelines — job submit/read, node read |
| `generate-certs.ps1` | PowerShell script — generates self-signed TLS certs for local dev |

## Quick start

### 1. Start Nomad with ACL

```powershell
# From the repo root:
nomad agent -config=infra-test\nomad-acl.hcl
```

This starts a single-node Nomad v2.x with:
- ACL authentication enabled
- `raw_exec` driver (no Docker needed)
- Data persisted in `./nomad-data/`

### 2. Bootstrap the ACL system (new terminal)

```powershell
nomad acl bootstrap
```

**Save the output!** You get one management token. Example output:

```
Accessor ID  = abc12345-...
Secret ID    = xyz78901-...    ← MANAGEMENT TOKEN — save this
```

### 3. Create a CI/CD deployer token

```powershell
# Set the management token so subsequent commands are authorized
$env:NOMAD_TOKEN = "<management-token-from-step-2>"

# Apply the deployer policy
nomad acl policy apply deployer infra-test\deployer-policy.hcl

# Create a token for CI/CD use
nomad acl token create -name="ci-cd" -policy=deployer -type=client
```

Save the `Secret ID` from this output — that's your **CI/CD token**.

### 4. Verify it works

```powershell
# Without token — should fail for job operations
$env:NOMAD_TOKEN = ""
go test -v -run TestSubmitJob_LocalNomad ./nomad/

# With bad token — should fail with 403
$env:NOMAD_TOKEN = "bad-token"
go test -v -run TestSubmitJob_WithAuthToken ./nomad/

# With real token — should succeed
$env:NOMAD_TOKEN = "<ci-cd-token>"
go test -v -run TestSubmitJob_WithAuthToken ./nomad/
```

### 5. Test the full binary

```powershell
$env:NOMAD_TOKEN = "<ci-cd-token>"
$env:NOMAD_ADDRESS = "http://127.0.0.1:4646"
$env:NOMAD_CUSTOM_NAME = "manual-test"
$env:DEPLOY_ENVIRONMENT = "dev"
$env:NUM_REPLICA = "1"
$env:PORT_NAME = "http"
$env:TARGET_PORT = "8080"
$env:IMAGE_URL = "nginx:latest"
$env:JOB_CPU = "50"
$env:JOB_MEMORY = "32"
$env:APP_HOST = "manual-test.local"

go run .
```

## ACL policy explained

The `deployer-policy.hcl` grants:

| Namespace | Access | Capabilities |
|---|---|---|
| `default` | `write` | `submit-job`, `read-job`, `list-jobs`, `alloc-exec` |
| `*` (all others) | `read` | `list-jobs` |
| Nodes | `read` | — |

This is the minimum needed for `nomad-ci-cd` to:
- **Register** jobs in the default namespace
- **Read** job status and node info
- **List** jobs across all namespaces

For production, lock this down further — remove `alloc-exec` if you don't
need remote task execution, and restrict the wildcard namespace.

## TLS with self-signed certificates

Nomad v2.x fully supports self-signed certs. Two modes:

| Mode | What it does | Client needs |
|---|---|---|
| **Server-only TLS** | Nomad presents a cert, client verifies it | `NOMAD_CACERT` (or `NOMAD_SKIP_VERIFY=true` for dev) |
| **mTLS** | Both server AND client present certs | `NOMAD_CACERT` + `NOMAD_CLIENT_CERT` + `NOMAD_CLIENT_KEY` |

### Prerequisites

Install OpenSSL:
- **Windows**: https://slproweb.com/products/Win32OpenSSL.html
- **Linux/macOS**: `apt install openssl` / `brew install openssl`

### Generate certs

```powershell
# Windows (PowerShell)
powershell -ExecutionPolicy Bypass -File infra-test\generate-certs.ps1
```

```bash
# Linux/macOS (bash)
mkdir -p infra-test/certs

# CA
openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 \
  -subj "/CN=nomad-ca" \
  -keyout infra-test/certs/nomad-ca-key.pem \
  -out infra-test/certs/nomad-ca.pem

# Server cert (with SANs for localhost)
cat > infra-test/certs/server.cnf <<'EOF'
[req] default_bits=4096 prompt=no default_md=sha256
distinguished_name=dn req_extensions=v3_req
[dn] CN=localhost
[v3_req] keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:localhost,DNS:nomad.local,IP:127.0.0.1,IP:::1
EOF
openssl req -new -newkey rsa:4096 -nodes \
  -config infra-test/certs/server.cnf \
  -keyout infra-test/certs/nomad-server-key.pem \
  -out infra-test/certs/nomad-server.csr
openssl x509 -req -days 3650 \
  -in infra-test/certs/nomad-server.csr \
  -CA infra-test/certs/nomad-ca.pem \
  -CAkey infra-test/certs/nomad-ca-key.pem -CAcreateserial \
  -extfile infra-test/certs/server.cnf -extensions v3_req \
  -out infra-test/certs/nomad-server.pem

# Client cert (for mTLS)
cat > infra-test/certs/client.cnf <<'EOF'
[req] default_bits=4096 prompt=no default_md=sha256
distinguished_name=dn req_extensions=v3_req
[dn] CN=nomad-ci-cd-client
[v3_req] keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF
openssl req -new -newkey rsa:4096 -nodes \
  -config infra-test/certs/client.cnf \
  -keyout infra-test/certs/nomad-client-key.pem \
  -out infra-test/certs/nomad-client.csr
openssl x509 -req -days 3650 \
  -in infra-test/certs/nomad-client.csr \
  -CA infra-test/certs/nomad-ca.pem \
  -CAkey infra-test/certs/nomad-ca-key.pem -CAcreateserial \
  -extfile infra-test/certs/client.cnf -extensions v3_req \
  -out infra-test/certs/nomad-client.pem

# Cleanup
rm infra-test/certs/*.csr infra-test/certs/*.cnf infra-test/certs/*.srl
```

### What gets generated

```
infra-test/certs/
├── nomad-ca.pem              # CA certificate — distribute to all clients
├── nomad-ca-key.pem          # CA private key — KEEP SECRET
├── nomad-server.pem          # Nomad server certificate (SAN: localhost, 127.0.0.1)
├── nomad-server-key.pem      # Nomad server private key
├── nomad-client.pem          # Client certificate (for mTLS)
└── nomad-client-key.pem      # Client private key (for mTLS)
```

### Enable TLS in Nomad

Uncomment the `tls {}` block in `nomad-acl.hcl`, then restart:

```hcl
tls {
  http = true
  rpc  = true

  ca_file   = "infra-test/certs/nomad-ca.pem"
  cert_file = "infra-test/certs/nomad-server.pem"
  key_file  = "infra-test/certs/nomad-server-key.pem"

  verify_https_client = false   # true = enable mTLS
}
```

### Configure the client

```powershell
# Server-only TLS (no mTLS):
$env:NOMAD_ADDR   = "https://127.0.0.1:4646"
$env:NOMAD_CACERT = "infra-test\certs\nomad-ca.pem"
$env:NOMAD_TOKEN  = "<ci-cd-token>"

# mTLS (client cert required):
$env:NOMAD_ADDR        = "https://127.0.0.1:4646"
$env:NOMAD_CACERT      = "infra-test\certs\nomad-ca.pem"
$env:NOMAD_CLIENT_CERT = "infra-test\certs\nomad-client.pem"
$env:NOMAD_CLIENT_KEY  = "infra-test\certs\nomad-client-key.pem"
$env:NOMAD_TOKEN       = "<ci-cd-token>"

# Then test:
go test -v -run TestSubmitJob_WithAuthToken ./nomad/
```

### How it works

```
┌──────────────┐         HTTPS (TLS)          ┌──────────────────┐
│ nomad-ci-cd  │ ──── nomad-server.pem ──────► │  Nomad v2.x     │
│ (client)     │ ◄─── nomad-ca.pem (verify) ── │  (server)       │
│              │                               │                  │
│ NOMAD_CACERT │                               │ ca_file          │
│ NOMAD_TOKEN  │ ──── X-Nomad-Token ──────────► │ acl { enabled } │
└──────────────┘                               └──────────────────┘

With mTLS (verify_https_client = true):
┌──────────────┐         HTTPS (mTLS)           ┌──────────────────┐
│ nomad-ci-cd  │ ──── nomad-client.pem ────────► │  Nomad v2.x     │
│              │ ◄─── nomad-server.pem ───────── │                  │
│ CLIENT_CERT  │                               │ verify_https_    │
│ CLIENT_KEY   │                               │   client = true  │
└──────────────┘                               └──────────────────┘
```

## Cleanup

```powershell
# Stop Nomad (Ctrl+C), then:
rm -r -force .\nomad-data\
```

## Troubleshooting

| Symptom | Fix |
|---|---|
| `Permission denied` on job submit | Check `NOMAD_TOKEN` is set to a valid CI token |
| `No such file or directory` | Run commands from the repo root |
| `ACL not enabled` | Make sure you're using `infra-test/nomad-acl.hcl`, not `-dev` |
| Token expired | Create a new token — management tokens don't expire, client tokens can |
| `x509: certificate signed by unknown authority` | Set `NOMAD_CACERT` to the CA cert, or `NOMAD_SKIP_VERIFY=true` for dev |
| `tls: bad certificate` | Regenerate certs — SANs may be missing. Check `subjectAltName` includes your address |
| `x509: certificate relies on legacy Common Name` | Add `subjectAltName` entries matching the address you're connecting to |
| `connection refused` on HTTPS | Make sure TLS block is uncommented in `nomad-acl.hcl` and Nomad restarted |
| `OpenSSL not found` | Install OpenSSL: https://slproweb.com/products/Win32OpenSSL.html (Windows) or `apt install openssl` (Linux) |
