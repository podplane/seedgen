// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seedgen

import (
	"bytes"
	"fmt"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/podplane/seedgen/internal/seedgen/defaults"
)

var (
	kubernetesProtobufPrefix = []byte{'k', '8', 's', 0}
	kubernetesCodecsOnce     sync.Once
	kubernetesCodecs         serializer.CodecFactory
	kubernetesCodecsErr      error
)

// transformValue applies seed transforms to either JSON or Kubernetes protobuf
// record values.
func transformValue(key, value []byte) ([]byte, error) {
	recordKey := string(key)
	if bytes.HasPrefix(value, kubernetesProtobufPrefix) {
		if !defaults.HasTransform(recordKey) {
			return value, nil
		}
		return transformKubernetesProtobufValue(recordKey, value)
	}
	return defaults.TransformValue(key, value)
}

// transformKubernetesProtobufValue applies seed transforms to Kubernetes
// protobuf storage values and preserves protobuf on output.
func transformKubernetesProtobufValue(key string, value []byte) ([]byte, error) {
	codecs, err := getKubernetesCodecs()
	if err != nil {
		return nil, err
	}
	obj, gvk, err := codecs.UniversalDeserializer().Decode(value, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("decode %s as Kubernetes protobuf: %w", key, err)
	}
	if gvk != nil {
		obj.GetObjectKind().SetGroupVersionKind(*gvk)
	}
	if !defaults.TransformObject(key, obj) {
		return value, nil
	}
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), runtime.ContentTypeProtobuf)
	if !ok {
		return nil, fmt.Errorf("Kubernetes protobuf serializer is unavailable")
	}
	gv := obj.GetObjectKind().GroupVersionKind().GroupVersion()
	if gvk != nil {
		gv = gvk.GroupVersion()
	}
	encoded, err := runtime.Encode(codecs.EncoderForVersion(info.Serializer, gv), obj)
	if err != nil {
		return nil, fmt.Errorf("encode transformed value for %s as Kubernetes protobuf: %w", key, err)
	}
	return encoded, nil
}

// getKubernetesCodecs returns the minimal Kubernetes codecs needed to decode
// and re-encode built-in resources that may be stored as protobuf in etcd.
func getKubernetesCodecs() (serializer.CodecFactory, error) {
	kubernetesCodecsOnce.Do(func() {
		scheme := runtime.NewScheme()
		if err := corev1.AddToScheme(scheme); err != nil {
			kubernetesCodecsErr = fmt.Errorf("register Kubernetes core/v1 scheme: %w", err)
			return
		}
		if err := appsv1.AddToScheme(scheme); err != nil {
			kubernetesCodecsErr = fmt.Errorf("register Kubernetes apps/v1 scheme: %w", err)
			return
		}
		kubernetesCodecs = serializer.NewCodecFactory(scheme)
	})
	return kubernetesCodecs, kubernetesCodecsErr
}
