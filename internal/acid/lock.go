// SPDX-License-Identifier: MIT

package acid

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// FileLocker implements file locking using flock.
type FileLocker struct {
	// mu protects the locks map
	mu sync.Mutex
	// locks tracks active locks
	locks map[string]*fileLock
}

// NewFileLocker creates a new FileLocker.
func NewFileLocker() *FileLocker {
	return &FileLocker{
		locks: make(map[string]*fileLock),
	}
}

// fileLock represents an active file lock.
type fileLock struct {
	flock     *flock.Flock
	path      string
	exclusive bool
}

// Lock acquires a lock on the specified path.
func (l *FileLocker) Lock(ctx context.Context, path string, opts LockOptions) (Lock, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create lock file if it doesn't exist
	if opts.CreateIfMissing {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create lock directory: %w", err)
		}

		// Touch the file
		f, err := os.OpenFile(absPath, os.O_CREATE|os.O_RDONLY, opts.Mode)
		if err != nil {
			return nil, fmt.Errorf("failed to create lock file: %w", err)
		}
		f.Close()
	}

	fl := flock.New(absPath)

	// Set up timeout context
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Try to acquire lock
	var locked bool
	if opts.Exclusive {
		locked, err = l.lockExclusive(ctx, fl, opts)
	} else {
		locked, err = l.lockShared(ctx, fl, opts)
	}

	if err != nil {
		return nil, err
	}

	if !locked {
		return nil, fmt.Errorf("failed to acquire lock on %s", absPath)
	}

	// Track the lock
	l.mu.Lock()
	lock := &fileLock{
		flock:     fl,
		path:      absPath,
		exclusive: opts.Exclusive,
	}
	l.locks[absPath] = lock
	l.mu.Unlock()

	return &lockHandle{
		locker: l,
		lock:   lock,
	}, nil
}

// lockExclusive attempts to acquire an exclusive lock.
func (l *FileLocker) lockExclusive(ctx context.Context, fl *flock.Flock, opts LockOptions) (bool, error) {
	if opts.Timeout == 0 {
		// Block indefinitely
		err := fl.Lock()
		return err == nil, err
	}

	// Try with retry
	ticker := time.NewTicker(opts.RetryInterval)
	defer ticker.Stop()

	for {
		locked, err := fl.TryLock()
		if err != nil {
			return false, err
		}
		if locked {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			// Continue trying
		}
	}
}

// lockShared attempts to acquire a shared lock.
func (l *FileLocker) lockShared(ctx context.Context, fl *flock.Flock, opts LockOptions) (bool, error) {
	if opts.Timeout == 0 {
		// Block indefinitely
		err := fl.RLock()
		return err == nil, err
	}

	// Try with retry
	ticker := time.NewTicker(opts.RetryInterval)
	defer ticker.Stop()

	for {
		locked, err := fl.TryRLock()
		if err != nil {
			return false, err
		}
		if locked {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			// Continue trying
		}
	}
}

// TryLock attempts to acquire a lock without blocking.
func (l *FileLocker) TryLock(ctx context.Context, path string, opts LockOptions) (Lock, bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create lock file if it doesn't exist
	if opts.CreateIfMissing {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, false, fmt.Errorf("failed to create lock directory: %w", err)
		}

		f, err := os.OpenFile(absPath, os.O_CREATE|os.O_RDONLY, opts.Mode)
		if err != nil {
			return nil, false, fmt.Errorf("failed to create lock file: %w", err)
		}
		f.Close()
	}

	fl := flock.New(absPath)

	var locked bool
	if opts.Exclusive {
		locked, err = fl.TryLock()
	} else {
		locked, err = fl.TryRLock()
	}

	if err != nil {
		return nil, false, err
	}

	if !locked {
		return nil, false, nil
	}

	// Track the lock
	l.mu.Lock()
	lock := &fileLock{
		flock:     fl,
		path:      absPath,
		exclusive: opts.Exclusive,
	}
	l.locks[absPath] = lock
	l.mu.Unlock()

	return &lockHandle{
		locker: l,
		lock:   lock,
	}, true, nil
}

// removeLock removes a lock from tracking.
func (l *FileLocker) removeLock(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.locks, path)
}

// lockHandle is the Lock implementation returned to callers.
type lockHandle struct {
	locker   *FileLocker
	lock     *fileLock
	unlocked bool
	mu       sync.Mutex
}

// Unlock releases the lock.
func (h *lockHandle) Unlock() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.unlocked {
		return nil
	}

	err := h.lock.flock.Unlock()
	h.locker.removeLock(h.lock.path)
	h.unlocked = true

	return err
}

// Path returns the path of the locked file.
func (h *lockHandle) Path() string {
	return h.lock.path
}

// IsExclusive returns true if this is an exclusive lock.
func (h *lockHandle) IsExclusive() bool {
	return h.lock.exclusive
}

// LockFile is a convenience function to lock a file for the duration of a function.
func LockFile(ctx context.Context, path string, exclusive bool, fn func() error) error {
	locker := NewFileLocker()
	opts := DefaultLockOptions()
	opts.Exclusive = exclusive

	lock, err := locker.Lock(ctx, path, opts)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	return fn()
}

// WithLock executes a function while holding a lock on the specified file.
// This is a higher-level convenience function that handles lock acquisition
// and release automatically.
func WithLock(ctx context.Context, path string, opts LockOptions, fn func() error) error {
	locker := NewFileLocker()

	lock, err := locker.Lock(ctx, path, opts)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	return fn()
}
