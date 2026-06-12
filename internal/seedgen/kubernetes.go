// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/podplane/seedgen/pkg/pipeline"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	flowcontrolv1 "k8s.io/api/flowcontrol/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	kubernetesProtobufPrefix = []byte{'k', '8', 's', 0}
	kubernetesCodecsOnce     sync.Once
	kubernetesCodecs         serializer.CodecFactory
	kubernetesCodecsErr      error
)

// standard timestamp to reset date/times to in seed records.
// why pick a boring date when you can use the kubernetes v1.0.0 tag timestamp?
const normalisedCreationTimestamp = "2015-07-11T04:02:31Z"

// transformValue applies seed transforms to either JSON or Kubernetes protobuf
// record values.
func transformValue(transforms pipeline.Transforms, key, value []byte) ([]byte, error) {
	recordKey := string(key)
	if bytes.HasPrefix(value, kubernetesProtobufPrefix) {
		return transformKubernetesProtobufValue(transforms, recordKey, value)
	}
	transformed, err := transforms.TransformValue(key, value)
	if err != nil {
		return nil, err
	}
	return normaliseKubernetesObjectMetadata(transformed)
}

// transformKubernetesProtobufValue applies seed transforms to Kubernetes
// protobuf storage values and emits decoded Kubernetes objects as JSON.
func transformKubernetesProtobufValue(transforms pipeline.Transforms, key string, value []byte) ([]byte, error) {
	codecs, err := getKubernetesCodecs()
	if err != nil {
		return nil, err
	}
	obj, gvk, err := codecs.UniversalDeserializer().Decode(value, nil, nil)
	if err != nil {
		if !transforms.HasTransform(key) {
			return value, nil
		}
		return nil, fmt.Errorf("decode %s as Kubernetes protobuf: %w", key, err)
	}
	if gvk != nil {
		obj.GetObjectKind().SetGroupVersionKind(*gvk)
	}
	transforms.TransformObject(key, obj)
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), runtime.ContentTypeJSON)
	if !ok {
		return nil, fmt.Errorf("Kubernetes JSON serializer is unavailable")
	}
	gv := obj.GetObjectKind().GroupVersionKind().GroupVersion()
	if gvk != nil {
		gv = gvk.GroupVersion()
	}
	encoded, err := runtime.Encode(codecs.EncoderForVersion(info.Serializer, gv), obj)
	if err != nil {
		return nil, fmt.Errorf("encode transformed value for %s as Kubernetes JSON: %w", key, err)
	}
	return normaliseKubernetesObjectMetadata(encoded)
}

// normaliseKubernetesObjectMetadata removes or resets volatile fields from JSON
// objects that have Kubernetes API object identity fields (apiVersion and kind).
// Seed records should contain desired/static object state; controller and
// apiserver observed fields are recomputed after bootstrap. Non-JSON values,
// JSON scalars/arrays, and JSON objects without apiVersion+kind are left
// untouched so opaque payloads such as Helm release Secret data are not modified.
func normaliseKubernetesObjectMetadata(value []byte) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal(value, &obj); err != nil {
		return value, nil
	}
	if _, ok := obj["apiVersion"].(string); !ok {
		return value, nil
	}
	if _, ok := obj["kind"].(string); !ok {
		return value, nil
	}
	changed := normaliseObjectMetadata(obj)
	if _, ok := obj["status"]; ok {
		delete(obj, "status")
		changed = true
	}
	if !changed {
		return value, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return nil, fmt.Errorf("encode normalised Kubernetes object: %w", err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// normaliseObjectMetadata mutates Kubernetes object metadata fields that are
// expected to vary between seed generation runs and reports whether it changed
// the object.
func normaliseObjectMetadata(obj map[string]any) bool {
	metadata, ok := obj["metadata"].(map[string]any)
	if !ok {
		return false
	}
	var changed bool
	if metadata["creationTimestamp"] != normalisedCreationTimestamp {
		metadata["creationTimestamp"] = normalisedCreationTimestamp
		changed = true
	}
	if _, ok := metadata["managedFields"]; ok {
		delete(metadata, "managedFields")
		changed = true
	}
	annotations, ok := metadata["annotations"].(map[string]any)
	if !ok {
		return changed
	}
	if _, ok := annotations["kubectl.kubernetes.io/last-applied-configuration"]; ok {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		changed = true
	}
	return changed
}

// getKubernetesCodecs returns the minimal Kubernetes codecs needed to decode
// and re-encode built-in resources that may be stored as protobuf in etcd.
func getKubernetesCodecs() (serializer.CodecFactory, error) {
	kubernetesCodecsOnce.Do(func() {
		scheme := runtime.NewScheme()
		registrations := []struct {
			name string
			add  func(*runtime.Scheme) error
		}{
			{"admissionregistration/v1", admissionregistrationv1.AddToScheme},
			{"apps/v1", appsv1.AddToScheme},
			{"batch/v1", batchv1.AddToScheme},
			{"core/v1", corev1.AddToScheme},
			{"flowcontrol/v1", flowcontrolv1.AddToScheme},
			{"networking/v1", networkingv1.AddToScheme},
			{"policy/v1", policyv1.AddToScheme},
			{"rbac/v1", rbacv1.AddToScheme},
			{"resource/v1beta1", resourcev1beta1.AddToScheme},
			{"scheduling/v1", schedulingv1.AddToScheme},
			{"storage/v1", storagev1.AddToScheme},
		}
		for _, registration := range registrations {
			if err := registration.add(scheme); err != nil {
				kubernetesCodecsErr = fmt.Errorf("register Kubernetes %s scheme: %w", registration.name, err)
				return
			}
		}
		kubernetesCodecs = serializer.NewCodecFactory(scheme)
	})
	return kubernetesCodecs, kubernetesCodecsErr
}
