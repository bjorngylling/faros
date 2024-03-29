package main

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/bep/debounce"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	gatev1 "sigs.k8s.io/gateway-api/apis/v1"
	gateclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gateinformer "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
)

const resyncPeriod = 10 * time.Minute
const controllerName = "github.com/bjorngylling/faros"

// Watcher reacts resource changes in the k8s cluster
type Watcher struct {
	client        kubernetes.Interface
	gatewayClient gateclient.Interface
	log           slog.Logger
}

func (w *Watcher) Run(ctx context.Context, routeAdder func(*gatev1.HTTPRoute)) error {
	factory := informers.NewSharedInformerFactory(w.client, resyncPeriod)
	_ = factory.Core().V1().Services().Lister()

	gateFactory := gateinformer.NewSharedInformerFactory(w.gatewayClient, resyncPeriod)
	gatewayClassLister := gateFactory.Gateway().V1().GatewayClasses().Lister()
	gatewayLister := gateFactory.Gateway().V1().Gateways().Lister()
	httpRouteLister := gateFactory.Gateway().V1().HTTPRoutes().Lister()

	onChange := func() {
		gwClasses, err := gatewayClassLister.List(labels.Everything())
		if err != nil {
			w.log.Error(err.Error())
		}
		gateways, err := gatewayLister.List(labels.Everything())
		if err != nil {
			w.log.Error(err.Error())
		}
		httpRoutes, err := httpRouteLister.List(labels.Everything())
		if err != nil {
			w.log.Error(err.Error())
		}
		for _, gwClass := range gwClasses {
			if gwClass.Spec.ControllerName == controllerName {
				for _, gw := range gateways {
					if gw.Spec.GatewayClassName != gatev1.ObjectName(gwClass.Name) {
						continue
					}
					gwMatcher := createGatewayMatcher(gw)
					routeCount := 0
					for _, route := range httpRoutes {
						if slices.ContainsFunc(route.Spec.ParentRefs, gwMatcher) {
							routeAdder(route)
							routeCount++
						}
					}
					w.log.Info("handled gateway",
						slog.String("gateway_class", gwClass.Name),
						slog.String("name", gw.Name),
						slog.String("namespace", gw.Namespace),
						slog.Int("route_count", routeCount))
				}
				UpdateGatewayClassStatus(w.gatewayClient.GatewayV1().GatewayClasses(), gwClass,
					metav1.Condition{
						Type:               string(gatev1.GatewayClassConditionStatusAccepted),
						Status:             metav1.ConditionTrue,
						Reason:             "Handled",
						Message:            "Handled by Faros",
						LastTransitionTime: metav1.Now(),
					})
			}
		}
	}

	d := debounce.New(time.Second)
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(_ interface{}) {
			d(onChange)
		},
		UpdateFunc: func(_, _ interface{}) {
			d(onChange)
		},
		DeleteFunc: func(_ interface{}) {
			d(onChange)
		},
	}

	factory.Core().V1().Services().Informer().AddEventHandler(handler)
	factory.Start(ctx.Done())

	gateFactory.Gateway().V1().GatewayClasses().Informer().AddEventHandler(handler)
	gateFactory.Gateway().V1().Gateways().Informer().AddEventHandler(handler)
	gateFactory.Gateway().V1().HTTPRoutes().Informer().AddEventHandler(handler)
	gateFactory.Start(ctx.Done())

	return nil
}

func createGatewayMatcher(gateway *gatev1.Gateway) func(gatev1.ParentReference) bool {
	return func(pr gatev1.ParentReference) bool {
		return pr.Name == gatev1.ObjectName(gateway.Name)
	}
}

type GatewayClassStatusUpdater interface {
	UpdateStatus(context.Context, *gatev1.GatewayClass, metav1.UpdateOptions) (*gatev1.GatewayClass, error)
}

func UpdateGatewayClassStatus(cl GatewayClassStatusUpdater, gwClass *gatev1.GatewayClass, condition metav1.Condition) error {
	gc := gwClass.DeepCopy()
	var newConditions []metav1.Condition
	for _, cond := range gc.Status.Conditions {
		if cond.Type == condition.Type && cond.Status == condition.Status {
			return nil
		}
		if cond.Type != condition.Type {
			newConditions = append(newConditions, cond)
		}
	}
	gc.Status.Conditions = append(newConditions, condition)
	_, err := cl.UpdateStatus(context.Background(), gc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
