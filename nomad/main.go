package nomad

import (
	"fmt"
	nomad "github.com/hashicorp/nomad/api"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func SubmitJob(address string) error {
	c, err := nomad.NewClient(&nomad.Config{
		Address: address,
	})
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
	return fmt.Sprintf(template, namaJob, os.Getenv("DEPLOY_ENVIRONMENT"), os.Getenv("NUM_REPLICA"), os.Getenv("PORT_NAME"), os.Getenv("TARGET_PORT"), generateDNSServer(), templateGenerator(), fmt.Sprintf("%v", currentTime.Format("2006-01-02 15:04:05.000000000")), os.Getenv("IMAGE_URL"), os.Getenv("PORT_NAME"), os.Getenv("JOB_CPU"), os.Getenv("JOB_MEMORY"), namaJob, os.Getenv("DEPLOY_ENVIRONMENT"), os.Getenv("PORT_NAME"), tagGenerator(), os.Getenv("PORT_NAME"))

}

func generateDNSServer() string {
	if os.Getenv("CONTAINER_DNS_SERVER") != "" {
		return fmt.Sprintf(`dns {
        servers = ["%s"]
      }`, os.Getenv("CONTAINER_DNS_SERVER"))
	} else {
		return ""
	}
}

func hostGenerator() string {
	ret := ""
	arrstr := strings.Split(os.Getenv("APP_HOST"), "#")
	for _, vhost := range arrstr {
		if vhost != "" {
			ret = ret + fmt.Sprintf("Host(\\\"%s\\\") || ", vhost)
		}
	}

	return ret[:len(ret)-3]
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
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls.domains[0].main=%s\",\n", os.Getenv("PORT_NAME"), hostGenerator())
	} else {
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s.rule=%s\",\n", os.Getenv("PORT_NAME"), hostGenerator())
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls=true\",\n", os.Getenv("PORT_NAME"))
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.rule=%s\",\n", os.Getenv("PORT_NAME"), hostGenerator())
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls.certresolver=myresolver\",\n", os.Getenv("PORT_NAME"))
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s-https.tls.domains[0].main=%s\",\n", os.Getenv("PORT_NAME"), hostGenerator())
	}

	if os.Getenv("TRAEFIK_PASSWORD") != "" {
		isMiddlewareEnabled = true
		tags = tags + fmt.Sprintf("\t\"traefik.http.middlewares.%v.basicauth.users=%v\",\n", os.Getenv("PORT_NAME"), os.Getenv("TRAEFIK_PASSWORD"))
	}

	if isMiddlewareEnabled {
		tags = tags + fmt.Sprintf("\t\"traefik.http.routers.%s.middlewares=%s@consulcatalog\",\n", os.Getenv("PORT_NAME"), os.Getenv("PORT_NAME"))
	}
	return tags
}

func templateGenerator() string {
	targetFile := ".env" // Defaulted to .env
	if os.Getenv("ENV_SOURCE") != "" {
		targetFile = os.Getenv("ENV_SOURCE")
	}

	content, err := ioutil.ReadFile(targetFile)
	if err != nil {
		fmt.Println(err)
		return ""
	} else {
		template := `template {
		data          = <<EOH
		%s
		EOH
		destination   = ".env"
		env           = false
	}`
		return fmt.Sprintf(template, string(content))
	}
}
