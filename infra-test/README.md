# infra-test

Configuration files and instructions for running a local Nomad v2.x cluster
with ACL authentication enabled — used for integration testing `nomad-ci-cd`.

## Files

| File | Purpose |
|---|---|
| `nomad-acl.hcl` | Nomad v2.x agent config — single server + client, ACL enabled, raw_exec driver |
| `deployer-policy.hcl` | ACL policy for CI/CD pipelines — job submit/read, node read |

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
