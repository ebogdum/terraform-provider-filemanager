// SPDX-License-Identifier: MIT

package acid

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileLocker_Lock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	locker := NewFileLocker()
	opts := DefaultLockOptions()

	lock, err := locker.Lock(context.Background(), path, opts)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	defer lock.Unlock()

	if lock.Path() != path {
		t.Errorf("Path mismatch: got %q, want %q", lock.Path(), path)
	}

	if !lock.IsExclusive() {
		t.Error("Expected exclusive lock")
	}
}

func TestFileLocker_TryLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	locker := NewFileLocker()
	opts := DefaultLockOptions()

	lock, acquired, err := locker.TryLock(context.Background(), path, opts)
	if err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}

	if !acquired {
		t.Fatal("Expected lock to be acquired")
	}

	defer lock.Unlock()
}

func TestFileLocker_SharedLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	locker := NewFileLocker()
	opts := DefaultLockOptions()
	opts.Exclusive = false // Shared lock

	lock, err := locker.Lock(context.Background(), path, opts)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	defer lock.Unlock()

	if lock.IsExclusive() {
		t.Error("Expected shared lock")
	}
}

func TestFileLocker_Timeout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	locker1 := NewFileLocker()
	locker2 := NewFileLocker()
	opts := DefaultLockOptions()

	// Acquire first lock
	lock1, err := locker1.Lock(context.Background(), path, opts)
	if err != nil {
		t.Fatalf("First lock failed: %v", err)
	}
	defer lock1.Unlock()

	// Try to acquire second lock with short timeout
	opts.Timeout = 100 * time.Millisecond
	_, acquired, err := locker2.TryLock(context.Background(), path, opts)
	if err != nil {
		t.Logf("TryLock returned error (expected on some platforms): %v", err)
	}

	if acquired {
		t.Error("Expected second lock to fail")
	}
}

func TestFileLocker_Concurrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	var wg sync.WaitGroup
	counter := 0
	iterations := 10

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			locker := NewFileLocker()
			opts := DefaultLockOptions()
			opts.Timeout = 5 * time.Second

			lock, err := locker.Lock(context.Background(), path, opts)
			if err != nil {
				t.Errorf("Lock failed: %v", err)
				return
			}
			defer lock.Unlock()

			// Critical section
			counter++
		}()
	}

	wg.Wait()

	if counter != iterations {
		t.Errorf("Counter mismatch: got %d, want %d", counter, iterations)
	}
}

func TestWithLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	executed := false
	err := WithLock(context.Background(), path, DefaultLockOptions(), func() error {
		executed = true
		return nil
	})

	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}

	if !executed {
		t.Error("Expected function to be executed")
	}
}
