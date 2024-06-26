package kgen

import (
	"fmt"
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type App interface {
	Scope
	WriteYAMLsToDisk() error
}

type AppProps struct {
	Outdir        string
	CacheDir      string
	DeleteOutDir  bool
	PatchObject   func(ApiObject) error
	SchemeBuilder runtime.SchemeBuilder
}

type app struct {
	Scope
	props AppProps
}

func NewApp(props AppProps) App {
	scheme := runtime.NewScheme()
	utilruntime.Must(props.SchemeBuilder.AddToScheme(scheme))
	codecFectory := serializer.NewCodecFactory(scheme)
	// deserializer := codecFectory.UniversalDeserializer()

	// info, _ := runtime.SerializerInfoForMediaType(codecFectory.SupportedMediaTypes(), runtime.ContentTypeYAML)
	// encoder := info.Serializer
	// serializer := codecFectory.EncoderForVersion(encoder, runtime.InternalGroupVersioner)

	yamlSerializer := k8sjson.NewSerializerWithOptions(
		k8sjson.DefaultMetaFactory, scheme, scheme,
		k8sjson.SerializerOptions{Pretty: true, Yaml: true, Strict: true},
	)
	jsonSerializer := k8sjson.NewSerializerWithOptions(
		k8sjson.DefaultMetaFactory, scheme, scheme,
		k8sjson.SerializerOptions{Pretty: true, Yaml: false, Strict: true},
	)

	return &app{
		Scope: newScope("$$root", ScopeProps{}, &globalContext{
			scheme:         scheme,
			codecFactory:   codecFectory,
			jsonSerializer: jsonSerializer,
			yamlSerializer: yamlSerializer,
		}),
		props: props,
	}
}

func (a *app) WriteYAMLsToDisk() error {
	fileNo := 0
	files := map[string][]ApiObject{}
	var prepareFiles func(scope *scope, currentChartID []string, level int) error
	prepareFiles = func(scope *scope, currentChartID []string, level int) error {
		if scope == nil {
			return nil
		}
		objects := []ApiObject{}
		chartCount := 0
		for _, object := range scope.objects {
			if a.props.PatchObject != nil {
				if err := a.props.PatchObject(object); err != nil {
					return fmt.Errorf("PatchObject: %w", err)
				}
			}
			objects = append(objects, object)
		}
		for _, childNode := range scope.children {
			// for i := 0; i < level; i++ {
			// 	fmt.Print("  ")
			// }
			// fmt.Println(node.id)
			thisChartID := currentChartID
			chartCount++
			thisChartID = append(thisChartID, fmt.Sprintf("%02d", chartCount), childNode.ID())
			// fmt.Println(strings.Join(currentChartID, "-"), reflect.TypeOf(childNode.value), thisChartID)
			if err := prepareFiles(childNode, thisChartID, level+1); err != nil {
				return fmt.Errorf("prepareFiles: %w", err)
			}
		}
		if len(objects) > 0 {
			currentChartID := strings.Join(currentChartID, "-")
			if _, ok := files[currentChartID]; !ok {
				fileNo++
			}
			files[currentChartID] = append(files[currentChartID], objects...)
		}
		return nil
	}
	if err := prepareFiles(a.Scope.(*scope), []string{}, 0); err != nil {
		return fmt.Errorf("prepareFiles: %w", err)
	}
	fileContents := map[string][]byte{}
	for _, currentChartID := range MapKeysSorted(files) {
		apiObjects := files[currentChartID]
		filePath := path.Join(a.props.Outdir, fmt.Sprintf("%s.yaml", currentChartID))
		// fmt.Println(filePath, len(apiObjects))
		for i, apiObject := range apiObjects {
			// fmt.Printf("  - %s/%s/%s\n", apiObject.GetAPIVersion(), apiObject.GetNamespace(), apiObject.GetName())
			if i != 0 {
				fileContents[filePath] = append(fileContents[filePath], []byte("---\n")...)
			}
			fileContents[filePath] = append(fileContents[filePath], apiObject.ToYAML()...)
		}
	}
	if a.props.DeleteOutDir {
		if err := os.RemoveAll(a.props.Outdir); err != nil {
			return fmt.Errorf("RemoveAll: %w", err)
		}
	}
	if err := os.MkdirAll(a.props.Outdir, 0755); err != nil {
		return fmt.Errorf("MkdirAll: %w", err)
	}
	for filePath, fileContent := range fileContents {
		if err := os.WriteFile(filePath, fileContent, 0644); err != nil {
			return fmt.Errorf("WriteFile: %w", err)
		}
	}
	return nil
}
