// SPDX-License-Identifier: MIT

// Package b2 implements a Backblaze B2 Cloud Storage backend.
package b2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/kurin/blazer/b2"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Backend implements the Backblaze B2 storage backend.
type Backend struct {
	client     *b2.Client
	bucket     *b2.Bucket
	bucketName string
	pathPrefix string
	connected  bool
}

// B2Backend extends the Backend interface with B2-specific operations.
type B2Backend interface {
	plugin.Backend

	// B2-specific operations
	CopyFile(ctx context.Context, srcKey, dstKey string) error
	GetFileInfo(ctx context.Context, key string) (*B2FileInfo, error)
	HideFile(ctx context.Context, key string) error
	UpdateFileLegalHold(ctx context.Context, key string, legalHold bool) error
	UpdateFileRetention(ctx context.Context, key string, mode string, retainUntilTimestamp int64) error
}

// B2FileInfo contains extended B2 file metadata.
type B2FileInfo struct {
	*plugin.FileInfo
	FileID           string
	Action           string
	UploadTimestamp  int64
	LegalHold        bool
	RetentionMode    string
	RetentionExpires int64
}

// New creates a new B2 backend instance.
func New() *Backend {
	return &Backend{}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "b2"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "b2"
}

// Connect establishes connection to Backblaze B2.
func (be *Backend) Connect(ctx context.Context, cfg plugin.BackendConfig) error {
	// Extract B2-specific settings from Extra
	bucketName, _ := cfg.Extra["bucket"].(string)
	if bucketName == "" {
		return errors.New("bucket is required for B2 backend")
	}
	be.bucketName = bucketName

	applicationKeyID, _ := cfg.Extra["application_key_id"].(string)
	applicationKey, _ := cfg.Extra["application_key"].(string)

	if applicationKeyID == "" || applicationKey == "" {
		return errors.New("application_key_id and application_key are required for B2 backend")
	}

	be.pathPrefix = cfg.BasePath

	// Create B2 client
	client, err := b2.NewClient(ctx, applicationKeyID, applicationKey)
	if err != nil {
		return fmt.Errorf("failed to create B2 client: %w", err)
	}

	be.client = client

	// Get bucket
	bucket, err := client.Bucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to get B2 bucket: %w", err)
	}

	be.bucket = bucket
	be.connected = true

	return nil
}

// Close closes the backend connection.
func (be *Backend) Close() error {
	be.connected = false
	return nil
}

// Ping checks if the backend is accessible.
func (be *Backend) Ping(ctx context.Context) error {
	if !be.connected {
		return errors.New("backend not connected")
	}

	// List one file to verify connection
	iter := be.bucket.List(ctx, b2.ListPageSize(1))
	if iter.Next() {
		// Success - at least we can list
		return nil
	}
	if err := iter.Err(); err != nil {
		return err
	}
	// Empty bucket but connection works
	return nil
}

// Read reads a file from B2.
func (be *Backend) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := be.fullPath(key)

	obj := be.bucket.Object(fullKey)
	reader := obj.NewReader(ctx)

	// Try to read attributes to check if file exists
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		reader.Close()
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get file attributes: %w", err)
	}

	_ = attrs // We just wanted to verify existence

	return reader, nil
}

// Write writes a file to B2.
func (be *Backend) Write(ctx context.Context, key string, r io.Reader, opts plugin.WriteOptions) error {
	fullKey := be.fullPath(key)

	obj := be.bucket.Object(fullKey)

	// Set up writer options
	writerOpts := []b2.WriterOption{}

	if opts.ContentType != "" {
		writerOpts = append(writerOpts, b2.WithAttrsOption(&b2.Attrs{
			ContentType: opts.ContentType,
			Info:        opts.Metadata,
		}))
	} else if len(opts.Metadata) > 0 {
		writerOpts = append(writerOpts, b2.WithAttrsOption(&b2.Attrs{
			Info: opts.Metadata,
		}))
	}

	writer := obj.NewWriter(ctx, writerOpts...)

	if _, err := io.Copy(writer, r); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize file write: %w", err)
	}

	return nil
}

// Delete removes a file from B2.
func (be *Backend) Delete(ctx context.Context, key string) error {
	fullKey := be.fullPath(key)

	obj := be.bucket.Object(fullKey)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if a file exists in B2.
func (be *Backend) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := be.fullPath(key)

	obj := be.bucket.Object(fullKey)
	_, err := obj.Attrs(ctx)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// Stat returns file information.
func (be *Backend) Stat(ctx context.Context, key string) (*plugin.FileInfo, error) {
	fullKey := be.fullPath(key)

	obj := be.bucket.Object(fullKey)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get file attributes: %w", err)
	}

	info := &plugin.FileInfo{
		Name:        path.Base(key),
		Path:        key,
		Size:        attrs.Size,
		ModTime:     attrs.UploadTimestamp,
		IsDir:       false,
		ContentType: attrs.ContentType,
		SHA256:      attrs.SHA1, // B2 uses SHA1
		Metadata:    attrs.Info,
	}

	return info, nil
}

// List lists files in B2.
func (be *Backend) List(ctx context.Context, prefix string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	fullPrefix := be.fullPath(prefix)
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	listOpts := []b2.ListOption{
		b2.ListPrefix(fullPrefix),
	}

	if !opts.Recursive {
		listOpts = append(listOpts, b2.ListDelimiter("/"))
	}

	if opts.MaxResults > 0 {
		listOpts = append(listOpts, b2.ListPageSize(opts.MaxResults))
	}

	var files []plugin.FileInfo
	iter := be.bucket.List(ctx, listOpts...)

	for iter.Next() {
		obj := iter.Object()
		attrs, _ := obj.Attrs(ctx)

		name := strings.TrimPrefix(obj.Name(), fullPrefix)
		if name == "" {
			continue // Skip the directory marker itself
		}

		// Check if it's a directory (prefix)
		isDir := strings.HasSuffix(name, "/")
		if isDir {
			name = strings.TrimSuffix(name, "/")
		}

		if attrs != nil {
			files = append(files, plugin.FileInfo{
				Name:        name,
				Path:        obj.Name(),
				Size:        attrs.Size,
				ModTime:     attrs.UploadTimestamp,
				IsDir:       isDir,
				ContentType: attrs.ContentType,
			})
		} else {
			files = append(files, plugin.FileInfo{
				Name:  name,
				Path:  obj.Name(),
				IsDir: isDir,
			})
		}

		if opts.MaxResults > 0 && len(files) >= opts.MaxResults {
			break
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}

// Mkdir creates a directory marker in B2.
func (be *Backend) Mkdir(ctx context.Context, prefix string, opts plugin.MkdirOptions) error {
	fullPrefix := be.fullPath(prefix)
	if !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	obj := be.bucket.Object(fullPrefix)
	writer := obj.NewWriter(ctx, b2.WithAttrsOption(&b2.Attrs{
		ContentType: "application/x-directory",
	}))

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to create directory marker: %w", err)
	}

	return nil
}

// Rmdir removes a directory and optionally its contents.
func (be *Backend) Rmdir(ctx context.Context, prefix string, recursive bool) error {
	fullPrefix := be.fullPath(prefix)
	if !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	if recursive {
		// List and delete all files with this prefix
		iter := be.bucket.List(ctx, b2.ListPrefix(fullPrefix))
		for iter.Next() {
			obj := iter.Object()
			if err := obj.Delete(ctx); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", obj.Name(), err)
			}
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("failed to list files for deletion: %w", err)
		}
	} else {
		// Just delete the directory marker
		obj := be.bucket.Object(fullPrefix)
		if err := obj.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete directory marker: %w", err)
		}
	}

	return nil
}

// Lock is not supported by B2.
func (be *Backend) Lock(ctx context.Context, path string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	return nil, plugin.ErrNotSupported
}

// Symlink is not supported by B2.
func (be *Backend) Symlink(ctx context.Context, target, link string) error {
	return plugin.ErrNotSupported
}

// Chmod is not supported by B2.
func (be *Backend) Chmod(ctx context.Context, path string, mode os.FileMode) error {
	return plugin.ErrNotSupported
}

// Chown is not supported by B2.
func (be *Backend) Chown(ctx context.Context, path string, uid, gid int) error {
	return plugin.ErrNotSupported
}

// Capabilities returns the backend capabilities.
func (be *Backend) Capabilities() plugin.BackendCapabilities {
	return plugin.BackendCapabilities{
		SupportsRead:             true,
		SupportsWrite:            true,
		SupportsDelete:           true,
		SupportsStat:             true,
		SupportsList:             true,
		SupportsMkdir:            true,
		SupportsRmdir:            true,
		SupportsLocking:          false,
		SupportsSymlinks:         false,
		SupportsChmod:            false,
		SupportsChown:            false,
		SupportsRangeRead:        true,
		SupportsMultipartUpload:  true,
		SupportsDirectTransfer:   false,
		SupportsStreamingWrite:   true,
		SupportsConcurrentAccess: true,
		SupportsMetadata:         true,
		SupportsVersioning:       true,
		SupportsChecksum:         true,
	}
}

// B2-specific operations

// CopyFile copies a file within the same bucket.
func (be *Backend) CopyFile(ctx context.Context, srcKey, dstKey string) error {
	fullSrcKey := be.fullPath(srcKey)
	fullDstKey := be.fullPath(dstKey)

	// B2 requires reading and rewriting for copy
	src := be.bucket.Object(fullSrcKey)
	dst := be.bucket.Object(fullDstKey)

	srcAttrs, err := src.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get source attributes: %w", err)
	}

	reader := src.NewReader(ctx)
	defer reader.Close()

	writer := dst.NewWriter(ctx, b2.WithAttrsOption(&b2.Attrs{
		ContentType: srcAttrs.ContentType,
		Info:        srcAttrs.Info,
	}))

	if _, err := io.Copy(writer, reader); err != nil {
		writer.Close()
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize copy: %w", err)
	}

	return nil
}

// GetFileInfo retrieves extended file information.
func (be *Backend) GetFileInfo(ctx context.Context, key string) (*B2FileInfo, error) {
	fullKey := be.fullPath(key)

	obj := be.bucket.Object(fullKey)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get file attributes: %w", err)
	}

	info := &B2FileInfo{
		FileInfo: &plugin.FileInfo{
			Name:        path.Base(key),
			Path:        key,
			Size:        attrs.Size,
			ModTime:     attrs.UploadTimestamp,
			IsDir:       false,
			ContentType: attrs.ContentType,
			SHA256:      attrs.SHA1,
			Metadata:    attrs.Info,
		},
		UploadTimestamp: attrs.UploadTimestamp.Unix(),
	}

	return info, nil
}

// HideFile hides a file version (B2 versioning).
func (be *Backend) HideFile(ctx context.Context, key string) error {
	fullKey := be.fullPath(key)

	obj := be.bucket.Object(fullKey)
	return obj.Hide(ctx)
}

// UpdateFileLegalHold updates the legal hold status.
func (be *Backend) UpdateFileLegalHold(ctx context.Context, key string, legalHold bool) error {
	// Note: B2 legal hold requires specific bucket configuration
	// This is a placeholder - actual implementation depends on B2 bucket settings
	return plugin.ErrNotSupported
}

// UpdateFileRetention updates the file retention settings.
func (be *Backend) UpdateFileRetention(ctx context.Context, key string, mode string, retainUntilTimestamp int64) error {
	// Note: B2 retention requires specific bucket configuration
	// This is a placeholder - actual implementation depends on B2 bucket settings
	return plugin.ErrNotSupported
}

// Helper functions

func (be *Backend) fullPath(key string) string {
	if be.pathPrefix == "" {
		return key
	}
	return path.Join(be.pathPrefix, key)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "no such file")
}

// Ensure Backend implements B2Backend.
var _ B2Backend = (*Backend)(nil)
