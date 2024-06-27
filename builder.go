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
	// Resources are split into seperate files by scope, while creating a folder for each scope.
	YamlOutputTypeFolderPerScopeFilePerLeafScope yamlOutputType = "folder-per-parent"
)

type RenderManifestsOptions struct {
	// The directory to write the YAML files to. If set to "-", the YAML files will be written to stdout.
	Outdir string
	// The output format for the YAML files.
	YamlOutputType yamlOutputType
	// Include a number in the filenames to maintain order.
	IncludeNumberInFilenames bool
	// Delete the output directory before writing the YAML files.
	DeleteOutDir bool
	// PatchObject is a function that can be used to modify the ApiObjects before they are rendered.
	PatchObject func(ApiObject) error
}

// Builder is the main interface for adding Kubernetes API objects and rendering them to YAML files.
type Builder interface {
	Scope
	// RenderManifests writes the Kubernetes API objects to disk or stdout in YAML format.
	RenderManifests(opts RenderManifestsOptions)
}

type BuilderOptions struct {
	// SchemeBuilder is used to add custom Kubernetes API types to the scheme.
	SchemeBuilder runtime.SchemeBuilder
	// Logger is used to log messages. If not set, a default logger is used.
	Logger Logger
}

type builder struct {
	Scope
}

type globalContext struct {
	scheme *runtime.Scheme
	logger Logger
}

// NewBuilder creates a new Builder instance.
func NewBuilder(opts BuilderOptions) Builder {
	if opts.SchemeBuilder == nil {
		panic("SchemeBuilder is required")
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(opts.SchemeBuilder.AddToScheme(scheme))

	if opts.Logger == nil {
		opts.Logger = NewCustomLogger(nil)
	}
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
	out := []string{}
	for _, part := range []string{obj.GetNamespace(), strings.ToLower(obj.GetKind()), obj.GetName()} {
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, "-")
}

func constructFilenameToApiObjectsMap(files map[string][]ApiObject, scope *scope, currentScopeID []string, level int, opts RenderManifestsOptions) {
	if scope == nil {
		return
	}
	sprintfWithNumber := func(n int, s string) string {
		if opts.IncludeNumberInFilenames {
			return fmt.Sprintf("%02d-%s", n, s)
		}
		return s
	}
	if len(scope.objects) > 0 {
		switch opts.YamlOutputType {
		case YamlOutputTypeSingleFile:
			filePath := "all"
			files[filePath] = append(files[filePath], scope.objects...)
		case YamlOutputTypeFilePerResource:
			for i, apiObject := range scope.objects {
				filePath := strings.Join([]string{strings.Join(currentScopeID, "-"), sprintfWithNumber(i+1, getObjectNameAndNamespace(apiObject))}, "-")
				files[filePath] = append(files[filePath], apiObject)
			}
		case YamlOutputTypeFilePerScope:
			filePath := strings.Join(currentScopeID, "-")
			files[filePath] = append(files[filePath], scope.objects...)
		case YamlOutputTypeFolderPerScopeFilePerResource:
			filePath := path.Join(currentScopeID...)
			if len(scope.children) > 0 {
				filePath = path.Join(filePath, sprintfWithNumber(0, scope.ID()))
			}
			for i, apiObject := range scope.objects {
				filePath := path.Join(filePath, sprintfWithNumber(i+1, getObjectNameAndNamespace(apiObject)))
				files[filePath] = append(files[filePath], apiObject)
			}
		case YamlOutputTypeFolderPerScopeFilePerLeafScope:
			filePath := path.Join(path.Join(currentScopeID...))
			if len(scope.children) > 0 {
				filePath = path.Join(filePath, sprintfWithNumber(0, scope.ID()))
			}
			files[filePath] = append(files[filePath], scope.objects...)
		}
	}
	for i, childScope := range scope.children {
		thisScopeID := append(currentScopeID, sprintfWithNumber(i+1, childScope.ID()))
		constructFilenameToApiObjectsMap(files, childScope, thisScopeID, level+1, opts)
	}
}

func (a *builder) RenderManifests(opts RenderManifestsOptions) {
	if opts.PatchObject != nil {
		if err := a.Scope.WalkApiObjects(opts.PatchObject); err != nil {
			a.Logger().Panicf("PatchObject: %v", err)
		}
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
		return
	}
	if opts.DeleteOutDir {
		if err := os.RemoveAll(opts.Outdir); err != nil {
			a.Logger().Panicf("RemoveAll: %v", err)
		}
	}
	for filePath, fileContent := range fileContents {
		parentDir := path.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			a.Logger().Panicf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filePath, fileContent, 0644); err != nil {
			a.Logger().Panicf("WriteFile: %v", err)
		}
	}
}
