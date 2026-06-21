# Nomad v2.x standalone server with ACL + TLS (self-signed certs).
#
# ---------------------------------------------------------------
# Quick start WITHOUT TLS (default):
#   nomad agent -config=infra-test/nomad-acl.hcl
#
# Quick start WITH TLS (generate certs first — see infra-test/README.md):
#   1. Run the cert generation script to create certs/
#   2. Uncomment the tls{} block below
#   3. nomad agent -config=infra-test/nomad-acl.hcl
# ---------------------------------------------------------------

data_dir = "./nomad-data"

bind_addr = "127.0.0.1"

# Advertise addresses — used when TLS is enabled so clients connect via HTTPS.
# Uncomment when enabling TLS:
# advertise {
#   http = "127.0.0.1"
#   rpc  = "127.0.0.1"
#   serf = "127.0.0.1"
# }

server {
  enabled          = true
  bootstrap_expect = 1
}

client {
  enabled = true
  options = {
    "driver.raw_exec.enable" = "true"
  }
}

acl {
  enabled = true
}

# ------------------------------------------------------------------
# TLS with self-signed certificates (uncomment after generating certs).
# verify_https_client = true  → requires mTLS (client must present cert)
# verify_https_client = false → server-only TLS (client just needs CA cert)
# ------------------------------------------------------------------

# tls {
#   http = true
#   rpc  = true
#
#   ca_file   = "infra-test/certs/nomad-ca.pem"
#   cert_file = "infra-test/certs/nomad-server.pem"
#   key_file  = "infra-test/certs/nomad-server-key.pem"
#
#   verify_https_client = false   # set to true for mTLS
# }
