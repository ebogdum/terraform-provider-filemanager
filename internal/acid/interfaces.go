// SPDX-License-Identifier: MIT

// Package acid provides ACID (Atomicity, Consistency, Isolation, Durability)
// guarantees for file operations.
package acid

import (
	"context"
	"io"
	"os"
	"time"
)

// Writer provides atomic write operations with ACID guarantees.
type Writer interface {
	// Write writes data atomically to the target path.
	// The write is performed to a temporary file first, then atomically
	// renamed to the target path.
	Write(ctx context.Context, path string, data []byte, opts WriteOptions) error

	// WriteFrom writes data from a reader atomically to the target path.
	WriteFrom(ctx context.Context, path string, r io.Reader, opts WriteOptions) error

	// WriteFunc writes data using a callback function that writes to the temp file.
	// This allows streaming writes while maintaining atomicity.
	WriteFunc(ctx context.Context, path string, fn func(w io.Writer) error, opts WriteOptions) error
}

// WriteOptions contains options for atomic write operations.
type WriteOptions struct {
	// Mode is the file permission mode for the target file.
	Mode os.FileMode

	// DirMode is the permission mode for any directories created.
	DirMode os.FileMode

	// CreateDirs specifies whether to create parent directories if they don't exist.
	CreateDirs bool

	// Sync specifies whether to sync the file content before rename.
	Sync bool

	// SyncDir specifies whether to sync the parent directory after rename.
	SyncDir bool

	// TempDir specifies the directory for temporary files.
	// If empty, the target file's directory is used.
	TempDir string

	// TempPrefix specifies the prefix for temporary files.
	TempPrefix string

	// VerifyAfterWrite specifies whether to verify the file content after write.
	VerifyAfterWrite bool

	// ExpectedChecksum is the expected checksum for verification.
	ExpectedChecksum string

	// ChecksumAlgo is the checksum algorithm (md5, sha256, sha512).
	ChecksumAlgo string

	// UID is the user ID for chown (Unix only, -1 to skip).
	UID int

	// GID is the group ID for chown (Unix only, -1 to skip).
	GID int
}

// DefaultWriteOptions returns default write options.
func DefaultWriteOptions() WriteOptions {
	return WriteOptions{
		Mode:       0644,
		DirMode:    0755,
		CreateDirs: false,
		Sync:       true,
		SyncDir:    true,
		TempPrefix: ".tmp.",
		UID:        -1,
		GID:        -1,
	}
}

// Locker provides file locking operations.
type Locker interface {
	// Lock acquires a lock on the specified path.
	Lock(ctx context.Context, path string, opts LockOptions) (Lock, error)

	// TryLock attempts to acquire a lock without blocking.
	TryLock(ctx context.Context, path string, opts LockOptions) (Lock, bool, error)
}

// Lock represents an acquired file lock.
type Lock interface {
	// Unlock releases the lock.
	Unlock() error

	// Path returns the path of the locked file.
	Path() string

	// IsExclusive returns true if this is an exclusive lock.
	IsExclusive() bool
}

// LockOptions contains options for lock operations.
type LockOptions struct {
	// Exclusive specifies whether to acquire an exclusive (write) lock.
	// If false, a shared (read) lock is acquired.
	Exclusive bool

	// Timeout is the maximum time to wait for the lock.
	// If zero, wait indefinitely.
	Timeout time.Duration

	// RetryInterval is the interval between lock retry attempts.
	RetryInterval time.Duration

	// CreateIfMissing creates the lock file if it doesn't exist.
	CreateIfMissing bool

	// Mode is the file mode for the lock file if created.
	Mode os.FileMode
}

// DefaultLockOptions returns default lock options.
func DefaultLockOptions() LockOptions {
	return LockOptions{
		Exclusive:       true,
		Timeout:         30 * time.Second,
		RetryInterval:   100 * time.Millisecond,
		CreateIfMissing: true,
		Mode:            0644,
	}
}

// Backup provides file backup operations.
type Backup interface {
	// Create creates a backup of the specified file.
	Create(ctx context.Context, path string, opts BackupOptions) (string, error)

	// Restore restores a file from backup.
	Restore(ctx context.Context, backupPath, targetPath string) error

	// List lists available backups for a file.
	List(ctx context.Context, path string) ([]BackupInfo, error)

	// Cleanup removes old backups according to retention policy.
	Cleanup(ctx context.Context, path string, retention int) error
}

// BackupOptions contains options for backup operations.
type BackupOptions struct {
	// BackupDir is the directory where backups are stored.
	// If empty, backups are stored alongside the original file.
	BackupDir string

	// Suffix is the suffix for backup files.
	Suffix string

	// IncludeTimestamp includes timestamp in backup filename.
	IncludeTimestamp bool

	// Compress compresses the backup.
	Compress bool

	// MaxBackups is the maximum number of backups to keep.
	MaxBackups int
}

// DefaultBackupOptions returns default backup options.
func DefaultBackupOptions() BackupOptions {
	return BackupOptions{
		Suffix:           ".bak",
		IncludeTimestamp: true,
		Compress:         false,
		MaxBackups:       5,
	}
}

// BackupInfo contains information about a backup.
type BackupInfo struct {
	Path       string
	Size       int64
	CreatedAt  time.Time
	Checksum   string
	Compressed bool
}

// Checksummer provides checksum calculation.
type Checksummer interface {
	// Calculate calculates the checksum of a file.
	Calculate(ctx context.Context, path string, algo string) (string, error)

	// CalculateReader calculates the checksum of data from a reader.
	CalculateReader(r io.Reader, algo string) (string, error)

	// CalculateBytes calculates the checksum of byte data.
	CalculateBytes(data []byte, algo string) string

	// Verify verifies a file's checksum.
	Verify(ctx context.Context, path string, expected string, algo string) error
}

// Transaction provides transactional file operations.
type Transaction interface {
	// Begin starts a new transaction.
	Begin(ctx context.Context) (Tx, error)
}

// Tx represents an active transaction.
type Tx interface {
	// Write queues a write operation.
	Write(path string, data []byte, opts WriteOptions) error

	// Delete queues a delete operation.
	Delete(path string) error

	// Mkdir queues a mkdir operation.
	Mkdir(path string, mode os.FileMode) error

	// Commit commits all queued operations atomically.
	Commit() error

	// Rollback rolls back all queued operations.
	Rollback() error
}

// Journal provides write-ahead logging for crash recovery.
type Journal interface {
	// Log logs an operation to the journal.
	Log(ctx context.Context, op Operation) error

	// Recover recovers from a crash using the journal.
	Recover(ctx context.Context) error

	// Checkpoint creates a checkpoint, removing committed entries.
	Checkpoint(ctx context.Context) error
}

// Operation represents a journaled operation.
type Operation struct {
	ID        string
	Type      OperationType
	Path      string
	TempPath  string
	Data      []byte
	Timestamp time.Time
	Status    OperationStatus
}

// OperationType is the type of journaled operation.
type OperationType string

const (
	OpWrite  OperationType = "write"
	OpDelete OperationType = "delete"
	OpMkdir  OperationType = "mkdir"
	OpRename OperationType = "rename"
)

// OperationStatus is the status of a journaled operation.
type OperationStatus string

const (
	OpStatusPending    OperationStatus = "pending"
	OpStatusCommitted  OperationStatus = "committed"
	OpStatusRolledBack OperationStatus = "rolled_back"
)
