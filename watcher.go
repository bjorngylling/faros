package main

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/bep/debounce"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Payload is a collection of k8s data loaded by Watcher.
type Payload struct {
	Ingresses       []IngressPayload
	TLSCertificates map[string]*tls.Certificate
}

// IngressPayload is an Ingress and its service ports.
type IngressPayload struct {
	Ingress      *networkingv1.Ingress
	ServicePorts map[string]map[string]int
}

func (p *IngressPayload) addBackend(backend networkingv1.IngressBackend, serviceLister v1.ServiceLister) {
	svc, err := serviceLister.Services(p.Ingress.Namespace).Get(backend.Service.Name)
	if err != nil {
		log.Printf("unknown service %s from ingress %s in namespace %s", backend.Service.Name, p.Ingress.Name, p.Ingress.Namespace)
	} else {
		m := make(map[string]int)
		for _, port := range svc.Spec.Ports {
			m[port.Name] = int(port.Port)
		}
		p.ServicePorts[svc.Name] = m
	}
}

// Watcher watches for changes to ingresses, services and secrets in the k8s cluster
type Watcher struct {
	client   kubernetes.Interface
	onChange func(*Payload)
}

func (w *Watcher) Run(ctx context.Context) error {
	factory := informers.NewSharedInformerFactory(w.client, time.Minute)
	ingressLister := factory.Networking().V1().Ingresses().Lister()
	secretLister := factory.Core().V1().Secrets().Lister()
	serviceLister := factory.Core().V1().Services().Lister()

	onChange := func() {
		ingresses, err := ingressLister.Ingresses("faros").List(labels.Everything())
		if err != nil {
			log.Fatalf("error listing ingress resources: %s", err)
		}

		payload := &Payload{TLSCertificates: map[string]*tls.Certificate{}}
		for _, ingress := range ingresses {
			for _, rec := range ingress.Spec.TLS {
				if rec.SecretName != "" {
					secret, err := secretLister.Secrets(ingress.Namespace).Get(rec.SecretName)
					if err != nil {
						log.Printf("unknown secret %s in namespace %s: %s", rec.SecretName, ingress.Namespace, err)
						continue
					}
					cert, err := tls.X509KeyPair(secret.Data["tls.crt"], secret.Data["tls.key"])
					if err != nil {
						log.Printf("unable to parse secret %s in namespace %s: %s", rec.SecretName, ingress.Namespace, err)
						continue
					}
					payload.TLSCertificates[rec.SecretName] = &cert
				}
			}
			ingressPayload := IngressPayload{Ingress: ingress, ServicePorts: make(map[string]map[string]int)}
			if ingress.Spec.DefaultBackend != nil {
				ingressPayload.addBackend(*ingress.Spec.DefaultBackend, serviceLister)
			}
			for _, rule := range ingress.Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					ingressPayload.addBackend(path.Backend, serviceLister)
				}
			}
			payload.Ingresses = append(payload.Ingresses, ingressPayload)
		}

		w.onChange(payload)
	}

	d := debounce.New(time.Second)
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			d(onChange)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			d(onChange)
		},
		DeleteFunc: func(obj interface{}) {
			d(onChange)
		},
	}

	go func() {
		informer := factory.Networking().V1().Ingresses().Informer()
		informer.AddEventHandler(handler)
		informer.Run(ctx.Done())
	}()

	go func() {
		informer := factory.Core().V1().Services().Informer()
		informer.AddEventHandler(handler)
		informer.Run(ctx.Done())
	}()

	go func() {
		informer := factory.Core().V1().Secrets().Informer()
		informer.AddEventHandler(handler)
		informer.Run(ctx.Done())
	}()

	return nil
}
