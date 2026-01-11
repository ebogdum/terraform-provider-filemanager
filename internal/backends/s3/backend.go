// SPDX-License-Identifier: MIT

// Package s3 implements an S3-compatible storage backend.
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Backend implements the S3 storage backend.
type Backend struct {
	client     *s3.Client
	bucket     string
	region     string
	endpoint   string
	pathPrefix string
	connected  bool
}

// S3Backend extends the Backend interface with S3-specific operations.
type S3Backend interface {
	plugin.Backend

	// S3-specific operations
	CopyObject(ctx context.Context, srcKey, dstKey string) error
	SetMetadata(ctx context.Context, key string, metadata map[string]string) error
	GetTags(ctx context.Context, key string) (map[string]string, error)
	SetTags(ctx context.Context, key string, tags map[string]string) error
	DeleteTags(ctx context.Context, key string) error
	SetStorageClass(ctx context.Context, key string, storageClass string) error
	RestoreObject(ctx context.Context, key string, days int32, tier string) error
	HeadObject(ctx context.Context, key string) (*S3ObjectInfo, error)
}

// S3ObjectInfo contains extended S3 object metadata.
type S3ObjectInfo struct {
	*plugin.FileInfo
	ReplicationStatus     string
	ObjectLockMode        string
	ObjectLockRetainUntil *time.Time
	LegalHold             bool
	RestoreStatus         string
	ServerSideEncryption  string
	SSEKMSKeyID           string
}

// New creates a new S3 backend instance.
func New() *Backend {
	return &Backend{}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "s3"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "s3"
}

// Connect establishes connection to the S3 service.
func (b *Backend) Connect(ctx context.Context, cfg plugin.BackendConfig) error {
	// Extract S3-specific settings from Extra
	bucket, _ := cfg.Extra["bucket"].(string)
	if bucket == "" {
		return errors.New("bucket is required for S3 backend")
	}
	b.bucket = bucket

	region, _ := cfg.Extra["region"].(string)
	if region == "" {
		region = "us-east-1"
	}
	b.region = region

	b.pathPrefix = cfg.BasePath
	b.endpoint, _ = cfg.Extra["endpoint"].(string)

	// Build AWS config
	var awsOpts []func(*config.LoadOptions) error

	awsOpts = append(awsOpts, config.WithRegion(region))

	// Check for explicit credentials
	accessKey, _ := cfg.Extra["access_key"].(string)
	secretKey, _ := cfg.Extra["secret_key"].(string)
	sessionToken, _ := cfg.Extra["session_token"].(string)

	if accessKey != "" && secretKey != "" {
		awsOpts = append(awsOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client options
	s3Opts := []func(*s3.Options){}

	// Custom endpoint for S3-compatible services
	if b.endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(b.endpoint)
			o.UsePathStyle = true // Required for most S3-compatible services
		})
	}

	b.client = s3.NewFromConfig(awsCfg, s3Opts...)
	b.connected = true

	return nil
}

// Close closes the backend connection.
func (b *Backend) Close() error {
	b.connected = false
	return nil
}

// Ping checks if the backend is accessible.
func (b *Backend) Ping(ctx context.Context) error {
	if !b.connected {
		return errors.New("backend not connected")
	}

	_, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.bucket),
	})
	return err
}

// Read reads a file from S3.
func (b *Backend) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := b.fullPath(key)

	output, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return output.Body, nil
}

// Write writes a file to S3.
func (b *Backend) Write(ctx context.Context, key string, r io.Reader, opts plugin.WriteOptions) error {
	fullKey := b.fullPath(key)

	input := &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
		Body:   r,
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	_, err := b.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

// Delete removes a file from S3.
func (b *Backend) Delete(ctx context.Context, key string) error {
	fullKey := b.fullPath(key)

	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// Exists checks if a file exists in S3.
func (b *Backend) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := b.fullPath(key)

	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to head object: %w", err)
	}

	return true, nil
}

// Stat returns file information.
func (b *Backend) Stat(ctx context.Context, key string) (*plugin.FileInfo, error) {
	fullKey := b.fullPath(key)

	output, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to head object: %w", err)
	}

	info := &plugin.FileInfo{
		Name:         path.Base(key),
		Path:         key,
		Size:         aws.ToInt64(output.ContentLength),
		ModTime:      aws.ToTime(output.LastModified),
		IsDir:        false,
		ETag:         strings.Trim(aws.ToString(output.ETag), "\""),
		ContentType:  aws.ToString(output.ContentType),
		StorageClass: string(output.StorageClass),
		VersionID:    aws.ToString(output.VersionId),
		Metadata:     output.Metadata,
	}

	// Get tags
	tagsOutput, err := b.client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err == nil && tagsOutput.TagSet != nil {
		info.Tags = make(map[string]string)
		for _, tag := range tagsOutput.TagSet {
			info.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return info, nil
}

// List lists objects in S3.
func (b *Backend) List(ctx context.Context, prefix string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	fullPrefix := b.fullPath(prefix)
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(fullPrefix),
	}

	if !opts.Recursive {
		input.Delimiter = aws.String("/")
	}

	if opts.MaxResults > 0 {
		input.MaxKeys = aws.Int32(int32(opts.MaxResults))
	}

	var files []plugin.FileInfo

	paginator := s3.NewListObjectsV2Paginator(b.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		// Add directories (common prefixes)
		for _, prefix := range output.CommonPrefixes {
			name := strings.TrimPrefix(aws.ToString(prefix.Prefix), fullPrefix)
			name = strings.TrimSuffix(name, "/")
			files = append(files, plugin.FileInfo{
				Name:  name,
				Path:  aws.ToString(prefix.Prefix),
				IsDir: true,
			})
		}

		// Add files
		for _, obj := range output.Contents {
			key := aws.ToString(obj.Key)
			name := strings.TrimPrefix(key, fullPrefix)
			if name == "" {
				continue // Skip the directory marker itself
			}
			files = append(files, plugin.FileInfo{
				Name:         name,
				Path:         key,
				Size:         aws.ToInt64(obj.Size),
				ModTime:      aws.ToTime(obj.LastModified),
				IsDir:        false,
				ETag:         strings.Trim(aws.ToString(obj.ETag), "\""),
				StorageClass: string(obj.StorageClass),
			})
		}

		if opts.MaxResults > 0 && len(files) >= opts.MaxResults {
			break
		}
	}

	return files, nil
}

// Mkdir creates a directory marker in S3.
func (b *Backend) Mkdir(ctx context.Context, prefix string, opts plugin.MkdirOptions) error {
	fullPrefix := b.fullPath(prefix)
	if !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullPrefix),
		Body:   strings.NewReader(""),
	})
	if err != nil {
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
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(b.bucket),
			Prefix: aws.String(fullPrefix),
		}

		paginator := s3.NewListObjectsV2Paginator(b.client, input)
		for paginator.HasMorePages() {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				return fmt.Errorf("failed to list objects for deletion: %w", err)
			}

			if len(output.Contents) == 0 {
				continue
			}

			// Batch delete
			var objects []types.ObjectIdentifier
			for _, obj := range output.Contents {
				objects = append(objects, types.ObjectIdentifier{
					Key: obj.Key,
				})
			}

			_, err = b.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(b.bucket),
				Delete: &types.Delete{
					Objects: objects,
					Quiet:   aws.Bool(true),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to delete objects: %w", err)
			}
		}
	} else {
		// Just delete the directory marker
		_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(b.bucket),
			Key:    aws.String(fullPrefix),
		})
		if err != nil {
			return fmt.Errorf("failed to delete directory marker: %w", err)
		}
	}

	return nil
}

// Lock is not supported by S3.
func (b *Backend) Lock(ctx context.Context, path string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	return nil, plugin.ErrNotSupported
}

// Symlink is not supported by S3.
func (b *Backend) Symlink(ctx context.Context, target, link string) error {
	return plugin.ErrNotSupported
}

// Chmod is not supported by S3.
func (b *Backend) Chmod(ctx context.Context, path string, mode os.FileMode) error {
	return plugin.ErrNotSupported
}

// Chown is not supported by S3.
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
		MaxFileSize:              5 * 1024 * 1024 * 1024 * 1024, // 5TB
	}
}

// S3-specific operations

// CopyObject copies an object within the same bucket.
func (b *Backend) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	fullSrcKey := b.fullPath(srcKey)
	fullDstKey := b.fullPath(dstKey)

	copySource := fmt.Sprintf("%s/%s", b.bucket, fullSrcKey)

	_, err := b.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(b.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(fullDstKey),
	})
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// SetMetadata updates object metadata (requires copy-to-self).
func (b *Backend) SetMetadata(ctx context.Context, key string, metadata map[string]string) error {
	fullKey := b.fullPath(key)
	copySource := fmt.Sprintf("%s/%s", b.bucket, fullKey)

	_, err := b.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:            aws.String(b.bucket),
		CopySource:        aws.String(copySource),
		Key:               aws.String(fullKey),
		Metadata:          metadata,
		MetadataDirective: types.MetadataDirectiveReplace,
	})
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

// GetTags retrieves object tags.
func (b *Backend) GetTags(ctx context.Context, key string) (map[string]string, error) {
	fullKey := b.fullPath(key)

	output, err := b.client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	tags := make(map[string]string)
	for _, tag := range output.TagSet {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return tags, nil
}

// SetTags sets object tags.
func (b *Backend) SetTags(ctx context.Context, key string, tags map[string]string) error {
	fullKey := b.fullPath(key)

	var tagSet []types.Tag
	for k, v := range tags {
		tagSet = append(tagSet, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := b.client.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
		Tagging: &types.Tagging{
			TagSet: tagSet,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set tags: %w", err)
	}

	return nil
}

// DeleteTags removes all tags from an object.
func (b *Backend) DeleteTags(ctx context.Context, key string) error {
	fullKey := b.fullPath(key)

	_, err := b.client.DeleteObjectTagging(ctx, &s3.DeleteObjectTaggingInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete tags: %w", err)
	}

	return nil
}

// SetStorageClass changes the object storage class (requires copy-to-self).
func (b *Backend) SetStorageClass(ctx context.Context, key string, storageClass string) error {
	fullKey := b.fullPath(key)
	copySource := fmt.Sprintf("%s/%s", b.bucket, fullKey)

	_, err := b.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:       aws.String(b.bucket),
		CopySource:   aws.String(copySource),
		Key:          aws.String(fullKey),
		StorageClass: types.StorageClass(storageClass),
	})
	if err != nil {
		return fmt.Errorf("failed to set storage class: %w", err)
	}

	return nil
}

// RestoreObject initiates a restore request for archived objects.
func (b *Backend) RestoreObject(ctx context.Context, key string, days int32, tier string) error {
	fullKey := b.fullPath(key)

	restoreTier := types.TierStandard
	switch strings.ToLower(tier) {
	case "expedited":
		restoreTier = types.TierExpedited
	case "bulk":
		restoreTier = types.TierBulk
	}

	_, err := b.client.RestoreObject(ctx, &s3.RestoreObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
		RestoreRequest: &types.RestoreRequest{
			Days: aws.Int32(days),
			GlacierJobParameters: &types.GlacierJobParameters{
				Tier: restoreTier,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to restore object: %w", err)
	}

	return nil
}

// HeadObject retrieves extended object metadata.
func (b *Backend) HeadObject(ctx context.Context, key string) (*S3ObjectInfo, error) {
	fullKey := b.fullPath(key)

	output, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to head object: %w", err)
	}

	info := &S3ObjectInfo{
		FileInfo: &plugin.FileInfo{
			Name:         path.Base(key),
			Path:         key,
			Size:         aws.ToInt64(output.ContentLength),
			ModTime:      aws.ToTime(output.LastModified),
			IsDir:        false,
			ETag:         strings.Trim(aws.ToString(output.ETag), "\""),
			ContentType:  aws.ToString(output.ContentType),
			StorageClass: string(output.StorageClass),
			VersionID:    aws.ToString(output.VersionId),
			Metadata:     output.Metadata,
		},
		ReplicationStatus:    string(output.ReplicationStatus),
		ServerSideEncryption: string(output.ServerSideEncryption),
		SSEKMSKeyID:          aws.ToString(output.SSEKMSKeyId),
	}

	if output.ObjectLockMode != "" {
		info.ObjectLockMode = string(output.ObjectLockMode)
	}
	if output.ObjectLockRetainUntilDate != nil {
		info.ObjectLockRetainUntil = output.ObjectLockRetainUntilDate
	}
	if output.ObjectLockLegalHoldStatus == types.ObjectLockLegalHoldStatusOn {
		info.LegalHold = true
	}
	if output.Restore != nil {
		info.RestoreStatus = aws.ToString(output.Restore)
	}

	// Get tags
	tagsOutput, err := b.client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err == nil && tagsOutput.TagSet != nil {
		info.FileInfo.Tags = make(map[string]string)
		for _, tag := range tagsOutput.TagSet {
			info.FileInfo.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
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

func isNotFoundError(err error) bool {
	var notFound *types.NotFound
	var noSuchKey *types.NoSuchKey
	return errors.As(err, &notFound) || errors.As(err, &noSuchKey)
}

// Ensure Backend implements S3Backend.
var _ S3Backend = (*Backend)(nil)
