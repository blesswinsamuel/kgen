package main

import (
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"

	"github.com/blesswinsamuel/kgen"
)

func main() {
	// 1. Create a schemeBuilder with all the custom resources you plan to generate.
	// It should also have "k8s.io/client-go/kubernetes/scheme".AddToScheme to include the core k8s resources.
	schemeBuilder := runtime.SchemeBuilder{
		scheme.AddToScheme,
		certmanagerv1.AddToScheme, // if you want to generate cert-manager resources
	}
	// 2. Create a builder instance passing `schemeBuilder` to start adding resources. You can also pass a custom logger here.
	builder := kgen.NewBuilder(kgen.BuilderOptions{
		SchemeBuilder: schemeBuilder,
	})
	// 3. Create a scope to organize resources. kgen can be configured to output the k8s resources added to each scope to separate files. See kgen.RenderManifestsOptions.
	// You can also set the kubernetes namespace for the scope.
	// If you don't set the namespace, the resources will have the default namespace, unless you set it in the resource objects themselves.
	whoamiScope := builder.CreateScope("whoami", kgen.ScopeProps{Namespace: "whoami"})
	// 4. Add resources to the scope. You can add any k8s resource object that implements the runtime.Object interface.
	whoamiScope.AddApiObject(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "whoami-deployment"},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "whoami"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "whoami"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "whoami-container",
							Image: "containous/whoami",
						},
					},
				},
			},
		},
	})
	// Notes:
	// - You can add more resources to the scope using whoamiScope.AddApiObject(...)
	// - You can create more scopes (can be nested as well) and add resources to them.
	// - You can also add resources to the builder directly without a scope using builder.AddApiObject(...).
	//   But the filenames might be empty in the output depending on the output format.
	//   So it's recommended to use scopes.
	// - You can also add existing helm charts to be rendered along with the resources. Check the complex example under examples directory for an example.

	// 5. Render the resources to k8s yaml files.
	builder.RenderManifests(kgen.RenderManifestsOptions{
		Outdir:       "k8s-rendered",
		DeleteOutDir: true,
	})
}
