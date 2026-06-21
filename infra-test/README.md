# infra-test

Local Nomad v2.x with ACL + self-signed TLS. Two terminals, three scripts.

## Files

| File | Purpose |
|---|---|
| `run.sh` | **Terminal 1** — generates certs if missing, starts Nomad on `https://0.0.0.0:4646` |
| `setup-acl.sh` | **Terminal 2** — bootstraps ACL, creates CI token, saves to `ci-token.txt` |
| `generate-certs.sh` | Called by `run.sh` — generates self-signed certs into `certs/` |
| `nomad-acl.hcl` | Nomad v2.x config — ACL + TLS on, `raw_exec` driver |
| `deployer-policy.hcl` | ACL policy — submit/read jobs in default ns, read nodes |
| `mgmt-token.txt` | Created by `setup-acl.sh` — management token |
| `ci-token.txt` | Created by `setup-acl.sh` — CI/CD token |

## Quick start

```bash
cd infra-test

# Terminal 1
bash run.sh

# Terminal 2 (once Nomad is up)
bash setup-acl.sh
```

That's it. `ci-token.txt` now contains your CI token.

## Test with nomad-ci-cd

```bash
cd infra-test

export NOMAD_ADDR=https://127.0.0.1:4646
export NOMAD_CACERT=certs/nomad-ca.pem
export NOMAD_CLIENT_CERT=certs/nomad-client.pem
export NOMAD_CLIENT_KEY=certs/nomad-client-key.pem
export NOMAD_TOKEN=$(cat ci-token.txt)

cd ..
go test -v -run TestSubmitJob_WithAuthToken ./nomad/
```

## What happens

```
Terminal 1                          Terminal 2
─────────                           ─────────
bash run.sh
  ├─ generate-certs.sh (if needed)
  │   └─► certs/
  │       ├── nomad-ca.pem
  │       ├── nomad-server.pem
  │       ├── nomad-server-key.pem
  │       ├── nomad-client.pem
  │       └── nomad-client-key.pem
  │
  └─ nomad agent -config=nomad-acl.hcl
      ├── 0.0.0.0:4646 (TLS)
      ├── ACL enabled               bash setup-acl.sh
      └── ready... ───────────────────► nomad acl bootstrap
                                        ├─► mgmt-token.txt
                                        ├─ nomad acl policy apply deployer
                                        └─ nomad acl token create ci-cd
                                           └─► ci-token.txt
```

## TLS modes

In `nomad-acl.hcl`:

| `verify_https_client` | Mode | Client needs |
|---|---|---|
| `false` (default) | Server-only TLS | `NOMAD_CACERT` |
| `true` | mTLS | `NOMAD_CACERT` + `NOMAD_CLIENT_CERT` + `NOMAD_CLIENT_KEY` |

## ACL policy

| Namespace | Access | Capabilities |
|---|---|---|
| `default` | `write` | `submit-job`, `read-job`, `list-jobs`, `alloc-exec` |
| `*` | `read` | `list-jobs` |
| Nodes | `read` | — |

## Cleanup

```bash
# Stop Nomad (Ctrl+C in terminal 1), then:
rm -rf certs/ mgmt-token.txt ci-token.txt
cd .. && rm -rf nomad-data/
```

## Troubleshooting

| Symptom | Fix |
|---|---|
| `Permission denied` | `NOMAD_TOKEN` missing — run `bash setup-acl.sh` or `cat ci-token.txt` |
| `x509: unknown authority` | `NOMAD_CACERT` not set — `export NOMAD_CACERT=certs/nomad-ca.pem` |
| `connection refused` | Nomad not running — `bash run.sh` in terminal 1 |
| `ACL not enabled` | You're not using `nomad-acl.hcl` — don't use `-dev` flag |
| Cert expired | `rm -rf certs/ && bash generate-certs.sh` then restart |
