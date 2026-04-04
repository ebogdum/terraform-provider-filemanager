// SPDX-License-Identifier: MIT

// Package ssh provides an SSH/SFTP backend implementation for remote file operations.
package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Backend implements the SSH/SFTP backend for remote file operations.
type Backend struct {
	config      plugin.BackendConfig
	connected   bool
	basePath    string
	sshClient   *ssh.Client
	sftpClient  *sftp.Client
	pool        *ConnectionPool
	agentCloser io.Closer // SSH agent connection; closed on backend Close
	mu          sync.RWMutex
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
	if "" == host {
		return fmt.Errorf("SSH host is required")
	}
	port := config.Port
	if 0 == port {
		port = 22
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// Set connection timeout
	timeout := config.Timeout
	if 0 == timeout {
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
		// Prefer dedicated passphrase from Extra map over the Password field
		passphrase := config.Password
		if pp, ok := config.Extra["passphrase"].(string); ok && "" != pp {
			passphrase = pp
		}
		signer, err := ParsePrivateKey(config.PrivateKey, passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Password authentication
	if "" != config.Password && 0 == len(config.PrivateKey) {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	// Try SSH agent if available and no other auth methods
	if 0 == len(authMethods) {
		if agentAuth, agentConn, err := SSHAgentAuth(); nil == err {
			authMethods = append(authMethods, agentAuth)
			b.agentCloser = agentConn
		}
	}

	if 0 == len(authMethods) {
		return nil, fmt.Errorf("no authentication method available")
	}

	username := config.Username
	if "" == username {
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
	// Insecure host key bypass is intentionally unsupported.
	if insecure, ok := config.Extra["insecure_skip_host_key_verification"].(bool); ok && insecure {
		return nil, fmt.Errorf("insecure_skip_host_key_verification is not supported")
	}

	// Check for explicit host key
	if hostKey, ok := config.Extra["host_key"].(string); ok && "" != hostKey {
		return HostKeyCallbackFromString(hostKey)
	}

	// Check for known_hosts file path
	if knownHostsPath, ok := config.Extra["known_hosts_file"].(string); ok && "" != knownHostsPath {
		return LoadKnownHostsFile(knownHostsPath)
	}

	// If TLS CA data provided (for known hosts data)
	if len(config.TLSCA) > 0 {
		callback, err := ParseKnownHosts(config.TLSCA)
		if nil == err {
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

	if nil != b.sftpClient {
		if err := b.sftpClient.Close(); nil != err {
			errs = append(errs, err)
		}
		b.sftpClient = nil
	}

	if nil != b.sshClient {
		if err := b.sshClient.Close(); nil != err {
			errs = append(errs, err)
		}
		b.sshClient = nil
	}

	if nil != b.agentCloser {
		if err := b.agentCloser.Close(); nil != err {
			errs = append(errs, err)
		}
		b.agentCloser = nil
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
func (b *Backend) resolvePath(p string) (string, error) {
	p = path.Clean(p)

	var resolved string
	if path.IsAbs(p) {
		resolved = p
	} else if "" != b.basePath {
		resolved = path.Join(b.basePath, p)
	} else {
		return p, nil
	}

	// Enforce basePath containment
	if "" != b.basePath {
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
	if nil != err {
		return nil, err
	}

	file, err := b.sftpClient.Open(absPath)
	if nil != err {
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

	absPath, err := b.resolvePath(filePath)
	if nil != err {
		return err
	}

	// Check if file exists and we're not overwriting
	if !opts.Overwrite {
		if _, err := b.sftpClient.Stat(absPath); nil == err {
			return plugin.ErrPathExists
		}
	}

	// Create parent directories if needed
	if opts.CreateDirs {
		dir := path.Dir(absPath)
		// Note: SFTP MkdirAll doesn't support custom directory modes
		if err := b.sftpClient.MkdirAll(dir); nil != err {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Determine file mode
	mode := opts.Mode
	if 0 == mode {
		mode = 0644
	}

	// Open file for writing
	file, err := b.sftpClient.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if nil != err {
		return err
	}
	defer file.Close()

	// Copy content
	if _, err := io.Copy(file, r); nil != err {
		return err
	}

	// Set permissions
	if err := b.sftpClient.Chmod(absPath, mode); nil != err {
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

	absPath, err := b.resolvePath(filePath)
	if nil != err {
		return err
	}

	info, err := b.sftpClient.Stat(absPath)
	if nil != err {
		if os.IsNotExist(err) {
			return plugin.ErrPathNotFound
		}
		return err
	}

	if info.IsDir() {
		return plugin.ErrNotAFile
	}

	if err := b.sftpClient.Remove(absPath); nil != err {
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

	absPath, err := b.resolvePath(filePath)
	if nil != err {
		return false, err
	}

	_, err = b.sftpClient.Stat(absPath)
	if nil != err {
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

	absPath, err := b.resolvePath(filePath)
	if nil != err {
		return nil, err
	}

	info, err := b.sftpClient.Lstat(absPath)
	if nil != err {
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
		if nil == err {
			fileInfo.LinkTarget = target
		}
	}

	// Try to get Unix-specific info from Sys()
	if sys := info.Sys(); nil != sys {
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

	absPath, err := b.resolvePath(dirPath)
	if nil != err {
		return nil, err
	}

	info, err := b.sftpClient.Stat(absPath)
	if nil != err {
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
	walker := b.sftpClient.Walk(absPath)
	for walker.Step() {
		if err := walker.Err(); nil != err {
			continue
		}

		info := walker.Stat()
		if shouldSkipSSHHidden(info.Name(), opts.IncludeHidden) {
			if info.IsDir() {
				walker.SkipDir()
			}
			continue
		}
		if !matchesSSHPattern(info.Name(), opts.Pattern) {
			continue
		}

		results = append(results, toSSHFileInfo(walker.Path(), info))
		if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
			break
		}
	}
	return results, nil
}

func (b *Backend) listShallow(absPath string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	entries, err := b.sftpClient.ReadDir(absPath)
	if nil != err {
		return nil, err
	}

	results := make([]plugin.FileInfo, 0, len(entries))
	for i, entry := range entries {
		if opts.Offset > 0 && i < opts.Offset {
			continue
		}
		if shouldSkipSSHHidden(entry.Name(), opts.IncludeHidden) {
			continue
		}
		if !matchesSSHPattern(entry.Name(), opts.Pattern) {
			continue
		}

		results = append(results, toSSHFileInfo(path.Join(absPath, entry.Name()), entry))
		if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
			break
		}
	}
	return results, nil
}

func shouldSkipSSHHidden(name string, includeHidden bool) bool {
	return !includeHidden && strings.HasPrefix(name, ".")
}

func matchesSSHPattern(name, pattern string) bool {
	if "" == pattern {
		return true
	}
	matched, err := path.Match(pattern, name)
	return nil == err && matched
}

func toSSHFileInfo(filePath string, info os.FileInfo) plugin.FileInfo {
	fileInfo := plugin.FileInfo{
		Name:    info.Name(),
		Path:    filePath,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
	if sys := info.Sys(); nil != sys {
		if stat, ok := sys.(*sftp.FileStat); ok {
			fileInfo.UID = int(stat.UID)
			fileInfo.GID = int(stat.GID)
		}
	}
	return fileInfo
}

// Mkdir creates a directory.
func (b *Backend) Mkdir(ctx context.Context, dirPath string, opts plugin.MkdirOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(dirPath)
	if nil != err {
		return err
	}

	mode := opts.Mode
	if 0 == mode {
		mode = 0755
	}

	if opts.Recursive {
		return b.sftpClient.MkdirAll(absPath)
	}

	if err := b.sftpClient.Mkdir(absPath); nil != err {
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

	absPath, err := b.resolvePath(dirPath)
	if nil != err {
		return err
	}

	info, err := b.sftpClient.Stat(absPath)
	if nil != err {
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

	if err := b.sftpClient.RemoveDirectory(absPath); nil != err {
		if strings.Contains(err.Error(), "not empty") ||
			strings.Contains(err.Error(), "directory not empty") {
			return plugin.ErrDirNotEmpty
		}
		return err
	}

	return nil
}

const maxRemoveDepth = 100

// removeAllRecursive removes a directory and all its contents.
func (b *Backend) removeAllRecursive(dirPath string, depth ...int) error {
	currentDepth := 0
	if len(depth) > 0 {
		currentDepth = depth[0]
	}
	if currentDepth > maxRemoveDepth {
		return fmt.Errorf("directory recursion depth exceeds maximum of %d", maxRemoveDepth)
	}

	entries, err := b.sftpClient.ReadDir(dirPath)
	if nil != err {
		return err
	}

	for _, entry := range entries {
		entryPath := path.Join(dirPath, entry.Name())
		// Skip symlinks to avoid following them into loops or outside the tree
		if entry.Mode()&os.ModeSymlink != 0 {
			if err := b.sftpClient.Remove(entryPath); nil != err {
				return err
			}
			continue
		}
		if entry.IsDir() {
			if err := b.removeAllRecursive(entryPath, currentDepth+1); nil != err {
				return err
			}
		} else {
			if err := b.sftpClient.Remove(entryPath); nil != err {
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

	absLink, err := b.resolvePath(link)
	if nil != err {
		return err
	}

	return b.sftpClient.Symlink(target, absLink)
}

// Chmod changes file permissions.
func (b *Backend) Chmod(ctx context.Context, filePath string, mode os.FileMode) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(filePath)
	if nil != err {
		return err
	}

	return b.sftpClient.Chmod(absPath, mode)
}

// Chown changes file ownership.
func (b *Backend) Chown(ctx context.Context, filePath string, uid, gid int) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absPath, err := b.resolvePath(filePath)
	if nil != err {
		return err
	}

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
// Commands containing shell metacharacters are rejected to prevent injection.
// Only simple commands without piping, chaining, or variable expansion are allowed.
func (b *Backend) Execute(ctx context.Context, command string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	// Reject commands with shell metacharacters to prevent injection
	if containsShellMetachars(command) {
		return nil, fmt.Errorf("command contains disallowed shell metacharacters")
	}

	session, err := b.sshClient.NewSession()
	if nil != err {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if nil != err {
		return output, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// containsShellMetachars returns true if the command string contains any shell
// metacharacters that could be used for command injection.
func containsShellMetachars(cmd string) bool {
	for _, c := range cmd {
		switch c {
		case ';', '|', '&', '$', '`', '\n', '\r', '(', ')', '{', '}', '<', '>', '!':
			return true
		}
	}
	return false
}

// CopyFile copies a file remotely (SFTP doesn't have native remote copy, so we stream).
func (b *Backend) CopyFile(ctx context.Context, src, dst string, opts plugin.WriteOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	absSrc, err := b.resolvePath(src)
	if nil != err {
		return err
	}
	absDst, err := b.resolvePath(dst)
	if nil != err {
		return err
	}

	// Open source file
	srcFile, err := b.sftpClient.Open(absSrc)
	if nil != err {
		if os.IsNotExist(err) {
			return plugin.ErrPathNotFound
		}
		return err
	}
	defer srcFile.Close()

	// Create parent directories if needed
	if opts.CreateDirs {
		dir := path.Dir(absDst)
		if err := b.sftpClient.MkdirAll(dir); nil != err {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Create destination file
	mode := opts.Mode
	if 0 == mode {
		// Try to preserve source mode
		if srcInfo, err := srcFile.Stat(); nil == err {
			mode = srcInfo.Mode()
		} else {
			mode = 0644
		}
	}

	dstFile, err := b.sftpClient.OpenFile(absDst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if nil != err {
		return err
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); nil != err {
		return err
	}

	// Set permissions
	_ = b.sftpClient.Chmod(absDst, mode)

	return nil
}
