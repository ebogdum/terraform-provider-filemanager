// SPDX-License-Identifier: MIT

package acid

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriter_Write(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("Hello, World!")

	writer := NewAtomicWriter()
	opts := DefaultWriteOptions()

	err := writer.Write(context.Background(), path, content, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("Content mismatch: got %q, want %q", got, content)
	}
}

func TestAtomicWriter_WriteWithCreateDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "test.txt")
	content := []byte("Hello, World!")

	writer := NewAtomicWriter()
	opts := DefaultWriteOptions()
	opts.CreateDirs = true

	err := writer.Write(context.Background(), path, content, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("Content mismatch: got %q, want %q", got, content)
	}
}

func TestAtomicWriter_WriteWithChecksum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("Hello, World!")

	writer := NewAtomicWriter()
	opts := DefaultWriteOptions()
	opts.VerifyAfterWrite = true
	opts.ChecksumAlgo = "sha256"

	err := writer.Write(context.Background(), path, content, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("Content mismatch: got %q, want %q", got, content)
	}
}

func TestAtomicWriter_WriteWithExpectedChecksum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("Hello, World!")

	// SHA-256 of "Hello, World!"
	expectedChecksum := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"

	writer := NewAtomicWriter()
	opts := DefaultWriteOptions()
	opts.VerifyAfterWrite = true
	opts.ChecksumAlgo = "sha256"
	opts.ExpectedChecksum = expectedChecksum

	err := writer.Write(context.Background(), path, content, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestAtomicWriter_WriteWithWrongChecksum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("Hello, World!")

	// Wrong checksum
	expectedChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	writer := NewAtomicWriter()
	opts := DefaultWriteOptions()
	opts.VerifyAfterWrite = true
	opts.ChecksumAlgo = "sha256"
	opts.ExpectedChecksum = expectedChecksum

	err := writer.Write(context.Background(), path, content, opts)
	if err == nil {
		t.Fatal("Expected checksum mismatch error")
	}
}

func TestFileChecksummer_Calculate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("Hello, World!")

	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	checksummer := NewChecksummer()

	tests := []struct {
		algo     string
		expected string
	}{
		{"sha256", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
		{"sha512", "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387"},
	}

	for _, tt := range tests {
		t.Run(tt.algo, func(t *testing.T) {
			checksum, err := checksummer.Calculate(context.Background(), path, tt.algo)
			if err != nil {
				t.Fatalf("Calculate failed: %v", err)
			}

			if checksum != tt.expected {
				t.Errorf("Checksum mismatch: got %q, want %q", checksum, tt.expected)
			}
		})
	}
}

func TestFileChecksummer_CalculateBytes(t *testing.T) {
	t.Parallel()

	content := []byte("Hello, World!")
	checksummer := NewChecksummer()

	tests := []struct {
		algo     string
		expected string
	}{
		{"sha256", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
		{"sha512", "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387"},
	}

	for _, tt := range tests {
		t.Run(tt.algo, func(t *testing.T) {
			checksum := checksummer.CalculateBytes(content, tt.algo)

			if checksum != tt.expected {
				t.Errorf("Checksum mismatch: got %q, want %q", checksum, tt.expected)
			}
		})
	}
}
