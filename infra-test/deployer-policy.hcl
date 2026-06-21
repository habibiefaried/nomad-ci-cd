# Nomad ACL policy for CI/CD deployer.
# Grants permission to register/read/stop jobs in the default namespace.
# Apply with: nomad acl policy apply deployer deployer-policy.hcl

namespace "default" {
  policy       = "write"
  capabilities = ["submit-job", "read-job", "list-jobs", "alloc-exec"]
}

namespace "*" {
  policy       = "read"
  capabilities = ["list-jobs"]
}

node {
  policy = "read"
}
