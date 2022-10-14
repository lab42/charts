//go:generate go run gen.go

package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/registry"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("Processing packages")

	helmCharts, _ := filepath.Glob("./src/*")
	linter := action.NewLint()
	log.Info().Msg("Linting")
	result := linter.Run(helmCharts, make(map[string]interface{}))

	for _, err := range result.Errors {
		if err != nil {
			log.Error().Err(err)
		}
	}

	for _, helmChart := range helmCharts {
		pkg := action.NewPackage()
		pkg.Destination = "./packages"

		if _, err := pkg.Run(helmChart, make(map[string]interface{})); err != nil {
			log.Error().Err(err)
		}
		log.Info().Msg("Packaged: " + helmChart)
	}

	var loginOptions []registry.LoginOption
	loginOptions = append(loginOptions, registry.LoginOptInsecure(true))
	loginOptions = append(loginOptions, registry.LoginOptBasicAuth("DanyHenriquez", "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"))

	client, err := registry.NewClient()
	if err != nil {
		log.Error().Err(err)
	}
	client.Login("ghcr.io/lab42/charts", loginOptions...)

	versionPattern := regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+)`)
	helmPackages, _ := filepath.Glob("./packages/*.tgz")

	for _, helmPackage := range helmPackages {
		b, err := ioutil.ReadFile(helmPackage)
		if err != nil {
			log.Error().Err(err)
		}

		helmPackageVersion := versionPattern.FindString(helmPackage)
		helmPackageName := strings.TrimSuffix(helmPackage, fmt.Sprintf("-%s.tgz", helmPackageVersion))

		info, err := client.Push(b, fmt.Sprintf("ghcr.io/lab42/charts/%s:%s", strings.TrimPrefix(helmPackageName, "packages/"), helmPackageVersion))
		if err != nil {
			log.Error().Err(err)
		}

		log.Info().Msg("Pushed: " + info.Ref)
		log.Info().Msg("Digest: " + info.Manifest.Digest)
	}
}
