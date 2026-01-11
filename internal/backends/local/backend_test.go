// SPDX-License-Identifier: MIT

package local

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ebogdum/filemanager/internal/plugin"
)

func TestBackend_Connect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	if err := backend.Ping(context.Background()); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestBackend_WriteAndRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	content := []byte("Hello, World!")
	path := "test.txt"

	// Write
	opts := plugin.WriteOptions{
		Mode:      0644,
		Overwrite: true,
		Atomic:    true,
	}

	if err := backend.Write(context.Background(), path, bytes.NewReader(content), opts); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	reader, err := backend.Read(context.Background(), path)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("Content mismatch: got %q, want %q", got, content)
	}
}

func TestBackend_Exists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	// Create a file
	path := "test.txt"
	if err := os.WriteFile(filepath.Join(dir, path), []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Check exists
	exists, err := backend.Exists(context.Background(), path)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Error("Expected file to exist")
	}

	// Check non-existent
	exists, err = backend.Exists(context.Background(), "nonexistent.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if exists {
		t.Error("Expected file to not exist")
	}
}

func TestBackend_Delete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	// Create a file
	path := "test.txt"
	if err := os.WriteFile(filepath.Join(dir, path), []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Delete
	if err := backend.Delete(context.Background(), path); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	exists, err := backend.Exists(context.Background(), path)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if exists {
		t.Error("Expected file to be deleted")
	}
}

func TestBackend_Stat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	// Create a file
	path := "test.txt"
	content := []byte("Hello, World!")
	if err := os.WriteFile(filepath.Join(dir, path), content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Stat
	info, err := backend.Stat(context.Background(), path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Name != "test.txt" {
		t.Errorf("Name mismatch: got %q, want %q", info.Name, "test.txt")
	}

	if info.Size != int64(len(content)) {
		t.Errorf("Size mismatch: got %d, want %d", info.Size, len(content))
	}

	if info.IsDir {
		t.Error("Expected file, not directory")
	}
}

func TestBackend_Mkdir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	// Create directory
	path := "subdir"
	opts := plugin.MkdirOptions{
		Mode: 0755,
	}

	if err := backend.Mkdir(context.Background(), path, opts); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Verify
	info, err := os.Stat(filepath.Join(dir, path))
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected directory")
	}
}

func TestBackend_MkdirRecursive(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	// Create nested directories
	path := "a/b/c"
	opts := plugin.MkdirOptions{
		Mode:      0755,
		Recursive: true,
	}

	if err := backend.Mkdir(context.Background(), path, opts); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Verify
	info, err := os.Stat(filepath.Join(dir, path))
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected directory")
	}
}

func TestBackend_List(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	// Create some files
	files := []string{"a.txt", "b.txt", "c.txt"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	// List
	opts := plugin.ListOptions{}
	entries, err := backend.List(context.Background(), ".", opts)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(entries) != len(files) {
		t.Errorf("Entry count mismatch: got %d, want %d", len(entries), len(files))
	}
}

func TestBackend_Capabilities(t *testing.T) {
	t.Parallel()

	backend := New()
	caps := backend.Capabilities()

	if !caps.SupportsRead {
		t.Error("Expected SupportsRead")
	}

	if !caps.SupportsWrite {
		t.Error("Expected SupportsWrite")
	}

	if !caps.SupportsAtomicWrite {
		t.Error("Expected SupportsAtomicWrite")
	}

	if !caps.SupportsLocking {
		t.Error("Expected SupportsLocking")
	}
}

func TestBackend_Symlink(t *testing.T) {
	t.Parallel()

	// Skip on Windows
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping symlink test on Windows")
	}

	dir := t.TempDir()
	backend := New()

	config := plugin.BackendConfig{
		BasePath: dir,
	}

	if err := backend.Connect(context.Background(), config); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	// Create a file
	if err := os.WriteFile(filepath.Join(dir, "target.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create symlink
	if err := backend.Symlink(context.Background(), "target.txt", "link.txt"); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	// Verify
	info, err := backend.Stat(context.Background(), "link.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if !info.IsSymlink {
		t.Error("Expected symlink")
	}

	if info.LinkTarget != "target.txt" {
		t.Errorf("LinkTarget mismatch: got %q, want %q", info.LinkTarget, "target.txt")
	}
}
