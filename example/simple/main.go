package main

import (
	"github.com/blesswinsamuel/kgen"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

// Credit: example translated from https://github.com/geraldcroes/kubernetes-traefik-article/tree/master

func main() {
	schemeBuilder := runtime.SchemeBuilder{
		scheme.AddToScheme,
	}
	builder := kgen.NewBuilder(kgen.BuilderOptions{
		SchemeBuilder: schemeBuilder,
	})
	builder.AddApiObject(&appsv1.Deployment{
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
	builder.AddApiObject(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "whoami-service"},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
			Selector: map[string]string{"app": "whoami"},
		},
	})
	builder.AddApiObject(&extensionsv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "whoami-ingress",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "traefik",
			},
		},
		Spec: extensionsv1.IngressSpec{
			Rules: []extensionsv1.IngressRule{
				{
					Host: "whoami.localhost",
					IngressRuleValue: extensionsv1.IngressRuleValue{
						HTTP: &extensionsv1.HTTPIngressRuleValue{
							Paths: []extensionsv1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extensionsv1.IngressBackend{
										ServiceName: "whoami-service",
										ServicePort: intstr.FromString("http"),
									},
								},
							},
						},
					},
				},
			},
		},
	})
	builder.RenderManifests(kgen.RenderManifestsOptions{
		Outdir:       "k8s-generated",
		DeleteOutDir: true,
	})
}
