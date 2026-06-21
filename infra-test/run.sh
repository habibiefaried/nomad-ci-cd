#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
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

# ── Start Nomad from THIS directory so HCL paths resolve correctly ──
cd "$SCRIPT_DIR"

echo "Starting Nomad v2.x with ACL + TLS (0.0.0.0:4646)..."
echo ""

echo "=== Open a NEW terminal, cd to infra-test/, and run: ==="
echo ""
echo "  bash setup-acl.sh"
echo ""

exec nomad agent -config="nomad-acl.hcl"
