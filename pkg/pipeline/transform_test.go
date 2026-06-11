// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import "testing"

// TestNormalizeImageRef verifies Docker Hub shorthand expansion and explicit
// registry preservation.
func TestNormalizeImageRef(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		image string
		want  string
	}{
		{name: "docker hub library shorthand", image: "caddy:2", want: "docker.io/library/caddy:2"},
		{name: "docker hub namespace shorthand", image: "coredns/coredns:v1.12.1", want: "docker.io/coredns/coredns:v1.12.1"},
		{name: "explicit registry", image: "ghcr.io/fluxcd/source-controller:v1.8.2", want: "ghcr.io/fluxcd/source-controller:v1.8.2"},
		{name: "explicit registry with port", image: "registry.local:5000/app:v1", want: "registry.local:5000/app:v1"},
		{name: "localhost registry", image: "localhost/app:v1", want: "localhost/app:v1"},
		{name: "trim whitespace", image: " caddy:2 ", want: "docker.io/library/caddy:2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeImageRef(tc.image); got != tc.want {
				t.Fatalf("NormalizeImageRef(%q) = %q, want %q", tc.image, got, tc.want)
			}
		})
	}
}
