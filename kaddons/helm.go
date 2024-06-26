package kaddons

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/blesswinsamuel/kgen"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type HelmChartInfo struct {
	Repo    string `json:"repo"`
	Chart   string `json:"chart"`
	Version string `json:"version"`
}

type HelmChartProps struct {
	ChartInfo           HelmChartInfo
	ChartFileNamePrefix string
	ReleaseName         string
	Namespace           string
	Values              map[string]interface{}
	PatchObject         func(resource *unstructured.Unstructured)
}

func AddHelmChart(scope kgen.Scope, props HelmChartProps) {
	opts := getOptions(scope)
	chartsCacheDir := path.Join(opts.CacheDir, "charts")
	if err := os.MkdirAll(chartsCacheDir, os.ModePerm); err != nil {
		log.Panic().Err(err).Msg("MkdirAll failed")
	}
	if _, err := exec.LookPath("helm"); err != nil {
		log.Panic().Err(err).Msg("helm LookPath failed")
	}
	if props.ChartInfo.Repo == "" {
		log.Panic().Msgf("helm chart repo is empty for %s", props.ReleaseName)
	}
	if props.ChartInfo.Chart == "" {
		log.Panic().Msgf("helm chart name is empty for %s", props.ReleaseName)
	}
	if props.ChartInfo.Version == "" {
		log.Panic().Msgf("helm chart version is empty for %s", props.ReleaseName)
	}
	chartFileName := props.ChartInfo.Chart + "-" + props.ChartInfo.Version + ".tgz"
	if props.ChartFileNamePrefix != "" {
		chartFileName = props.ChartFileNamePrefix + props.ChartInfo.Version + ".tgz"
	}
	chartPath := fmt.Sprintf("%s/%s", chartsCacheDir, chartFileName)
	if _, err := os.Stat(chartPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Info().Msgf("Fetching chart '%s' from repo '%s' version '%s'...", props.ChartInfo.Chart, props.ChartInfo.Repo, props.ChartInfo.Version)
			cmd := exec.Command("helm", "pull", props.ChartInfo.Chart, "--repo", props.ChartInfo.Repo, "--destination", chartsCacheDir, "--version", props.ChartInfo.Version)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println(string(out))
				log.Panic().Err(err).Msg("Error occured during helm pull command")
			} else {
				if len(out) > 0 {
					log.Warn().Str("output", string(out)).Msgf("Received unexpected output from helm pull command for chart '%s'", props.ChartInfo.Chart)
				}
			}
		} else {
			log.Panic().Err(err).Msg("Error occured while checking if chart exists in cache")
		}
	}
	namespace := props.Namespace
	if namespace == "" {
		namespace = scope.Namespace()
	}

	cmd := exec.Command(
		"helm",
		"template",
		props.ReleaseName,
		chartPath,
		"--namespace",
		namespace,
		"--kube-version",
		opts.HelmKubeVersion,
		"--include-crds",
		"--skip-tests",
		"--no-hooks",
		"--values",
		"-",
	)
	valuesJson, err := json.Marshal(props.Values)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to convert to JSON")
	}
	cmd.Stdin = strings.NewReader(string(valuesJson))
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(ee.Stderr))
		}
		log.Panic().Err(err).Msg("helm template failed")
	}

	dec := yaml.NewDecoder(bytes.NewReader(out))
	i := 0
	for {
		i++
		var obj map[string]any
		err := dec.Decode(&obj)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Panic().Err(err).Msg("Error decoding yaml")
		}
		if len(obj) == 0 {
			continue
		}
		runtimeObj := &unstructured.Unstructured{Object: obj}
		if props.PatchObject != nil {
			props.PatchObject(runtimeObj)
		}
		scope.AddApiObject(runtimeObj)
	}
}
