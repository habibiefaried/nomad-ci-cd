package helper

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

// Version is set at build time via ldflags (git tag or short commit SHA).
// Default "dev" means built locally without goreleaser.
var Version = "dev"

func RunCommandExec(cmdinput string) (string, error) {
	fmt.Println("[DEBUG] Executing " + cmdinput)
	cmd := exec.Command("/bin/sh", "-c", cmdinput)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.New(fmt.Sprint(err) + ": " + stderr.String())
	} else {
		return out.String(), nil
	}
}
