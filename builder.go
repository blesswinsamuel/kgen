package kgen

import (
	"fmt"
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type WriteOpts struct {
	Outdir       string
	DeleteOutDir bool
	PatchObject  func(ApiObject) error
}

type Builder interface {
	Scope
	WriteYAMLsToDisk(opts WriteOpts) error
}

type BuilderOptions struct {
	SchemeBuilder runtime.SchemeBuilder
}

type builder struct {
	Scope
	opts BuilderOptions
}

func NewBuilder(opts BuilderOptions) Builder {
	scheme := runtime.NewScheme()
	utilruntime.Must(opts.SchemeBuilder.AddToScheme(scheme))

	return &builder{
		Scope: newScope("__root__", ScopeProps{}, &globalContext{
			scheme: scheme,
		}),
		opts: opts,
	}
}

func constructFilenameToApiObjectsMap(files map[string][]ApiObject, scope *scope, currentScopeID []string, level int) error {
	if scope == nil {
		return nil
	}
	if len(scope.objects) > 0 {
		currentScopeID := strings.Join(currentScopeID, "-")
		files[currentScopeID] = append(files[currentScopeID], scope.objects...)
	}
	for i, childNode := range scope.children {
		thisScopeID := append(currentScopeID, fmt.Sprintf("%02d", i+1), childNode.ID())
		if err := constructFilenameToApiObjectsMap(files, childNode, thisScopeID, level+1); err != nil {
			return fmt.Errorf("constructFilenameToApiObjectsMap: %w", err)
		}
	}
	return nil
}

func (a *builder) WriteYAMLsToDisk(opts WriteOpts) error {
	a.Scope.(*scope).patchObjects(opts.PatchObject)
	files := map[string][]ApiObject{} // map[filename]apiObjects
	if err := constructFilenameToApiObjectsMap(files, a.Scope.(*scope), []string{}, 0); err != nil {
		return fmt.Errorf("constructFilenameToApiObjectsMap: %w", err)
	}
	fileContents := map[string][]byte{}
	for _, currentScopeID := range MapKeysSorted(files) {
		apiObjects := files[currentScopeID]
		filePath := path.Join(opts.Outdir, fmt.Sprintf("%s.yaml", currentScopeID))
		for i, apiObject := range apiObjects {
			if i > 0 {
				fileContents[filePath] = append(fileContents[filePath], []byte("---\n")...)
			}
			fileContents[filePath] = append(fileContents[filePath], apiObject.ToYAML()...)
		}
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
