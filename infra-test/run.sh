#!/usr/bin/env bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/certs"

# ── Pre-flight checks ───────────────────────────────────────────────
if ! command -v nomad &>/dev/null; then
  echo "ERROR: nomad not found in PATH. Install Nomad v2.x first."
  exit 1
fi

if [ ! -f "$CERT_DIR/nomad-server.pem" ]; then
  echo "=== Certs not found — generating... ==="
  bash "$SCRIPT_DIR/generate-certs.sh"
  echo ""
fi

NOMAD_VERSION=$(nomad version 2>/dev/null | head -1)

# ── Start Nomad from THIS directory ─────────────────────────────────
cd "$SCRIPT_DIR"

echo "=============================================="
echo "  $NOMAD_VERSION"
echo "=============================================="
echo ""
echo "  Bind   : 0.0.0.0:4646"
echo "  TLS    : ON  (self-signed)"
echo "  ACL    : ON"
echo "  Driver : raw_exec"
echo ""
echo "  UI     : https://127.0.0.1:4646/ui/jobs"
echo "  API    : https://127.0.0.1:4646/v1/..."
echo ""
echo "  ⚠  Use HTTPS, not HTTP — TLS is enabled."
echo "  ⚠  Browser will warn about self-signed cert — accept it."
echo "  ⚠  For remote access, add your IP to SANs in generate-certs.sh"
echo ""
echo "=============================================="
echo "  Open a NEW terminal and run:"
echo "=============================================="
echo ""
echo "  cd infra-test"
echo "  bash setup-acl.sh"
echo ""

exec nomad agent -config="nomad-acl.hcl"
