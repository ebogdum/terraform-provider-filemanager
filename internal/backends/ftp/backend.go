// SPDX-License-Identifier: MIT

// Package ftp provides an FTP/FTPS backend implementation for remote file operations.
package ftp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/jlaffaye/ftp"
)

// Backend implements the FTP/FTPS backend for remote file operations.
type Backend struct {
	config    plugin.BackendConfig
	connected bool
	basePath  string
	conn      *ftp.ServerConn
	mu        sync.RWMutex
}

// New creates a new FTP backend.
func New() *Backend {
	return &Backend{}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "ftp"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "ftp"
}

// Connect initializes the backend with the given configuration.
func (b *Backend) Connect(ctx context.Context, config plugin.BackendConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connected {
		return plugin.ErrAlreadyConnected
	}

	b.config = config
	b.basePath = config.BasePath

	// Determine host and port
	host := config.Host
	if host == "" {
		return fmt.Errorf("FTP host is required")
	}
	port := config.Port
	if port == 0 {
		port = 21
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// Set connection timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Build dial options
	var dialOpts []ftp.DialOption
	dialOpts = append(dialOpts, ftp.DialWithTimeout(timeout))
	dialOpts = append(dialOpts, ftp.DialWithContext(ctx))

	// TLS configuration
	if config.TLSEnabled {
		tlsConfig, err := buildTLSConfig(config)
		if err != nil {
			return fmt.Errorf("failed to build TLS config: %w", err)
		}
		if explicit, ok := config.Extra["explicit_tls"].(bool); ok && explicit {
			dialOpts = append(dialOpts, ftp.DialWithExplicitTLS(tlsConfig))
		} else {
			dialOpts = append(dialOpts, ftp.DialWithTLS(tlsConfig))
		}
	}

	if !config.TLSEnabled && config.Username != "" && config.Username != "anonymous" {
		log.Printf("[WARN] FTP connection to %s is using plaintext authentication. Credentials may be transmitted unencrypted. Consider enabling TLS.", config.Host)
	}

	// Passive mode (default)
	if passive, ok := config.Extra["passive_mode"].(bool); !ok || passive {
		dialOpts = append(dialOpts, ftp.DialWithDisabledEPSV(false))
	}

	// Connect to FTP server
	conn, err := ftp.Dial(addr, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP server %s: %w", addr, err)
	}

	// Login
	username := config.Username
	if username == "" {
		username = "anonymous"
	}
	password := config.Password
	if password == "" && username == "anonymous" {
		password = "anonymous@"
	}

	if err := conn.Login(username, password); err != nil {
		_ = conn.Quit()
		return fmt.Errorf("failed to login to FTP server: %w", err)
	}

	b.conn = conn
	b.connected = true
	return nil
}

// Close closes the backend connection.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil
	}

	var err error
	if b.conn != nil {
		err = b.conn.Quit()
		b.conn = nil
	}

	b.connected = false
	return err
}

// Ping checks if the backend is accessible.
func (b *Backend) Ping(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	// Try to get current directory as a connectivity check
	_, err := b.conn.CurrentDir()
	return err
}

// resolvePath resolves a relative path against the base path.
func (b *Backend) resolvePath(p string) (string, error) {
	p = path.Clean(p)

	var resolved string
	if path.IsAbs(p) {
		resolved = p
	} else if b.basePath != "" {
		resolved = path.Join(b.basePath, p)
	} else {
		return p, nil
	}

	// Enforce basePath containment
	if b.basePath != "" {
		if resolved != b.basePath && !strings.HasPrefix(resolved+"/", b.basePath+"/") {
			return "", fmt.Errorf("path %q escapes base path %q", p, b.basePath)
		}
	}

	return resolved, nil
}

// Read reads a file and returns an io.ReadCloser.
func (b *Backend) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(filePath)
	if err != nil {
		return nil, err
	}

	resp, err := b.conn.Retr(absPath)
	if err != nil {
		if isFTPNotFound(err) {
			return nil, plugin.ErrPathNotFound
		}
		if isFTPPermissionDenied(err) {
			return nil, plugin.ErrPermissionDenied
		}
		return nil, err
	}

	return resp, nil
}

// Write writes data to a file.
func (b *Backend) Write(ctx context.Context, filePath string, r io.Reader, opts plugin.WriteOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(filePath)
	if err != nil {
		return err
	}

	// Check if file exists and we're not overwriting
	if !opts.Overwrite {
		_, err := b.conn.FileSize(absPath)
		if err == nil {
			return plugin.ErrPathExists
		}
	}

	// Create parent directories if needed
	if opts.CreateDirs {
		dir := path.Dir(absPath)
		if err := b.mkdirAll(dir); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Store file
	if err := b.conn.Stor(absPath, r); err != nil {
		return err
	}

	return nil
}

// mkdirAll creates directories recursively.
func (b *Backend) mkdirAll(dirPath string) error {
	if dirPath == "/" || dirPath == "." || dirPath == "" {
		return nil
	}

	// Check if directory exists
	_, err := b.conn.List(dirPath)
	if err == nil {
		return nil // Directory exists
	}

	// Create parent first
	parent := path.Dir(dirPath)
	if parent != dirPath {
		if err := b.mkdirAll(parent); err != nil {
			return err
		}
	}

	// Create this directory
	return b.conn.MakeDir(dirPath)
}

// Delete deletes a file.
func (b *Backend) Delete(ctx context.Context, filePath string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(filePath)
	if err != nil {
		return err
	}

	// Check if it's a directory
	entries, err := b.conn.List(absPath)
	if err == nil && len(entries) == 1 && entries[0].Type == ftp.EntryTypeFolder {
		return plugin.ErrNotAFile
	}

	if err := b.conn.Delete(absPath); err != nil {
		if isFTPNotFound(err) {
			return plugin.ErrPathNotFound
		}
		if isFTPPermissionDenied(err) {
			return plugin.ErrPermissionDenied
		}
		return err
	}

	return nil
}

// Exists checks if a path exists.
func (b *Backend) Exists(ctx context.Context, filePath string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return false, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(filePath)
	if err != nil {
		return false, err
	}

	// Try to get file size (works for files)
	_, err = b.conn.FileSize(absPath)
	if err == nil {
		return true, nil
	}

	// Try to list (works for directories)
	_, err = b.conn.List(absPath)
	if err == nil {
		return true, nil
	}

	if isFTPNotFound(err) {
		return false, nil
	}
	return false, err
}

// Stat returns file information.
func (b *Backend) Stat(ctx context.Context, filePath string) (*plugin.FileInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(filePath)
	if err != nil {
		return nil, err
	}

	// Use LIST on parent directory to get file info
	parentDir := path.Dir(absPath)
	fileName := path.Base(absPath)

	entries, err := b.conn.List(parentDir)
	if err != nil {
		if isFTPNotFound(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.Name == fileName {
			fileInfo := &plugin.FileInfo{
				Name:    entry.Name,
				Path:    absPath,
				Size:    clampUint64ToInt64(entry.Size),
				ModTime: entry.Time,
				IsDir:   entry.Type == ftp.EntryTypeFolder,
			}

			// FTP doesn't provide Unix permissions reliably
			if entry.Type == ftp.EntryTypeFolder {
				fileInfo.Mode = os.ModeDir | 0755
			} else {
				fileInfo.Mode = 0644
			}

			// Check for symlink
			if entry.Type == ftp.EntryTypeLink {
				fileInfo.IsSymlink = true
				fileInfo.LinkTarget = entry.Target
			}

			return fileInfo, nil
		}
	}

	return nil, plugin.ErrPathNotFound
}

// List lists directory contents.
func (b *Backend) List(ctx context.Context, dirPath string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(dirPath)
	if err != nil {
		return nil, err
	}

	var results []plugin.FileInfo

	if opts.Recursive {
		err := b.walkDir(absPath, &results, opts)
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := b.conn.List(absPath)
		if err != nil {
			if isFTPNotFound(err) {
				return nil, plugin.ErrPathNotFound
			}
			return nil, err
		}

		for i, entry := range entries {
			if shouldSkipListEntry(entry, i, opts) {
				continue
			}

			results = append(results, ftpEntryToFileInfo(entry, absPath))

			// Check max results
			if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
				break
			}
		}
	}

	return results, nil
}

const maxRecursionDepth = 100

// walkDir recursively walks a directory.
func (b *Backend) walkDir(dirPath string, results *[]plugin.FileInfo, opts plugin.ListOptions, depth ...int) error {
	currentDepth := 0
	if len(depth) > 0 {
		currentDepth = depth[0]
	}
	if currentDepth > maxRecursionDepth {
		return fmt.Errorf("directory recursion depth exceeds maximum of %d", maxRecursionDepth)
	}

	entries, err := b.conn.List(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if shouldSkipRecursiveEntry(entry, opts) {
			continue
		}

		entryPath := path.Join(dirPath, entry.Name)
		*results = append(*results, ftpEntryToFileInfo(entry, dirPath))

		// Check max results
		if opts.MaxResults > 0 && len(*results) >= opts.MaxResults {
			return nil
		}

		// Recurse into directories
		if entry.Type == ftp.EntryTypeFolder {
			if err := b.walkDir(entryPath, results, opts, currentDepth+1); err != nil {
				continue // Skip errors in recursive listing
			}
		}
	}

	return nil
}

// Mkdir creates a directory.
func (b *Backend) Mkdir(ctx context.Context, dirPath string, opts plugin.MkdirOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(dirPath)
	if err != nil {
		return err
	}

	if opts.Recursive {
		return b.mkdirAll(absPath)
	}

	if err := b.conn.MakeDir(absPath); err != nil {
		if strings.Contains(err.Error(), "exists") {
			return plugin.ErrPathExists
		}
		if isFTPPermissionDenied(err) {
			return plugin.ErrPermissionDenied
		}
		return err
	}

	return nil
}

// Rmdir removes a directory.
func (b *Backend) Rmdir(ctx context.Context, dirPath string, recursive bool) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(dirPath)
	if err != nil {
		return err
	}

	if recursive {
		return b.removeAllRecursive(absPath)
	}

	if err := b.conn.RemoveDir(absPath); err != nil {
		if isFTPNotFound(err) {
			return plugin.ErrPathNotFound
		}
		if strings.Contains(err.Error(), "not empty") {
			return plugin.ErrDirNotEmpty
		}
		return err
	}

	return nil
}

// removeAllRecursive removes a directory and all its contents.
func (b *Backend) removeAllRecursive(dirPath string, depth ...int) error {
	currentDepth := 0
	if len(depth) > 0 {
		currentDepth = depth[0]
	}
	if currentDepth > maxRecursionDepth {
		return fmt.Errorf("directory recursion depth exceeds maximum of %d", maxRecursionDepth)
	}

	entries, err := b.conn.List(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		entryPath := path.Join(dirPath, entry.Name)
		if entry.Type == ftp.EntryTypeFolder {
			if err := b.removeAllRecursive(entryPath, currentDepth+1); err != nil {
				return err
			}
		} else {
			if err := b.conn.Delete(entryPath); err != nil {
				return err
			}
		}
	}

	return b.conn.RemoveDir(dirPath)
}

// Lock acquires a file lock.
// Note: FTP doesn't support file locking.
func (b *Backend) Lock(ctx context.Context, filePath string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	return nil, plugin.ErrNotSupported
}

// Symlink creates a symbolic link.
// Note: Standard FTP doesn't support symlink creation.
func (b *Backend) Symlink(ctx context.Context, target, link string) error {
	return plugin.ErrNotSupported
}

// Chmod changes file permissions.
// Note: Standard FTP doesn't support CHMOD in a portable way.
func (b *Backend) Chmod(ctx context.Context, filePath string, mode os.FileMode) error {
	// FTP protocol doesn't have a standard CHMOD command
	// Some servers support SITE CHMOD but it's not universal
	return plugin.ErrNotSupported
}

// Chown changes file ownership.
// Note: FTP doesn't support chown.
func (b *Backend) Chown(ctx context.Context, filePath string, uid, gid int) error {
	return plugin.ErrNotSupported
}

// Capabilities returns the backend's capabilities.
func (b *Backend) Capabilities() plugin.BackendCapabilities {
	return plugin.BackendCapabilities{
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
		SupportsLocking:  false, // No locking support
		SupportsSymlinks: false, // Limited symlink support
		SupportsChmod:    false, // No standard support
		SupportsChown:    false, // No chown support

		// Transfer optimizations
		SupportsRangeRead:        false,
		SupportsMultipartUpload:  false,
		SupportsDirectTransfer:   false,
		SupportsSendfile:         false,
		SupportsSplice:           false,
		SupportsCopyFileRange:    false,
		SupportsMemoryMapping:    false,
		SupportsStreamingWrite:   true,
		SupportsAtomicWrite:      false,
		SupportsConcurrentAccess: false, // FTP connections are typically single-threaded

		// Metadata
		SupportsMetadata:   false,
		SupportsVersioning: false,
		SupportsChecksum:   false,

		// Limits
		MaxFileSize:      0,    // Unlimited (server dependent)
		MaxPathLength:    4096, // Common limit
		MaxFilenameBytes: 255,
	}
}

// CopyFile copies a file (FTP doesn't have native remote copy, so we download and upload).
func (b *Backend) CopyFile(ctx context.Context, src, dst string, opts plugin.WriteOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

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

	// Download source file to memory
	resp, err := b.conn.Retr(absSrc)
	if err != nil {
		if isFTPNotFound(err) {
			return plugin.ErrPathNotFound
		}
		return err
	}

	// Read all content
	content, err := io.ReadAll(resp)
	resp.Close()
	if err != nil {
		return err
	}

	// Create parent directories if needed
	if opts.CreateDirs {
		dir := path.Dir(absDst)
		if err := b.mkdirAll(dir); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Upload to destination
	return b.conn.Stor(absDst, bytes.NewReader(content))
}

// Rename renames/moves a file on the FTP server.
func (b *Backend) Rename(ctx context.Context, src, dst string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

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

	return b.conn.Rename(absSrc, absDst)
}

// Helper functions

func isFTPNotFound(err error) bool {
	if nil == err {
		return false
	}
	errStr := strings.ToLower(err.Error())
	// Check explicit "not found" messages first, then fall back to 550
	// which is ambiguous (used for both not-found and permission-denied).
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such file") {
		return true
	}
	// Only treat 550 as not-found if it doesn't look like a permission error
	if strings.Contains(errStr, "550") && !strings.Contains(errStr, "permission") {
		return true
	}
	return false
}

func isFTPPermissionDenied(err error) bool {
	if nil == err {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "553") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "permission")
}

func clampUint64ToInt64(v uint64) int64 {
	if v > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(v)
}

func shouldSkipListEntry(entry *ftp.Entry, index int, opts plugin.ListOptions) bool {
	if isFTPTraversalEntry(entry.Name) {
		return true
	}
	if opts.Offset > 0 && index < opts.Offset {
		return true
	}
	return !matchesEntryFilters(entry, opts, false)
}

func shouldSkipRecursiveEntry(entry *ftp.Entry, opts plugin.ListOptions) bool {
	if isFTPTraversalEntry(entry.Name) {
		return true
	}
	return !matchesEntryFilters(entry, opts, true)
}

func isFTPTraversalEntry(name string) bool {
	return name == "." || name == ".."
}

func matchesEntryFilters(entry *ftp.Entry, opts plugin.ListOptions, allowDirPatternMiss bool) bool {
	if !opts.IncludeHidden && strings.HasPrefix(entry.Name, ".") {
		return false
	}
	if opts.Pattern == "" {
		return true
	}

	matched, err := path.Match(opts.Pattern, entry.Name)
	if err != nil {
		return false
	}
	if matched {
		return true
	}
	return allowDirPatternMiss && entry.Type == ftp.EntryTypeFolder
}

func ftpEntryToFileInfo(entry *ftp.Entry, parentPath string) plugin.FileInfo {
	fileInfo := plugin.FileInfo{
		Name:    entry.Name,
		Path:    path.Join(parentPath, entry.Name),
		Size:    clampUint64ToInt64(entry.Size),
		ModTime: entry.Time,
		IsDir:   entry.Type == ftp.EntryTypeFolder,
	}

	if entry.Type == ftp.EntryTypeFolder {
		fileInfo.Mode = os.ModeDir | 0755
	} else {
		fileInfo.Mode = 0644
	}

	if entry.Type == ftp.EntryTypeLink {
		fileInfo.IsSymlink = true
		fileInfo.LinkTarget = entry.Target
	}
	return fileInfo
}
