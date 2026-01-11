// SPDX-License-Identifier: MIT

// Package gcs implements a Google Cloud Storage backend.
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Backend implements the Google Cloud Storage backend.
type Backend struct {
	client     *storage.Client
	bucket     *storage.BucketHandle
	bucketName string
	projectID  string
	pathPrefix string
	connected  bool
}

// GCSBackend extends the Backend interface with GCS-specific operations.
type GCSBackend interface {
	plugin.Backend

	// GCS-specific operations
	CopyObject(ctx context.Context, srcKey, dstKey string) error
	SetMetadata(ctx context.Context, key string, metadata map[string]string) error
	SetStorageClass(ctx context.Context, key string, storageClass string) error
	SetTemporaryHold(ctx context.Context, key string, hold bool) error
	SetEventBasedHold(ctx context.Context, key string, hold bool) error
	GetObjectAttrs(ctx context.Context, key string) (*GCSObjectInfo, error)
}

// GCSObjectInfo contains extended GCS object metadata.
type GCSObjectInfo struct {
	*plugin.FileInfo
	Generation        int64
	Metageneration    int64
	TemporaryHold     bool
	EventBasedHold    bool
	RetentionExpires  string
	KMSKeyName        string
	CustomerKeySHA256 string
}

// New creates a new GCS backend instance.
func New() *Backend {
	return &Backend{}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "gcs"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "gs"
}

// Connect establishes connection to Google Cloud Storage.
func (b *Backend) Connect(ctx context.Context, cfg plugin.BackendConfig) error {
	// Extract GCS-specific settings from Extra
	bucketName, _ := cfg.Extra["bucket"].(string)
	if bucketName == "" {
		return errors.New("bucket is required for GCS backend")
	}
	b.bucketName = bucketName

	b.projectID, _ = cfg.Extra["project"].(string)
	b.pathPrefix = cfg.BasePath

	// Build client options
	var clientOpts []option.ClientOption

	// Check for credentials
	credentialsFile, _ := cfg.Extra["credentials_file"].(string)
	credentialsJSON, _ := cfg.Extra["credentials_json"].(string)

	if credentialsFile != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(credentialsFile))
	} else if credentialsJSON != "" {
		clientOpts = append(clientOpts, option.WithCredentialsJSON([]byte(credentialsJSON)))
	}
	// Otherwise, use default credentials (ADC, env vars, etc.)

	client, err := storage.NewClient(ctx, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}

	b.client = client
	b.bucket = client.Bucket(bucketName)
	b.connected = true

	return nil
}

// Close closes the backend connection.
func (b *Backend) Close() error {
	if b.client != nil {
		b.client.Close()
	}
	b.connected = false
	return nil
}

// Ping checks if the backend is accessible.
func (b *Backend) Ping(ctx context.Context) error {
	if !b.connected {
		return errors.New("backend not connected")
	}

	_, err := b.bucket.Attrs(ctx)
	return err
}

// Read reads an object from GCS.
func (b *Backend) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := b.fullPath(key)

	reader, err := b.bucket.Object(fullKey).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return reader, nil
}

// Write writes an object to GCS.
func (b *Backend) Write(ctx context.Context, key string, r io.Reader, opts plugin.WriteOptions) error {
	fullKey := b.fullPath(key)

	obj := b.bucket.Object(fullKey)
	writer := obj.NewWriter(ctx)

	if opts.ContentType != "" {
		writer.ContentType = opts.ContentType
	}

	if len(opts.Metadata) > 0 {
		writer.Metadata = opts.Metadata
	}

	if _, err := io.Copy(writer, r); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write object: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize object write: %w", err)
	}

	return nil
}

// Delete removes an object from GCS.
func (b *Backend) Delete(ctx context.Context, key string) error {
	fullKey := b.fullPath(key)

	if err := b.bucket.Object(fullKey).Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// Exists checks if an object exists in GCS.
func (b *Backend) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := b.fullPath(key)

	_, err := b.bucket.Object(fullKey).Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// Stat returns object information.
func (b *Backend) Stat(ctx context.Context, key string) (*plugin.FileInfo, error) {
	fullKey := b.fullPath(key)

	attrs, err := b.bucket.Object(fullKey).Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	info := &plugin.FileInfo{
		Name:         path.Base(key),
		Path:         key,
		Size:         attrs.Size,
		ModTime:      attrs.Updated,
		IsDir:        false,
		ETag:         attrs.Etag,
		ContentType:  attrs.ContentType,
		StorageClass: attrs.StorageClass,
		MD5:          fmt.Sprintf("%x", attrs.MD5),
		CRC32:        fmt.Sprintf("%d", attrs.CRC32C),
		Metadata:     attrs.Metadata,
	}

	return info, nil
}

// List lists objects in GCS.
func (b *Backend) List(ctx context.Context, prefix string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	fullPrefix := b.fullPath(prefix)
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	query := &storage.Query{
		Prefix: fullPrefix,
	}

	if !opts.Recursive {
		query.Delimiter = "/"
	}

	var files []plugin.FileInfo

	it := b.bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Handle virtual directories (prefixes)
		if attrs.Prefix != "" {
			name := strings.TrimPrefix(attrs.Prefix, fullPrefix)
			name = strings.TrimSuffix(name, "/")
			files = append(files, plugin.FileInfo{
				Name:  name,
				Path:  attrs.Prefix,
				IsDir: true,
			})
			continue
		}

		name := strings.TrimPrefix(attrs.Name, fullPrefix)
		if name == "" {
			continue // Skip the directory marker itself
		}

		files = append(files, plugin.FileInfo{
			Name:         name,
			Path:         attrs.Name,
			Size:         attrs.Size,
			ModTime:      attrs.Updated,
			IsDir:        false,
			ETag:         attrs.Etag,
			ContentType:  attrs.ContentType,
			StorageClass: attrs.StorageClass,
		})

		if opts.MaxResults > 0 && len(files) >= opts.MaxResults {
			break
		}
	}

	return files, nil
}

// Mkdir creates a directory marker in GCS.
func (b *Backend) Mkdir(ctx context.Context, prefix string, opts plugin.MkdirOptions) error {
	fullPrefix := b.fullPath(prefix)
	if !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	writer := b.bucket.Object(fullPrefix).NewWriter(ctx)
	writer.ContentType = "application/x-directory"

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to create directory marker: %w", err)
	}

	return nil
}

// Rmdir removes a directory and optionally its contents.
func (b *Backend) Rmdir(ctx context.Context, prefix string, recursive bool) error {
	fullPrefix := b.fullPath(prefix)
	if !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	if recursive {
		// List and delete all objects with this prefix
		query := &storage.Query{Prefix: fullPrefix}
		it := b.bucket.Objects(ctx, query)
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to list objects for deletion: %w", err)
			}

			if err := b.bucket.Object(attrs.Name).Delete(ctx); err != nil {
				return fmt.Errorf("failed to delete object %s: %w", attrs.Name, err)
			}
		}
	} else {
		// Just delete the directory marker
		if err := b.bucket.Object(fullPrefix).Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete directory marker: %w", err)
		}
	}

	return nil
}

// Lock is not supported by GCS.
func (b *Backend) Lock(ctx context.Context, path string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	return nil, plugin.ErrNotSupported
}

// Symlink is not supported by GCS.
func (b *Backend) Symlink(ctx context.Context, target, link string) error {
	return plugin.ErrNotSupported
}

// Chmod is not supported by GCS.
func (b *Backend) Chmod(ctx context.Context, path string, mode os.FileMode) error {
	return plugin.ErrNotSupported
}

// Chown is not supported by GCS.
func (b *Backend) Chown(ctx context.Context, path string, uid, gid int) error {
	return plugin.ErrNotSupported
}

// Capabilities returns the backend capabilities.
func (b *Backend) Capabilities() plugin.BackendCapabilities {
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
		SupportsDirectTransfer:   true,
		SupportsStreamingWrite:   true,
		SupportsConcurrentAccess: true,
		SupportsMetadata:         true,
		SupportsVersioning:       true,
		SupportsChecksum:         true,
	}
}

// GCS-specific operations

// CopyObject copies an object within the same bucket.
func (b *Backend) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	fullSrcKey := b.fullPath(srcKey)
	fullDstKey := b.fullPath(dstKey)

	src := b.bucket.Object(fullSrcKey)
	dst := b.bucket.Object(fullDstKey)

	_, err := dst.CopierFrom(src).Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// SetMetadata updates object metadata.
func (b *Backend) SetMetadata(ctx context.Context, key string, metadata map[string]string) error {
	fullKey := b.fullPath(key)

	_, err := b.bucket.Object(fullKey).Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

// SetStorageClass changes the object storage class.
func (b *Backend) SetStorageClass(ctx context.Context, key, storageClass string) error {
	fullKey := b.fullPath(key)

	// Storage class change requires rewriting the object
	src := b.bucket.Object(fullKey)
	dst := b.bucket.Object(fullKey)

	copier := dst.CopierFrom(src)
	copier.StorageClass = storageClass

	_, err := copier.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to set storage class: %w", err)
	}

	return nil
}

// SetTemporaryHold sets or removes a temporary hold on the object.
func (b *Backend) SetTemporaryHold(ctx context.Context, key string, hold bool) error {
	fullKey := b.fullPath(key)

	_, err := b.bucket.Object(fullKey).Update(ctx, storage.ObjectAttrsToUpdate{
		TemporaryHold: hold,
	})
	if err != nil {
		return fmt.Errorf("failed to set temporary hold: %w", err)
	}

	return nil
}

// SetEventBasedHold sets or removes an event-based hold on the object.
func (b *Backend) SetEventBasedHold(ctx context.Context, key string, hold bool) error {
	fullKey := b.fullPath(key)

	_, err := b.bucket.Object(fullKey).Update(ctx, storage.ObjectAttrsToUpdate{
		EventBasedHold: hold,
	})
	if err != nil {
		return fmt.Errorf("failed to set event-based hold: %w", err)
	}

	return nil
}

// GetObjectAttrs retrieves extended object attributes.
func (b *Backend) GetObjectAttrs(ctx context.Context, key string) (*GCSObjectInfo, error) {
	fullKey := b.fullPath(key)

	attrs, err := b.bucket.Object(fullKey).Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	info := &GCSObjectInfo{
		FileInfo: &plugin.FileInfo{
			Name:         path.Base(key),
			Path:         key,
			Size:         attrs.Size,
			ModTime:      attrs.Updated,
			IsDir:        false,
			ETag:         attrs.Etag,
			ContentType:  attrs.ContentType,
			StorageClass: attrs.StorageClass,
			MD5:          fmt.Sprintf("%x", attrs.MD5),
			CRC32:        fmt.Sprintf("%d", attrs.CRC32C),
			Metadata:     attrs.Metadata,
			CreationTime: attrs.Created,
		},
		Generation:        attrs.Generation,
		Metageneration:    attrs.Metageneration,
		TemporaryHold:     attrs.TemporaryHold,
		EventBasedHold:    attrs.EventBasedHold,
		KMSKeyName:        attrs.KMSKeyName,
		CustomerKeySHA256: attrs.CustomerKeySHA256,
	}

	if !attrs.RetentionExpirationTime.IsZero() {
		info.RetentionExpires = attrs.RetentionExpirationTime.String()
	}

	return info, nil
}

// Helper functions

func (b *Backend) fullPath(key string) string {
	if b.pathPrefix == "" {
		return key
	}
	return path.Join(b.pathPrefix, key)
}

// Ensure Backend implements GCSBackend.
var _ GCSBackend = (*Backend)(nil)
