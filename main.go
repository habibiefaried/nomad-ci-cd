package main

import (
	"fmt"
	"github.com/habibiefaried/nomad-ci-cd/helper"
	nomadcicd "github.com/habibiefaried/nomad-ci-cd/nomad"
	"os"
)

func main() {
	out, err := helper.RunCommandExec("env")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(out)

	out, err = helper.RunCommandExec("curl https://api.ipify.org")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(out)

	err = helper.DockerBuildAndPush()
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	// Sync IMAGE_URL to the dynamically-tagged version so the Nomad job
	// references the same image that DockerBuildAndPush just pushed,
	// never "latest".
	taggedURL := helper.TagImageURL()
	os.Setenv("IMAGE_URL", taggedURL)
	fmt.Printf("[INFO] Nomad job will use image: %s\n", taggedURL)

	if os.Getenv("NOMAD_ADDRESS") == "" {
		fmt.Println("skip nomad deployment")
	} else {
		err = nomadcicd.SubmitJob(os.Getenv("NOMAD_ADDRESS"))

		if err != nil {
			fmt.Println(err)
			os.Exit(5)
		} else {
			fmt.Println("Success!")
		}
	}
}
