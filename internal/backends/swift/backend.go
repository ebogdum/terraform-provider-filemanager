// SPDX-License-Identifier: MIT

// Package swift provides an OpenStack Swift backend implementation for object storage.
package swift

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
	"github.com/ncw/swift/v2"
)

// Backend implements the OpenStack Swift backend for object storage operations.
type Backend struct {
	config    plugin.BackendConfig
	connected bool
	basePath  string
	container string
	conn      *swift.Connection
	mu        sync.RWMutex
}

// New creates a new Swift backend.
func New() *Backend {
	return &Backend{}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "swift"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "swift"
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

	// Get container name from Extra
	container, ok := config.Extra["container"].(string)
	if !ok || container == "" {
		return fmt.Errorf("swift container name is required")
	}
	b.container = container

	// Get auth URL
	authURL := config.Endpoint
	if authURL == "" {
		if url, ok := config.Extra["auth_url"].(string); ok {
			authURL = url
		}
	}
	if authURL == "" {
		return fmt.Errorf("swift auth URL is required")
	}

	// Set connection timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Build Swift connection
	conn := &swift.Connection{
		AuthUrl:  authURL,
		UserName: config.Username,
		ApiKey:   config.Password,
		Timeout:  timeout,
	}

	// Optional configuration
	if tenant, ok := config.Extra["tenant"].(string); ok {
		conn.Tenant = tenant
	}
	if tenantID, ok := config.Extra["tenant_id"].(string); ok {
		conn.TenantId = tenantID
	}
	if domain, ok := config.Extra["domain"].(string); ok {
		conn.Domain = domain
	}
	if domainID, ok := config.Extra["domain_id"].(string); ok {
		conn.DomainId = domainID
	}
	if region, ok := config.Extra["region"].(string); ok {
		conn.Region = region
	}
	if authVersion, ok := config.Extra["auth_version"].(int); ok {
		conn.AuthVersion = authVersion
	}

	// Use application credential if provided
	if appCredID, ok := config.Extra["application_credential_id"].(string); ok {
		conn.ApplicationCredentialId = appCredID
	}
	if appCredSecret, ok := config.Extra["application_credential_secret"].(string); ok {
		conn.ApplicationCredentialSecret = appCredSecret
	}

	// Token-based authentication
	if config.Token != "" {
		conn.AuthToken = config.Token
		if storageURL, ok := config.Extra["storage_url"].(string); ok {
			conn.StorageUrl = storageURL
		}
	}

	// Authenticate
	if err := conn.Authenticate(ctx); err != nil {
		return fmt.Errorf("failed to authenticate with Swift: %w", err)
	}

	// Verify container exists (or create if specified)
	_, _, err := conn.Container(ctx, container)
	if err != nil {
		if err == swift.ContainerNotFound {
			if createContainer, ok := config.Extra["create_container"].(bool); ok && createContainer {
				if err := conn.ContainerCreate(ctx, container, nil); err != nil {
					return fmt.Errorf("failed to create container %s: %w", container, err)
				}
			} else {
				return fmt.Errorf("container %s not found", container)
			}
		} else {
			return fmt.Errorf("failed to access container %s: %w", container, err)
		}
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

	// Swift connections are stateless, just mark as disconnected
	b.conn = nil
	b.connected = false
	return nil
}

// Ping checks if the backend is accessible.
func (b *Backend) Ping(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	// Check if container is accessible
	_, _, err := b.conn.Container(ctx, b.container)
	return err
}

// resolvePath resolves a relative path against the base path and normalizes for Swift.
func (b *Backend) resolvePath(p string) string {
	p = path.Clean(p)
	// Remove leading slash for Swift objects
	p = strings.TrimPrefix(p, "/")
	if b.basePath != "" {
		basePath := strings.TrimPrefix(b.basePath, "/")
		if basePath != "" {
			return basePath + "/" + p
		}
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

	objectPath := b.resolvePath(filePath)

	contents, _, err := b.conn.ObjectOpen(ctx, b.container, objectPath, false, nil)
	if err != nil {
		if err == swift.ObjectNotFound {
			return nil, plugin.ErrPathNotFound
		}
		return nil, err
	}

	return contents, nil
}

// Write writes data to a file.
func (b *Backend) Write(ctx context.Context, filePath string, r io.Reader, opts plugin.WriteOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	objectPath := b.resolvePath(filePath)

	// Check if object exists and we're not overwriting
	if !opts.Overwrite {
		_, _, err := b.conn.Object(ctx, b.container, objectPath)
		if err == nil {
			return plugin.ErrPathExists
		}
		if err != swift.ObjectNotFound {
			return err
		}
	}

	// Create parent "directories" if needed (pseudo-directories in Swift)
	if opts.CreateDirs {
		dir := path.Dir(objectPath)
		if err := b.createPseudoDir(ctx, dir); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Set content type
	contentType := opts.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Build headers
	headers := swift.Headers{}
	headers["Content-Type"] = contentType

	// Add metadata
	for key, value := range opts.Metadata {
		headers["X-Object-Meta-"+key] = value
	}

	// Upload object
	_, err := b.conn.ObjectPut(ctx, b.container, objectPath, r, false, "", contentType, headers)
	if err != nil {
		return err
	}

	return nil
}

// createPseudoDir creates pseudo-directories (empty objects with trailing slash).
func (b *Backend) createPseudoDir(ctx context.Context, dirPath string) error {
	if dirPath == "" || dirPath == "." {
		return nil
	}

	// Check if pseudo-dir exists
	_, _, err := b.conn.Object(ctx, b.container, dirPath+"/")
	if err == nil {
		return nil
	}

	// Create parent first
	parent := path.Dir(dirPath)
	if parent != "." && parent != "" && parent != dirPath {
		if err := b.createPseudoDir(ctx, parent); err != nil {
			return err
		}
	}

	// Create pseudo-directory (empty object with trailing slash)
	_, err = b.conn.ObjectPut(ctx, b.container, dirPath+"/", strings.NewReader(""), false, "", "application/directory", nil)
	return err
}

// Delete deletes a file.
func (b *Backend) Delete(ctx context.Context, filePath string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	objectPath := b.resolvePath(filePath)

	// Check if it's a directory (pseudo-directory in Swift)
	objs, err := b.conn.ObjectsAll(ctx, b.container, &swift.ObjectsOpts{
		Prefix: objectPath + "/",
		Limit:  1,
	})
	if err == nil && len(objs) > 0 {
		return plugin.ErrNotAFile
	}

	if err := b.conn.ObjectDelete(ctx, b.container, objectPath); err != nil {
		if err == swift.ObjectNotFound {
			return plugin.ErrPathNotFound
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

	objectPath := b.resolvePath(filePath)

	// Check as file
	_, _, err := b.conn.Object(ctx, b.container, objectPath)
	if err == nil {
		return true, nil
	}
	if err != swift.ObjectNotFound {
		return false, err
	}

	// Check as directory (look for objects with this prefix)
	objs, err := b.conn.ObjectsAll(ctx, b.container, &swift.ObjectsOpts{
		Prefix: objectPath + "/",
		Limit:  1,
	})
	if err != nil {
		return false, err
	}

	return len(objs) > 0, nil
}

// Stat returns file information.
func (b *Backend) Stat(ctx context.Context, filePath string) (*plugin.FileInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	objectPath := b.resolvePath(filePath)

	info, headers, err := b.conn.Object(ctx, b.container, objectPath)
	if err != nil {
		if err == swift.ObjectNotFound {
			// Check if it's a directory
			objs, err := b.conn.ObjectsAll(ctx, b.container, &swift.ObjectsOpts{
				Prefix: objectPath + "/",
				Limit:  1,
			})
			if err != nil {
				return nil, err
			}
			if len(objs) > 0 {
				return &plugin.FileInfo{
					Name:  path.Base(objectPath),
					Path:  objectPath,
					IsDir: true,
					Mode:  os.ModeDir | 0755,
				}, nil
			}
			return nil, plugin.ErrPathNotFound
		}
		return nil, err
	}

	fileInfo := &plugin.FileInfo{
		Name:        path.Base(objectPath),
		Path:        objectPath,
		Size:        info.Bytes,
		ModTime:     info.LastModified,
		IsDir:       false,
		Mode:        0644,
		ContentType: info.ContentType,
		ETag:        info.Hash,
	}

	// Extract metadata
	fileInfo.Metadata = make(map[string]string)
	for key, value := range headers {
		if strings.HasPrefix(key, "X-Object-Meta-") {
			metaKey := strings.TrimPrefix(key, "X-Object-Meta-")
			fileInfo.Metadata[metaKey] = value
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

	prefix := b.resolvePath(dirPath)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	listOpts := &swift.ObjectsOpts{
		Prefix: prefix,
	}

	if !opts.Recursive {
		listOpts.Delimiter = '/'
	}

	if opts.MaxResults > 0 {
		listOpts.Limit = opts.MaxResults
	}

	objs, err := b.conn.ObjectsAll(ctx, b.container, listOpts)
	if err != nil {
		return nil, err
	}

	var results []plugin.FileInfo
	seenDirs := make(map[string]bool)

	for _, obj := range objs {
		name := strings.TrimPrefix(obj.Name, prefix)
		if name == "" {
			continue
		}

		// Handle non-recursive listing with delimiter
		if !opts.Recursive && strings.Contains(name, "/") {
			// This is a pseudo-directory
			dirName := strings.Split(name, "/")[0]
			if seenDirs[dirName] {
				continue
			}
			seenDirs[dirName] = true

			// Skip hidden directories if not included
			if !opts.IncludeHidden && strings.HasPrefix(dirName, ".") {
				continue
			}

			results = append(results, plugin.FileInfo{
				Name:  dirName,
				Path:  path.Join(prefix, dirName),
				IsDir: true,
				Mode:  os.ModeDir | 0755,
			})
			continue
		}

		// Skip hidden files if not included
		baseName := path.Base(name)
		if !opts.IncludeHidden && strings.HasPrefix(baseName, ".") {
			continue
		}

		// Apply pattern filter
		if opts.Pattern != "" {
			matched, err := path.Match(opts.Pattern, baseName)
			if err != nil || !matched {
				continue
			}
		}

		isDir := strings.HasSuffix(obj.Name, "/") || obj.ContentType == "application/directory"

		fileInfo := plugin.FileInfo{
			Name:        baseName,
			Path:        obj.Name,
			Size:        obj.Bytes,
			ModTime:     obj.LastModified,
			IsDir:       isDir,
			ContentType: obj.ContentType,
			ETag:        obj.Hash,
		}

		if isDir {
			fileInfo.Mode = os.ModeDir | 0755
		} else {
			fileInfo.Mode = 0644
		}

		results = append(results, fileInfo)

		if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
			break
		}
	}

	return results, nil
}

// Mkdir creates a pseudo-directory.
func (b *Backend) Mkdir(ctx context.Context, dirPath string, opts plugin.MkdirOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	objectPath := b.resolvePath(dirPath)

	if opts.Recursive {
		return b.createPseudoDir(ctx, objectPath)
	}

	// Create pseudo-directory
	_, err := b.conn.ObjectPut(ctx, b.container, objectPath+"/", strings.NewReader(""), false, "", "application/directory", nil)
	return err
}

// Rmdir removes a pseudo-directory.
func (b *Backend) Rmdir(ctx context.Context, dirPath string, recursive bool) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	objectPath := b.resolvePath(dirPath)
	if !strings.HasSuffix(objectPath, "/") {
		objectPath += "/"
	}

	// List objects in directory
	objs, err := b.conn.ObjectsAll(ctx, b.container, &swift.ObjectsOpts{
		Prefix: objectPath,
	})
	if err != nil {
		return err
	}

	if len(objs) == 0 {
		return plugin.ErrPathNotFound
	}

	if !recursive && len(objs) > 1 {
		return plugin.ErrDirNotEmpty
	}

	// Delete all objects
	for _, obj := range objs {
		if err := b.conn.ObjectDelete(ctx, b.container, obj.Name); err != nil {
			if err != swift.ObjectNotFound {
				return err
			}
		}
	}

	return nil
}

// Lock acquires a file lock.
// Note: Swift doesn't support file locking.
func (b *Backend) Lock(ctx context.Context, filePath string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	return nil, plugin.ErrNotSupported
}

// Symlink creates a symbolic link.
// Note: Object storage doesn't support symlinks.
func (b *Backend) Symlink(ctx context.Context, target, link string) error {
	return plugin.ErrNotSupported
}

// Chmod changes file permissions.
// Note: Object storage doesn't support Unix permissions.
func (b *Backend) Chmod(ctx context.Context, filePath string, mode os.FileMode) error {
	return plugin.ErrNotSupported
}

// Chown changes file ownership.
// Note: Object storage doesn't support Unix ownership.
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
		SupportsLocking:  false,
		SupportsSymlinks: false,
		SupportsChmod:    false,
		SupportsChown:    false,

		// Transfer optimizations
		SupportsRangeRead:        true, // Swift supports Range header
		SupportsMultipartUpload:  true, // Swift supports SLO/DLO
		SupportsDirectTransfer:   false,
		SupportsSendfile:         false,
		SupportsSplice:           false,
		SupportsCopyFileRange:    false,
		SupportsMemoryMapping:    false,
		SupportsStreamingWrite:   true,
		SupportsAtomicWrite:      true, // Object storage is atomic
		SupportsConcurrentAccess: true,

		// Metadata
		SupportsMetadata:   true,
		SupportsVersioning: true, // With versioning enabled on container
		SupportsChecksum:   true, // ETag/MD5

		// Limits
		MaxFileSize:      5 * 1024 * 1024 * 1024 * 1024, // 5TB with large objects
		MaxPathLength:    1024,
		MaxFilenameBytes: 1024,
	}
}

// CopyFile copies a file using server-side copy.
func (b *Backend) CopyFile(ctx context.Context, src, dst string, opts plugin.WriteOptions) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	srcPath := b.resolvePath(src)
	dstPath := b.resolvePath(dst)

	// Create parent directories if needed
	if opts.CreateDirs {
		dir := path.Dir(dstPath)
		if err := b.createPseudoDir(ctx, dir); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Server-side copy
	_, err := b.conn.ObjectCopy(ctx, b.container, srcPath, b.container, dstPath, nil)
	if err != nil {
		if err == swift.ObjectNotFound {
			return plugin.ErrPathNotFound
		}
		return err
	}

	return nil
}

// SetMetadata sets custom metadata on an object.
func (b *Backend) SetMetadata(ctx context.Context, filePath string, metadata map[string]string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return plugin.ErrNotConnected
	}

	objectPath := b.resolvePath(filePath)

	headers := swift.Headers{}
	for key, value := range metadata {
		headers["X-Object-Meta-"+key] = value
	}

	return b.conn.ObjectUpdate(ctx, b.container, objectPath, headers)
}

// GetMetadata retrieves custom metadata from an object.
func (b *Backend) GetMetadata(ctx context.Context, filePath string) (map[string]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return nil, plugin.ErrNotConnected
	}

	objectPath := b.resolvePath(filePath)

	_, headers, err := b.conn.Object(ctx, b.container, objectPath)
	if err != nil {
		if err == swift.ObjectNotFound {
			return nil, plugin.ErrPathNotFound
		}
		return nil, err
	}

	metadata := make(map[string]string)
	for key, value := range headers {
		if strings.HasPrefix(key, "X-Object-Meta-") {
			metaKey := strings.TrimPrefix(key, "X-Object-Meta-")
			metadata[metaKey] = value
		}
	}

	return metadata, nil
}
