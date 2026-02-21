// SPDX-License-Identifier: MIT

package hcl

import "testing"

func TestDelete_RemovesArrayElementFromMap(t *testing.T) {
	t.Parallel()

	f := New()
	input := map[string]any{
		"arr": []any{"a", "b", "c"},
	}

	out, err := f.Delete(input, "arr.1")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	m, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("expected map output, got %T", out)
	}

	arr, ok := m["arr"].([]any)
	if !ok {
		t.Fatalf("expected arr to be []any, got %T", m["arr"])
	}

	if len(arr) != 2 || arr[0] != "a" || arr[1] != "c" {
		t.Fatalf("unexpected array value: %#v", arr)
	}
}

func TestDelete_RemovesArrayElementFromRootArray(t *testing.T) {
	t.Parallel()

	f := New()
	input := []any{"x", "y", "z"}

	out, err := f.Delete(input, "1")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	arr, ok := out.([]any)
	if !ok {
		t.Fatalf("expected []any output, got %T", out)
	}

	if len(arr) != 2 || arr[0] != "x" || arr[1] != "z" {
		t.Fatalf("unexpected array value: %#v", arr)
	}
}
