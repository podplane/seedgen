// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"github.com/podplane/seedgen/pkg/pipeline"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const envoyGatewayDeploymentKey = "/registry/deployments/platform-envoy-gateway/envoy-gateway"

// RecommendedTransforms extends the built-in transforms with
// recommended-profile bootstrap behavior.
var RecommendedTransforms = append(append(pipeline.Transforms{}, Transforms...),
	pipeline.KeyTransform{Key: envoyGatewayDeploymentKey, JSONTransforms: []pipeline.JSONTransform{scaleDeploymentToZero}, ProtobufTransforms: []pipeline.ProtobufTransform{scaleDeploymentToZeroObject}},
)

// scaleDeploymentToZero prevents a seeded Deployment from creating Pods before
// its runtime dependencies have been reconciled.
func scaleDeploymentToZero(obj map[string]any) bool {
	spec, ok := obj["spec"].(map[string]any)
	if !ok || spec["replicas"] == float64(0) {
		return false
	}
	spec["replicas"] = float64(0)
	return true
}

// scaleDeploymentToZeroObject prevents a protobuf-decoded seeded Deployment
// from creating Pods before its runtime dependencies have been reconciled.
func scaleDeploymentToZeroObject(obj runtime.Object) bool {
	deployment := obj.(*appsv1.Deployment)
	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == 0 {
		return false
	}
	replicas := int32(0)
	deployment.Spec.Replicas = &replicas
	return true
}
