// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/seedgen/pkg/pipeline"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// TestWriteSnapshotRenumbersAndNormalises verifies output records are rewritten
// as fresh contiguous seed records.
func TestWriteSnapshotRenumbersAndNormalises(t *testing.T) {
	t.Parallel()
	createdAt := time.Now()
	input := []*datafile.Record{
		{Revision: 17, Key: []byte("/a"), Value: []byte("va"), Deleted: false, CreateRevision: 17, PrevRevision: 0, Version: 1, Lease: 99, Dek: 7, CreatedAt: &createdAt, LeaderID: "old-node"},
		{Revision: 23, Key: []byte("/b"), Value: []byte("vb"), Deleted: true, CreateRevision: 18, PrevRevision: 18, Version: 2, Lease: 0, Dek: 0, CreatedAt: &createdAt, LeaderID: "old-node"},
		{Revision: 99, Key: []byte("/c"), Value: []byte("vc"), Deleted: false, CreateRevision: 50, PrevRevision: 60, Version: 3, Lease: 0, Dek: 0, CreatedAt: &createdAt, LeaderID: "old-node"},
	}
	var buf bytes.Buffer
	if err := WriteSnapshot(&buf, input, WriteOptions{LeaderID: "seed"}); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	got, err := datafile.ReadSnapshot(&buf)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if len(got) != len(input) {
		t.Fatalf("want %d records, got %d", len(input), len(got))
	}
	for i, record := range got {
		wantRev := int64(i + 1)
		if record.Revision != wantRev {
			t.Errorf("record %d: Revision = %d, want %d", i, record.Revision, wantRev)
		}
		if record.CreateRevision != wantRev {
			t.Errorf("record %d: CreateRevision = %d, want %d", i, record.CreateRevision, wantRev)
		}
		if record.PrevRevision != 0 {
			t.Errorf("record %d: PrevRevision = %d, want 0", i, record.PrevRevision)
		}
		if record.Version != 1 {
			t.Errorf("record %d: Version = %d, want 1", i, record.Version)
		}
		if !record.Created {
			t.Errorf("record %d: Created = false, want true", i)
		}
		if record.Deleted {
			t.Errorf("record %d: Deleted = true, want false", i)
		}
		if record.Lease != 0 {
			t.Errorf("record %d: Lease = %d, want 0", i, record.Lease)
		}
		if record.Dek != 0 {
			t.Errorf("record %d: Dek = %d, want 0", i, record.Dek)
		}
		if record.LeaderID != "seed" {
			t.Errorf("record %d: LeaderID = %q, want seed", i, record.LeaderID)
		}
	}

	// Bootstrap integrity invariant: count == max revision.
	maxRev := int64(0)
	for _, record := range got {
		if record.Revision > maxRev {
			maxRev = record.Revision
		}
	}
	if int64(len(got)) != maxRev {
		t.Fatalf("integrity: len(records)=%d != max revision=%d", len(got), maxRev)
	}
}

// TestWriteSnapshotAppliesSeedTransforms verifies JSON seed transforms for
// status reset and service dual-stack defaults.
func TestWriteSnapshotAppliesSeedTransforms(t *testing.T) {
	t.Parallel()
	input := []*datafile.Record{
		{
			Revision: 1,
			Key:      []byte("/registry/cert-manager.io/certificates/platform-trust-manager/platform-trust-manager"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "cert-manager.io/v1",
				"kind":       "Certificate",
				"status": map[string]any{"conditions": []any{
					map[string]any{"type": "Ready", "status": "True", "reason": "Ready"},
					map[string]any{"type": "Issuing", "status": "False"},
				}},
			}),
		},
		{
			Revision: 2,
			Key:      []byte("/registry/helm.toolkit.fluxcd.io/helmreleases/platform-trust-manager/trust-manager"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "helm.toolkit.fluxcd.io/v2",
				"kind":       "HelmRelease",
				"status":     map[string]any{"conditions": []any{map[string]any{"type": "Ready", "status": "True"}}},
			}),
		},
		{
			Revision: 3,
			Key:      []byte("/registry/policy.cert-manager.io/certificaterequestpolicies/trust-manager-policy"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "policy.cert-manager.io/v1alpha1",
				"kind":       "CertificateRequestPolicy",
				"status":     map[string]any{"conditions": []any{map[string]any{"type": "Ready", "status": "True"}}},
			}),
		},
		{
			Revision: 4,
			Key:      []byte("/registry/deployments/platform-trust-manager/platform-trust-manager"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"status":     map[string]any{"conditions": []any{map[string]any{"type": "Available", "status": "True"}}},
			}),
		},
		{
			Revision: 5,
			Key:      []byte("/registry/daemonsets/platform-traefik/platform-traefik"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "DaemonSet",
				"status":     map[string]any{"numberReady": 1, "numberAvailable": 1},
			}),
		},
		{
			Revision: 6,
			Key:      []byte("/registry/services/specs/default/kubernetes"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"spec": map[string]any{
					"clusterIP":      "198.18.0.1",
					"clusterIPs":     []any{"198.18.0.1"},
					"ipFamilies":     []any{"IPv4"},
					"ipFamilyPolicy": "SingleStack",
				},
			}),
		},
		{
			Revision: 7,
			Key:      []byte("/registry/services/specs/platform-cert-manager/platform-cert-manager"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"spec": map[string]any{
					"clusterIP":      "198.18.10.10",
					"clusterIPs":     []any{"198.18.10.10"},
					"ipFamilies":     []any{"IPv4"},
					"ipFamilyPolicy": "SingleStack",
				},
			}),
		},
		{
			Revision: 8,
			Key:      []byte("/registry/example.io/widgets/default/widget"),
			Value: mustJSON(t, map[string]any{
				"apiVersion": "example.io/v1",
				"kind":       "Certificate",
				"status":     map[string]any{"conditions": []any{map[string]any{"type": "Ready", "status": "True"}}},
			}),
		},
	}
	var buf bytes.Buffer
	writeOpts := WriteOptions{LeaderID: "seed", Transforms: testTransforms()}
	if err := WriteSnapshot(&buf, input, writeOpts); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	got, err := datafile.ReadSnapshot(&buf)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if conditionStatus(t, got[0].Value, "Ready") != "False" {
		t.Fatalf("Certificate Ready status = %s, want False", conditionStatus(t, got[0].Value, "Ready"))
	}
	if conditionStatus(t, got[0].Value, "Issuing") != "False" {
		t.Fatalf("Certificate Issuing status = %s, want unchanged False", conditionStatus(t, got[0].Value, "Issuing"))
	}
	if conditionStatus(t, got[1].Value, "Ready") != "False" {
		t.Fatalf("HelmRelease Ready status = %s, want False", conditionStatus(t, got[1].Value, "Ready"))
	}
	if conditionStatus(t, got[2].Value, "Ready") != "False" {
		t.Fatalf("CertificateRequestPolicy Ready status = %s, want False", conditionStatus(t, got[2].Value, "Ready"))
	}
	if conditionStatus(t, got[3].Value, "Available") != "False" {
		t.Fatalf("Deployment Available status = %s, want False", conditionStatus(t, got[3].Value, "Available"))
	}
	var daemonSet map[string]any
	decodeValue(t, got[4].Value, &daemonSet)
	status := daemonSet["status"].(map[string]any)
	if status["numberReady"] != float64(0) || status["numberAvailable"] != float64(0) {
		t.Fatalf("DaemonSet status = %#v, want ready/available counters zero", status)
	}
	var service map[string]any
	decodeValue(t, got[5].Value, &service)
	spec := service["spec"].(map[string]any)
	if spec["ipFamilyPolicy"] != "PreferDualStack" {
		t.Fatalf("Service ipFamilyPolicy = %v, want PreferDualStack", spec["ipFamilyPolicy"])
	}
	clusterIPs := spec["clusterIPs"].([]any)
	if len(clusterIPs) != 2 || clusterIPs[0] != "198.18.0.1" || clusterIPs[1] != "fdc6::1" {
		t.Fatalf("Service clusterIPs = %#v, want IPv4 and IPv6", clusterIPs)
	}
	var genericService map[string]any
	decodeValue(t, got[6].Value, &genericService)
	genericSpec := genericService["spec"].(map[string]any)
	if genericSpec["ipFamilyPolicy"] != "PreferDualStack" {
		t.Fatalf("Generic Service ipFamilyPolicy = %v, want PreferDualStack", genericSpec["ipFamilyPolicy"])
	}
	genericClusterIPs := genericSpec["clusterIPs"].([]any)
	if len(genericClusterIPs) != 1 || genericClusterIPs[0] != "198.18.10.10" {
		t.Fatalf("Generic Service clusterIPs = %#v, want original cluster IP only", genericClusterIPs)
	}
	if string(got[7].Value) != string(input[7].Value) {
		t.Fatalf("unrelated resource value changed: %s", got[7].Value)
	}
}

// TestWriteSnapshotEmitsJSONForKubernetesProtobufSeedTransforms verifies that
// protobuf storage values are transformed and emitted as JSON.
func TestWriteSnapshotEmitsJSONForKubernetesProtobufSeedTransforms(t *testing.T) {
	t.Parallel()
	singleStack := corev1.IPFamilyPolicySingleStack
	input := []*datafile.Record{
		{
			Revision: 1,
			Key:      []byte("/registry/deployments/platform-trust-manager/platform-trust-manager"),
			Value: kubernetesProtobufValue(t, &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
				Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
				}},
			}, appsv1.SchemeGroupVersion),
		},
		{
			Revision: 2,
			Key:      []byte("/registry/daemonsets/platform-traefik/platform-traefik"),
			Value: kubernetesProtobufValue(t, &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
				Status:   appsv1.DaemonSetStatus{NumberReady: 1, NumberAvailable: 1},
			}, appsv1.SchemeGroupVersion),
		},
		{
			Revision: 3,
			Key:      []byte("/registry/services/specs/default/kubernetes"),
			Value: kubernetesProtobufValue(t, &corev1.Service{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
				Spec: corev1.ServiceSpec{
					ClusterIP:      "198.18.0.1",
					ClusterIPs:     []string{"198.18.0.1"},
					IPFamilies:     []corev1.IPFamily{corev1.IPv4Protocol},
					IPFamilyPolicy: &singleStack,
				},
			}, corev1.SchemeGroupVersion),
		},
		{
			Revision: 4,
			Key:      []byte("/registry/services/specs/platform-cert-manager/platform-cert-manager"),
			Value: kubernetesProtobufValue(t, &corev1.Service{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
				Spec: corev1.ServiceSpec{
					ClusterIP:      "198.18.10.10",
					ClusterIPs:     []string{"198.18.10.10"},
					IPFamilies:     []corev1.IPFamily{corev1.IPv4Protocol},
					IPFamilyPolicy: &singleStack,
				},
			}, corev1.SchemeGroupVersion),
		},
	}
	var buf bytes.Buffer
	writeOpts := WriteOptions{LeaderID: "seed", Transforms: testTransforms()}
	if err := WriteSnapshot(&buf, input, writeOpts); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}
	got, err := datafile.ReadSnapshot(&buf)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	var deployment appsv1.Deployment
	decodeValue(t, got[0].Value, &deployment)
	if bytes.HasPrefix(got[0].Value, kubernetesProtobufPrefix) {
		t.Fatalf("Deployment value remained Kubernetes protobuf, want JSON")
	}
	if deployment.Status.Conditions[0].Status != corev1.ConditionFalse {
		t.Fatalf("Deployment Available status = %s, want False", deployment.Status.Conditions[0].Status)
	}
	var daemonSet appsv1.DaemonSet
	decodeValue(t, got[1].Value, &daemonSet)
	if daemonSet.Status.NumberReady != 0 || daemonSet.Status.NumberAvailable != 0 {
		t.Fatalf("DaemonSet status = %#v, want ready/available counters zero", daemonSet.Status)
	}
	var service corev1.Service
	decodeValue(t, got[2].Value, &service)
	if service.Spec.IPFamilyPolicy == nil || *service.Spec.IPFamilyPolicy != corev1.IPFamilyPolicyPreferDualStack {
		t.Fatalf("Service ipFamilyPolicy = %v, want PreferDualStack", service.Spec.IPFamilyPolicy)
	}
	if !slices.Equal(service.Spec.ClusterIPs, []string{"198.18.0.1", "fdc6::1"}) {
		t.Fatalf("Service clusterIPs = %#v, want IPv4 and IPv6", service.Spec.ClusterIPs)
	}
	var genericService corev1.Service
	decodeValue(t, got[3].Value, &genericService)
	if genericService.Spec.IPFamilyPolicy == nil || *genericService.Spec.IPFamilyPolicy != corev1.IPFamilyPolicyPreferDualStack {
		t.Fatalf("Generic Service ipFamilyPolicy = %v, want PreferDualStack", genericService.Spec.IPFamilyPolicy)
	}
	if !slices.Equal(genericService.Spec.ClusterIPs, []string{"198.18.10.10"}) {
		t.Fatalf("Generic Service clusterIPs = %#v, want original cluster IP only", genericService.Spec.ClusterIPs)
	}
}

// TestWriteSnapshotFailsForInvalidKubernetesProtobufTransformTarget verifies
// protobuf decode failures are reported for transform targets.
func TestWriteSnapshotFailsForInvalidKubernetesProtobufTransformTarget(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	writeOpts := WriteOptions{LeaderID: "seed", Transforms: testTransforms()}
	err := WriteSnapshot(&buf, []*datafile.Record{
		{
			Revision: 1,
			Key:      []byte("/registry/services/specs/default/kubernetes"),
			Value:    []byte("k8s\x00protobuf"),
		},
	}, writeOpts)
	if err == nil {
		t.Fatalf("WriteSnapshot succeeded for invalid Kubernetes protobuf transform target")
	}
	if !strings.Contains(err.Error(), "decode /registry/services/specs/default/kubernetes as Kubernetes protobuf") {
		t.Fatalf("WriteSnapshot error = %v, want Kubernetes protobuf decode error", err)
	}
}

// TestWriteSnapshotAllowsNonJSONUntransformedValue verifies unrelated binary or
// textual values pass through unchanged.
func TestWriteSnapshotAllowsNonJSONUntransformedValue(t *testing.T) {
	t.Parallel()
	input := []*datafile.Record{
		{
			Revision: 1,
			Key:      []byte("/registry/secrets/default/not-transformed"),
			Value:    []byte("not json"),
		},
	}
	var buf bytes.Buffer
	if err := WriteSnapshot(&buf, input, WriteOptions{LeaderID: "seed"}); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}
	got, err := datafile.ReadSnapshot(&buf)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if string(got[0].Value) != string(input[0].Value) {
		t.Fatalf("value = %q, want %q", got[0].Value, input[0].Value)
	}
}

// conditionStatus returns the status for a named condition in a JSON object.
func conditionStatus(t *testing.T, value []byte, conditionType string) string {
	t.Helper()
	var obj map[string]any
	decodeValue(t, value, &obj)
	status := obj["status"].(map[string]any)
	conditions := status["conditions"].([]any)
	for _, item := range conditions {
		condition := item.(map[string]any)
		if condition["type"] == conditionType {
			return condition["status"].(string)
		}
	}
	t.Fatalf("condition %s not found in %s", conditionType, value)
	return ""
}

// decodeValue decodes a JSON record value into dst for test assertions.
func decodeValue(t *testing.T, value []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(value, dst); err != nil {
		t.Fatalf("unmarshal %s: %v", value, err)
	}
}

// mustJSON marshals a test fixture to JSON or fails the test.
func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}

// kubernetesProtobufValue encodes a typed Kubernetes object as a protobuf
// fixture using the same serializer family used by Kubernetes storage.
func kubernetesProtobufValue(t *testing.T, obj runtime.Object, gv schema.GroupVersion) []byte {
	t.Helper()
	codecs := testKubernetesCodecs(t)
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), runtime.ContentTypeProtobuf)
	if !ok {
		t.Fatalf("Kubernetes protobuf serializer is unavailable")
	}
	data, err := runtime.Encode(codecs.EncoderForVersion(info.Serializer, gv), obj)
	if err != nil {
		t.Fatalf("encode Kubernetes protobuf fixture: %v", err)
	}
	return data
}

// decodeKubernetesProtobufValue decodes a Kubernetes protobuf record value into
// the provided typed object for test assertions.
func decodeKubernetesProtobufValue(t *testing.T, value []byte, into runtime.Object) {
	t.Helper()
	codecs := testKubernetesCodecs(t)
	if _, _, err := codecs.UniversalDeserializer().Decode(value, nil, into); err != nil {
		t.Fatalf("decode Kubernetes protobuf value: %v", err)
	}
}

// testKubernetesCodecs returns a test-local scheme with the built-in API groups
// covered by seed transforms.
func testKubernetesCodecs(t *testing.T) serializer.CodecFactory {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register core/v1 scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register apps/v1 scheme: %v", err)
	}
	return serializer.NewCodecFactory(scheme)
}

// testTransforms returns the minimal transform table exercised by writer tests.
func testTransforms() pipeline.Transforms {
	setCondition := func(conditionType string) func(map[string]any) bool {
		return func(obj map[string]any) bool {
			status, ok := obj["status"].(map[string]any)
			if !ok {
				return false
			}
			conditions, ok := status["conditions"].([]any)
			if !ok {
				return false
			}
			var changed bool
			for _, item := range conditions {
				condition := item.(map[string]any)
				if condition["type"] == conditionType && condition["status"] == "True" {
					condition["status"] = "False"
					changed = true
				}
			}
			return changed
		}
	}
	resetDaemonSet := func(obj map[string]any) bool {
		status := obj["status"].(map[string]any)
		status["numberReady"] = float64(0)
		status["numberAvailable"] = float64(0)
		return true
	}
	setService := func(ipv4, ipv6 string) func(map[string]any) bool {
		return func(obj map[string]any) bool {
			spec := obj["spec"].(map[string]any)
			spec["ipFamilies"] = []string{"IPv4", "IPv6"}
			spec["ipFamilyPolicy"] = "PreferDualStack"
			if ipv4 != "" {
				spec["clusterIP"] = ipv4
				clusterIPs := []string{ipv4}
				if ipv6 != "" {
					clusterIPs = append(clusterIPs, ipv6)
				}
				spec["clusterIPs"] = clusterIPs
			}
			return true
		}
	}
	preferService := setService("", "")
	setTypedService := func(ipv4, ipv6 string) func(runtime.Object) bool {
		return func(obj runtime.Object) bool {
			service := obj.(*corev1.Service)
			service.Spec.IPFamilies = []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}
			policy := corev1.IPFamilyPolicyPreferDualStack
			service.Spec.IPFamilyPolicy = &policy
			if ipv4 != "" {
				service.Spec.ClusterIP = ipv4
				service.Spec.ClusterIPs = []string{ipv4}
				if ipv6 != "" {
					service.Spec.ClusterIPs = append(service.Spec.ClusterIPs, ipv6)
				}
			}
			return true
		}
	}
	return pipeline.Transforms{
		pipeline.PrefixTransform{Prefix: "/registry/cert-manager.io/certificates/", APIPrefix: "cert-manager.io/", Kind: "Certificate", JSONTransforms: []pipeline.JSONTransform{setCondition("Ready")}},
		pipeline.PrefixTransform{Prefix: "/registry/helm.toolkit.fluxcd.io/helmreleases/", APIPrefix: "helm.toolkit.fluxcd.io/", Kind: "HelmRelease", JSONTransforms: []pipeline.JSONTransform{setCondition("Ready")}},
		pipeline.PrefixTransform{Prefix: "/registry/policy.cert-manager.io/certificaterequestpolicies/", APIPrefix: "policy.cert-manager.io/", Kind: "CertificateRequestPolicy", JSONTransforms: []pipeline.JSONTransform{setCondition("Ready")}},
		pipeline.PrefixTransform{Prefix: "/registry/deployments/", APIVersion: "apps/v1", Kind: "Deployment", JSONTransforms: []pipeline.JSONTransform{setCondition("Available")}, ProtobufTransforms: []pipeline.ProtobufTransform{func(obj runtime.Object) bool {
			deployment := obj.(*appsv1.Deployment)
			deployment.Status.Conditions[0].Status = corev1.ConditionFalse
			return true
		}}},
		pipeline.PrefixTransform{Prefix: "/registry/daemonsets/", APIVersion: "apps/v1", Kind: "DaemonSet", JSONTransforms: []pipeline.JSONTransform{resetDaemonSet}, ProtobufTransforms: []pipeline.ProtobufTransform{func(obj runtime.Object) bool {
			daemonSet := obj.(*appsv1.DaemonSet)
			daemonSet.Status.NumberReady = 0
			daemonSet.Status.NumberAvailable = 0
			return true
		}}},
		pipeline.PrefixTransform{Prefix: "/registry/services/specs/", APIVersion: "v1", Kind: "Service", JSONTransforms: []pipeline.JSONTransform{preferService}, ProtobufTransforms: []pipeline.ProtobufTransform{setTypedService("", "")}},
		pipeline.KeyTransform{Key: "/registry/services/specs/default/kubernetes", JSONTransforms: []pipeline.JSONTransform{setService("198.18.0.1", "fdc6::1")}, ProtobufTransforms: []pipeline.ProtobufTransform{setTypedService("198.18.0.1", "fdc6::1")}},
	}
}
