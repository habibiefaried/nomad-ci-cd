# infra-test

Local Nomad v2.x with ACL + self-signed TLS (server-only, no mTLS).
Two terminals, three scripts.

## Files

| File | Purpose |
|---|---|
| `run.sh` | **Terminal 1** — generates certs if missing, starts Nomad on `https://0.0.0.0:4646` |
| `setup-acl.sh` | **Terminal 2** — bootstraps ACL, creates CI token, saves to `ci-token.txt` |
| `generate-certs.sh` | Called by `run.sh` — generates CA + server cert into `certs/` |
| `nomad-acl.hcl` | Nomad v2.x config — ACL on, TLS on, `raw_exec` driver |
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

`ci-token.txt` now contains your CI token.

## Remote access

To access the Nomad UI from another machine, regenerate certs with your IP:

```bash
cd infra-test
EXTERNAL_IP=194.233.68.255 bash generate-certs.sh
bash run.sh   # restart
```

Then open `https://194.233.68.255:4646/ui/jobs` — accept the self-signed cert warning.

## Test with nomad-ci-cd

```bash
export NOMAD_ADDR=https://127.0.0.1:4646
export NOMAD_CACERT=infra-test/certs/nomad-ca.pem
export NOMAD_TOKEN=$(cat infra-test/ci-token.txt)

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
  │       ├── nomad-ca-key.pem
  │       ├── nomad-server.pem
  │       └── nomad-server-key.pem
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

## TLS

Server-only TLS — Nomad presents a self-signed cert, the client verifies it with the CA. No client certificate needed.

| Env var | Purpose |
|---|---|
| `NOMAD_ADDR` | `https://127.0.0.1:4646` |
| `NOMAD_CACERT` | Path to `certs/nomad-ca.pem` |
| `NOMAD_TOKEN` | ACL token for auth |

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
| `x509: unknown authority` | `NOMAD_CACERT` not set or wrong path |
| `connection refused` | Nomad not running — `bash run.sh` in terminal 1 |
| Browser says "untrusted" | Self-signed cert — click "Advanced" → "Proceed" |
| Remote IP not in cert | Regenerate with `EXTERNAL_IP=<your-ip> bash generate-certs.sh` |
| Cert expired | `rm -rf certs/ && bash generate-certs.sh` then restart |
