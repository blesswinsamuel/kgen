package kgen

import (
	"github.com/blesswinsamuel/infra-base/infrahelpers"
	"github.com/rs/zerolog/log"
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
	id       string
	props    ScopeProps
	context  map[string]any
	parent   *scope
	children []*scope
	objects  []ApiObject
}

func newScope(id string, props ScopeProps) Scope {
	scope := &scope{
		id:      id,
		props:   props,
		context: map[string]any{},
	}
	if props.Namespace != "" {
		scope.context["namespace"] = props.Namespace
	}
	return scope
}

func (c *scope) SetContext(key string, value any) {
	c.context[key] = value
}

func (c *scope) GetContext(key string) any {
	for s := c; s != nil; s = s.parent {
		if ctx, ok := s.context[key]; ok {
			return ctx
		}
	}
	return c.context[key]
}

func (c *scope) ID() string {
	return c.id
}

func (c *scope) CreateScope(id string, props ScopeProps) Scope {
	childScope := newScope(id, props).(*scope)
	childScope.parent = c
	c.children = append(c.children, childScope)
	return childScope
}

func (c *scope) Namespace() string {
	return c.GetContext("namespace").(string)
}

func (c *scope) AddApiObject(obj runtime.Object) ApiObject {
	groupVersionKinds, _, err := infrahelpers.Scheme.ObjectKinds(obj)
	if err != nil {
		log.Panic().Err(err).Msg("ObjectKinds")
	}
	if len(groupVersionKinds) != 1 {
		log.Panic().Msgf("expected 1 groupVersionKind, got %d: %v", len(groupVersionKinds), groupVersionKinds)
	}
	groupVersion := groupVersionKinds[0]
	mobj := infrahelpers.K8sObjectToMap(obj)
	mobj["apiVersion"] = groupVersion.GroupVersion().String()
	mobj["kind"] = groupVersion.Kind
	return c.AddApiObjectFromMap(mobj)
}

func (c *scope) AddApiObjectFromMap(obj map[string]any) ApiObject {
	props := ApiObjectProps{Unstructured: unstructured.Unstructured{Object: obj}}
	if props.GetNamespace() == "" {
		if namespaceCtx := getNamespaceContext(c); namespaceCtx != "" {
			props.SetNamespace(namespaceCtx)
		}
	}

	apiObject := &apiObject{ApiObjectProps: props}

	// fmt.Println(apiObject.GetKind(), apiObject.GetAPIVersion())
	c.objects = append(c.objects, apiObject)
	return apiObject
}
