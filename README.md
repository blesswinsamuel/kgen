# kgen

> write helm templates using Go

kgen is a Go library to generate Kubernetes manifests using Go code.

## Why?

Kubernetes resources are defined in YAML files. This is a good thing, because it allows us to use the same tooling to manage them as we use to manage our application code. However, it also means that we lose the ability to use the type system to validate our resources.

Kube GoGen allows us to define our Kubernetes resources in Go code, and then generate the YAML files from that code. This means that we can use the type system to validate our resources, and we can use our IDE to provide autocompletion and documentation.

## Documentation

Coming soon. This package will be moved out into a separate repository. In the meantime, see how this is being used in the k8sbase package.
