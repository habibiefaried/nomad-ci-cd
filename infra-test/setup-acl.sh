#!/usr/bin/env bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/certs"

# ── TLS env vars (server-only TLS, no mTLS) ─────────────────────────
export NOMAD_ADDR="https://127.0.0.1:4646"
export NOMAD_CACERT="$CERT_DIR/nomad-ca.pem"

echo "=== Bootstrapping Nomad ACL ==="

# ── 0. Wait for Nomad to be ready ───────────────────────────────────
echo ""
echo "[0/4] Waiting for Nomad to be ready..."
for i in $(seq 1 30); do
  if curl -sk --cacert "$NOMAD_CACERT" \
          "$NOMAD_ADDR/v1/status/leader" 2>/dev/null | grep -q "."; then
    echo "   -> Nomad is ready"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: Nomad did not become ready after 30 seconds."
    echo "Is run.sh still running in terminal 1?"
    exit 1
  fi
  sleep 1
done

# ── 1. Bootstrap ────────────────────────────────────────────────────
echo ""
echo "[1/4] Bootstrapping ACL..."
BOOTSTRAP_OUT=$(nomad acl bootstrap 2>&1) || true
echo "$BOOTSTRAP_OUT"

MGMT_TOKEN=$(echo "$BOOTSTRAP_OUT" | grep -i "Secret ID" | awk '{print $NF}' || true)
if [ -z "$MGMT_TOKEN" ]; then
  echo ""
  echo "ERROR: Could not extract management token."
  echo "This usually means ACL was already bootstrapped. Try:"
  echo "  cat $SCRIPT_DIR/mgmt-token.txt"
  echo "If that file exists with a token, use it."
  echo "If not, check the raw output above for the Secret ID."
  exit 1
fi

echo "$MGMT_TOKEN" > "$SCRIPT_DIR/mgmt-token.txt"
echo "   -> saved to mgmt-token.txt"

export NOMAD_TOKEN="$MGMT_TOKEN"

# ── 2. Create deployer policy ───────────────────────────────────────
echo ""
echo "[2/4] Creating deployer policy..."
nomad acl policy apply deployer "$SCRIPT_DIR/deployer-policy.hcl" || {
  echo "   -> policy may already exist, continuing..."
}
echo "   -> deployer policy ready"

# ── 3. Create CI token ──────────────────────────────────────────────
echo ""
echo "[3/4] Creating CI/CD token..."
TOKEN_OUT=$(nomad acl token create -name="ci-cd" -policy="deployer" -type="client" 2>&1) || true
echo "$TOKEN_OUT"

CI_TOKEN=$(echo "$TOKEN_OUT" | grep -i "Secret ID" | awk '{print $NF}' || true)
if [ -z "$CI_TOKEN" ]; then
  echo ""
  echo "ERROR: Could not create CI token."
  echo "Raw output above — you may need to set NOMAD_TOKEN to the management token first."
  echo "Try: export NOMAD_TOKEN=\$(cat $SCRIPT_DIR/mgmt-token.txt)"
  exit 1
fi

echo "$CI_TOKEN" > "$SCRIPT_DIR/ci-token.txt"
echo "   -> saved to ci-token.txt"

# ── 4. Verify ───────────────────────────────────────────────────────
echo ""
echo "[4/4] Verifying CI token..."
export NOMAD_TOKEN="$CI_TOKEN"
if nomad job list 2>&1; then
  echo "   -> CI token works!"
else
  echo "   -> Token created but job list failed (may be ok if no jobs exist)"
fi

# ── Done ────────────────────────────────────────────────────────────
echo ""
echo "=============================================="
echo "  Tokens saved in infra-test/"
echo "=============================================="
echo ""
echo "  mgmt-token.txt : $MGMT_TOKEN"
echo "  ci-token.txt   : $CI_TOKEN"
echo ""
echo "=============================================="
echo "  For nomad-ci-cd, export:"
echo "=============================================="
echo ""
echo "  export NOMAD_ADDR=https://127.0.0.1:4646"
echo "  export NOMAD_CACERT=$CERT_DIR/nomad-ca.pem"
echo "  export NOMAD_TOKEN=\$(cat $SCRIPT_DIR/ci-token.txt)"
echo ""
