package kaddons

import (
	"github.com/blesswinsamuel/kgen"
)

type Options struct {
	CacheDir        string
	HelmKubeVersion string
}

var optionsContextKey = kgen.GenerateContextKey()

func SetOptions(scope kgen.Scope, opts Options) {
	if opts.HelmKubeVersion == "" {
		opts.HelmKubeVersion = "v1.30.2"
	}
	scope.SetContext(optionsContextKey, opts)
}

func getOptions(scope kgen.Scope) Options {
	return scope.GetContext(optionsContextKey).(Options)
}
