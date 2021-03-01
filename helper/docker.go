package helper

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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

func DockerBuildAndPush() error {
	err := isEligible()
	if err != nil {
		return err
	} else {
		cmdStr := fmt.Sprintf("docker build -f %v -t %v .", os.Getenv("DOCKERFILE"), os.Getenv("IMAGE_URL"))
		_, err := RunCommandExec(cmdStr)
		if err != nil {
			return err
		}

		cmdStr = fmt.Sprintf("echo %s |docker login --username %s --password-stdin", os.Getenv("DOCKER_LOGIN_PASSWORD"), os.Getenv("DOCKER_LOGIN_USERNAME"))
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
	}

	return nil
}
