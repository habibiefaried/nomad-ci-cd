package nomad

import (
	"fmt"
	nomad "github.com/hashicorp/nomad/api"
	"os"
	"strings"
	"time"
)

func SubmitJob(address string) error {
	// Set NOMAD_ADDR so DefaultConfig picks up the correct address
	// along with TLS config.
	if address != "" {
		os.Setenv("NOMAD_ADDR", address)
	}
	config := nomad.DefaultConfig()

	// Only skip TLS verification when explicitly opted in.
	// Use NOMAD_CACERT or NOMAD_CAPATH for proper CA verification.
	if os.Getenv("NOMAD_SKIP_VERIFY") == "true" {
		if config.TLSConfig == nil {
			config.TLSConfig = &nomad.TLSConfig{}
		}
		config.TLSConfig.Insecure = true
	}

	c, err := nomad.NewClient(config)
	if err != nil {
		return err
	}

	jobs := c.Jobs()
	s := jobGeneration()
	fmt.Println(s)
	job, err := jobs.ParseHCL(s, true)
	if err != nil {
		return err
	}

	_, _, err = jobs.Register(job, nil)
	return err
}

func jobGeneration() string {
	var namaJob string
	// Traefik: To set the value of a rule, use backticks ` or escaped double-quotes \".
	currentTime := time.Now()
	namaJob = os.Getenv("NOMAD_CUSTOM_NAME")

	template := `
job %s--%s {
  datacenters = ["dc1"]
  %s
  group "app" {
    count = %s
    network {
      port "%s" { to = %s }
      %s
    }

    task "server" {
      %s

      driver = "docker"
      env {
        BUILDNUMBER = "%s"
      }
      config {
        image = "%s"
        ports = ["%s"]
        force_pull = true
        %s
      }

      resources {
        cpu    = %s
        memory = %s
      }
    }

    service {
      name = "%s--%s"
      port = "%s"
      tags = [
      	%s
      ]
      check {
        port        = "%s"
        type        = "tcp"
        interval    = "15s"
        timeout     = "14s"
      }
    }
  }
}`
	return fmt.Sprintf(template,
		namaJob,
		os.Getenv("DEPLOY_ENVIRONMENT"),
		constraintGenerator(),
		os.Getenv("NUM_REPLICA"),
		os.Getenv("PORT_NAME"),
		os.Getenv("TARGET_PORT"),
		generateDNSServer(),
		templateGenerator(),
		fmt.Sprintf("%v", currentTime.Format("2006-01-02 15:04:05.000000000")),
		os.Getenv("IMAGE_URL"),
		os.Getenv("PORT_NAME"),
		authGenerator(),
		os.Getenv("JOB_CPU"),
		os.Getenv("JOB_MEMORY"),
		namaJob,
		os.Getenv("DEPLOY_ENVIRONMENT"),
		os.Getenv("PORT_NAME"),
		tagGenerator(),
		os.Getenv("PORT_NAME"),
	)
}

func generateDNSServer() string {
	if os.Getenv("CONTAINER_DNS_SERVER") != "" {
		return fmt.Sprintf(`dns {
        servers = ["%s"]
      }`, os.Getenv("CONTAINER_DNS_SERVER"))
	}
	return ""
}

func hostGenerator() string {
	ret := ""
	arrstr := strings.Split(os.Getenv("APP_HOST"), "#")
	for _, vhost := range arrstr {
		if vhost != "" {
			ret = ret + fmt.Sprintf("Host(\\\"%s\\\") || ", vhost)
		}
	}

	if ret == "" {
		return ""
	}
	// Remove trailing " || " (4 characters)
	return ret[:len(ret)-4]
}

// authGenerator returns a Nomad Docker auth block for private registry pull.
// Reads NOMAD_REGISTRY_USERNAME / NOMAD_REGISTRY_PASSWORD, falling back to
// DOCKER_LOGIN_USERNAME / DOCKER_LOGIN_PASSWORD.
// Returns empty string if no credentials are set.
func authGenerator() string {
	username := os.Getenv("NOMAD_REGISTRY_USERNAME")
	if username == "" {
		username = os.Getenv("DOCKER_LOGIN_USERNAME")
	}
	password := os.Getenv("NOMAD_REGISTRY_PASSWORD")
	if password == "" {
		password = os.Getenv("DOCKER_LOGIN_PASSWORD")
	}

	if username != "" && password != "" {
		return fmt.Sprintf(`auth {
        username = "%s"
        password = "%s"
      }`, username, password)
	}
	return ""
}

func tagGenerator() string {
	tags := "\"traefik.enable=true\",\n"
	isMiddlewareEnabled := false

	if os.Getenv("APP_PREFIX_REGEX") != "" {
		isMiddlewareEnabled = true
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s.rule=%s && PathPrefix(\\\"%s\\\")\",\n", os.Getenv("PORT_NAME"), hostGenerator(), os.Getenv("APP_PREFIX_REGEX"))
		tags = tags + fmt.Sprintf("\t\"traefik.http.middlewares.%s.stripprefix.prefixes=%s\",\n", os.Getenv("PORT_NAME"), os.Getenv("APP_PREFIX_REGEX"))
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls=true\",\n", os.Getenv("PORT_NAME"))
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.rule=%s\" && PathPrefix(\\\"%s\\\")\",\n", os.Getenv("PORT_NAME"), hostGenerator(), os.Getenv("APP_PREFIX_REGEX"))
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls.certresolver=myresolver\",\n", os.Getenv("PORT_NAME"))
	} else {
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s.rule=%s\",\n", os.Getenv("PORT_NAME"), hostGenerator())
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls=true\",\n", os.Getenv("PORT_NAME"))
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.rule=%s\",\n", os.Getenv("PORT_NAME"), hostGenerator())
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls.certresolver=myresolver\",\n", os.Getenv("PORT_NAME"))
	}

	if os.Getenv("TRAEFIK_PASSWORD") != "" {
		isMiddlewareEnabled = true
		tags = tags + fmt.Sprintf("\t\"traefik.http.middlewares.%v.basicauth.users=%v\",\n", os.Getenv("PORT_NAME"), os.Getenv("TRAEFIK_PASSWORD"))
	}

	if isMiddlewareEnabled {
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s.middlewares=%s@consulcatalog\",\n", os.Getenv("PORT_NAME"), os.Getenv("PORT_NAME"))
	}

	tags = tags + fmt.Sprintf("\t\"traefik.http.middlewares.%s-https.redirectscheme.scheme=https\",\n", os.Getenv("PORT_NAME"))
	tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s.middlewares=%s-https\",\n", os.Getenv("PORT_NAME"), os.Getenv("PORT_NAME"))
	return tags
}

func templateGenerator() string {
	targetFile := ".env" // Defaulted to .env
	if os.Getenv("ENV_SOURCE") != "" {
		targetFile = os.Getenv("ENV_SOURCE")
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	template := `template {
        data          = <<EOH
%s
        EOH
        destination   = ".env"
        env           = false
      }`
	return fmt.Sprintf(template, string(content))
}

func constraintGenerator() string {
	if os.Getenv("CONS_ATTR") != "" && os.Getenv("CONS_OP") != "" && os.Getenv("CONS_VALUE") != "" {
		template := `constraint {
        attribute = "${%s}"
        operator  = "%s"
        value     = "%s"
      }`

		return fmt.Sprintf(template, os.Getenv("CONS_ATTR"), os.Getenv("CONS_OP"), os.Getenv("CONS_VALUE"))
	}
	return ""
}
