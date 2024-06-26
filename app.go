package kgen

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/blesswinsamuel/infra-base/infrahelpers"
)

type App interface {
	Scope
	OutDir() string
	WriteYAMLsToDisk()
}

type AppProps struct {
	Outdir       string
	CacheDir     string
	DeleteOutDir bool
	PatchNdots   bool
}

type app struct {
	Scope
	props AppProps
}

func NewApp(props AppProps) App {
	return &app{
		Scope: newScope("$$root", ScopeProps{}),
		props: props,
	}
}

func patchObject(apiObject ApiObject) {
	dnsConfig := &corev1.PodDNSConfig{
		Options: []corev1.PodDNSConfigOption{
			{Name: "ndots", Value: infrahelpers.Ptr("1")},
		},
	}
	switch apiObject.GetKind() {
	case "Deployment":
		modifyObj(apiObject, func(deployment *appsv1.Deployment) {
			deployment.Spec.Template.Spec.DNSConfig = dnsConfig
		})
	case "StatefulSet":
		modifyObj(apiObject, func(statefulset *appsv1.StatefulSet) {
			statefulset.Spec.Template.Spec.DNSConfig = dnsConfig
		})
	case "DaemonSet":
		modifyObj(apiObject, func(statefulset *appsv1.DaemonSet) {
			statefulset.Spec.Template.Spec.DNSConfig = dnsConfig
		})
	case "CronJob":
		modifyObj(apiObject, func(cronjob *batchv1.CronJob) {
			cronjob.Spec.JobTemplate.Spec.Template.Spec.DNSConfig = dnsConfig
		})
	}
}

func modifyObj[T any](apiObject ApiObject, f func(*T)) {
	var res T
	statefulsetUnstructured := apiObject.GetObject().(*unstructured.Unstructured)
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(statefulsetUnstructured.UnstructuredContent(), &res)
	if err != nil {
		log.Fatalf("FromUnstructured: %v", err)
	}
	f(&res)
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&res)
	if err != nil {
		log.Fatalf("ToUnstructured: %v", err)
	}
	apiObject.SetObject(unstructured.Unstructured{Object: unstructuredObj})
}

func (a *app) WriteYAMLsToDisk() {
	fileNo := 0
	files := map[string][]ApiObject{}
	var prepareFiles func(scope *scope, currentChartID []string, level int)
	prepareFiles = func(scope *scope, currentChartID []string, level int) {
		if scope == nil {
			return
		}
		objects := []ApiObject{}
		chartCount := 0
		for _, object := range scope.objects {
			if a.props.PatchNdots {
				patchObject(object)
			}
			objects = append(objects, object)
		}
		for _, childNode := range scope.children {
			// for i := 0; i < level; i++ {
			// 	fmt.Print("  ")
			// }
			// fmt.Println(node.id)
			thisChartID := currentChartID
			chartCount++
			thisChartID = append(thisChartID, fmt.Sprintf("%02d", chartCount), childNode.ID())
			// fmt.Println(strings.Join(currentChartID, "-"), reflect.TypeOf(childNode.value), thisChartID)
			prepareFiles(childNode, thisChartID, level+1)
		}
		if len(objects) > 0 {
			currentChartID := strings.Join(currentChartID, "-")
			if _, ok := files[currentChartID]; !ok {
				fileNo++
			}
			files[currentChartID] = append(files[currentChartID], objects...)
		}
	}
	prepareFiles(a.Scope.(*scope), []string{}, 0)
	fileContents := map[string][]byte{}
	for _, currentChartID := range infrahelpers.MapKeys(files) {
		apiObjects := files[currentChartID]
		filePath := path.Join(a.props.Outdir, fmt.Sprintf("%s.yaml", currentChartID))
		// fmt.Println(filePath, len(apiObjects))
		for i, apiObject := range apiObjects {
			// fmt.Printf("  - %s/%s/%s\n", apiObject.GetAPIVersion(), apiObject.GetNamespace(), apiObject.GetName())
			if i != 0 {
				fileContents[filePath] = append(fileContents[filePath], []byte("---\n")...)
			}
			fileContents[filePath] = append(fileContents[filePath], apiObject.ToYAML()...)
		}
	}
	if a.props.DeleteOutDir {
		if err := os.RemoveAll(a.props.Outdir); err != nil {
			log.Fatalf("RemoveAll: %v", err)
		}
	}
	if err := os.MkdirAll(a.props.Outdir, 0755); err != nil {
		log.Fatalf("MkdirAll: %v", err)
	}
	for filePath, fileContent := range fileContents {
		if err := os.WriteFile(filePath, fileContent, 0644); err != nil {
			log.Fatalf("WriteFile: %v", err)
		}
	}
}

func (a *app) OutDir() string {
	return a.props.Outdir
}
