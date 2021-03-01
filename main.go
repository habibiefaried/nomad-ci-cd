package main

import (
	"fmt"
	"github.com/habibiefaried/nomad-ci-cd/helper"
	nomadcicd "github.com/habibiefaried/nomad-ci-cd/nomad"
	"os"
)

func main() {
	fmt.Println("Main program called")
	err := nomadcicd.SubmitJob(os.Getenv("NOMAD_ADDRESS"))
	out, err := helper.RunCommandExec("env")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(out)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Success!")
	}
}
