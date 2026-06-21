# Nomad v2.x standalone server — ACL + TLS with self-signed certs.
#
# Quick start (do this once):
#   bash infra-test/generate-certs.sh     # generate TLS certs
#   bash infra-test/run.sh                # start Nomad
#
# The run.sh script handles all env setup. After Nomad starts, open a
# new terminal and run:  nomad acl bootstrap

data_dir = "./nomad-data"

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

  ca_file   = "infra-test/certs/nomad-ca.pem"
  cert_file = "infra-test/certs/nomad-server.pem"
  key_file  = "infra-test/certs/nomad-server-key.pem"

  verify_https_client = false   # set to true for mTLS
}
