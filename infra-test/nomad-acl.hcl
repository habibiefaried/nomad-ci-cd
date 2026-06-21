# Nomad v2.x standalone server with ACL enabled.
# Start with: nomad agent -config=nomad-acl.hcl

data_dir = "./nomad-data"

bind_addr = "127.0.0.1"

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

# Optional: enable TLS for production (requires certs)
# tls {
#   http = true
#   rpc  = true
#   ca_file   = "/path/to/ca.pem"
#   cert_file = "/path/to/nomad.pem"
#   key_file  = "/path/to/nomad-key.pem"
# }
