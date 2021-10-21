package main

import (
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkingv1 "k8s.io/api/networking/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersv1 "k8s.io/client-go/listers/core/v1"
)

type serviceNamespaceLister struct{}

func (s serviceNamespaceLister) List(_ labels.Selector) (ret []*corev1.Service, err error) {
	panic("should not be called")
}

func (s serviceNamespaceLister) Get(_ string) (*corev1.Service, error) {
	return &corev1.Service{
		ObjectMeta: v1.ObjectMeta{Name: "svc"},
		Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "http", Port: 80}}},
	}, nil
}

type serviceLister struct{}

func (s serviceLister) List(_ labels.Selector) (ret []*corev1.Service, err error) {
	panic("should not be called")
}

func (s serviceLister) Services(_ string) listersv1.ServiceNamespaceLister {
	return serviceNamespaceLister{}
}

func TestIngressPayload_addBackend(t *testing.T) {
	p := IngressPayload{
		Ingress:      &networkingv1.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: ""}},
		ServicePorts: map[string]map[string]int{},
	}
	s := &serviceLister{}
	ib := networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "svc"}}
	p.addBackend(ib, s)

	portMappings, ok := p.ServicePorts["svc"]
	if !ok {
		t.Errorf("Service svc missing from IngressPayload.")
	}
	port, ok := portMappings["http"]
	if !ok {
		t.Errorf("Service svc: missing port named http")
	}
	if port != 80 {
		t.Errorf("Service svc: wrong value for port named http, got=%d, want=%d", port, 80)
	}
}
