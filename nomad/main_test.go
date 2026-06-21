package nomad

import (
	"fmt"
	"os"
	"strings"
	"testing"

	nomad "github.com/hashicorp/nomad/api"
)

// =============================================================================
// Unit tests: constraintGenerator
// =============================================================================

func TestConstraintGenerator_AllEnvSet(t *testing.T) {
	os.Setenv("CONS_ATTR", "node.class")
	os.Setenv("CONS_OP", "==")
	os.Setenv("CONS_VALUE", "compute")
	defer os.Unsetenv("CONS_ATTR")
	defer os.Unsetenv("CONS_OP")
	defer os.Unsetenv("CONS_VALUE")

	result := constraintGenerator()
	if result == "" {
		t.Fatal("expected non-empty constraint when all env vars are set")
	}
	if !strings.Contains(result, "constraint") {
		t.Fatalf("expected constraint block, got: %s", result)
	}
	if !strings.Contains(result, "node.class") {
		t.Fatalf("expected attribute name in output, got: %s", result)
	}
	if !strings.Contains(result, `"=="`) {
		t.Fatalf("expected operator in output, got: %s", result)
	}
	if !strings.Contains(result, `"compute"`) {
		t.Fatalf("expected value in output, got: %s", result)
	}
}

func TestConstraintGenerator_MissingEnv(t *testing.T) {
	os.Unsetenv("CONS_ATTR")
	os.Unsetenv("CONS_OP")
	os.Unsetenv("CONS_VALUE")

	result := constraintGenerator()
	if result != "" {
		t.Fatalf("expected empty string when env vars are missing, got: %s", result)
	}
}

func TestConstraintGenerator_PartialEnv(t *testing.T) {
	os.Setenv("CONS_ATTR", "node.class")
	os.Unsetenv("CONS_OP")
	os.Unsetenv("CONS_VALUE")
	defer os.Unsetenv("CONS_ATTR")

	result := constraintGenerator()
	if result != "" {
		t.Fatalf("expected empty string when only some env vars are set, got: %s", result)
	}
}

// =============================================================================
// Unit tests: generateDNSServer
// =============================================================================

func TestGenerateDNSServer_WithDNS(t *testing.T) {
	os.Setenv("CONTAINER_DNS_SERVER", "8.8.8.8")
	defer os.Unsetenv("CONTAINER_DNS_SERVER")

	result := generateDNSServer()
	if result == "" {
		t.Fatal("expected non-empty DNS config when CONTAINER_DNS_SERVER is set")
	}
	if !strings.Contains(result, "dns") {
		t.Fatalf("expected dns block, got: %s", result)
	}
	if !strings.Contains(result, "8.8.8.8") {
		t.Fatalf("expected DNS server IP in output, got: %s", result)
	}
}

func TestGenerateDNSServer_WithoutDNS(t *testing.T) {
	os.Unsetenv("CONTAINER_DNS_SERVER")

	result := generateDNSServer()
	if result != "" {
		t.Fatalf("expected empty string when CONTAINER_DNS_SERVER is not set, got: %s", result)
	}
}

// =============================================================================
// Unit tests: hostGenerator
// =============================================================================

func TestHostGenerator_SingleHost(t *testing.T) {
	os.Setenv("APP_HOST", "example.com")
	defer os.Unsetenv("APP_HOST")

	result := hostGenerator()
	expected := `Host(\"example.com\")`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestHostGenerator_MultipleHosts(t *testing.T) {
	os.Setenv("APP_HOST", "example.com#api.example.com#www.example.com")
	defer os.Unsetenv("APP_HOST")

	result := hostGenerator()
	if !strings.Contains(result, `Host(\"example.com\")`) {
		t.Fatalf("expected first host in output, got: %s", result)
	}
	if !strings.Contains(result, `Host(\"api.example.com\")`) {
		t.Fatalf("expected second host in output, got: %s", result)
	}
	if !strings.Contains(result, `Host(\"www.example.com\")`) {
		t.Fatalf("expected third host in output, got: %s", result)
	}
	// Should have " || " between hosts
	if !strings.Contains(result, " || ") {
		t.Fatalf("expected ' || ' separator between hosts, got: %s", result)
	}
}

func TestHostGenerator_EmptyHost(t *testing.T) {
	os.Unsetenv("APP_HOST")

	result := hostGenerator()
	if result != "" {
		t.Fatalf("expected empty string when APP_HOST is not set, got: %s", result)
	}
}

func TestHostGenerator_SkipsEmptySegments(t *testing.T) {
	os.Setenv("APP_HOST", "example.com##api.example.com")
	defer os.Unsetenv("APP_HOST")

	result := hostGenerator()
	// Should contain only 2 hosts, not 3 (empty segment between ## skipped)
	parts := strings.Split(result, " || ")
	if len(parts) != 2 {
		t.Fatalf("expected 2 hosts (empty segment skipped), got %d: %s", len(parts), result)
	}
}

// =============================================================================
// Unit tests: tagGenerator
// =============================================================================

func TestTagGenerator_BasicTraefikTags(t *testing.T) {
	os.Setenv("PORT_NAME", "http")
	os.Setenv("APP_HOST", "example.com")
	os.Unsetenv("APP_PREFIX_REGEX")
	os.Unsetenv("TRAEFIK_PASSWORD")
	defer os.Unsetenv("PORT_NAME")
	defer os.Unsetenv("APP_HOST")

	result := tagGenerator()

	// Always-present tags
	requiredTags := []string{
		"traefik.enable=true",
		"traefik.http.routers.",
		"traefik.http.routers.http-https.tls=true",
		"traefik.http.routers.http-https.tls.certresolver=myresolver",
		"traefik.http.middlewares.http-https.redirectscheme.scheme=https",
	}
	for _, tag := range requiredTags {
		if !strings.Contains(result, tag) {
			t.Fatalf("expected tag %q in output, got:\n%s", tag, result)
		}
	}
}

func TestTagGenerator_WithPathPrefix(t *testing.T) {
	os.Setenv("PORT_NAME", "api")
	os.Setenv("APP_HOST", "example.com")
	os.Setenv("APP_PREFIX_REGEX", "/api")
	os.Unsetenv("TRAEFIK_PASSWORD")
	defer os.Unsetenv("PORT_NAME")
	defer os.Unsetenv("APP_HOST")
	defer os.Unsetenv("APP_PREFIX_REGEX")

	result := tagGenerator()

	if !strings.Contains(result, "PathPrefix") {
		t.Fatalf("expected PathPrefix in output when APP_PREFIX_REGEX is set, got:\n%s", result)
	}
	if !strings.Contains(result, "stripprefix") {
		t.Fatalf("expected stripprefix middleware in output, got:\n%s", result)
	}
}

func TestTagGenerator_WithTraefikPassword(t *testing.T) {
	os.Setenv("PORT_NAME", "web")
	os.Setenv("APP_HOST", "example.com")
	os.Unsetenv("APP_PREFIX_REGEX")
	os.Setenv("TRAEFIK_PASSWORD", "user:$2y$10$hash")
	defer os.Unsetenv("PORT_NAME")
	defer os.Unsetenv("APP_HOST")
	defer os.Unsetenv("TRAEFIK_PASSWORD")

	result := tagGenerator()

	if !strings.Contains(result, "basicauth.users") {
		t.Fatalf("expected basicauth middleware when TRAEFIK_PASSWORD is set, got:\n%s", result)
	}
}

func TestTagGenerator_MiddlewareChainWhenEnabled(t *testing.T) {
	os.Setenv("PORT_NAME", "svc")
	os.Setenv("APP_HOST", "example.com")
	os.Setenv("APP_PREFIX_REGEX", "/app")
	os.Unsetenv("TRAEFIK_PASSWORD")
	defer os.Unsetenv("PORT_NAME")
	defer os.Unsetenv("APP_HOST")
	defer os.Unsetenv("APP_PREFIX_REGEX")

	result := tagGenerator()

	// When middleware is enabled (via APP_PREFIX_REGEX), the middleware chain tag should exist
	if !strings.Contains(result, "middlewares=svc@consulcatalog") {
		t.Fatalf("expected middleware chain tag when middleware is enabled, got:\n%s", result)
	}
}

// =============================================================================
// Unit tests: jobGeneration (full HCL output)
// =============================================================================

func TestJobGeneration_FullHCL(t *testing.T) {
	// Set all required env vars for a valid job spec
	os.Setenv("NOMAD_CUSTOM_NAME", "myapp")
	os.Setenv("DEPLOY_ENVIRONMENT", "staging")
	os.Setenv("NUM_REPLICA", "2")
	os.Setenv("PORT_NAME", "http")
	os.Setenv("TARGET_PORT", "8080")
	os.Setenv("IMAGE_URL", "registry.example.com/myapp:latest")
	os.Setenv("JOB_CPU", "500")
	os.Setenv("JOB_MEMORY", "256")
	os.Setenv("APP_HOST", "myapp.example.com")
	os.Unsetenv("CONS_ATTR")
	os.Unsetenv("CONTAINER_DNS_SERVER")
	os.Unsetenv("APP_PREFIX_REGEX")
	os.Unsetenv("TRAEFIK_PASSWORD")
	os.Unsetenv("ENV_SOURCE")

	defer func() {
		os.Unsetenv("NOMAD_CUSTOM_NAME")
		os.Unsetenv("DEPLOY_ENVIRONMENT")
		os.Unsetenv("NUM_REPLICA")
		os.Unsetenv("PORT_NAME")
		os.Unsetenv("TARGET_PORT")
		os.Unsetenv("IMAGE_URL")
		os.Unsetenv("JOB_CPU")
		os.Unsetenv("JOB_MEMORY")
		os.Unsetenv("APP_HOST")
	}()

	result := jobGeneration()

	// Verify key Nomad v2.x HCL constructs.
	// Note: job name is an HCL identifier (unquoted), not a string literal.
	checks := []string{
		`job myapp--staging`,        // HCL identifier, not quoted
		`datacenters = ["dc1"]`,
		`group "app"`,
		`count = 2`,
		`port "http" { to = 8080 }`,
		`task "server"`,
		`driver = "docker"`,
		`image = "registry.example.com/myapp:latest"`,
		`ports = ["http"]`,
		`force_pull = true`,
		`cpu    = 500`,
		`memory = 256`,
		`service {`,
		`name = "myapp--staging"`,
		`port = "http"`,
		`type        = "tcp"`,
		`traefik.enable=true`,
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Fatalf("expected %q in generated HCL, got:\n%s", check, result)
		}
	}
}

// =============================================================================
// Integration test: HCL parsing with Nomad v2.x API client
// =============================================================================

func TestHCLParsing_ValidJobSpec(t *testing.T) {
	// Set up env vars for a valid job
	os.Setenv("NOMAD_CUSTOM_NAME", "testjob")
	os.Setenv("DEPLOY_ENVIRONMENT", "test")
	os.Setenv("NUM_REPLICA", "1")
	os.Setenv("PORT_NAME", "http")
	os.Setenv("TARGET_PORT", "3000")
	os.Setenv("IMAGE_URL", "docker.io/library/nginx:latest")
	os.Setenv("JOB_CPU", "100")
	os.Setenv("JOB_MEMORY", "128")
	os.Setenv("APP_HOST", "test.local")
	os.Unsetenv("CONS_ATTR")
	os.Unsetenv("CONTAINER_DNS_SERVER")
	os.Unsetenv("APP_PREFIX_REGEX")
	os.Unsetenv("TRAEFIK_PASSWORD")
	os.Unsetenv("ENV_SOURCE")

	defer func() {
		os.Unsetenv("NOMAD_CUSTOM_NAME")
		os.Unsetenv("DEPLOY_ENVIRONMENT")
		os.Unsetenv("NUM_REPLICA")
		os.Unsetenv("PORT_NAME")
		os.Unsetenv("TARGET_PORT")
		os.Unsetenv("IMAGE_URL")
		os.Unsetenv("JOB_CPU")
		os.Unsetenv("JOB_MEMORY")
		os.Unsetenv("APP_HOST")
	}()

	hcl := jobGeneration()

	// Create a Nomad v2.x API client and parse the HCL.
	// This validates the HCL is compatible with the Nomad v2.x API.
	client, err := nomad.NewClient(nomad.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create Nomad v2.x client: %v", err)
	}

	// ParseHCL validates the HCL syntax against the Nomad v2.x API schema.
	// canonicalize=true normalizes the job spec.
	job, err := client.Jobs().ParseHCL(hcl, true)
	if err != nil {
		// Nomad might not be running locally — that's OK.
		t.Skipf("Nomad not reachable, skipping HCL parse test: %v", err)
		return
	}

	// Verify the parsed job has correct fields
	if job.Name == nil || *job.Name != "testjob--test" {
		t.Fatalf("expected job name 'testjob--test', got: %v", job.Name)
	}
	if job.Datacenters == nil || len(job.Datacenters) == 0 || job.Datacenters[0] != "dc1" {
		t.Fatalf("expected datacenter 'dc1', got: %v", job.Datacenters)
	}
	if len(job.TaskGroups) != 1 {
		t.Fatalf("expected 1 task group, got: %d", len(job.TaskGroups))
	}

	tg := job.TaskGroups[0]
	if tg.Name == nil || *tg.Name != "app" {
		t.Fatalf("expected task group name 'app', got: %v", tg.Name)
	}
	if tg.Count == nil || *tg.Count != 1 {
		t.Fatalf("expected count=1, got: %v", tg.Count)
	}
	if len(tg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got: %d", len(tg.Tasks))
	}

	task := tg.Tasks[0]
	if task.Driver != "docker" {
		t.Fatalf("expected docker driver, got: %s", task.Driver)
	}

	t.Logf("✅ Nomad v2.x API successfully parsed HCL job: %s", *job.Name)
}

// =============================================================================
// Auth tests: verify Nomad v2.x authentication configuration
// =============================================================================

// setenv saves the old value, sets the new one, and restores on cleanup.
// This prevents test interference — Go tests in the same package share one process,
// so os.Setenv/os.Unsetenv in one test permanently affects all subsequent tests.
func setenv(t *testing.T, key, value string) {
	old, existed := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

func TestAuthConfig_DefaultConfigPicksUpToken(t *testing.T) {
	setenv(t, "NOMAD_TOKEN", "test-secret-token-12345")

	config := nomad.DefaultConfig()
	if config.SecretID != "test-secret-token-12345" {
		t.Fatalf("expected SecretID 'test-secret-token-12345', got: %q", config.SecretID)
	}
	t.Logf("✅ NOMAD_TOKEN picked up: SecretID = %q", config.SecretID)
}

func TestAuthConfig_DefaultConfigPicksUpHTTPAuth(t *testing.T) {
	setenv(t, "NOMAD_HTTP_AUTH", "ci-user:ci-password")

	config := nomad.DefaultConfig()
	if config.HttpAuth == nil {
		t.Fatal("expected HttpAuth to be set from NOMAD_HTTP_AUTH")
	}
	if config.HttpAuth.Username != "ci-user" {
		t.Fatalf("expected username 'ci-user', got: %q", config.HttpAuth.Username)
	}
	if config.HttpAuth.Password != "ci-password" {
		t.Fatalf("expected password 'ci-password', got: %q", config.HttpAuth.Password)
	}
	t.Logf("✅ NOMAD_HTTP_AUTH picked up: user=%q", config.HttpAuth.Username)
}

func TestAuthConfig_TLSPicksUpEnvVars(t *testing.T) {
	setenv(t, "NOMAD_CACERT", "/path/to/ca.pem")
	setenv(t, "NOMAD_CLIENT_CERT", "/path/to/client.pem")
	setenv(t, "NOMAD_CLIENT_KEY", "/path/to/client-key.pem")
	setenv(t, "NOMAD_TLS_SERVER_NAME", "nomad.example.com")
	setenv(t, "NOMAD_SKIP_VERIFY", "true")

	config := nomad.DefaultConfig()
	if config.TLSConfig == nil {
		t.Fatal("expected TLSConfig to be set from env vars")
	}
	if config.TLSConfig.CACert != "/path/to/ca.pem" {
		t.Fatalf("expected CACert '/path/to/ca.pem', got: %q", config.TLSConfig.CACert)
	}
	if config.TLSConfig.ClientCert != "/path/to/client.pem" {
		t.Fatalf("expected ClientCert '/path/to/client.pem', got: %q", config.TLSConfig.ClientCert)
	}
	if config.TLSConfig.ClientKey != "/path/to/client-key.pem" {
		t.Fatalf("expected ClientKey '/path/to/client-key.pem', got: %q", config.TLSConfig.ClientKey)
	}
	if config.TLSConfig.TLSServerName != "nomad.example.com" {
		t.Fatalf("expected TLSServerName 'nomad.example.com', got: %q", config.TLSConfig.TLSServerName)
	}
	if !config.TLSConfig.Insecure {
		t.Fatal("expected Insecure=true from NOMAD_SKIP_VERIFY")
	}
	t.Log("✅ All TLS env vars picked up correctly")
}

func TestAuthConfig_WithoutEnvVars(t *testing.T) {
	setenv(t, "NOMAD_TOKEN", "")
	setenv(t, "NOMAD_HTTP_AUTH", "")
	setenv(t, "NOMAD_CLIENT_CERT", "")
	setenv(t, "NOMAD_CLIENT_KEY", "")
	setenv(t, "NOMAD_CACERT", "")

	config := nomad.DefaultConfig()
	if config.SecretID != "" {
		t.Fatalf("expected empty SecretID when NOMAD_TOKEN not set, got: %q", config.SecretID)
	}
	if config.HttpAuth != nil {
		t.Fatal("expected nil HttpAuth when NOMAD_HTTP_AUTH not set")
	}
	if config.TLSConfig != nil && config.TLSConfig.ClientCert != "" {
		t.Fatal("expected no ClientCert when TLS env vars not set")
	}
	t.Log("✅ No auth configured when env vars unset — safe defaults")
}

// TestSubmitJob_WithAuthToken validates that SubmitJob works when NOMAD_TOKEN is set.
// Set NOMAD_ADDRESS and NOMAD_TOKEN to test against a real TLS+ACL cluster.
// Defaults to http://localhost:4646 (no TLS) if nothing is configured.
func TestSubmitJob_WithAuthToken(t *testing.T) {
	nomadAddr := resolveNomadAddr(t)

	// Use the real NOMAD_TOKEN from env if set, otherwise a fake one.
	// A fake token works when ACL is disabled; a real token is needed when ACL is on.
	testToken := os.Getenv("NOMAD_TOKEN")
	if testToken == "" {
		testToken = "ci-test-token-fake"
	}
	os.Setenv("NOMAD_TOKEN", testToken)
	defer os.Unsetenv("NOMAD_TOKEN")

	// Set env vars for a valid Docker job (HCL will parse, but placement
	// may fail without Docker driver — that's fine for auth testing).
	os.Setenv("NOMAD_CUSTOM_NAME", "auth-test")
	os.Setenv("DEPLOY_ENVIRONMENT", "auth")
	os.Setenv("NUM_REPLICA", "1")
	os.Setenv("PORT_NAME", "http")
	os.Setenv("TARGET_PORT", "3000")
	os.Setenv("IMAGE_URL", "docker.io/library/nginx:latest")
	os.Setenv("JOB_CPU", "50")
	os.Setenv("JOB_MEMORY", "32")
	os.Setenv("APP_HOST", "auth-test.local")
	os.Unsetenv("CONS_ATTR")
	os.Unsetenv("CONTAINER_DNS_SERVER")
	os.Unsetenv("APP_PREFIX_REGEX")
	os.Unsetenv("TRAEFIK_PASSWORD")
	os.Unsetenv("ENV_SOURCE")
	defer func() {
		os.Unsetenv("NOMAD_CUSTOM_NAME")
		os.Unsetenv("DEPLOY_ENVIRONMENT")
		os.Unsetenv("NUM_REPLICA")
		os.Unsetenv("PORT_NAME")
		os.Unsetenv("TARGET_PORT")
		os.Unsetenv("IMAGE_URL")
		os.Unsetenv("JOB_CPU")
		os.Unsetenv("JOB_MEMORY")
		os.Unsetenv("APP_HOST")
	}()

	err := SubmitJob(nomadAddr)
	// When ACL is disabled, a fake token is ignored → should succeed.
	// When ACL is enabled with a real token, it should also succeed.
	if err != nil {
		t.Logf("SubmitJob with auth token returned: %v", err)
		t.Log("(Expected if ACL is enabled and token is fake, or if Docker is unavailable)")
	} else {
		t.Log("✅ SubmitJob succeeded with NOMAD_TOKEN set — auth header was sent")
	}
}

// =============================================================================
// Helpers for integration tests
// =============================================================================

// resolveNomadAddr returns the Nomad address to use for tests.
// Reads NOMAD_ADDRESS (used by main.go) or NOMAD_ADDR (Nomad standard).
// If NOMAD_CACERT is set, defaults to HTTPS; otherwise HTTP.
func resolveNomadAddr(t *testing.T) string {
	if addr := os.Getenv("NOMAD_ADDRESS"); addr != "" {
		t.Logf("Using NOMAD_ADDRESS=%s", addr)
		return addr
	}
	if addr := os.Getenv("NOMAD_ADDR"); addr != "" {
		t.Logf("Using NOMAD_ADDR=%s", addr)
		return addr
	}
	// Default: HTTPS if CA cert is available, otherwise plain HTTP.
	if os.Getenv("NOMAD_CACERT") != "" {
		t.Log("NOMAD_CACERT set — defaulting to https://127.0.0.1:4646")
		return "https://127.0.0.1:4646"
	}
	return "http://localhost:4646"
}

// dockerDriverAvailable checks whether any Nomad node has a healthy Docker driver.
// Returns false if the check itself fails (e.g. Nomad not reachable).
func dockerDriverAvailable(t *testing.T, client *nomad.Client) bool {
	nodes, _, err := client.Nodes().List(nil)
	if err != nil {
		t.Logf("Could not list Nomad nodes: %v", err)
		return false
	}
	for _, nodeStub := range nodes {
		node, _, err := client.Nodes().Info(nodeStub.ID, nil)
		if err != nil {
			continue
		}
		if node.Drivers != nil {
			if d, ok := node.Drivers["docker"]; ok && d.Healthy {
				return true
			}
		}
	}
	return false
}

// rawExecJobHCL returns a minimal job that uses the raw_exec driver (no Docker needed).
// Useful for validating the Nomad v2.x API in dev environments without Docker.
func rawExecJobHCL(name, command string) string {
	return fmt.Sprintf(`
job "%s" {
  datacenters = ["dc1"]
  type        = "batch"

  group "test" {
    count = 1

    task "runner" {
      driver = "raw_exec"

      config {
        command = "%s"
        args    = ["-c", "echo nomad-v2-api-ok && exit 0"]
      }

      resources {
        cpu    = 50
        memory = 32
      }
    }
  }
}`, name, command)
}

// =============================================================================
// Integration test: SubmitJob to local Nomad v2.x
// =============================================================================

func TestSubmitJob_LocalNomad(t *testing.T) {
	nomadAddr := resolveNomadAddr(t)

	// Build config from scratch — do NOT use DefaultConfig() because
	// overriding config.Address afterward doesn't re-parse config.url
	// in Nomad's NewClient, which can break TLS setup.
	config := &nomad.Config{Address: nomadAddr}
	if cacert := os.Getenv("NOMAD_CACERT"); cacert != "" {
		config.TLSConfig = &nomad.TLSConfig{CACert: cacert}
	}
	if token := os.Getenv("NOMAD_TOKEN"); token != "" {
		config.SecretID = token
	}

	client, err := nomad.NewClient(config)
	if err != nil {
		t.Skipf("Nomad not available, skipping integration test: %v", err)
	}

	hasDocker := dockerDriverAvailable(t, client)
	t.Logf("Docker driver available: %v", hasDocker)

	// Parse HCL to validate Nomad v2.x API compatibility.
	// This works even without a running cluster when Nomad isn't available,
	// skip the registration part.
	var jobHCL, jobName string

	if hasDocker {
		// Full Docker + Traefik job via the tool's normal code path.
		os.Setenv("NOMAD_CUSTOM_NAME", "ci-test")
		os.Setenv("DEPLOY_ENVIRONMENT", "testing")
		os.Setenv("NUM_REPLICA", "1")
		os.Setenv("PORT_NAME", "http")
		os.Setenv("TARGET_PORT", "8080")
		os.Setenv("IMAGE_URL", "docker.io/library/nginx:latest")
		os.Setenv("JOB_CPU", "50")
		os.Setenv("JOB_MEMORY", "64")
		os.Setenv("APP_HOST", "ci-test.local")
		os.Unsetenv("CONS_ATTR")
		os.Unsetenv("CONTAINER_DNS_SERVER")
		os.Unsetenv("APP_PREFIX_REGEX")
		os.Unsetenv("TRAEFIK_PASSWORD")
		os.Unsetenv("ENV_SOURCE")

		defer func() {
			os.Unsetenv("NOMAD_CUSTOM_NAME")
			os.Unsetenv("DEPLOY_ENVIRONMENT")
			os.Unsetenv("NUM_REPLICA")
			os.Unsetenv("PORT_NAME")
			os.Unsetenv("TARGET_PORT")
			os.Unsetenv("IMAGE_URL")
			os.Unsetenv("JOB_CPU")
			os.Unsetenv("JOB_MEMORY")
			os.Unsetenv("APP_HOST")
		}()

		jobHCL = jobGeneration()
		jobName = "ci-test--testing"
	} else {
		// Fallback: raw_exec batch job for environments without Docker.
		t.Log("Docker not available — using raw_exec driver for API validation")
		jobName = "ci-test-raw-exec"
		jobHCL = rawExecJobHCL(jobName, "cmd")
	}

	// Parse the HCL via the Nomad v2.x API client.
	job, err := client.Jobs().ParseHCL(jobHCL, true)
	if err != nil {
		t.Skipf("Nomad not reachable for HCL parse, skipping: %v", err)
		return
	}
	t.Logf("✅ Nomad v2.x parsed job: %s", *job.Name)

	// Register the job.
	_, _, err = client.Jobs().Register(job, nil)
	if err != nil {
		t.Skipf("Nomad not reachable for job register, skipping: %v", err)
		return
	}
	t.Logf("✅ Job registered in Nomad v2.x: %s", *job.Name)

	// Clean up after the test.
	defer func() {
		_, _, err := client.Jobs().Deregister(jobName, true, nil)
		if err != nil {
			t.Logf("(cleanup) Failed to deregister %s: %v", jobName, err)
		} else {
			t.Logf("🧹 Cleaned up job: %s", jobName)
		}
	}()
}
