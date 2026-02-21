// SPDX-License-Identifier: MIT

// Package local provides a local filesystem backend implementation.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ebogdum/filemanager/internal/acid"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Backend implements the local filesystem backend.
type Backend struct {
	config       plugin.BackendConfig
	connected    bool
	basePath     string
	atomicWriter *acid.AtomicWriter
	locker       *acid.FileLocker
	checksummer  *acid.FileChecksummer
	zeroCopyImpl *ZeroCopy
}

// New creates a new local backend.
func New() *Backend {
	return &Backend{
		atomicWriter: acid.NewAtomicWriter(),
		locker:       acid.NewFileLocker(),
		checksummer:  acid.NewChecksummer(),
		zeroCopyImpl: NewZeroCopy(),
	}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "local"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "file"
}

// Connect initializes the backend with the given configuration.
func (b *Backend) Connect(ctx context.Context, config plugin.BackendConfig) error {
	if b.connected {
		return plugin.ErrAlreadyConnected
	}

	b.config = config
	b.basePath = config.BasePath

	// Validate base path if set
	if b.basePath != "" {
		absPath, err := filepath.Abs(b.basePath)
		if err != nil {
			return fmt.Errorf("invalid base path: %w", err)
		}
		b.basePath = absPath

		// Check if base path exists (optional - could create it)
		info, err := os.Stat(b.basePath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat base path: %w", err)
		}
		if err == nil && !info.IsDir() {
			return fmt.Errorf("base path is not a directory: %s", b.basePath)
		}
	}

	b.connected = true
	return nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	b.connected = false
	return nil
}

// Ping checks if the backend is accessible.
func (b *Backend) Ping(ctx context.Context) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	if b.basePath != "" {
		_, err := os.Stat(b.basePath)
		return err
	}

	return nil
}

// resolvePath resolves a relative path against the base path.
func (b *Backend) resolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	if b.basePath != "" {
		return filepath.Join(b.basePath, path), nil
	}

	return filepath.Abs(path)
}

// Read reads a file and returns an io.ReadCloser.
func (b *Backend) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, plugin.ErrPathNotFound
		}
		if os.IsPermission(err) {
			return nil, plugin.ErrPermissionDenied
		}
		return nil, err
	}

	return file, nil
}

// Write writes data to a file.
func (b *Backend) Write(ctx context.Context, path string, r io.Reader, opts plugin.WriteOptions) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return err
	}

	// Check if file exists and we're not overwriting
	if !opts.Overwrite {
		if _, err := os.Stat(absPath); err == nil {
			return plugin.ErrPathExists
		}
	}

	// Use atomic writer for safe writes
	writeOpts := acid.WriteOptions{
		Mode:             opts.Mode,
		DirMode:          opts.DirMode,
		CreateDirs:       opts.CreateDirs,
		Sync:             true,
		SyncDir:          true,
		VerifyAfterWrite: opts.VerifyAfterWrite,
		ExpectedChecksum: opts.Checksum,
		ChecksumAlgo:     opts.ChecksumAlgo,
		UID:              -1, // -1 means don't change ownership
		GID:              -1,
	}

	if opts.Mode == 0 {
		writeOpts.Mode = 0644
	}
	if opts.DirMode == 0 {
		writeOpts.DirMode = 0755
	}

	if opts.Atomic {
		return b.atomicWriter.WriteFrom(ctx, absPath, r, writeOpts)
	}

	// Non-atomic write
	return b.directWrite(ctx, absPath, r, writeOpts)
}

// directWrite performs a direct (non-atomic) write.
func (b *Backend) directWrite(ctx context.Context, path string, r io.Reader, opts acid.WriteOptions) error {
	if opts.CreateDirs {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, opts.DirMode); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	file, err := os.OpenFile(path, flags, opts.Mode)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return err
	}

	if opts.Sync {
		return file.Sync()
	}

	return nil
}

// Delete deletes a file.
func (b *Backend) Delete(ctx context.Context, path string) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return plugin.ErrPathNotFound
		}
		return err
	}

	if info.IsDir() {
		return plugin.ErrNotAFile
	}

	if err := os.Remove(absPath); err != nil {
		if os.IsPermission(err) {
			return plugin.ErrPermissionDenied
		}
		return err
	}

	return nil
}

// Exists checks if a path exists.
func (b *Backend) Exists(ctx context.Context, path string) (bool, error) {
	if !b.connected {
		return false, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Stat returns file information.
func (b *Backend) Stat(ctx context.Context, path string) (*plugin.FileInfo, error) {
	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, err
	}

	fileInfo := &plugin.FileInfo{
		Name:    info.Name(),
		Path:    absPath,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}

	// Check for symlink
	if info.Mode()&os.ModeSymlink != 0 {
		fileInfo.IsSymlink = true
		target, err := os.Readlink(absPath)
		if err == nil {
			fileInfo.LinkTarget = target
		}
	}

	// Get Unix-specific info
	b.fillUnixInfo(info, fileInfo)

	return fileInfo, nil
}

// List lists directory contents.
func (b *Backend) List(ctx context.Context, path string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, plugin.ErrNotADirectory
	}

	if opts.Recursive {
		return b.listRecursive(absPath, opts)
	}

	return b.listShallow(absPath, opts)
}

func (b *Backend) listRecursive(absPath string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	results := make([]plugin.FileInfo, 0)
	err := filepath.WalkDir(absPath, func(walkPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if shouldSkipLocalEntry(d.Name(), opts) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesLocalPattern(d.Name(), opts.Pattern) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		results = append(results, b.toLocalFileInfo(info, walkPath))
		if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (b *Backend) listShallow(absPath string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	results := make([]plugin.FileInfo, 0, len(entries))
	for i, entry := range entries {
		if opts.Offset > 0 && i < opts.Offset {
			continue
		}
		if shouldSkipLocalEntry(entry.Name(), opts) {
			continue
		}
		if !matchesLocalPattern(entry.Name(), opts.Pattern) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		results = append(results, b.toLocalFileInfo(info, filepath.Join(absPath, info.Name())))
		if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
			break
		}
	}

	return results, nil
}

func (b *Backend) toLocalFileInfo(info os.FileInfo, filePath string) plugin.FileInfo {
	fileInfo := plugin.FileInfo{
		Name:    info.Name(),
		Path:    filePath,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
	b.fillUnixInfo(info, &fileInfo)
	return fileInfo
}

func shouldSkipLocalEntry(name string, opts plugin.ListOptions) bool {
	if opts.IncludeHidden || !strings.HasPrefix(name, ".") {
		return false
	}
	return true
}

func matchesLocalPattern(name, pattern string) bool {
	if pattern == "" {
		return true
	}
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

// Mkdir creates a directory.
func (b *Backend) Mkdir(ctx context.Context, path string, opts plugin.MkdirOptions) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return err
	}

	mode := opts.Mode
	if mode == 0 {
		mode = 0755
	}

	if opts.Recursive {
		return os.MkdirAll(absPath, mode)
	}

	if err := os.Mkdir(absPath, mode); err != nil {
		if os.IsExist(err) {
			return plugin.ErrPathExists
		}
		if os.IsPermission(err) {
			return plugin.ErrPermissionDenied
		}
		return err
	}

	return nil
}

// Rmdir removes a directory.
func (b *Backend) Rmdir(ctx context.Context, path string, recursive bool) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return plugin.ErrPathNotFound
		}
		return err
	}

	if !info.IsDir() {
		return plugin.ErrNotADirectory
	}

	if recursive {
		return os.RemoveAll(absPath)
	}

	if err := os.Remove(absPath); err != nil {
		if strings.Contains(err.Error(), "not empty") ||
			strings.Contains(err.Error(), "directory not empty") {
			return plugin.ErrDirNotEmpty
		}
		return err
	}

	return nil
}

// Lock acquires a file lock.
func (b *Backend) Lock(ctx context.Context, path string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return nil, err
	}

	lockOpts := acid.LockOptions{
		Exclusive:       opts.Exclusive,
		Timeout:         opts.Timeout,
		RetryInterval:   100 * time.Millisecond,
		CreateIfMissing: true,
		Mode:            0644,
	}

	return b.locker.Lock(ctx, absPath, lockOpts)
}

// Symlink creates a symbolic link.
func (b *Backend) Symlink(ctx context.Context, target, link string) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	absLink, err := b.resolvePath(link)
	if err != nil {
		return err
	}

	// Target is used as-is - resolution is handled by the caller
	return os.Symlink(target, absLink)
}

// Chmod changes file permissions.
func (b *Backend) Chmod(ctx context.Context, path string, mode os.FileMode) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return err
	}

	return os.Chmod(absPath, mode)
}

// Chown changes file ownership.
func (b *Backend) Chown(ctx context.Context, path string, uid, gid int) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	if runtime.GOOS == "windows" {
		return plugin.ErrNotSupported
	}

	absPath, err := b.resolvePath(path)
	if err != nil {
		return err
	}

	return os.Chown(absPath, uid, gid)
}

// Capabilities returns the backend's capabilities.
func (b *Backend) Capabilities() plugin.BackendCapabilities {
	caps := plugin.BackendCapabilities{
		// File operations
		SupportsRead:   true,
		SupportsWrite:  true,
		SupportsDelete: true,
		SupportsStat:   true,

		// Directory operations
		SupportsList:  true,
		SupportsMkdir: true,
		SupportsRmdir: true,

		// Advanced operations
		SupportsLocking:  true,
		SupportsSymlinks: runtime.GOOS != "windows",
		SupportsChmod:    true,
		SupportsChown:    runtime.GOOS != "windows",

		// Transfer optimizations
		SupportsRangeRead:        true,
		SupportsMultipartUpload:  false,
		SupportsDirectTransfer:   true,
		SupportsStreamingWrite:   true,
		SupportsAtomicWrite:      true,
		SupportsConcurrentAccess: true,

		// Metadata
		SupportsMetadata:   false,
		SupportsVersioning: false,
		SupportsChecksum:   true,

		// Limits
		MaxFileSize:      0, // Unlimited (filesystem dependent)
		MaxPathLength:    4096,
		MaxFilenameBytes: 255,
	}

	// Add zero-copy capabilities based on platform
	b.addZeroCopyCapabilities(&caps)

	return caps
}

// addZeroCopyCapabilities adds platform-specific zero-copy capabilities.
func (b *Backend) addZeroCopyCapabilities(caps *plugin.BackendCapabilities) {
	caps.SupportsSendfile = b.zeroCopyImpl.SupportsSendfile()
	caps.SupportsSplice = b.zeroCopyImpl.SupportsSplice()
	caps.SupportsCopyFileRange = b.zeroCopyImpl.SupportsCopyFileRange()
	caps.SupportsMemoryMapping = true
}

// CopyFile copies a file using the most efficient method available.
func (b *Backend) CopyFile(ctx context.Context, src, dst string, opts plugin.WriteOptions) error {
	if !b.connected {
		return plugin.ErrNotConnected
	}

	absSrc, err := b.resolvePath(src)
	if err != nil {
		return err
	}

	absDst, err := b.resolvePath(dst)
	if err != nil {
		return err
	}

	// Try zero-copy methods first
	if err := b.zeroCopyImpl.CopyFile(ctx, absSrc, absDst); err == nil {
		return nil
	}

	// Fall back to regular copy
	srcFile, err := os.Open(absSrc)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	return b.Write(ctx, dst, srcFile, opts)
}
