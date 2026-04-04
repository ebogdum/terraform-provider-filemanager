// SPDX-License-Identifier: MIT

package acid

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// AtomicWriter implements atomic file writes with ACID guarantees.
type AtomicWriter struct {
	// Fallback to non-atomic write on cross-device rename
	FallbackOnCrossDevice bool
}

// NewAtomicWriter creates a new AtomicWriter.
func NewAtomicWriter() *AtomicWriter {
	return &AtomicWriter{
		FallbackOnCrossDevice: true,
	}
}

// Write writes data atomically to the target path.
func (w *AtomicWriter) Write(ctx context.Context, path string, data []byte, opts WriteOptions) error {
	return w.WriteFrom(ctx, path, bytes.NewReader(data), opts)
}

// WriteFrom writes data from a reader atomically to the target path.
func (w *AtomicWriter) WriteFrom(ctx context.Context, path string, r io.Reader, opts WriteOptions) error {
	return w.WriteFunc(ctx, path, func(w io.Writer) error {
		_, err := io.Copy(w, r)
		return err
	}, opts)
}

// WriteFunc writes data using a callback function that writes to the temp file.
func (w *AtomicWriter) WriteFunc(ctx context.Context, path string, fn func(w io.Writer) error, opts WriteOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	absPath, dir, tempDir, err := w.resolveWritePaths(path, opts)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(tempDir, opts.TempPrefix+"*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	success := false
	defer cleanupTempFileOnError(tempFile, tempPath, &success)

	writer, hasher, err := buildWriteWriter(tempFile, opts)
	if err != nil {
		return err
	}

	if err := fn(writer); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	if err := w.finalizeTempFile(tempFile, tempPath, opts); err != nil {
		return err
	}

	if err := verifyExpectedChecksum(opts, hasher); err != nil {
		return err
	}

	if err := w.commitTempFile(tempPath, absPath, dir, opts); err != nil {
		return err
	}

	if err := verifyWrittenChecksum(absPath, opts, hasher); err != nil {
		return err
	}

	success = true
	return nil
}

func (w *AtomicWriter) resolveWritePaths(path string, opts WriteOptions) (absPath, dir, tempDir string, err error) {
	absPath, err = filepath.Abs(path)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	dir = filepath.Dir(absPath)

	if opts.CreateDirs {
		if err := os.MkdirAll(dir, opts.DirMode); err != nil {
			return "", "", "", fmt.Errorf("failed to create directories: %w", err)
		}
	}

	tempDir = opts.TempDir
	if tempDir == "" {
		tempDir = dir
	}

	return absPath, dir, tempDir, nil
}

func cleanupTempFileOnError(tempFile *os.File, tempPath string, success *bool) {
	if *success {
		return
	}
	_ = tempFile.Close()
	_ = os.Remove(tempPath)
}

func buildWriteWriter(tempFile *os.File, opts WriteOptions) (io.Writer, hash.Hash, error) {
	if !opts.VerifyAfterWrite || opts.ChecksumAlgo == "" {
		return tempFile, nil, nil
	}

	hasher, err := newHasher(opts.ChecksumAlgo)
	if err != nil {
		return nil, nil, err
	}

	return io.MultiWriter(tempFile, hasher), hasher, nil
}

func (w *AtomicWriter) finalizeTempFile(tempFile *os.File, tempPath string, opts WriteOptions) error {
	if opts.Sync {
		if err := tempFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync file: %w", err)
		}
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if opts.Mode != 0 {
		if err := os.Chmod(tempPath, opts.Mode); err != nil {
			return fmt.Errorf("failed to set file mode: %w", err)
		}
	}

	if err := w.setOwnership(tempPath, opts.UID, opts.GID); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	return nil
}

func verifyExpectedChecksum(opts WriteOptions, hasher hash.Hash) error {
	if opts.ExpectedChecksum == "" {
		return nil
	}

	if hasher == nil {
		return fmt.Errorf("expected checksum requires verify_after_write with a supported checksum algorithm")
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != opts.ExpectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", opts.ExpectedChecksum, actualChecksum)
	}

	return nil
}

func (w *AtomicWriter) commitTempFile(tempPath, absPath, dir string, opts WriteOptions) error {
	if err := os.Rename(tempPath, absPath); err != nil {
		if w.FallbackOnCrossDevice && isCrossDeviceError(err) {
			if err := w.copyFile(tempPath, absPath, opts.Mode); err != nil {
				return fmt.Errorf("failed to copy file across devices: %w", err)
			}
			_ = os.Remove(tempPath)
		} else {
			return fmt.Errorf("failed to rename temp file: %w", err)
		}
	}

	if opts.SyncDir {
		_ = syncDir(dir)
	}

	return nil
}

func verifyWrittenChecksum(absPath string, opts WriteOptions, hasher hash.Hash) error {
	if !opts.VerifyAfterWrite || opts.ChecksumAlgo == "" || hasher == nil {
		return nil
	}

	expectedChecksum := hex.EncodeToString(hasher.Sum(nil))
	actualChecksum, err := calculateFileChecksum(absPath, opts.ChecksumAlgo)
	if err != nil {
		return fmt.Errorf("failed to verify written file: %w", err)
	}
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("written file verification failed: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// setOwnership sets the file ownership on Unix systems.
func (w *AtomicWriter) setOwnership(path string, uid, gid int) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if uid < 0 && gid < 0 {
		return nil
	}
	return os.Chown(path, uid, gid)
}

// copyFile copies a file when rename across devices fails.
// Uses a temporary file + rename to preserve atomicity even on cross-device writes.
func (w *AtomicWriter) copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstDir := filepath.Dir(dst)
	tmpFile, err := os.CreateTemp(dstDir, ".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if mode != 0 {
		if err := os.Chmod(tmpPath, mode); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return err
	}
	success = true
	return nil
}

// syncDir syncs a directory to ensure rename is durable.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		// Windows doesn't support syncing directories
		return nil
	}

	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	return d.Sync()
}

// isCrossDeviceError checks if an error is a cross-device link error.
func isCrossDeviceError(err error) bool {
	if err == nil {
		return false
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return errors.Is(linkErr.Err, syscall.EXDEV)
	}
	return strings.Contains(err.Error(), "invalid cross-device link")
}

// newHasher creates a new hash.Hash for the given algorithm.
func newHasher(algo string) (hash.Hash, error) {
	switch strings.ToLower(strings.TrimSpace(algo)) {
	case "", "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum algorithm %q (supported: sha256, sha512)", algo)
	}
}

// calculateFileChecksum calculates the checksum of a file.
func calculateFileChecksum(path string, algo string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h, err := newHasher(algo)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileChecksummer implements the Checksummer interface.
type FileChecksummer struct{}

// NewChecksummer creates a new FileChecksummer.
func NewChecksummer() *FileChecksummer {
	return &FileChecksummer{}
}

// Calculate calculates the checksum of a file.
func (c *FileChecksummer) Calculate(ctx context.Context, path string, algo string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return calculateFileChecksum(path, algo)
}

// CalculateReader calculates the checksum of data from a reader.
func (c *FileChecksummer) CalculateReader(r io.Reader, algo string) (string, error) {
	h, err := newHasher(algo)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CalculateBytes calculates the checksum of byte data.
func (c *FileChecksummer) CalculateBytes(data []byte, algo string) string {
	h, err := newHasher(algo)
	if err != nil {
		h = sha256.New()
	}
	_, _ = h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Verify verifies a file's checksum.
func (c *FileChecksummer) Verify(ctx context.Context, path string, expected string, algo string) error {
	actual, err := c.Calculate(ctx, path, algo)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}
