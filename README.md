![Go](https://github.com/habibiefaried/nomad-ci-cd/workflows/Go/badge.svg)

# Description
Nomad ci/cd generator to generate all CI/CD command in 1 file

# Features
1. Build and push docker image
2. Publish nomad job with API

# Environment variables guide
* NOMAD_CUSTOM_NAME: Nomad job to be run
* DEPLOY_ENVIRONMENT: Set env name (can be "staging", "prod", etc)
* NUM_REPLICA: Container replica for nomad job
* PORT_NAME: Port name that must be unique for entire cluster
* TARGET_PORT: Port that will be used by container
* IMAGE_URL: Docker image URL
* JOB_CPU: CPU for this container in MHz
* JOB_MEMORY: Memory for this container in MB
* APP_PREFIX_REGEX: *Optional*, if you have predefined prefix/path on your system
* APP_HOST: DNS of this app. Can use "#" to use multiple domains
* TRAEFIK_PASSWORD: *Optional*, if you want to protect this container with apache2 compliant basic authentication
* ENV_SOURCE: *Optional*, if you have env file that want to be sourced as container environment variables. If not set, then by default it will try to find `.env` file
* DOCKER_LOGIN_PASSWORD: Password to login (only dockerhub.com supported)
* DOCKER_LOGIN_USERNAME: Username to login (only dockerhub.com supported)
* DOCKERFILE: Dockerfile name
* NOMAD_ADDRESS: Address of nomad server, will skip deployment if this is not set
* CONTAINER_DNS_SERVER: DNS Server that container should be using