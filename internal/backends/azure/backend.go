// SPDX-License-Identifier: MIT

// Package azure implements an Azure Blob Storage backend.
package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Backend implements the Azure Blob Storage backend.
type Backend struct {
	client        *azblob.Client
	containerName string
	accountName   string
	pathPrefix    string
	connected     bool
}

// AzureBackend extends the Backend interface with Azure-specific operations.
type AzureBackend interface {
	plugin.Backend

	// Azure-specific operations
	CopyBlob(ctx context.Context, srcBlob, dstBlob string) error
	SetMetadata(ctx context.Context, blobName string, metadata map[string]*string) error
	GetTags(ctx context.Context, blobName string) (map[string]string, error)
	SetTags(ctx context.Context, blobName string, tags map[string]string) error
	SetAccessTier(ctx context.Context, blobName string, tier string) error
	AcquireLease(ctx context.Context, blobName string, duration int32) (string, error)
	ReleaseLease(ctx context.Context, blobName, leaseID string) error
	HeadBlob(ctx context.Context, blobName string) (*AzureBlobInfo, error)
}

// AzureBlobInfo contains extended Azure blob metadata.
type AzureBlobInfo struct {
	*plugin.FileInfo
	AccessTier        string
	LeaseStatus       string
	LeaseState        string
	LeaseDuration     string
	BlobType          string
	ServerEncrypted   bool
	RehydratePriority string
	ArchiveStatus     string
}

// New creates a new Azure backend instance.
func New() *Backend {
	return &Backend{}
}

// Name returns the backend name.
func (b *Backend) Name() string {
	return "azure"
}

// Scheme returns the URI scheme.
func (b *Backend) Scheme() string {
	return "azure"
}

// Connect establishes connection to Azure Blob Storage.
func (b *Backend) Connect(ctx context.Context, cfg plugin.BackendConfig) error {
	// Extract Azure-specific settings from Extra
	containerName, _ := cfg.Extra["container"].(string)
	if containerName == "" {
		return errors.New("container is required for Azure backend")
	}
	b.containerName = containerName

	accountName, _ := cfg.Extra["account_name"].(string)
	if accountName == "" {
		return errors.New("account_name is required for Azure backend")
	}
	b.accountName = accountName

	b.pathPrefix = cfg.BasePath

	// Build service URL
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	// Check for explicit credentials
	accountKey, _ := cfg.Extra["account_key"].(string)
	connectionString, _ := cfg.Extra["connection_string"].(string)
	sasToken, _ := cfg.Extra["sas_token"].(string)

	var client *azblob.Client
	var err error

	if connectionString != "" {
		// Use connection string
		client, err = azblob.NewClientFromConnectionString(connectionString, nil)
	} else if accountKey != "" {
		// Use shared key credential
		cred, credErr := azblob.NewSharedKeyCredential(accountName, accountKey)
		if credErr != nil {
			return fmt.Errorf("failed to create shared key credential: %w", credErr)
		}
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	} else if sasToken != "" {
		// Use SAS token
		urlWithSAS := serviceURL + "?" + sasToken
		client, err = azblob.NewClientWithNoCredential(urlWithSAS, nil)
	} else {
		// Use default Azure identity (managed identity, env vars, etc.)
		cred, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return fmt.Errorf("failed to create default Azure credential: %w", credErr)
		}
		client, err = azblob.NewClient(serviceURL, cred, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to create Azure client: %w", err)
	}

	b.client = client
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

	containerClient := b.client.ServiceClient().NewContainerClient(b.containerName)
	_, err := containerClient.GetProperties(ctx, nil)
	return err
}

// Read reads a blob from Azure.
func (b *Backend) Read(ctx context.Context, blobName string) (io.ReadCloser, error) {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to download blob: %w", err)
	}

	return resp.Body, nil
}

// Write writes a blob to Azure.
func (b *Backend) Write(ctx context.Context, blobName string, r io.Reader, opts plugin.WriteOptions) error {
	fullPath := b.fullPath(blobName)

	// Read all content (Azure requires knowing content length for block blobs)
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	blockBlobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlockBlobClient(fullPath)

	uploadOpts := &blockblob.UploadOptions{}

	if opts.ContentType != "" {
		uploadOpts.HTTPHeaders = &blob.HTTPHeaders{
			BlobContentType: &opts.ContentType,
		}
	}

	if len(opts.Metadata) > 0 {
		metadataPtr := make(map[string]*string)
		for k, v := range opts.Metadata {
			val := v
			metadataPtr[k] = &val
		}
		uploadOpts.Metadata = metadataPtr
	}

	_, err = blockBlobClient.Upload(ctx, newReadSeekCloser(data), uploadOpts)
	if err != nil {
		return fmt.Errorf("failed to upload blob: %w", err)
	}

	return nil
}

// Delete removes a blob from Azure.
func (b *Backend) Delete(ctx context.Context, blobName string) error {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	_, err := blobClient.Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	return nil
}

// Exists checks if a blob exists in Azure.
func (b *Backend) Exists(ctx context.Context, blobName string) (bool, error) {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get blob properties: %w", err)
	}

	return true, nil
}

// Stat returns blob information.
func (b *Backend) Stat(ctx context.Context, blobName string) (*plugin.FileInfo, error) {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get blob properties: %w", err)
	}

	info := &plugin.FileInfo{
		Name:        path.Base(blobName),
		Path:        blobName,
		Size:        *props.ContentLength,
		ModTime:     *props.LastModified,
		IsDir:       false,
		ETag:        string(*props.ETag),
		ContentType: *props.ContentType,
		VersionID:   safeString(props.VersionID),
	}

	if props.Metadata != nil {
		info.Metadata = make(map[string]string)
		for k, v := range props.Metadata {
			if v != nil {
				info.Metadata[k] = *v
			}
		}
	}

	// Get tags
	tagsResp, err := blobClient.GetTags(ctx, nil)
	if err == nil && tagsResp.BlobTagSet != nil {
		info.Tags = make(map[string]string)
		for _, tag := range tagsResp.BlobTagSet {
			if tag.Key != nil && tag.Value != nil {
				info.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return info, nil
}

// List lists blobs in Azure.
func (b *Backend) List(ctx context.Context, prefix string, opts plugin.ListOptions) ([]plugin.FileInfo, error) {
	fullPrefix := b.fullPath(prefix)
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}

	containerClient := b.client.ServiceClient().NewContainerClient(b.containerName)

	listOpts := &container.ListBlobsHierarchyOptions{
		Prefix: &fullPrefix,
	}

	if opts.MaxResults > 0 {
		maxResults := int32(opts.MaxResults)
		listOpts.MaxResults = &maxResults
	}

	var files []plugin.FileInfo
	delimiter := "/"

	if opts.Recursive {
		// Flat listing for recursive
		pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
			Prefix:     &fullPrefix,
			MaxResults: listOpts.MaxResults,
		})

		for pager.More() {
			resp, err := pager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list blobs: %w", err)
			}

			for _, item := range resp.Segment.BlobItems {
				name := strings.TrimPrefix(*item.Name, fullPrefix)
				if name == "" {
					continue
				}
				files = append(files, plugin.FileInfo{
					Name:        name,
					Path:        *item.Name,
					Size:        *item.Properties.ContentLength,
					ModTime:     *item.Properties.LastModified,
					IsDir:       false,
					ETag:        string(*item.Properties.ETag),
					ContentType: safeString(item.Properties.ContentType),
				})
			}

			if opts.MaxResults > 0 && len(files) >= opts.MaxResults {
				break
			}
		}
	} else {
		// Hierarchical listing
		pager := containerClient.NewListBlobsHierarchyPager(delimiter, listOpts)

		for pager.More() {
			resp, err := pager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list blobs: %w", err)
			}

			// Add virtual directories
			for _, prefix := range resp.Segment.BlobPrefixes {
				name := strings.TrimPrefix(*prefix.Name, fullPrefix)
				name = strings.TrimSuffix(name, "/")
				files = append(files, plugin.FileInfo{
					Name:  name,
					Path:  *prefix.Name,
					IsDir: true,
				})
			}

			// Add blobs
			for _, item := range resp.Segment.BlobItems {
				name := strings.TrimPrefix(*item.Name, fullPrefix)
				if name == "" {
					continue
				}
				files = append(files, plugin.FileInfo{
					Name:        name,
					Path:        *item.Name,
					Size:        *item.Properties.ContentLength,
					ModTime:     *item.Properties.LastModified,
					IsDir:       false,
					ETag:        string(*item.Properties.ETag),
					ContentType: safeString(item.Properties.ContentType),
				})
			}

			if opts.MaxResults > 0 && len(files) >= opts.MaxResults {
				break
			}
		}
	}

	return files, nil
}

// Mkdir creates a virtual directory (empty blob with trailing slash).
func (b *Backend) Mkdir(ctx context.Context, dirPath string, opts plugin.MkdirOptions) error {
	fullPath := b.fullPath(dirPath)
	if !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}

	blockBlobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlockBlobClient(fullPath)
	_, err := blockBlobClient.Upload(ctx, newReadSeekCloser([]byte{}), nil)
	if err != nil {
		return fmt.Errorf("failed to create directory marker: %w", err)
	}

	return nil
}

// Rmdir removes a virtual directory and optionally its contents.
func (b *Backend) Rmdir(ctx context.Context, dirPath string, recursive bool) error {
	fullPath := b.fullPath(dirPath)
	if !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}

	containerClient := b.client.ServiceClient().NewContainerClient(b.containerName)

	if recursive {
		// List and delete all blobs with this prefix
		pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
			Prefix: &fullPath,
		})

		for pager.More() {
			resp, err := pager.NextPage(ctx)
			if err != nil {
				return fmt.Errorf("failed to list blobs for deletion: %w", err)
			}

			for _, item := range resp.Segment.BlobItems {
				blobClient := containerClient.NewBlobClient(*item.Name)
				_, err := blobClient.Delete(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to delete blob %s: %w", *item.Name, err)
				}
			}
		}
	} else {
		// Just delete the directory marker
		blobClient := containerClient.NewBlobClient(fullPath)
		_, err := blobClient.Delete(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to delete directory marker: %w", err)
		}
	}

	return nil
}

// Lock is not supported by Azure Blob Storage (use lease instead).
func (b *Backend) Lock(ctx context.Context, path string, opts plugin.LockOptions) (plugin.Unlocker, error) {
	return nil, plugin.ErrNotSupported
}

// Symlink is not supported by Azure Blob Storage.
func (b *Backend) Symlink(ctx context.Context, target, link string) error {
	return plugin.ErrNotSupported
}

// Chmod is not supported by Azure Blob Storage.
func (b *Backend) Chmod(ctx context.Context, path string, mode os.FileMode) error {
	return plugin.ErrNotSupported
}

// Chown is not supported by Azure Blob Storage.
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

// Azure-specific operations

// CopyBlob copies a blob within the same container.
func (b *Backend) CopyBlob(ctx context.Context, srcBlob, dstBlob string) error {
	srcPath := b.fullPath(srcBlob)
	dstPath := b.fullPath(dstBlob)

	srcBlobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(srcPath)
	dstBlobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(dstPath)

	_, err := dstBlobClient.StartCopyFromURL(ctx, srcBlobClient.URL(), nil)
	if err != nil {
		return fmt.Errorf("failed to copy blob: %w", err)
	}

	return nil
}

// SetMetadata updates blob metadata.
func (b *Backend) SetMetadata(ctx context.Context, blobName string, metadata map[string]*string) error {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	_, err := blobClient.SetMetadata(ctx, metadata, nil)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

// GetTags retrieves blob tags.
func (b *Backend) GetTags(ctx context.Context, blobName string) (map[string]string, error) {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	resp, err := blobClient.GetTags(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	tags := make(map[string]string)
	if resp.BlobTagSet != nil {
		for _, tag := range resp.BlobTagSet {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	return tags, nil
}

// SetTags sets blob tags.
func (b *Backend) SetTags(ctx context.Context, blobName string, tags map[string]string) error {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)

	_, err := blobClient.SetTags(ctx, tags, nil)
	if err != nil {
		return fmt.Errorf("failed to set tags: %w", err)
	}

	return nil
}

// SetAccessTier changes the blob access tier.
func (b *Backend) SetAccessTier(ctx context.Context, blobName string, tier string) error {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)

	accessTier := blob.AccessTier(tier)
	_, err := blobClient.SetTier(ctx, accessTier, nil)
	if err != nil {
		return fmt.Errorf("failed to set access tier: %w", err)
	}

	return nil
}

// AcquireLease acquires a lease on the blob.
func (b *Backend) AcquireLease(ctx context.Context, blobName string, duration int32) (string, error) {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	leaseClient, err := lease.NewBlobClient(blobClient, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create lease client: %w", err)
	}

	resp, err := leaseClient.AcquireLease(ctx, duration, nil)
	if err != nil {
		return "", fmt.Errorf("failed to acquire lease: %w", err)
	}

	return *resp.LeaseID, nil
}

// ReleaseLease releases a lease on the blob.
func (b *Backend) ReleaseLease(ctx context.Context, blobName, leaseID string) error {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	leaseClient, err := lease.NewBlobClient(blobClient, &lease.BlobClientOptions{LeaseID: &leaseID})
	if err != nil {
		return fmt.Errorf("failed to create lease client: %w", err)
	}

	_, err = leaseClient.ReleaseLease(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to release lease: %w", err)
	}

	return nil
}

// HeadBlob retrieves extended blob metadata.
func (b *Backend) HeadBlob(ctx context.Context, blobName string) (*AzureBlobInfo, error) {
	fullPath := b.fullPath(blobName)

	blobClient := b.client.ServiceClient().NewContainerClient(b.containerName).NewBlobClient(fullPath)
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if isNotFoundError(err) {
			return nil, plugin.ErrPathNotFound
		}
		return nil, fmt.Errorf("failed to get blob properties: %w", err)
	}

	info := &AzureBlobInfo{
		FileInfo: &plugin.FileInfo{
			Name:        path.Base(blobName),
			Path:        blobName,
			Size:        *props.ContentLength,
			ModTime:     *props.LastModified,
			IsDir:       false,
			ETag:        string(*props.ETag),
			ContentType: *props.ContentType,
			VersionID:   safeString(props.VersionID),
		},
		AccessTier:      ptrToString(props.AccessTier),
		LeaseStatus:     ptrToString(props.LeaseStatus),
		LeaseState:      ptrToString(props.LeaseState),
		BlobType:        ptrToString(props.BlobType),
		ServerEncrypted: safeServerEncrypted(props.IsServerEncrypted),
	}

	if props.Metadata != nil {
		info.FileInfo.Metadata = make(map[string]string)
		for k, v := range props.Metadata {
			if v != nil {
				info.FileInfo.Metadata[k] = *v
			}
		}
	}

	// Get tags
	tagsResp, err := blobClient.GetTags(ctx, nil)
	if err == nil && tagsResp.BlobTagSet != nil {
		info.FileInfo.Tags = make(map[string]string)
		for _, tag := range tagsResp.BlobTagSet {
			if tag.Key != nil && tag.Value != nil {
				info.FileInfo.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return info, nil
}

// Helper functions

func (b *Backend) fullPath(blobName string) string {
	if b.pathPrefix == "" {
		return blobName
	}
	return path.Join(b.pathPrefix, blobName)
}

func isNotFoundError(err error) bool {
	return bloberror.HasCode(err, bloberror.BlobNotFound) ||
		bloberror.HasCode(err, bloberror.ContainerNotFound)
}

// readSeekCloser wraps bytes.Reader to implement io.ReadSeekCloser.
type readSeekCloser struct {
	*bytes.Reader
}

func (r *readSeekCloser) Close() error {
	return nil
}

func newReadSeekCloser(data []byte) io.ReadSeekCloser {
	return &readSeekCloser{Reader: bytes.NewReader(data)}
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrToString(v any) string {
	if v == nil {
		return ""
	}
	// Handle various string-based types from Azure SDK
	switch t := v.(type) {
	case *string:
		if t == nil {
			return ""
		}
		return *t
	default:
		// For generated types like *LeaseStatusType, *LeaseStateType, *BlobType
		// which are string-based but not directly accessible
		return fmt.Sprintf("%v", v)
	}
}

func safeServerEncrypted(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// Ensure Backend implements AzureBackend.
var _ AzureBackend = (*Backend)(nil)
