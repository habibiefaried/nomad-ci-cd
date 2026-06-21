# Generate self-signed TLS certificates for Nomad v2.x local development.
# Requires OpenSSL (https://slproweb.com/products/Win32OpenSSL.html).
#
# Usage:  powershell -ExecutionPolicy Bypass -File infra-test\generate-certs.ps1
# Output: infra-test\certs\

$ErrorActionPreference = "Stop"
$CERT_DIR = "infra-test\certs"

Write-Host "=== Generating self-signed certs for Nomad v2.x ===" -ForegroundColor Cyan

# Create output directory
New-Item -ItemType Directory -Force -Path $CERT_DIR | Out-Null

# ── Step 1: CA certificate ──────────────────────────────────────────
Write-Host "[1/4] Generating CA cert..." -ForegroundColor Yellow
openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 `
  -subj "/C=US/ST=Dev/L=Local/O=Nomad-CI-CD/CN=nomad-ca" `
  -keyout "$CERT_DIR\nomad-ca-key.pem" `
  -out "$CERT_DIR\nomad-ca.pem"

# ── Step 2: Server certificate ──────────────────────────────────────
Write-Host "[2/4] Generating server cert..." -ForegroundColor Yellow

# SANs — Nomad validates these. Add your hostname/IP here.
$SERVER_CNF = @"
[req]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C=US
ST=Dev
L=Local
O=Nomad-CI-CD
CN=localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = nomad.local
IP.1  = 127.0.0.1
IP.2  = ::1
"@

$SERVER_CNF | Out-File -Encoding ascii "$CERT_DIR\server.cnf"

# Generate server key + CSR
openssl req -new -newkey rsa:4096 -nodes `
  -config "$CERT_DIR\server.cnf" `
  -keyout "$CERT_DIR\nomad-server-key.pem" `
  -out "$CERT_DIR\nomad-server.csr"

# Sign with CA
openssl x509 -req -days 3650 `
  -in "$CERT_DIR\nomad-server.csr" `
  -CA "$CERT_DIR\nomad-ca.pem" `
  -CAkey "$CERT_DIR\nomad-ca-key.pem" `
  -CAcreateserial `
  -extfile "$CERT_DIR\server.cnf" `
  -extensions v3_req `
  -out "$CERT_DIR\nomad-server.pem"

# ── Step 3: Client certificate (for mTLS) ───────────────────────────
Write-Host "[3/4] Generating client cert (for mTLS)..." -ForegroundColor Yellow

$CLIENT_CNF = @"
[req]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C=US
ST=Dev
L=Local
O=Nomad-CI-CD
CN=nomad-ci-cd-client

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
"@

$CLIENT_CNF | Out-File -Encoding ascii "$CERT_DIR\client.cnf"

# Generate client key + CSR
openssl req -new -newkey rsa:4096 -nodes `
  -config "$CERT_DIR\client.cnf" `
  -keyout "$CERT_DIR\nomad-client-key.pem" `
  -out "$CERT_DIR\nomad-client.csr"

# Sign with CA
openssl x509 -req -days 3650 `
  -in "$CERT_DIR\nomad-client.csr" `
  -CA "$CERT_DIR\nomad-ca.pem" `
  -CAkey "$CERT_DIR\nomad-ca-key.pem" `
  -CAcreateserial `
  -extfile "$CERT_DIR\client.cnf" `
  -extensions v3_req `
  -out "$CERT_DIR\nomad-client.pem"

# ── Step 4: Cleanup ─────────────────────────────────────────────────
Write-Host "[4/4] Cleaning up..." -ForegroundColor Yellow
Remove-Item "$CERT_DIR\*.csr", "$CERT_DIR\*.cnf", "$CERT_DIR\*.srl" -Force

# ── Summary ─────────────────────────────────────────────────────────
Write-Host ""
Write-Host "=== Certificates generated in $CERT_DIR\" -ForegroundColor Green
Write-Host ""
Get-ChildItem $CERT_DIR | ForEach-Object { Write-Host "  $($_.Name)" }
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Uncomment the tls{} block in infra-test\nomad-acl.hcl"
Write-Host "  2. Start Nomad: nomad agent -config=infra-test\nomad-acl.hcl"
Write-Host "  3. Set client env vars:"
Write-Host ""
Write-Host "  # Server-only TLS (no mTLS):"
Write-Host '  $env:NOMAD_ADDR = "https://127.0.0.1:4646"'
Write-Host '  $env:NOMAD_CACERT = "infra-test\certs\nomad-ca.pem"'
Write-Host ""
Write-Host "  # mTLS (client cert required):"
Write-Host '  $env:NOMAD_ADDR = "https://127.0.0.1:4646"'
Write-Host '  $env:NOMAD_CACERT = "infra-test\certs\nomad-ca.pem"'
Write-Host '  $env:NOMAD_CLIENT_CERT = "infra-test\certs\nomad-client.pem"'
Write-Host '  $env:NOMAD_CLIENT_KEY = "infra-test\certs\nomad-client-key.pem"'
