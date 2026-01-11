// SPDX-License-Identifier: MIT

package acid

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
	// Ensure context is not cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	dir := filepath.Dir(absPath)

	// Create parent directories if needed
	if opts.CreateDirs {
		if err := os.MkdirAll(dir, opts.DirMode); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Determine temp directory
	tempDir := opts.TempDir
	if tempDir == "" {
		tempDir = dir
	}

	// Create temporary file
	tempFile, err := os.CreateTemp(tempDir, opts.TempPrefix+"*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup on any error
	success := false
	defer func() {
		if !success {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	// Set up writer with optional checksum calculation
	var writer io.Writer = tempFile
	var hasher hash.Hash

	if opts.VerifyAfterWrite && opts.ChecksumAlgo != "" {
		hasher = newHasher(opts.ChecksumAlgo)
		if hasher != nil {
			writer = io.MultiWriter(tempFile, hasher)
		}
	}

	// Write content
	if err := fn(writer); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Sync to disk
	if opts.Sync {
		if err := tempFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync file: %w", err)
		}
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set file mode
	if opts.Mode != 0 {
		if err := os.Chmod(tempPath, opts.Mode); err != nil {
			return fmt.Errorf("failed to set file mode: %w", err)
		}
	}

	// Set ownership (Unix only)
	if err := w.setOwnership(tempPath, opts.UID, opts.GID); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	// Verify checksum if expected
	if opts.ExpectedChecksum != "" && hasher != nil {
		actualChecksum := hex.EncodeToString(hasher.Sum(nil))
		if actualChecksum != opts.ExpectedChecksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", opts.ExpectedChecksum, actualChecksum)
		}
	}

	// Atomic rename
	if err := os.Rename(tempPath, absPath); err != nil {
		// Handle cross-device link error
		if w.FallbackOnCrossDevice && isCrossDeviceError(err) {
			if err := w.copyFile(tempPath, absPath, opts.Mode); err != nil {
				return fmt.Errorf("failed to copy file across devices: %w", err)
			}
			os.Remove(tempPath)
		} else {
			return fmt.Errorf("failed to rename temp file: %w", err)
		}
	}

	// Sync parent directory
	if opts.SyncDir {
		if err := syncDir(dir); err != nil {
			// Non-fatal, just log
			_ = err
		}
	}

	// Verify after write if requested
	if opts.VerifyAfterWrite && opts.ChecksumAlgo != "" && hasher != nil {
		expectedChecksum := hex.EncodeToString(hasher.Sum(nil))
		actualChecksum, err := calculateFileChecksum(absPath, opts.ChecksumAlgo)
		if err != nil {
			return fmt.Errorf("failed to verify written file: %w", err)
		}
		if actualChecksum != expectedChecksum {
			return fmt.Errorf("written file verification failed: expected %s, got %s", expectedChecksum, actualChecksum)
		}
	}

	success = true
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
func (w *AtomicWriter) copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
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
	// Check for EXDEV error (cross-device link)
	return os.IsNotExist(err) || // Shouldn't happen but check anyway
		err.Error() == "invalid cross-device link" ||
		err.Error() == "rename: invalid cross-device link"
}

// newHasher creates a new hash.Hash for the given algorithm.
func newHasher(algo string) hash.Hash {
	switch algo {
	case "md5":
		return md5.New()
	case "sha256":
		return sha256.New()
	case "sha512":
		return sha512.New()
	default:
		return sha256.New() // Default to SHA-256
	}
}

// calculateFileChecksum calculates the checksum of a file.
func calculateFileChecksum(path string, algo string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := newHasher(algo)
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
	h := newHasher(algo)
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CalculateBytes calculates the checksum of byte data.
func (c *FileChecksummer) CalculateBytes(data []byte, algo string) string {
	h := newHasher(algo)
	h.Write(data)
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
