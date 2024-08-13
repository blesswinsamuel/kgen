package kgen

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type Scope interface {
	// ID returns the identifier of the scope.
	ID() string
	// Namespace returns the namespace of the scope. It searches the current scope and its parents.
	Namespace() string
	// CreateScope creates a new scope, nested under the current scope.
	CreateScope(id string, props ScopeProps) Scope
	// GetContext returns the value of the given context key. It searches the current scope and its parents.
	GetContext(key string) any
	// SetContext sets the value of the given context key.
	SetContext(key string, value any)
	// AddApiObject adds a new API object to the scope.
	AddApiObject(obj runtime.Object) ApiObject
	// AddApiObjectFromMap adds a new API object to the scope from an arbitrary map.
	AddApiObjectFromMap(props map[string]any) ApiObject
	// WalkApiObjects walks through all the API objects in the scope and its children.
	WalkApiObjects(walkFn func(ApiObject) error) error
	// Children returns the child scopes of the current scope.
	Children() []Scope
	// Logger returns the logger that was passed to the builder.
	Logger() Logger
}

// ScopeProps is the properties for creating a new scope.
type ScopeProps struct {
	// Namespace is the default kubernetes namespace that should be used for the k8s resources in the scope.
	Namespace string
}

type scope struct {
	id            string
	globalContext *globalContext
	context       map[string]any
	parent        *scope
	children      []*scope
	objects       []ApiObject
}

func newScope(id string, props ScopeProps, globalContext *globalContext) Scope {
	scope := &scope{
		id:            id,
		context:       map[string]any{},
		globalContext: globalContext,
	}
	if props.Namespace != "" {
		scope.context[namespaceContextKey] = props.Namespace
	}
	return scope
}

func (s *scope) SetContext(key string, value any) {
	s.context[key] = value
}

func (s *scope) GetContext(key string) any {
	for s := s; s != nil; s = s.parent {
		if ctx, ok := s.context[key]; ok {
			return ctx
		}
	}
	return s.context[key]
}

func (s *scope) ID() string {
	return s.id
}

func (s *scope) CreateScope(id string, props ScopeProps) Scope {
	childScope := newScope(id, props, s.globalContext).(*scope)
	childScope.parent = s
	s.children = append(s.children, childScope)
	return childScope
}

func (s *scope) Namespace() string {
	return s.GetContext(namespaceContextKey).(string)
}

func (s *scope) addApiObject(obj runtime.Object) (ApiObject, error) {
	groupVersionKinds, _, err := s.globalContext.scheme.ObjectKinds(obj)
	if err != nil {
		return nil, fmt.Errorf("ObjectKinds: %w", err)
	}
	if len(groupVersionKinds) != 1 {
		return nil, fmt.Errorf("expected 1 groupVersionKind, got %d: %v", len(groupVersionKinds), groupVersionKinds)
	}
	groupVersion := groupVersionKinds[0]
	mobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("k8sObjectToMap: %w", err)
	}
	mobj["apiVersion"] = groupVersion.GroupVersion().String()
	mobj["kind"] = groupVersion.Kind
	return s.AddApiObjectFromMap(mobj), nil
}

func (s *scope) AddApiObject(obj runtime.Object) ApiObject {
	if apiObject, err := s.addApiObject(obj); err != nil {
		s.Logger().Panicf("failed to add api object: %v", err)
		return nil
	} else {
		return apiObject
	}
}

func (s *scope) AddApiObjectFromMap(obj map[string]any) ApiObject {
	props := apiObjectProps{Unstructured: &unstructured.Unstructured{Object: obj}}
	if props.GetNamespace() == "" {
		namespaceCtx, _ := s.GetContext(namespaceContextKey).(string)
		if namespaceCtx != "" {
			props.SetNamespace(namespaceCtx)
		}
	}

	apiObject := &apiObject{apiObjectProps: props, globalContext: s.globalContext}

	s.objects = append(s.objects, apiObject)
	return apiObject
}

func (s *scope) WalkApiObjects(walkFn func(ApiObject) error) error {
	for _, object := range s.objects {
		if err := walkFn(object); err != nil {
			return fmt.Errorf("walk objects: %w", err)
		}
	}
	for _, childNode := range s.children {
		if err := childNode.WalkApiObjects(walkFn); err != nil {
			return fmt.Errorf("walk children: %w", err)
		}
	}
	return nil
}

func (s *scope) Children() []Scope {
	children := make([]Scope, 0, len(s.children))
	for _, childNode := range s.children {
		children = append(children, childNode)
	}
	return children
}

func (s *scope) Logger() Logger {
	return s.globalContext.logger
}
