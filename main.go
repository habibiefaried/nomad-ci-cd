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

	err = helper.DockerBuildAndPush()
	if err != nil {
		fmt.Println(err)
	}

	if os.Getenv("NOMAD_ADDRESS") == "" {
		fmt.Println("skip nomad deployment")
	} else {
		err = nomadcicd.SubmitJob(os.Getenv("NOMAD_ADDRESS"))

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Success!")
		}
	}
}
