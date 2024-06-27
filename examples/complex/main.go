package main

import (
	"github.com/blesswinsamuel/kgen"
	whoami "github.com/blesswinsamuel/kgen/example/like-a-helm-chart"
	"github.com/blesswinsamuel/kgen/kaddons"
	certmanageracmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

// Credit: example partially translated from https://github.com/geraldcroes/kubernetes-traefik-article/tree/master

func main() {
	schemeBuilder := runtime.SchemeBuilder{
		scheme.AddToScheme,
		certmanagerv1.AddToScheme,
	}
	builder := kgen.NewBuilder(kgen.BuilderOptions{
		SchemeBuilder: schemeBuilder,
	})

	whoScope := builder.CreateScope("who", kgen.ScopeProps{Namespace: "who"})
	whoami.AddWhoAmi(whoScope, whoami.WhoAmiProps{Replicas: 1, EnableIngress: true})
	AddWhoAreYou(whoScope)

	certManagerScope := builder.CreateScope("certmanager", kgen.ScopeProps{Namespace: "cert-manager"})
	AddCertManager(certManagerScope, CertManagerProps{Version: "v1.14.5"})

	myCertScope := builder.CreateScope("my-cert", kgen.ScopeProps{Namespace: "my-cert"})
	AddMyCert(myCertScope)

	builder.RenderManifests(kgen.RenderManifestsOptions{
		Outdir:                   "k8s-rendered",
		YamlOutputType:           kgen.YamlOutputTypeFilePerScope,
		DeleteOutDir:             true,
		IncludeNumberInFilenames: true,
	})
}

func AddWhoAreYou(scope kgen.Scope) {
	scope.AddApiObject(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "whoareyou-deployment"},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](2),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "whoareyou"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "whoareyou"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "whoareyou-container",
							Image: "containous/whoami",
						},
					},
				},
			},
		},
	})
	scope.AddApiObject(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "whoareyou-service"},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
			Selector: map[string]string{"app": "whoareyou"},
		},
	})
	scope.AddApiObject(&extensionsv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "whoareyou-ingress",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "traefik",
			},
		},
		Spec: extensionsv1.IngressSpec{
			Rules: []extensionsv1.IngressRule{
				{
					Host: "whoareyou.localhost",
					IngressRuleValue: extensionsv1.IngressRuleValue{
						HTTP: &extensionsv1.HTTPIngressRuleValue{
							Paths: []extensionsv1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extensionsv1.IngressBackend{
										ServiceName: "whoareyou-service",
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
}

type CertManagerProps struct {
	Version string
}

func AddCertManager(scope kgen.Scope, props CertManagerProps) {
	kaddons.AddHelmChart(scope, kaddons.HelmChartProps{
		ChartInfo: kaddons.HelmChartInfo{
			Repo:    "https://charts.jetstack.io",
			Chart:   "cert-manager",
			Version: props.Version,
		},
		ReleaseName: "cert-manager",
		Values: map[string]interface{}{
			"installCRDs": "true",
		},
	})
	scope.AddApiObject(&certmanagerv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: "letsencrypt-prod"},
		Spec: certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				ACME: &certmanageracmev1.ACMEIssuer{
					Email:  "example@example.com",
					Server: "https://acme-v02.api.letsencrypt.org/directory",
					Solvers: []certmanageracmev1.ACMEChallengeSolver{
						{
							HTTP01: &certmanageracmev1.ACMEChallengeSolverHTTP01{
								Ingress: &certmanageracmev1.ACMEChallengeSolverHTTP01Ingress{
									Class: ptr.To("traefik"),
								},
							},
						},
					},
				},
			},
		},
	})
}

func AddMyCert(scope kgen.Scope) {
	scope.AddApiObject(&certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cert"},
		Spec: certmanagerv1.CertificateSpec{
			DNSNames:   []string{"whoami.localhost"},
			SecretName: "whoami-tls",
			IssuerRef: certmanagermetav1.ObjectReference{
				Name: "letsencrypt-prod",
				Kind: "ClusterIssuer",
			},
		},
	})
}
