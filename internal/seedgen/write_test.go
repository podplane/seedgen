// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/netsy-dev/netsy/pkg/datafile"
)

func TestWriteSnapshotRenumbersAndNormalises(t *testing.T) {
	t.Parallel()
	createdAt := time.Now()
	input := []*datafile.Record{
		{Revision: 17, Key: []byte("/a"), Value: []byte("va"), Deleted: false, CreateRevision: 17, PrevRevision: 0, Version: 1, Lease: 99, Dek: 7, CreatedAt: &createdAt, LeaderID: "old-node"},
		{Revision: 23, Key: []byte("/b"), Value: []byte("vb"), Deleted: true, CreateRevision: 18, PrevRevision: 18, Version: 2, Lease: 0, Dek: 0, CreatedAt: &createdAt, LeaderID: "old-node"},
		{Revision: 99, Key: []byte("/c"), Value: []byte("vc"), Deleted: false, CreateRevision: 50, PrevRevision: 60, Version: 3, Lease: 0, Dek: 0, CreatedAt: &createdAt, LeaderID: "old-node"},
	}
	var buf bytes.Buffer
	if err := WriteSnapshot(&buf, input, "seed"); err != nil {
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
	if err := WriteSnapshot(&buf, input, "seed"); err != nil {
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

func TestWriteSnapshotFailsForNonJSONTransformTarget(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := WriteSnapshot(&buf, []*datafile.Record{
		{
			Revision: 1,
			Key:      []byte("/registry/services/specs/default/kubernetes"),
			Value:    []byte("k8s\x00protobuf"),
		},
	}, "seed")
	if err == nil {
		t.Fatalf("WriteSnapshot succeeded for non-JSON transform target")
	}
	if !strings.Contains(err.Error(), "decode /registry/services/specs/default/kubernetes as JSON") {
		t.Fatalf("WriteSnapshot error = %v, want transform target decode error", err)
	}
}

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
	if err := WriteSnapshot(&buf, input, "seed"); err != nil {
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

func decodeValue(t *testing.T, value []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(value, dst); err != nil {
		t.Fatalf("unmarshal %s: %v", value, err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}
