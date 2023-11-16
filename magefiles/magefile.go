package main

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/mg"
)

var (
	packageName = "github.com/bjorngylling/faros"
)

func Build() error {
	return sh.Run("go", "build", packageName)
}

func Run() error {
	mg.Deps(Build)
	return sh.Run("./faros", "-port", "8080")
}

func DockerBuild() error {
	return sh.Run("docker", "build",
		"--build-arg", fmt.Sprintf(`PACKAGE=%s`, packageName),
		"-t", "faros:latest", ".")
}

func Test() error {
	return sh.RunV("go", "test", "./...")
}
