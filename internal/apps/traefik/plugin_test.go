// SPDX-License-Identifier: MIT

package traefik

import "testing"

func TestIsValidDockerEndpoint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid unix socket", input: "unix:///var/run/docker.sock", expected: true},
		{name: "valid tcp host port", input: "tcp://127.0.0.1:2375", expected: true},
		{name: "valid http endpoint", input: "http://docker.internal:2375", expected: true},
		{name: "invalid empty tcp host", input: "tcp://", expected: false},
		{name: "invalid tcp without port", input: "tcp://docker.internal", expected: false},
		{name: "invalid scheme", input: "ftp://docker.internal:21", expected: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isValidDockerEndpoint(tc.input)
			if got != tc.expected {
				t.Fatalf("isValidDockerEndpoint(%q)=%v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid", input: "ops@example.com", expected: true},
		{name: "invalid missing at", input: "ops.example.com", expected: false},
		{name: "invalid display name", input: "Ops <ops@example.com>", expected: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isValidEmail(tc.input)
			if got != tc.expected {
				t.Fatalf("isValidEmail(%q)=%v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}
