//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/repo"
	// mg contains helpful utility functions, like Deps
)

// Lint Helm packages
func Lint() error {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	helmCharts, _ := filepath.Glob("./src/*")
	linter := action.NewLint()
	result := linter.Run(helmCharts, make(map[string]interface{}))

	for _, err := range result.Errors {
		if err != nil {
			return err
		}
	}

	return nil
}

// Update Helm registry
func Update() error {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Info().Msg("Cloning registry")
	url := fmt.Sprintf("https://%s:%s@github.com/lab42/registry.git", os.Getenv("USERNAME"), os.Getenv("TOKEN"))
	repository, err := git.PlainClone("./charts", false, &git.CloneOptions{
		URL:           url,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName("refs/heads/main"),
		SingleBranch:  true,
	})

	if err != nil {
		return err
	}

	helmCharts, err := filepath.Glob("./src/*")
	if err != nil {
		return err
	}

	for _, helmChart := range helmCharts {
		pkg := action.NewPackage()
		pkg.Destination = "./charts"

		if _, err := pkg.Run(helmChart, make(map[string]interface{})); err != nil {
			return err
		}
		log.Info().Msg("Packaged: " + helmChart)
	}

	if err := index(url); err != nil {
		return err
	}

	workTree, err := repository.Worktree()
	workTree.AddGlob("*.tgz")
	workTree.AddGlob("*.yaml")

	hash, err := workTree.Commit(os.Getenv("GITHUB_SHA"),
		&git.CommitOptions{
			All: true,
			Author: &object.Signature{
				Name:  "Dany Henriquez",
				Email: "danyhenriquez86@gmail.com",
				When:  time.Now(),
			},
		},
	)
	if err != nil {
		return nil
	}

	log.Info().Msg("Commit hash: " + hash.String())

	log.Info().Msg("Pushing registry")
	repository.Push(&git.PushOptions{
		Progress: os.Stdout,
	})
	return nil
}

func index(url string) error {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	out := filepath.Join("charts", "index.yaml")
	mergeTo := ""

	if _, err := os.Stat(out); err == nil {
		mergeTo = out
	}

	i, err := repo.IndexDirectory("charts", url)
	if err != nil {
		return err
	}

	if mergeTo != "" {
		// if index.yaml is missing then create an empty one to merge into
		var i2 *repo.IndexFile
		if _, err := os.Stat(mergeTo); os.IsNotExist(err) {
			i2 = repo.NewIndexFile()
			i2.WriteFile(mergeTo, 0644)
		} else {
			i2, err = repo.LoadIndexFile(mergeTo)
			if err != nil {
				return errors.Wrap(err, "merge failed")
			}
		}
		i.Merge(i2)
	}
	i.SortEntries()
	return i.WriteFile(out, 0644)
}
