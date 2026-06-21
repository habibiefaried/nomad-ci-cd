#!/usr/bin/env bash
set -eu

# Generate self-signed TLS certificates for Nomad v2.x local development.
#
# Usage:
#   bash generate-certs.sh                          # localhost only
#   EXTERNAL_IP=194.233.68.255 bash generate-certs.sh  # add remote IP
#
# The EXTERNAL_IP env var adds an extra SAN entry so you can access
# the Nomad UI/API from another machine without TLS errors.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/certs"

echo "=== Generating self-signed certs for Nomad v2.x ==="

mkdir -p "$CERT_DIR"

# ── CA ──────────────────────────────────────────────────────────────
echo "[1/4] Generating CA cert..."
openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 \
  -subj "/CN=nomad-ca" \
  -keyout "$CERT_DIR/nomad-ca-key.pem" \
  -out    "$CERT_DIR/nomad-ca.pem"

# ── Server cert ─────────────────────────────────────────────────────
echo "[2/3] Generating server cert..."

# Build SAN list. Always include localhost/127.0.0.1.
# Add EXTERNAL_IP if set.
SAN_ENTRIES="DNS.1 = localhost
DNS.2 = nomad.local
IP.1  = 127.0.0.1"

if [ -n "${EXTERNAL_IP:-}" ]; then
  SAN_ENTRIES="$SAN_ENTRIES
IP.3  = ${EXTERNAL_IP}"
  echo "   -> Adding EXTERNAL_IP=$EXTERNAL_IP to certificate SANs"
fi

cat > "$CERT_DIR/server.cnf" <<EOF
[req]
default_bits  = 4096
prompt        = no
default_md    = sha256
distinguished_name = dn
req_extensions     = v3_req

[dn]
CN = nomad-server

[v3_req]
keyUsage         = digitalSignature, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName   = @alt_names

[alt_names]
$SAN_ENTRIES
EOF

openssl req -new -newkey rsa:4096 -nodes \
  -config "$CERT_DIR/server.cnf" \
  -keyout "$CERT_DIR/nomad-server-key.pem" \
  -out    "$CERT_DIR/nomad-server.csr"

openssl x509 -req -days 3650 \
  -in      "$CERT_DIR/nomad-server.csr" \
  -CA      "$CERT_DIR/nomad-ca.pem" \
  -CAkey   "$CERT_DIR/nomad-ca-key.pem" \
  -CAcreateserial \
  -extfile "$CERT_DIR/server.cnf" \
  -extensions v3_req \
  -out     "$CERT_DIR/nomad-server.pem"

# ── Cleanup ─────────────────────────────────────────────────────────
echo "[3/3] Cleaning up..."
rm -f "$CERT_DIR"/*.csr "$CERT_DIR"/*.cnf "$CERT_DIR"/*.srl

# ── Summary ─────────────────────────────────────────────────────────
echo ""
echo "=== Certs generated in $CERT_DIR/ ==="
ls -1 "$CERT_DIR"
echo ""
echo "Next: bash run.sh"
