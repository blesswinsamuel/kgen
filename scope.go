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
}

type ScopeProps struct {
	Namespace string
}

type scope struct {
	id            string
	props         ScopeProps
	globalContext *globalContext
	context       map[string]any
	parent        *scope
	children      []*scope
	objects       []ApiObject
}

func newScope(id string, props ScopeProps, globalContext *globalContext) Scope {
	scope := &scope{
		id:            id,
		props:         props,
		context:       map[string]any{},
		globalContext: globalContext,
	}
	if props.Namespace != "" {
		scope.context["namespace"] = props.Namespace
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
	return s.GetContext("namespace").(string)
}

func (s *scope) AddApiObject(obj runtime.Object) ApiObject {
	groupVersionKinds, _, err := s.globalContext.scheme.ObjectKinds(obj)
	if err != nil {
		panic(fmt.Errorf("ObjectKinds: %w", err))
	}
	if len(groupVersionKinds) != 1 {
		panic(fmt.Errorf("expected 1 groupVersionKind, got %d: %v", len(groupVersionKinds), groupVersionKinds))
	}
	groupVersion := groupVersionKinds[0]
	mobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		panic(fmt.Errorf("k8sObjectToMap: %w", err))
	}
	mobj["apiVersion"] = groupVersion.GroupVersion().String()
	mobj["kind"] = groupVersion.Kind
	return s.AddApiObjectFromMap(mobj)
}

func (s *scope) AddApiObjectFromMap(obj map[string]any) ApiObject {
	props := ApiObjectProps{Unstructured: &unstructured.Unstructured{Object: obj}}
	if props.GetNamespace() == "" {
		namespaceCtx, _ := s.GetContext("namespace").(string)
		if namespaceCtx != "" {
			props.SetNamespace(namespaceCtx)
		}
	}

	apiObject := &apiObject{ApiObjectProps: props, globalContext: s.globalContext}

	s.objects = append(s.objects, apiObject)
	return apiObject
}

func (s *scope) patchObjects(patchFn func(ApiObject) error) error {
	for _, object := range s.objects {
		if err := patchFn(object); err != nil {
			return fmt.Errorf("PatchObject: %w", err)
		}
	}
	for _, childNode := range s.children {
		if err := childNode.patchObjects(patchFn); err != nil {
			return fmt.Errorf("patchObjects: %w", err)
		}
	}
	return nil
}
