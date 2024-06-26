package kgen

import (
	"bytes"
	"sort"

	"github.com/goccy/go-yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	// "sigs.k8s.io/yaml"
)

type ApiObject interface {
	metav1.Type
	metav1.Object
	ToYAML() []byte
	GetObject() runtime.Object
	ReplaceObject(v runtime.Object)
}

type ApiObjectProps struct {
	*unstructured.Unstructured
}

type apiObject struct {
	ApiObjectProps
	globalContext *globalContext
}

var _ ApiObject = &apiObject{}

func (a *apiObject) GetObject() runtime.Object {
	return a.Unstructured
}

func (a *apiObject) ReplaceObject(obj runtime.Object) {
	if objUnstructured, ok := obj.(*unstructured.Unstructured); ok {
		a.Unstructured = objUnstructured
		return
	}
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		panic(err)
	}
	a.Unstructured = &unstructured.Unstructured{Object: unstructuredObj}
}

func (a *apiObject) ToYAML() []byte {
	// // reference: https://github.com/kubernetes/cli-runtime/blob/8e480ebaa098dffbb0bd05f3d7b47b1d1d2d4847/pkg/printers/yaml.go#L75-L84
	// if a.Unstructured.GetObjectKind().GroupVersionKind().Empty() {
	// 	panic("missing apiVersion or kind; try GetObjectKind().SetGroupVersionKind() if you know the type")
	// }

	// output, err := yaml.Marshal(a.Unstructured)
	// if err != nil {
	// 	panic(fmt.Errorf("yaml.Marshal: %w", err))
	// }
	// return output

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
