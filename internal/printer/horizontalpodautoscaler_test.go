/*
Copyright (c) 2019 VMware, Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package printer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/vmware/octant/internal/testutil"
	"github.com/vmware/octant/pkg/view/component"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_HorizontalPodAutoscalerListHandler(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tpo := newTestPrinterOptions(controller)
	printOptions := tpo.ToOptions()

	objectLabels := map[string]string{
		"foo": "bar",
	}

	now := testutil.Time()
	var minReplicas int32 = 1

	object := &autoscalingv1.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v1",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "horizontalpodautoscaler",
			Namespace: "default",
			CreationTimestamp: metav1.Time{
				Time: now,
			},
			Labels: objectLabels,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			MinReplicas: &minReplicas,
			MaxReplicas: 10,
		},
		Status: autoscalingv1.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 2,
		},
	}

	tpo.PathForObject(object, object.Name, "/path")

	list := &autoscalingv1.HorizontalPodAutoscalerList{
		Items: []autoscalingv1.HorizontalPodAutoscaler{*object},
	}

	ctx := context.Background()
	got, err := HorizontalPodAutoscalerListHandler(ctx, list, printOptions)
	require.NoError(t, err)

	cols := component.NewTableCols("Name", "Labels", "Targets", "Minimum Pods", "Maximum Pods", "Replicas", "Age")
	expected := component.NewTable("Horizontal Pod Autoscalers", "We couldn't find any horizontal pod autoscalers", cols)
	expected.Add(component.TableRow{
		"Name":         component.NewLink("", "horizontalpodautoscaler", "/path"),
		"Labels":       component.NewLabels(objectLabels),
		"Targets":      component.NewText("placeholder"),
		"Minimum Pods": component.NewText("1"),
		"Maximum Pods": component.NewText("10"),
		"Replicas":     component.NewText("2"),
		"Age":          component.NewTimestamp(now),
	})

	component.AssertEqual(t, expected, got)
}

func Test_HorizontalPodAutoscalerConfiguration(t *testing.T) {
	var replicas int32 = 1
	hpa := testutil.CreateHorizontalPodAutoscaler("hpa")
	hpa.Spec.MinReplicas = &replicas
	hpa.Spec.MaxReplicas = 10

	deployment := testutil.CreateDeployment("deployment")
	hpa.Spec.ScaleTargetRef = autoscalingv1.CrossVersionObjectReference{
		Kind:       deployment.Kind,
		APIVersion: deployment.APIVersion,
		Name:       deployment.Name,
	}

	cases := []struct {
		name                    string
		horizontalPodAutoscaler *autoscalingv1.HorizontalPodAutoscaler
		expected                component.Component
		isErr                   bool
	}{
		{
			name:                    "general",
			horizontalPodAutoscaler: hpa,
			expected: component.NewSummary("Configuration", []component.SummarySection{
				{
					Header:  "Reference",
					Content: component.NewLink("", "deployment", "/deployment"),
				},
				{
					Header:  "Min Replicas",
					Content: component.NewText("1"),
				},
				{
					Header:  "Max Replicas",
					Content: component.NewText("10"),
				},
			}...),
		},
		{
			name:                    "nil horizontalpodautoscaler",
			horizontalPodAutoscaler: nil,
			isErr:                   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			tpo := newTestPrinterOptions(controller)
			printOptions := tpo.ToOptions()

			hc := NewHorizontalPodAutoscalerConfiguration(tc.horizontalPodAutoscaler)

			scaleTarget := component.NewLink("", "deployment", "/deployment")
			tpo.link.EXPECT().
				ForGVK("namespace", "apps/v1", "Deployment", "deployment", "deployment").
				Return(scaleTarget, nil).
				AnyTimes()

			summary, err := hc.Create(printOptions)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			component.AssertEqual(t, tc.expected, summary)
		})
	}
}
