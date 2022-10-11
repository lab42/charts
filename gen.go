//go:build ignore

package main

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

func main() {
	charts, _ := filepath.Glob("./src/*")
	for _, c := range charts {
		if chartHasUpdates(c) {
			cmd := exec.Command("helm", "package", "--destination", "./packages", c)
			if err := cmd.Run(); err != nil {
				fmt.Println(err)
			}
		}
	}
}

type Chart struct {
	APIVersion  string `yaml:"apiVersion"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Version     string `yaml:"version"`
}

func chartHasUpdates(path string) bool {
	cmd := exec.Command("git", "diff", "--exit-code", "HEAD^", path)

	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return true
		}
	}

	return false
}
