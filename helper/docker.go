package helper

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// isEligible return true if docker binary, env variable DOCKERFILE, and IMAGE_URL is available
func isEligible() error {
	if os.Getenv("DOCKERFILE") == "" {
		return errors.New("DOCKERFILE env var not found, skipping")
	}

	if os.Getenv("IMAGE_URL") == "" {
		return errors.New("IMAGE_URL env var not found, skipping")
	}

	cmdStr := "which docker"
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		return err
	}

	if string(out) == "" {
		return errors.New("docker binary not found")
	}

	return nil
}

// getRegistry extracts the registry host from IMAGE_URL.
// Returns the registry URL if present, empty string for Docker Hub images.
//
// Examples:
//
//	registry.example.com/namespace/repo:tag  →  registry.example.com
//	namespace/repo:tag                        →  ""  (Docker Hub)
//
// Set DOCKER_REGISTRY to override auto-detection.
func getRegistry() string {
	if r := os.Getenv("DOCKER_REGISTRY"); r != "" {
		return r
	}

	imageURL := os.Getenv("IMAGE_URL")
	parts := strings.Split(imageURL, "/")

	// Registry is present if the first segment contains a dot (hostname)
	// or a colon (hostname:port). Docker Hub images have no host segment.
	if len(parts) > 1 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		return parts[0]
	}

	return ""
}

func DockerBuildAndPush() error {
	err := isEligible()
	if err != nil {
		return err
	}

	cmdStr := fmt.Sprintf("docker build -f %v -t %v .", os.Getenv("DOCKERFILE"), os.Getenv("IMAGE_URL"))
	_, err = RunCommandExec(cmdStr)
	if err != nil {
		return err
	}

	registry := getRegistry()
	if registry != "" {
		cmdStr = fmt.Sprintf("echo %s | docker login --username %s --password-stdin %s",
			os.Getenv("DOCKER_LOGIN_PASSWORD"),
			os.Getenv("DOCKER_LOGIN_USERNAME"),
			registry)
	} else {
		cmdStr = fmt.Sprintf("echo %s | docker login --username %s --password-stdin",
			os.Getenv("DOCKER_LOGIN_PASSWORD"),
			os.Getenv("DOCKER_LOGIN_USERNAME"))
	}

	out, err := RunCommandExec(cmdStr)
	if err != nil {
		return err
	}
	fmt.Println(out)

	cmdStr = fmt.Sprintf("docker push %v", os.Getenv("IMAGE_URL"))
	out, err = RunCommandExec(cmdStr)
	if err != nil {
		return err
	}
	fmt.Println(out)

	return nil
}
