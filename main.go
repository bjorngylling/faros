package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/cache"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Payload is a collection of Kubernetes data loaded by the watcher.
type Payload struct {
	Ingresses       []IngressPayload
	TLSCertificates map[string]*tls.Certificate
}

// IngressPayload is an ingress + its service ports.
type IngressPayload struct {
	Ingress      *v1.Ingress
	ServicePorts map[string]map[string]int
}

// Watcher watches for ingresses in the kubernetes cluster
type Watcher struct {
	client   kubernetes.Interface
	onChange func(*Payload)
}

func (w *Watcher) Run(ctx context.Context) error {
	factory := informers.NewSharedInformerFactory(w.client, time.Minute)
	ingressLister := factory.Networking().V1().Ingresses().Lister()

	onChange := func() {
		ingresses, err := ingressLister.Ingresses("faros").List(labels.Everything())
		if err != nil {
			log.Fatalf("error listing Ingress resources: %s", err)
		}

		payload := &Payload{}
		for _, ingress := range ingresses {
			payload.Ingresses = append(payload.Ingresses, IngressPayload{Ingress: ingress})
		}

		w.onChange(payload)
	}

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			onChange()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			onChange()
		},
		DeleteFunc: func(obj interface{}) {
			onChange()
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		informer := factory.Networking().V1().Ingresses().Informer()
		informer.AddEventHandler(handler)
		informer.Run(ctx.Done())
		wg.Done()
	}()
	wg.Wait()

	return nil
}

func main() {
	fmt.Println("faros - a k8s ingress-controller")

	config, err := rest.InClusterConfig()
	if err != nil {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatalf("failed to build k8s config: %s", err)
		}
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %s", err)
	}

	w := Watcher{client: client, onChange: func(payload *Payload) {
		fmt.Printf("payload: %+v\n", *payload)
	}}

	err = w.Run(context.Background())
	if err != nil {
		log.Fatalf("%s", err)
	}
}
