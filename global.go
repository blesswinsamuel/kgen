package kgen

import (
	"k8s.io/apimachinery/pkg/runtime"
)

type globalContext struct {
	scheme *runtime.Scheme
}
