// SPDX-License-Identifier: MIT

// Package ssh provides an SSH/SFTP backend implementation for remote file operations.
package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Backend implements the SSH/SFTP backend for remote file operations.
type Backend struct {
	config     plugin.BackendConfig
	connected  bool
	basePath   string
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	pool       *ConnectionPool
	mu         sync.RWMutex
}

// New creates a new SSH backend.
func New() *Backend {
	return &Backend{
		pool: NewConnectionPool(DefaultPoolConfig()),
	}
}

// NewWithConfig creates a new SSH backend with custom pool configuration.
func NewWithConfig(poolConfig PoolConfig) *Backend {
	return &Backend{
		pool: NewConnectionPool(poolConfig),
	}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "ssh"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "ssh"
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

	// Build SSH client configuration
	sshConfig, err := b.buildSSHConfig(config)
	if err != nil {
		return fmt.Errorf("failed to build SSH config: %w", err)
	}

	// Determine host and port
	host := config.Host
	if host == "" {
		return fmt.Errorf("SSH host is required")
	}
	port := config.Port
	if port == 0 {
		port = 22
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	// Set connection timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	sshConfig.Timeout = timeout

	// Connect to SSH server
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server %s: %w", addr, err)
	}
	b.sshClient = client

	// Create SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	b.sftpClient = sftpClient

	b.connected = true
	return nil
}

// buildSSHConfig builds an SSH client configuration from the backend config.
func (b *Backend) buildSSHConfig(config plugin.BackendConfig) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	// Try private key authentication first
	if len(config.PrivateKey) > 0 {
		signer, err := ParsePrivateKey(config.PrivateKey, config.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Password authentication
	if config.Password != "" && len(config.PrivateKey) == 0 {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	// Try SSH agent if available and no other auth methods
	if len(authMethods) == 0 {
		if agentAuth, err := SSHAgentAuth(); err == nil {
			authMethods = append(authMethods, agentAuth)
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available")
	}

	username := config.Username
	if username == "" {
		return nil, fmt.Errorf("SSH username is required")
	}

	// Build host key callback
	hostKeyCallback, err := b.buildHostKeyCallback(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build host key callback: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	return sshConfig, nil
}

// buildHostKeyCallback builds a host key callback based on configuration.
func (b *Backend) buildHostKeyCallback(config plugin.BackendConfig) (ssh.HostKeyCallback, error) {
	// Check for insecure mode (must be explicitly enabled)
	if insecure, ok := config.Extra["insecure_skip_host_key_verification"].(bool); ok && insecure {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// Check for explicit host key
	if hostKey, ok := config.Extra["host_key"].(string); ok && hostKey != "" {
		return HostKeyCallbackFromString(hostKey)
	}

	// Check for known_hosts file path
	if knownHostsPath, ok := config.Extra["known_hosts_file"].(string); ok && knownHostsPath != "" {
		return LoadKnownHostsFile(knownHostsPath)
	}

	// If TLS CA data provided (for known hosts data)
	if len(config.TLSCA) > 0 {
		callback, err := ParseKnownHosts(config.TLSCA)
		if err == nil {
			return callback, nil
		}
		// Fall through to default if parsing fails
	}

	// Use default known_hosts file (~/.ssh/known_hosts)
	return DefaultKnownHostsCallback()
}

// Close closes the backend connection.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil
	}

	var errs []error

	if b.sftpClient != nil {
		if err := b.sftpClient.Close(); err != nil {
			errs = append(errs, err)
		}
		b.sftpClient = nil
	}

	if b.sshClient != nil {
		if err := b.sshClient.Close(); err != nil {
			errs = append(errs, err)
		}
		b.sshClient = nil
	}

	b.connected = false

	if len(errs) > 0 {
		return fmt.Errorf("errors closing SSH connection: %v", errs)
	}
	return nil
}

// Ping checks if the backend is accessible.
func (b *Backend) Ping(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	// Try to get working directory as a simple connectivity check
	_, err := b.sftpClient.Getwd()
	return err
}

// resolvePath resolves a relative path against the base path.
func (b *Backend) resolvePath(p string) string {
	// Clean the path using POSIX path semantics
	p = path.Clean(p)

	// If absolute path, use as-is
	if path.IsAbs(p) {
		return p
	}

	// Join with base path
	if b.basePath != "" {
		return path.Join(b.basePath, p)
	}

	return p
}

// Read reads a file and returns an io.ReadCloser.
func (b *Backend) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath := b.resolvePath(filePath)

	file, err := b.sftpClient.Open(absPath)
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
func (b *Backend) Write(ctx context.Context, filePath string, r io.Reader, opts plugin.WriteOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath := b.resolvePath(filePath)

	// Check if file exists and we're not overwriting
	if !opts.Overwrite {
		if _, err := b.sftpClient.Stat(absPath); err == nil {
			return plugin.ErrPathExists
		}
	}

	// Create parent directories if needed
	if opts.CreateDirs {
		dir := path.Dir(absPath)
		// Note: SFTP MkdirAll doesn't support custom directory modes
		if err := b.sftpClient.MkdirAll(dir); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Determine file mode
	mode := opts.Mode
	if mode == 0 {
		mode = 0644
	}

	// Open file for writing
	file, err := b.sftpClient.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy content
	if _, err := io.Copy(file, r); err != nil {
		return err
	}

	// Set permissions
	if err := b.sftpClient.Chmod(absPath, mode); err != nil {
		// Non-fatal, some servers may not support this
		_ = err
	}

	return nil
}

// Delete deletes a file.
func (b *Backend) Delete(ctx context.Context, filePath string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath := b.resolvePath(filePath)

	info, err := b.sftpClient.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return plugin.ErrPathNotFound
		}
		return err
	}

	if info.IsDir() {
		return plugin.ErrNotAFile
	}

	if err := b.sftpClient.Remove(absPath); err != nil {
		if os.IsPermission(err) {
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

	absPath := b.resolvePath(filePath)

	_, err := b.sftpClient.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Stat returns file information.
func (b *Backend) Stat(ctx context.Context, filePath string) (*plugin.FileInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath := b.resolvePath(filePath)

	info, err := b.sftpClient.Lstat(absPath)
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
		target, err := b.sftpClient.ReadLink(absPath)
		if err == nil {
			fileInfo.LinkTarget = target
		}
	}

	// Try to get Unix-specific info from Sys()
	if sys := info.Sys(); sys != nil {
		if stat, ok := sys.(*sftp.FileStat); ok {
			fileInfo.UID = int(stat.UID)
			fileInfo.GID = int(stat.GID)
		}
	}

	return fileInfo, nil
}

// List lists directory contents.
func (b *Backend) List(ctx context.Context, dirPath string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	absPath := b.resolvePath(dirPath)

	info, err := b.sftpClient.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, plugin.ErrNotADirectory
	}

	var results []plugin.FileInfo

	if opts.Recursive {
		walker := b.sftpClient.Walk(absPath)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				continue
			}

			walkPath := walker.Path()
			info := walker.Stat()

			// Skip hidden files if not included
			if !opts.IncludeHidden && strings.HasPrefix(info.Name(), ".") {
				if info.IsDir() {
					walker.SkipDir()
				}
				continue
			}

			// Apply pattern filter
			if opts.Pattern != "" {
				matched, err := path.Match(opts.Pattern, info.Name())
				if err != nil || !matched {
					continue
				}
			}

			fileInfo := plugin.FileInfo{
				Name:    info.Name(),
				Path:    walkPath,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime(),
				IsDir:   info.IsDir(),
			}

			// Try to get Unix-specific info
			if sys := info.Sys(); sys != nil {
				if stat, ok := sys.(*sftp.FileStat); ok {
					fileInfo.UID = int(stat.UID)
					fileInfo.GID = int(stat.GID)
				}
			}

			results = append(results, fileInfo)

			// Check max results
			if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
				break
			}
		}
	} else {
		entries, err := b.sftpClient.ReadDir(absPath)
		if err != nil {
			return nil, err
		}

		for i, entry := range entries {
			// Apply offset
			if opts.Offset > 0 && i < opts.Offset {
				continue
			}

			// Skip hidden files if not included
			if !opts.IncludeHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			// Apply pattern filter
			if opts.Pattern != "" {
				matched, err := path.Match(opts.Pattern, entry.Name())
				if err != nil || !matched {
					continue
				}
			}

			fileInfo := plugin.FileInfo{
				Name:    entry.Name(),
				Path:    path.Join(absPath, entry.Name()),
				Size:    entry.Size(),
				Mode:    entry.Mode(),
				ModTime: entry.ModTime(),
				IsDir:   entry.IsDir(),
			}

			// Try to get Unix-specific info
			if sys := entry.Sys(); sys != nil {
				if stat, ok := sys.(*sftp.FileStat); ok {
					fileInfo.UID = int(stat.UID)
					fileInfo.GID = int(stat.GID)
				}
			}

			results = append(results, fileInfo)

			// Check max results
			if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
				break
			}
		}
	}

	return results, nil
}

// Mkdir creates a directory.
func (b *Backend) Mkdir(ctx context.Context, dirPath string, opts plugin.MkdirOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath := b.resolvePath(dirPath)

	mode := opts.Mode
	if mode == 0 {
		mode = 0755
	}

	if opts.Recursive {
		return b.sftpClient.MkdirAll(absPath)
	}

	if err := b.sftpClient.Mkdir(absPath); err != nil {
		if os.IsExist(err) {
			return plugin.ErrPathExists
		}
		if os.IsPermission(err) {
			return plugin.ErrPermissionDenied
		}
		return err
	}

	// Try to set permissions (may fail on some servers)
	_ = b.sftpClient.Chmod(absPath, mode)

	return nil
}

// Rmdir removes a directory.
func (b *Backend) Rmdir(ctx context.Context, dirPath string, recursive bool) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath := b.resolvePath(dirPath)

	info, err := b.sftpClient.Stat(absPath)
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
		return b.removeAllRecursive(absPath)
	}

	if err := b.sftpClient.RemoveDirectory(absPath); err != nil {
		if strings.Contains(err.Error(), "not empty") ||
			strings.Contains(err.Error(), "directory not empty") {
			return plugin.ErrDirNotEmpty
		}
		return err
	}

	return nil
}

// removeAllRecursive removes a directory and all its contents.
func (b *Backend) removeAllRecursive(dirPath string) error {
	entries, err := b.sftpClient.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := path.Join(dirPath, entry.Name())
		if entry.IsDir() {
			if err := b.removeAllRecursive(entryPath); err != nil {
				return err
			}
		} else {
			if err := b.sftpClient.Remove(entryPath); err != nil {
				return err
			}
		}
	}

	return b.sftpClient.RemoveDirectory(dirPath)
}

// Lock acquires a file lock.
// Note: SSH/SFTP doesn't natively support file locking, so we use advisory locking via lock files.
func (b *Backend) Lock(ctx context.Context, filePath string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	// SSH/SFTP doesn't support native file locking
	// Return ErrNotSupported or implement advisory locking via .lock files
	return nil, plugin.ErrNotSupported
}

// Symlink creates a symbolic link.
func (b *Backend) Symlink(ctx context.Context, target, link string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absLink := b.resolvePath(link)

	return b.sftpClient.Symlink(target, absLink)
}

// Chmod changes file permissions.
func (b *Backend) Chmod(ctx context.Context, filePath string, mode os.FileMode) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath := b.resolvePath(filePath)

	return b.sftpClient.Chmod(absPath, mode)
}

// Chown changes file ownership.
func (b *Backend) Chown(ctx context.Context, filePath string, uid, gid int) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath := b.resolvePath(filePath)

	return b.sftpClient.Chown(absPath, uid, gid)
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
		SupportsLocking:  false, // No native locking support
		SupportsSymlinks: true,
		SupportsChmod:    true,
		SupportsChown:    true,

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
		SupportsConcurrentAccess: true,

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

// Execute runs a command on the remote system and returns the output.
// This implements the plugin.CommandExecutor interface.
func (b *Backend) Execute(ctx context.Context, command string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	session, err := b.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// CopyFile copies a file remotely (SFTP doesn't have native remote copy, so we stream).
func (b *Backend) CopyFile(ctx context.Context, src, dst string, opts plugin.WriteOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absSrc := b.resolvePath(src)
	absDst := b.resolvePath(dst)

	// Open source file
	srcFile, err := b.sftpClient.Open(absSrc)
	if err != nil {
		if os.IsNotExist(err) {
			return plugin.ErrPathNotFound
		}
		return err
	}
	defer srcFile.Close()

	// Create parent directories if needed
	if opts.CreateDirs {
		dir := path.Dir(absDst)
		if err := b.sftpClient.MkdirAll(dir); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Create destination file
	mode := opts.Mode
	if mode == 0 {
		// Try to preserve source mode
		if srcInfo, err := srcFile.Stat(); err == nil {
			mode = srcInfo.Mode()
		} else {
			mode = 0644
		}
	}

	dstFile, err := b.sftpClient.OpenFile(absDst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Set permissions
	_ = b.sftpClient.Chmod(absDst, mode)

	return nil
}
