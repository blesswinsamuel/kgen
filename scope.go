package kgen

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type Scope interface {
	ID() string
	Namespace() string
	CreateScope(id string, props ScopeProps) Scope
	GetContext(key string) any
	SetContext(key string, value any)
	AddApiObject(obj runtime.Object) ApiObject
	AddApiObjectFromMap(props map[string]any) ApiObject
	WalkApiObjects(walkFn func(ApiObject) error) error
	Logger() Logger
}

type ScopeProps struct {
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
	props := ApiObjectProps{Unstructured: &unstructured.Unstructured{Object: obj}}
	if props.GetNamespace() == "" {
		namespaceCtx, _ := s.GetContext(namespaceContextKey).(string)
		if namespaceCtx != "" {
			props.SetNamespace(namespaceCtx)
		}
	}

	apiObject := &apiObject{ApiObjectProps: props, globalContext: s.globalContext}

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

func (s *scope) Logger() Logger {
	return s.globalContext.logger
}
