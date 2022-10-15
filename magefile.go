//go:build mage
// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/registry"
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
			log.Error().Err(err)
			return err
		}
	}

	return nil
}

type HelmChart struct {
	APIVersion  string `yaml:"apiVersion"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Version     string `yaml:"version"`
}

// Return Helm charts info
func chartInfo(glob string) ([]HelmChart, error) {
	out := []HelmChart{}

	helmCharts, err := filepath.Glob(glob)
	if err != nil {
		return out, err
	}

	for _, helmChart := range helmCharts {
		b, err := ioutil.ReadFile(
			fmt.Sprintf("%s/Chart.yaml", helmChart),
		)
		if err != nil {
			return out, err
		}

		var helmChartYaml HelmChart
		err = yaml.Unmarshal(b, &helmChartYaml)
		if err != nil {
			return out, err
		}

		out = append(out, helmChartYaml)
	}

	return out, nil
}

func Build() error {
	helmCharts, err := chartInfo("./src/*")
	if err != nil {
		log.Error().Err(err)
		return err
	}

	for _, helmChartInfo := range helmCharts {
		pkg := action.NewPackage()
		pkg.Destination = "./charts"

		if _, err := pkg.Run(fmt.Sprintf("./src/%s", helmChartInfo.Name), make(map[string]interface{})); err != nil {
			log.Error().Err(err)
			return err
		}
		log.Info().Msg(fmt.Sprintf("Packaged: %s %s", helmChartInfo.Name, helmChartInfo.Version))
	}

	return nil
}

func Push() error {
	fmt.Println(os.Getenv("TOKEN"))
	loginOptions := []registry.LoginOption{
		registry.LoginOptBasicAuth(os.Getenv("USERNAME"), os.Getenv("TOKEN")),
	}

	client, err := registry.NewClient(registry.ClientOptDebug(true), registry.ClientOptEnableCache(false))
	if err != nil {
		return err
	}

	if err := client.Login("ghcr.io/lab42/charts", loginOptions...); err != nil {
		return err
	}

	helmCharts, err := chartInfo("./src/*")
	if err != nil {
		log.Error().Err(err)
		return err
	}

	for _, helmChartInfo := range helmCharts {
		b, err := ioutil.ReadFile(fmt.Sprintf("./charts/%s-%s.tgz", helmChartInfo.Name, helmChartInfo.Version))
		if err != nil {
			log.Error().Err(err)
			return err
		}

		info, err := client.Push(b, fmt.Sprintf("ghcr.io/lab42/charts/%s:%s", helmChartInfo.Name, helmChartInfo.Version))
		if err != nil {
			log.Error().Err(err)
			return err
		}

		log.Info().Msg("Pushed: " + info.Ref)
		log.Info().Msg("Digest: " + info.Manifest.Digest)
	}

	return nil
}
