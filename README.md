# kgen

kgen is a Go library to generate Kubernetes manifests using Go code.

## Why?

I used Helm for a while to generate Kubernetes manifests for my homelab. Helm is a great tool, but maintaining the complex helm templates became a pain, especially without the IDE's autocompletion and type checking.

I tried cdk8s, which is a great tool that provides autocompletion and type checking. It was a huge step up over helm, but it requires code generation for custom resources, and the huge number of files it generated caused my IDE to slow down, and I found it slow to run as it uses JavaScript under the hood to generate the manifests.

Since all if not most of the operators that have CRDs are written in Go (feel free to raise an issue if I'm wrong), and the Go structs are already available, I thought it would be great to use Go to generate the manifests. That's why I created kgen to generate Kubernetes manifests using Go code.

## Features

- Write Kubernetes manifests in Go. Allows you to create your own abstractions for Kubernetes resources using the full power of the Go programming language.
- Because it's Go, you can use the Go structs from the Kubernetes client-go library directly. IDEs can provide autocompletion and type checking.
- Supports custom resources without any code generation. Just import and use the Go structs from the 3rd party repositories. Check [this example](./examples/complex/main.go).
- Allows adding existing helm charts. Check [this example](./examples/complex/main.go).
- Relatively faster than cdk8s in my experience, especially when templating a large number of resources, thanks to Go's fast compilation.
- Has the potential to replace Helm for distributing kubernetes resources. Instead of helm templates, applications would need to have a go package that adds the kubernetes resources based on props. See [here](examples/like-a-helm-chart/main.go) for an example.

The tradeoff - helm doesn't need a Go compiler to run, but kgen requires Go to be installed.

## Examples

Check the [examples](./examples) directory for examples.

## Documentation

Check [godoc](https://pkg.go.dev/github.com/blesswinsamuel/kgen) for the documentation.

## Basic Usage

kgen is a Go library. Use it in your Go code to generate Kubernetes manifests.

```go
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
	//    You can also customize the output format (one YAML file per scope, one YAML file per resource, etc). See kgen.RenderManifestsOptions in godoc.
	builder.RenderManifests(kgen.RenderManifestsOptions{
		Outdir:       "k8s-rendered",
		DeleteOutDir: true,
	})
}
```

Run `go run .` to generate the k8s manifests in the `k8s-rendered` directory. See [here](examples/readme-example/k8s-rendered) for the generated manifests.

## Bonus tips

- It's a good idea to commit the rendered manifests in Git, ideally in the CI. [Here](https://akuity.io/blog/the-rendered-manifests-pattern/) is a great read on this topic.
- Use [kapp](https://get-kapp.io/) to apply the manifests on the cluster. It's a great tool that can also show the diff against the cluster state, and also faster than helm in my experience.
  Here is a command I use: `kapp deploy -a homelab -f k8s-rendered --diff-changes --diff-mask=false --diff-context=2 --apply-default-update-strategy=fallback-on-replace --color`
- Build your own abstractions for Kubernetes resources using the full power of the Go programming language. See [here](https://github.com/blesswinsamuel/infra-base/blob/main/k8sapp/application.go) for a complex (ugly) example.

## FAQ

**Why name it kgen?**

I use kapp for applying the manifests, so I thought kgen would be a good name for the tool that generates the manifests.

**Why panic everywhere?**

Although this is meant to be used as a library, the program that uses this library is meant to be run as a CLI tool in a CI/CD pipeline or locally.
If there is an error, we want the pipeline to fail loudly.
If you have an use case where you don't want to panic, feel free to open an issue and I'll reconsider it.

## Contributing

Feel free to open an issue or a PR for any feature requests, bug reports, or improvements.
