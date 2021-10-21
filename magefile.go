//go:build mage

package main

import (
	"time"

	"github.com/magefile/mage/sh"
)

var (
	packageName = "github.com/bjorngylling/faros"
	ldflags     = "-X main.commitHash=$COMMIT_HASH -X main.buildDate=$BUILD_DATE"
)

func Build() error {
	return sh.RunWith(flagEnv(), "go", "build", "-ldflags", ldflags, packageName)
}

func Test() error {
	return sh.RunV("go", "test", "./...")
}

func flagEnv() map[string]string {
	hash, _ := sh.Output("git", "rev-parse", "--short", "HEAD")
	return map[string]string{
		"COMMIT_HASH": hash,
		"BUILD_DATE":  time.Now().Format("2006-01-02T15:04:05Z0700"),
	}
}
