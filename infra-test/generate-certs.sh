#!/usr/bin/env bash
set -euo pipefail

# Generate self-signed TLS certificates for Nomad v2.x local development.
# Usage:  bash infra-test/generate-certs.sh
# Output: infra-test/certs/

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
echo "[2/4] Generating server cert..."
cat > "$CERT_DIR/server.cnf" <<'EOF'
[req]
default_bits  = 4096
prompt        = no
default_md    = sha256
distinguished_name = dn
req_extensions     = v3_req

[dn]
CN = nomad-server

[v3_req]
keyUsage         = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName   = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = nomad.local
IP.1  = 127.0.0.1
IP.2  = 0.0.0.0
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

# ── Client cert (for mTLS) ──────────────────────────────────────────
echo "[3/4] Generating client cert (for mTLS)..."
cat > "$CERT_DIR/client.cnf" <<'EOF'
[req]
default_bits  = 4096
prompt        = no
default_md    = sha256
distinguished_name = dn
req_extensions     = v3_req

[dn]
CN = nomad-ci-cd-client

[v3_req]
keyUsage         = keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF

openssl req -new -newkey rsa:4096 -nodes \
  -config "$CERT_DIR/client.cnf" \
  -keyout "$CERT_DIR/nomad-client-key.pem" \
  -out    "$CERT_DIR/nomad-client.csr"

openssl x509 -req -days 3650 \
  -in      "$CERT_DIR/nomad-client.csr" \
  -CA      "$CERT_DIR/nomad-ca.pem" \
  -CAkey   "$CERT_DIR/nomad-ca-key.pem" \
  -CAcreateserial \
  -extfile "$CERT_DIR/client.cnf" \
  -extensions v3_req \
  -out     "$CERT_DIR/nomad-client.pem"

# ── Cleanup ─────────────────────────────────────────────────────────
echo "[4/4] Cleaning up..."
rm -f "$CERT_DIR"/*.csr "$CERT_DIR"/*.cnf "$CERT_DIR"/*.srl

# ── Summary ─────────────────────────────────────────────────────────
echo ""
echo "=== Certificates generated in $CERT_DIR/ ==="
ls -1 "$CERT_DIR"
echo ""
echo "Next: bash infra-test/run.sh"
