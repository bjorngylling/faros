//go:build mage

package main

import (
	"github.com/magefile/mage/sh"
)

var (
	packageName = "github.com/bjorngylling/faros"
)

func Build() error {
	return sh.Run("go", "build", packageName)
}

func Test() error {
	return sh.RunV("go", "test", "./...")
}
