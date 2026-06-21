# Nomad v2.x standalone server — ACL + TLS with self-signed certs.
#
# All paths are relative to infra-test/ (the working directory when
# run.sh starts Nomad). Do NOT run nomad directly — use the scripts:
#
#   cd infra-test
#   bash run.sh          # terminal 1
#   bash setup-acl.sh    # terminal 2 (after run.sh is up)

data_dir = "../nomad-data"

bind_addr = "0.0.0.0"

advertise {
  http = "127.0.0.1"
  rpc  = "127.0.0.1"
  serf = "127.0.0.1"
}

ports {
  http = 4646
  rpc  = 4647
  serf = 4648
}

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

tls {
  http = true
  rpc  = true

  ca_file   = "certs/nomad-ca.pem"
  cert_file = "certs/nomad-server.pem"
  key_file  = "certs/nomad-server-key.pem"

  verify_https_client = false   # server-only TLS (no mTLS)
}
