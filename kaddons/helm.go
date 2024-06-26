package kaddons

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
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

	CacheDir        string
	HelmKubeVersion string
	Logger          Logger
}

type Logger interface {
	Infof(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
}

type CustomLogger struct {
	InfoFn func(msg string, args ...interface{})
	WarnFn func(msg string, args ...interface{})
}

func (l CustomLogger) Infof(msg string, args ...interface{}) {
	if l.InfoFn == nil {
		l.InfoFn = log.Printf
	}
	l.InfoFn(msg, args...)
}

func (l CustomLogger) Warnf(msg string, args ...interface{}) {
	if l.WarnFn == nil {
		l.WarnFn = log.Printf
	}
	l.WarnFn(msg, args...)
}

func ExecHelmTemplateAndGetObjects(props HelmChartProps) ([]runtime.Object, error) {
	if props.Logger == nil {
		props.Logger = CustomLogger{}
	}
	if err := os.MkdirAll(props.CacheDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("MkdirAll failed: %w", err)
	}
	if _, err := exec.LookPath("helm"); err != nil {
		return nil, fmt.Errorf("helm not found in PATH: %w", err)
	}
	if props.ChartInfo.Repo == "" {
		return nil, errors.New("helm chart repo is empty")
	}
	if props.ChartInfo.Chart == "" {
		return nil, errors.New("helm chart name is empty")
	}
	if props.ChartInfo.Version == "" {
		return nil, errors.New("helm chart version is empty")
	}
	chartFileName := props.ChartInfo.Chart + "-" + props.ChartInfo.Version + ".tgz"
	if props.ChartFileNamePrefix != "" {
		chartFileName = props.ChartFileNamePrefix + props.ChartInfo.Version + ".tgz"
	}
	chartPath := path.Join(props.CacheDir, chartFileName)
	if _, err := os.Stat(chartPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			props.Logger.Infof("Fetching chart '%s' from repo '%s' version '%s'...", props.ChartInfo.Chart, props.ChartInfo.Repo, props.ChartInfo.Version)
			cmd := exec.Command("helm", "pull", props.ChartInfo.Chart, "--repo", props.ChartInfo.Repo, "--destination", props.CacheDir, "--version", props.ChartInfo.Version)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println(string(out))
				return nil, fmt.Errorf("helm pull failed: %w", err)
			} else {
				if len(out) > 0 {
					fmt.Println(string(out))
					props.Logger.Warnf("Received unexpected output from helm pull command for chart '%s'", props.ChartInfo.Chart)
				}
			}
		} else {
			return nil, fmt.Errorf("error occured while checking if chart exists in cache: %w", err)
		}
	}
	cmd := exec.Command(
		"helm",
		"template",
		props.ReleaseName,
		chartPath,
		"--namespace",
		props.Namespace,
		"--kube-version",
		props.HelmKubeVersion,
		"--include-crds",
		"--skip-tests",
		"--no-hooks",
		"--values",
		"-",
	)
	valuesJson, err := json.Marshal(props.Values)
	if err != nil {
		return nil, fmt.Errorf("json marshal failed: %w", err)
	}
	cmd.Stdin = strings.NewReader(string(valuesJson))
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(ee.Stderr))
		}
		return nil, fmt.Errorf("helm template failed: %w", err)
	}

	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(out)))
	var objects []runtime.Object
	for {
		var obj map[string]any

		bytes, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error decoding yaml: %w", err)
		}
		if len(bytes) == 0 {
			continue
		}

		if err := yaml.Unmarshal(bytes, &obj); err != nil {
			return nil, fmt.Errorf("error decoding yaml: %w", err)
		}
		if len(obj) == 0 {
			continue
		}
		runtimeObj := &unstructured.Unstructured{Object: obj}
		objects = append(objects, runtimeObj)
	}
	return objects, nil
}
