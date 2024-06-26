package kgen

import (
	"bytes"
	"sort"

	"github.com/goccy/go-yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type ApiObject interface { // Object
	metav1.Type
	metav1.Object
	ToYAML() []byte
	GetObject() runtime.Object
	SetObject(v unstructured.Unstructured)
}

type ApiObjectProps struct {
	unstructured.Unstructured
}

func getNamespaceContext(scope Scope) string {
	ns, _ := scope.GetContext("namespace").(string)
	return ns
}

type apiObject struct {
	ApiObjectProps
}

func (a *apiObject) GetObject() runtime.Object {
	return &a.Unstructured
}

func (a *apiObject) SetObject(v unstructured.Unstructured) {
	a.Unstructured = v
}

func (a *apiObject) ToYAML() []byte {
	b := bytes.NewBuffer(nil)
	enc := yaml.NewEncoder(b, yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true), yaml.UseSingleQuote(false))
	sortedMap := yaml.MapSlice{}
	keys := []string{}
	for k := range a.Object {
		if k == "apiVersion" || k == "kind" || k == "metadata" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	keys = append([]string{"apiVersion", "kind", "metadata"}, keys...)
	for _, key := range keys {
		sortedMap = append(sortedMap, yaml.MapItem{
			Key:   key,
			Value: a.Object[key],
		})
	}
	err := enc.Encode(sortedMap)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}
