package kaddons

import (
	"os"
	"path"

	"github.com/blesswinsamuel/kgen"
)

type Options struct {
	CacheDir        string
	HelmKubeVersion string
	logger          kgen.Logger
}

var optionsContextKey = kgen.GenerateContextKey()

func SetOptions(scope kgen.Scope, opts Options) {
	defaultOptions := getOptions(scope)
	if opts.CacheDir == "" {
		opts.CacheDir = defaultOptions.CacheDir
	}
	if opts.HelmKubeVersion == "" {
		opts.HelmKubeVersion = defaultOptions.HelmKubeVersion
	}
	if opts.logger == nil {
		opts.logger = defaultOptions.logger
	}
	scope.SetContext(optionsContextKey, opts)
}

func getOptions(scope kgen.Scope) Options {
	if v := scope.GetContext(optionsContextKey); v == nil {
		return Options{
			CacheDir:        path.Join(os.TempDir(), "kgen-cache"),
			HelmKubeVersion: "v1.30.2",
			logger:          scope.Logger(),
		}
	}
	return scope.GetContext(optionsContextKey).(Options)
}
