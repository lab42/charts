//go:build mage
// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
	AppVersion  string `yaml:"appVersion"`
}

type GithubRelease struct {
	TagName         string    `json:"tag_name"`
	Name            string    `json:"name"`
	TargetCommitish string    `json:"target_commitish"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
}

// Return Helm charts info
func chartInfo(glob string) ([]HelmChart, error) {
	out := []HelmChart{}

	helmCharts, err := filepath.Glob(glob)
	if err != nil {
		log.Error().Err(err)
		return out, err
	}

	for _, helmChart := range helmCharts {
		b, err := ioutil.ReadFile(
			fmt.Sprintf("%s/Chart.yaml", helmChart),
		)
		if err != nil {
			log.Error().Err(err)
			return out, err
		}

		var helmChartYaml HelmChart
		err = yaml.Unmarshal(b, &helmChartYaml)
		if err != nil {
			log.Error().Err(err)
			return out, err
		}

		out = append(out, helmChartYaml)
	}

	return out, nil
}

func latestReleaseVersion(repository string) string {
	url := fmt.Sprintf("https://api.github.com/repos/lab42/%s/releases/latest", repository)
	var bearer = "Bearer " + os.Getenv("TOKEN")

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Accept", "application/vnd.github+json")

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Panic().Err(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic().Err(err)
	}

	var githubRelease GithubRelease
	if err := yaml.Unmarshal(body, &githubRelease); err != nil {
		log.Panic().Err(err)
	}

	return githubRelease.Name
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
		if helmChartInfo.Name != "namespace" {
			pkg.AppVersion = latestReleaseVersion(helmChartInfo.Name)
		}

		if _, err := pkg.Run(fmt.Sprintf("./src/%s", helmChartInfo.Name), make(map[string]interface{})); err != nil {
			log.Error().Err(err)
			return err
		}
		log.Info().Msg(fmt.Sprintf("Packaged: %s %s", helmChartInfo.Name, helmChartInfo.Version))
	}

	return nil
}

func Push() error {

	client, err := registry.NewClient()
	if err != nil {
		return err
	}

	loginAction := action.NewRegistryLogin(&action.Configuration{RegistryClient: client})
	if err := loginAction.Run(os.Stdout, "ghcr.io/lab42/charts", os.Getenv("USERNAME"), os.Getenv("TOKEN"), false); err != nil {
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
