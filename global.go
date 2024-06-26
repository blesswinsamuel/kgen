package kgen

import (
	"bytes"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type globalContext struct {
	scheme         *runtime.Scheme
	codecFactory   serializer.CodecFactory
	jsonSerializer runtime.Serializer
	yamlSerializer runtime.Serializer
}

func (gc *globalContext) k8sObjectToMap(obj runtime.Object) map[string]any {
	b := bytes.NewBuffer(nil)
	if err := gc.jsonSerializer.Encode(obj, b); err != nil {
		panic(fmt.Errorf("k8sObjectToMap: %w", err))
	}
	// fmt.Println(string(b.Bytes()))

	var out map[string]any
	if err := json.Unmarshal(b.Bytes(), &out); err != nil {
		panic(fmt.Errorf("k8sObjectToMap: %w", err))
	}
	return out
}

func (gc *globalContext) yamlToK8sObject(data []byte) runtime.Object {
	obj, _, err := gc.codecFactory.UniversalDeserializer().Decode(data, nil, nil)
	if err != nil {
		// log.Println(string(data))
		panic(fmt.Errorf("yamlToK8sObject: %w", err))
	}
	return obj
}
