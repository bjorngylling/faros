package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatev1 "sigs.k8s.io/gateway-api/apis/v1"
)

type gwStatusUpdater struct {
	conditions []metav1.Condition
}

func (g *gwStatusUpdater) UpdateStatus(ctx context.Context, gwClass *gatev1.GatewayClass, _ metav1.UpdateOptions) (*gatev1.GatewayClass, error) {
	g.conditions = gwClass.Status.Conditions
	return nil, nil
}

func TestUpdateGatewayClassStatus(t *testing.T) {
	updater := &gwStatusUpdater{}
	before := []metav1.Condition{
		{
			Type:               string(gatev1.GatewayClassConditionStatusAccepted),
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Time{},
		},
		{
			Type:               string(gatev1.GatewayClassConditionStatusSupportedVersion),
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Time{},
		},
	}
	now := metav1.Now()
	UpdateGatewayClassStatus(updater, &gatev1.GatewayClass{Status: gatev1.GatewayClassStatus{Conditions: before}}, metav1.Condition{
		Type:               string(gatev1.GatewayClassConditionStatusAccepted),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
	})
	got := updater.conditions
	want := []metav1.Condition{
		{
			Type:               string(gatev1.GatewayClassConditionStatusSupportedVersion),
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Time{},
		},
		{
			Type:               string(gatev1.GatewayClassConditionStatusAccepted),
			Status:             metav1.ConditionTrue,
			LastTransitionTime: now,
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("UpdateGatewayClassStatus() mismatch (-want +got):\n%s", diff)
	}
}
