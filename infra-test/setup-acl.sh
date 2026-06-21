#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/certs"

# ── TLS env vars (needed to talk to Nomad over HTTPS) ───────────────
export NOMAD_ADDR="https://127.0.0.1:4646"
export NOMAD_CACERT="$CERT_DIR/nomad-ca.pem"
export NOMAD_CLIENT_CERT="$CERT_DIR/nomad-client.pem"
export NOMAD_CLIENT_KEY="$CERT_DIR/nomad-client-key.pem"

echo "=== Bootstrapping Nomad ACL ==="

# ── 1. Bootstrap ────────────────────────────────────────────────────
echo ""
echo "[1/4] Bootstrapping ACL..."
BOOTSTRAP_OUT=$(nomad acl bootstrap 2>&1)
echo "$BOOTSTRAP_OUT"

MGMT_TOKEN=$(echo "$BOOTSTRAP_OUT" | grep -i "Secret ID" | awk '{print $NF}')
if [ -z "$MGMT_TOKEN" ]; then
  echo "ERROR: failed to extract management token from bootstrap output."
  echo "Raw output above — copy the Secret ID manually."
  exit 1
fi

echo "$MGMT_TOKEN" > "$SCRIPT_DIR/mgmt-token.txt"
echo "   -> saved to mgmt-token.txt"

export NOMAD_TOKEN="$MGMT_TOKEN"

# ── 2. Create deployer policy ───────────────────────────────────────
echo ""
echo "[2/4] Creating deployer policy..."
nomad acl policy apply deployer "$SCRIPT_DIR/deployer-policy.hcl"
echo "   -> deployer policy created"

# ── 3. Create CI token ──────────────────────────────────────────────
echo ""
echo "[3/4] Creating CI/CD token..."
TOKEN_OUT=$(nomad acl token create -name="ci-cd" -policy="deployer" -type="client" 2>&1)
echo "$TOKEN_OUT"

CI_TOKEN=$(echo "$TOKEN_OUT" | grep -i "Secret ID" | awk '{print $NF}')
if [ -z "$CI_TOKEN" ]; then
  echo "ERROR: failed to extract CI token."
  echo "Raw output above — copy the Secret ID manually."
  exit 1
fi

echo "$CI_TOKEN" > "$SCRIPT_DIR/ci-token.txt"
echo "   -> saved to ci-token.txt"

# ── 4. Verify ───────────────────────────────────────────────────────
echo ""
echo "[4/4] Verifying..."
export NOMAD_TOKEN="$CI_TOKEN"
if nomad job list &>/dev/null; then
  echo "   -> CI token works!"
else
  echo "   -> WARNING: CI token check failed (maybe no jobs yet — that's ok)"
fi

# ── Done ────────────────────────────────────────────────────────────
echo ""
echo "=== Ready ==="
echo ""
echo "  Management token : $(cat "$SCRIPT_DIR/mgmt-token.txt")"
echo "  CI/CD token      : $(cat "$SCRIPT_DIR/ci-token.txt")"
echo ""
echo "To use the CI token with nomad-ci-cd:"
echo ""
echo "  export NOMAD_ADDR=https://127.0.0.1:4646"
echo "  export NOMAD_CACERT=$CERT_DIR/nomad-ca.pem"
echo "  export NOMAD_CLIENT_CERT=$CERT_DIR/nomad-client.pem"
echo "  export NOMAD_CLIENT_KEY=$CERT_DIR/nomad-client-key.pem"
echo "  export NOMAD_TOKEN=\$(cat $SCRIPT_DIR/ci-token.txt)"
echo ""
echo "  cd .. && go test -v -run TestSubmitJob_WithAuthToken ./nomad/"
