#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CERT_DIR="$SCRIPT_DIR/certs"

# ── Pre-flight checks ───────────────────────────────────────────────
if ! command -v nomad &>/dev/null; then
  echo "ERROR: nomad not found in PATH. Install Nomad v2.x first."
  exit 1
fi

if [ ! -f "$CERT_DIR/nomad-server.pem" ]; then
  echo "Certs not found — generating them now..."
  bash "$SCRIPT_DIR/generate-certs.sh"
fi

NOMAD_VERSION=$(nomad version 2>/dev/null | head -1)
echo "=== $NOMAD_VERSION ==="
echo ""

# ── Export env vars for nomad CLI ───────────────────────────────────
export NOMAD_ADDR="https://127.0.0.1:4646"
export NOMAD_CACERT="$CERT_DIR/nomad-ca.pem"
export NOMAD_CLIENT_CERT="$CERT_DIR/nomad-client.pem"
export NOMAD_CLIENT_KEY="$CERT_DIR/nomad-client-key.pem"

# ── Clean up stale data (optional: remove to start fresh) ──────────
# rm -rf "$REPO_ROOT/nomad-data"

# ── Start Nomad ─────────────────────────────────────────────────────
cd "$REPO_ROOT"
echo "Starting Nomad v2.x with ACL + TLS (0.0.0.0:4646)..."
echo ""
echo "  NOMAD_ADDR       = $NOMAD_ADDR"
echo "  NOMAD_CACERT     = $NOMAD_CACERT"
echo "  NOMAD_CLIENT_CERT = $NOMAD_CLIENT_CERT"
echo "  NOMAD_CLIENT_KEY  = $NOMAD_CLIENT_KEY"
echo ""
echo "=== Open a NEW terminal and bootstrap ACL: ==="
echo ""
echo "  export NOMAD_ADDR=https://127.0.0.1:4646"
echo "  export NOMAD_CACERT=$CERT_DIR/nomad-ca.pem"
echo "  export NOMAD_CLIENT_CERT=$CERT_DIR/nomad-client.pem"
echo "  export NOMAD_CLIENT_KEY=$CERT_DIR/nomad-client-key.pem"
echo ""
echo "  nomad acl bootstrap"
echo "  # Save the Secret ID, then:"
echo "  export NOMAD_TOKEN=<secret-id>"
echo "  nomad acl policy apply deployer $SCRIPT_DIR/deployer-policy.hcl"
echo "  nomad acl token create -name=ci-cd -policy=deployer -type=client"
echo ""
echo "  # Then test the pipeline:"
echo "  export NOMAD_TOKEN=<ci-cd-secret-id>"
echo "  cd $REPO_ROOT && go test -v -run TestSubmitJob_WithAuthToken ./nomad/"
echo ""

exec nomad agent -config="$SCRIPT_DIR/nomad-acl.hcl"
