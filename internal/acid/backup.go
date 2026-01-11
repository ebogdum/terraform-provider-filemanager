// SPDX-License-Identifier: MIT

package acid

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileBackup implements the Backup interface for local files.
type FileBackup struct {
	checksummer Checksummer
}

// NewFileBackup creates a new FileBackup.
func NewFileBackup() *FileBackup {
	return &FileBackup{
		checksummer: NewChecksummer(),
	}
}

// Create creates a backup of the specified file.
func (b *FileBackup) Create(ctx context.Context, path string, opts BackupOptions) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if source exists
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat source file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("cannot backup directory: %s", absPath)
	}

	// Determine backup path
	backupPath := b.generateBackupPath(absPath, opts)

	// Create backup directory if needed
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create backup file
	var dstFile *os.File
	if opts.Compress {
		backupPath += ".gz"
		dstFile, err = os.Create(backupPath)
		if err != nil {
			return "", fmt.Errorf("failed to create backup file: %w", err)
		}
		defer dstFile.Close()

		gzWriter := gzip.NewWriter(dstFile)
		defer gzWriter.Close()

		if _, err := io.Copy(gzWriter, srcFile); err != nil {
			os.Remove(backupPath)
			return "", fmt.Errorf("failed to compress backup: %w", err)
		}

		if err := gzWriter.Close(); err != nil {
			os.Remove(backupPath)
			return "", fmt.Errorf("failed to close gzip writer: %w", err)
		}
	} else {
		dstFile, err = os.Create(backupPath)
		if err != nil {
			return "", fmt.Errorf("failed to create backup file: %w", err)
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			os.Remove(backupPath)
			return "", fmt.Errorf("failed to copy to backup: %w", err)
		}
	}

	if err := dstFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync backup file: %w", err)
	}

	// Clean up old backups if needed
	if opts.MaxBackups > 0 {
		if err := b.Cleanup(ctx, absPath, opts.MaxBackups); err != nil {
			// Non-fatal, just log
			_ = err
		}
	}

	return backupPath, nil
}

// generateBackupPath generates the backup file path.
func (b *FileBackup) generateBackupPath(path string, opts BackupOptions) string {
	dir := filepath.Dir(path)
	if opts.BackupDir != "" {
		dir = opts.BackupDir
	}

	base := filepath.Base(path)
	suffix := opts.Suffix
	if suffix == "" {
		suffix = ".bak"
	}

	var backupName string
	if opts.IncludeTimestamp {
		timestamp := time.Now().Format("20060102-150405")
		backupName = fmt.Sprintf("%s.%s%s", base, timestamp, suffix)
	} else {
		backupName = base + suffix
	}

	return filepath.Join(dir, backupName)
}

// Restore restores a file from backup.
func (b *FileBackup) Restore(ctx context.Context, backupPath, targetPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute backup path: %w", err)
	}

	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute target path: %w", err)
	}

	// Check if backup exists
	if _, err := os.Stat(absBackupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Open backup file
	srcFile, err := os.Open(absBackupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer srcFile.Close()

	// Create target directory if needed
	targetDir := filepath.Dir(absTargetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Use atomic writer for restoration
	writer := NewAtomicWriter()
	opts := DefaultWriteOptions()
	opts.Sync = true

	// Check if backup is compressed
	var reader io.Reader = srcFile
	if strings.HasSuffix(absBackupPath, ".gz") {
		gzReader, err := gzip.NewReader(srcFile)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	return writer.WriteFrom(ctx, absTargetPath, reader, opts)
}

// List lists available backups for a file.
func (b *FileBackup) List(ctx context.Context, path string) ([]BackupInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	dir := filepath.Dir(absPath)
	base := filepath.Base(absPath)

	// Find matching backup files
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if this looks like a backup of our file
		if !strings.HasPrefix(name, base+".") {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		backupPath := filepath.Join(dir, name)
		compressed := strings.HasSuffix(name, ".gz")

		// Calculate checksum
		checksum, _ := b.checksummer.Calculate(ctx, backupPath, "sha256")

		backups = append(backups, BackupInfo{
			Path:       backupPath,
			Size:       info.Size(),
			CreatedAt:  info.ModTime(),
			Checksum:   checksum,
			Compressed: compressed,
		})
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// Cleanup removes old backups according to retention policy.
func (b *FileBackup) Cleanup(ctx context.Context, path string, retention int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if retention <= 0 {
		return nil
	}

	backups, err := b.List(ctx, path)
	if err != nil {
		return err
	}

	// Delete excess backups (oldest first)
	if len(backups) > retention {
		for _, backup := range backups[retention:] {
			if err := os.Remove(backup.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove old backup %s: %w", backup.Path, err)
			}
		}
	}

	return nil
}
