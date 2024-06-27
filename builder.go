package kgen

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/blesswinsamuel/kgen/internal"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type yamlOutputType string

// inspired by cdk8s (https://cdk8s.io/docs/latest/reference/cdk8s/python/#yamloutputtype)
const (
	// All resources are output into a single YAML file.
	YamlOutputTypeSingleFile yamlOutputType = "single"
	// Resources are split into seperate files by scope.
	YamlOutputTypeFilePerScope yamlOutputType = "scope"
	// Each resource is output to its own file.
	YamlOutputTypeFilePerResource yamlOutputType = "resource"
	// Each resource is output to its own file in a folder named after the scope.
	YamlOutputTypeFolderPerScopeFilePerResource yamlOutputType = "folder"
	// Each resource is output to its own file in a folder named after the parent scope.
	YamlOutputTypeFolderPerParentScopeFilePerLeafScope yamlOutputType = "folder-per-parent"
)

type RenderManifestsOptions struct {
	Outdir         string
	YamlOutputType yamlOutputType
	DeleteOutDir   bool
	PatchObject    func(ApiObject) error
}

type Builder interface {
	Scope
	RenderManifests(opts RenderManifestsOptions) error
}

type BuilderOptions struct {
	SchemeBuilder runtime.SchemeBuilder
	Logger        Logger
}

type builder struct {
	Scope
}

type globalContext struct {
	scheme *runtime.Scheme
	logger Logger
}

func NewBuilder(opts BuilderOptions) Builder {
	scheme := runtime.NewScheme()
	utilruntime.Must(opts.SchemeBuilder.AddToScheme(scheme))

	scope := newScope("__root__", ScopeProps{}, &globalContext{
		scheme: scheme,
		logger: opts.Logger,
	})
	return &builder{
		Scope: scope,
	}
}

func getObjectNameAndNamespace(apiObject ApiObject) string {
	obj := apiObject.GetObject().(*unstructured.Unstructured)
	namespace := obj.GetNamespace()
	name := obj.GetName()
	if namespace != "" {
		return fmt.Sprintf("%s-%s", namespace, name)
	}
	return name
}

func constructFilenameToApiObjectsMap(files map[string][]ApiObject, scope *scope, currentScopeID []string, level int, opts RenderManifestsOptions) {
	if scope == nil {
		return
	}
	if len(scope.objects) > 0 {
		switch opts.YamlOutputType {
		case YamlOutputTypeSingleFile:
			filePath := "all"
			files[filePath] = append(files[filePath], scope.objects...)
		case YamlOutputTypeFilePerResource:
			for i, apiObject := range scope.objects {
				filePath := fmt.Sprintf("%s-%02d-%s", strings.Join(currentScopeID, "-"), i+1, getObjectNameAndNamespace(apiObject))
				files[filePath] = append(files[filePath], apiObject)
			}
		case YamlOutputTypeFilePerScope:
			filePath := strings.Join(currentScopeID, "-")
			files[filePath] = append(files[filePath], scope.objects...)
		case YamlOutputTypeFolderPerScopeFilePerResource:
			filePath := path.Join(currentScopeID...)
			for i, apiObject := range scope.objects {
				filePath := path.Join(filePath, fmt.Sprintf("%02d-%s", i+1, getObjectNameAndNamespace(apiObject)))
				files[filePath] = append(files[filePath], apiObject)
			}
		case YamlOutputTypeFolderPerParentScopeFilePerLeafScope:
			filePath := path.Join(currentScopeID...)
			files[filePath] = append(files[filePath], scope.objects...)
		}
	}
	for i, childNode := range scope.children {
		thisScopeID := append(currentScopeID, fmt.Sprintf("%02d-%s", i+1, childNode.ID()))
		constructFilenameToApiObjectsMap(files, childNode, thisScopeID, level+1, opts)
	}
}

// RenderManifests writes the ApiObjects to disk or stdout in YAML format.
func (a *builder) RenderManifests(opts RenderManifestsOptions) error {
	if opts.PatchObject != nil {
		a.Scope.WalkApiObjects(opts.PatchObject)
	}
	if opts.YamlOutputType == "" {
		opts.YamlOutputType = YamlOutputTypeSingleFile
	}

	files := map[string][]ApiObject{} // map[filename]apiObjects
	constructFilenameToApiObjectsMap(files, a.Scope.(*scope), []string{}, 0, opts)

	fileContents := map[string][]byte{}
	for _, currentScopeID := range internal.MapKeysSorted(files) {
		apiObjects := files[currentScopeID]
		filePath := path.Join(opts.Outdir, fmt.Sprintf("%s.yaml", currentScopeID))
		for i, apiObject := range apiObjects {
			if i > 0 {
				fileContents[filePath] = append(fileContents[filePath], []byte("---\n")...)
			}
			fileContents[filePath] = append(fileContents[filePath], apiObject.ToYAML()...)
		}
	}
	if opts.Outdir == "-" || opts.Outdir == "" {
		for i, filePath := range internal.MapKeysSorted(fileContents) {
			fileContent := fileContents[filePath]
			if i > 0 {
				fmt.Println("---")
			}
			fmt.Println(string(fileContent))
		}
		return nil
	}
	if opts.DeleteOutDir {
		if err := os.RemoveAll(opts.Outdir); err != nil {
			return fmt.Errorf("RemoveAll: %w", err)
		}
	}
	if err := os.MkdirAll(opts.Outdir, 0755); err != nil {
		return fmt.Errorf("MkdirAll: %w", err)
	}
	for filePath, fileContent := range fileContents {
		if err := os.WriteFile(filePath, fileContent, 0644); err != nil {
			return fmt.Errorf("WriteFile: %w", err)
		}
	}
	return nil
}
