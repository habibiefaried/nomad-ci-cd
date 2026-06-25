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

// tagImageURL returns IMAGE_URL with its tag replaced by Version.
// If IMAGE_URL has no tag, Version is appended.
// The registry port (e.g. registry:5000) is never mistaken for a tag
// because ports always appear in the host segment, before the first "/".
//
// Examples (Version = "abc123def"):
//
//	registry:5000/myimage:latest   →  registry:5000/myimage:abc123def
//	registry.example.com/repo:tag  →  registry.example.com/repo:abc123def
//	myimage:latest                 →  myimage:abc123def
//	myimage                        →  myimage:abc123def
func tagImageURL() string {
	imageURL := os.Getenv("IMAGE_URL")

	// Split into segments: [host(:port), path, ..., image(:tag)]
	parts := strings.Split(imageURL, "/")
	last := parts[len(parts)-1]

	// Strip existing tag from the last segment (image name), if present.
	// A ":" in the first segment is a registry port, not a tag.
	if idx := strings.LastIndex(last, ":"); idx != -1 {
		parts[len(parts)-1] = last[:idx]
	}

	return strings.Join(parts, "/") + ":" + Version
}

func DockerBuildAndPush() error {
	err := isEligible()
	if err != nil {
		return err
	}

	taggedURL := tagImageURL()
	fmt.Printf("[INFO] Docker image tag: %s\n", taggedURL)

	cmdStr := fmt.Sprintf("docker build -f %v -t %v .", os.Getenv("DOCKERFILE"), taggedURL)
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

	cmdStr = fmt.Sprintf("docker push %v", taggedURL)
	out, err = RunCommandExec(cmdStr)
	if err != nil {
		return err
	}
	fmt.Println(out)

	return nil
}
