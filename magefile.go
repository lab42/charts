//go:build mage
// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/registry"
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

// Build Helm packages
func Package() error {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	helmCharts, err := filepath.Glob("./src/*")
	if err != nil {
		return err
	}

	for _, helmChart := range helmCharts {
		pkg := action.NewPackage()
		pkg.Destination = "./packages"

		if _, err := pkg.Run(helmChart, make(map[string]interface{})); err != nil {
			return err
		}
		log.Info().Msg("Packaged: " + helmChart)
	}

	return nil
}

// Index Helm packages
func Index() error {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	out := filepath.Join("charts", "index.yaml")
	mergeTo := ""

	if _, err := os.Stat(out); err == nil {
		mergeTo = out
	}

	i, err := repo.IndexDirectory("charts", "https://lab42.github.io/charts")
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

func Push() error {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	var loginOptions []registry.LoginOption
	loginOptions = append(loginOptions, registry.LoginOptBasicAuth(os.Getenv("HELM_USERNAME"), os.Getenv("HELM_PASSWORD")))

	client, err := registry.NewClient()
	if err != nil {
		return err
	}
	client.Login("https://lab42.github.io/charts", loginOptions...)

	versionPattern := regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+)`)
	helmPackages, _ := filepath.Glob("./packages/*.tgz")

	for _, helmPackage := range helmPackages {
		b, err := ioutil.ReadFile(helmPackage)
		if err != nil {
			return err
		}

		helmPackageVersion := versionPattern.FindString(helmPackage)
		helmPackageName := strings.TrimSuffix(helmPackage, fmt.Sprintf("-%s.tgz", helmPackageVersion))

		info, err := client.Push(b, fmt.Sprintf("https://lab42.github.io/charts/%s:%s", strings.TrimPrefix(helmPackageName, "packages/"), helmPackageVersion))
		if err != nil {
			return err
		}

		log.Info().Msg("Pushed: " + info.Ref)
		log.Info().Msg("Digest: " + info.Manifest.Digest)
	}

	return nil
}
