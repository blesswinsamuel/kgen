package kaddons

import (
	"os"
	"path"

	"github.com/blesswinsamuel/kgen"
)

// Options for kaddons.
type Options struct {
	// CacheDir is the directory where the cache (downloaded helm charts) is stored. Default is os.TempDir() + "/kgen-cache".
	CacheDir string
	// HelmKubeVersion is the kubernetes version passed to helm (kube-version arg) while running helm template. Default is "v1.30.2".
	HelmKubeVersion string
	// Logger is the logger for kaddons. Default is the logger passed to the builder.
	logger kgen.Logger
}

var configContextKey = kgen.GenerateContextKey()

// SetOptions sets the options for kaddons.
func SetOptions(scope kgen.Scope, opts Options) {
	defaultOptions := getAddonsConfig(scope)
	if opts.CacheDir == "" {
		opts.CacheDir = defaultOptions.CacheDir
	}
	if opts.HelmKubeVersion == "" {
		opts.HelmKubeVersion = defaultOptions.HelmKubeVersion
	}
	if opts.logger == nil {
		opts.logger = defaultOptions.logger
	}
	scope.SetContext(configContextKey, opts)
}

func getAddonsConfig(scope kgen.Scope) Options {
	if v := scope.GetContext(configContextKey); v == nil {
		return Options{
			CacheDir:        path.Join(os.TempDir(), "kgen-cache"),
			HelmKubeVersion: "v1.30.2",
			logger:          scope.Logger(),
		}
	}
	return scope.GetContext(configContextKey).(Options)
}
