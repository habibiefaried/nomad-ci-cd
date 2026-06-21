# infra-test

Local Nomad v2.x cluster with ACL + TLS via self-signed certs — used for
integration testing `nomad-ci-cd`. Two scripts, one config, one policy.

## Files

| File | Purpose |
|---|---|
| `run.sh` | **Entry point** — generates certs if needed, starts Nomad, prints next steps |
| `generate-certs.sh` | Generates self-signed CA + server + client certs into `certs/` |
| `nomad-acl.hcl` | Nomad v2.x config — `0.0.0.0:4646`, ACL on, TLS on, `raw_exec` driver |
| `deployer-policy.hcl` | ACL policy for CI/CD — submit/read jobs in default ns, read nodes |

## Quick start

```bash
# Terminal 1 — start Nomad
bash infra-test/run.sh
```

`run.sh` will:
1. Check nomad is installed
2. Auto-generate certs if `certs/nomad-server.pem` doesn't exist
3. Print the exact commands for the next step
4. Start Nomad on `https://0.0.0.0:4646`

```bash
# Terminal 2 — bootstrap ACL and create a CI token
export NOMAD_ADDR=https://127.0.0.1:4646
export NOMAD_CACERT=infra-test/certs/nomad-ca.pem
export NOMAD_CLIENT_CERT=infra-test/certs/nomad-client.pem
export NOMAD_CLIENT_KEY=infra-test/certs/nomad-client-key.pem

nomad acl bootstrap
# → Save the Secret ID (management token)

export NOMAD_TOKEN=<management-token>
nomad acl policy apply deployer infra-test/deployer-policy.hcl
nomad acl token create -name=ci-cd -policy=deployer -type=client
# → Save this Secret ID (CI/CD token)
```

```bash
# Test with nomad-ci-cd
export NOMAD_TOKEN=<ci-cd-token>
go test -v -run TestSubmitJob_WithAuthToken ./nomad/
```

## What runs where

```
$ bash infra-test/run.sh
        │
        ├─► generate-certs.sh  (if certs/ missing)
        │   └─► infra-test/certs/
        │       ├── nomad-ca.pem
        │       ├── nomad-ca-key.pem
        │       ├── nomad-server.pem
        │       ├── nomad-server-key.pem
        │       ├── nomad-client.pem
        │       └── nomad-client-key.pem
        │
        └─► nomad agent -config=infra-test/nomad-acl.hcl
            ├── 0.0.0.0:4646  (TLS, self-signed server cert)
            ├── ACL enabled
            └── raw_exec driver (no Docker needed)
```

## TLS modes

| Config | What Nomad requires |
|---|---|
| `verify_https_client = false` (default) | Server-only TLS — client only needs `NOMAD_CACERT` |
| `verify_https_client = true` | mTLS — client must also present `NOMAD_CLIENT_CERT` + `NOMAD_CLIENT_KEY` |

Switch modes in `nomad-acl.hcl`.

## ACL policy

| Namespace | Access | Capabilities |
|---|---|---|
| `default` | `write` | `submit-job`, `read-job`, `list-jobs`, `alloc-exec` |
| `*` | `read` | `list-jobs` |
| Nodes | `read` | — |

## Cleanup

```bash
rm -rf ./nomad-data/
```

To regenerate certs:
```bash
rm -rf infra-test/certs/
bash infra-test/generate-certs.sh
```

## Troubleshooting

| Symptom | Fix |
|---|---|
| `Permission denied` | `NOMAD_TOKEN` is missing or invalid |
| `x509: certificate signed by unknown authority` | `NOMAD_CACERT` not set or wrong path |
| `connection refused` | Nomad not started — run `bash infra-test/run.sh` |
| `No such file or directory` | Run commands from the repo root |
