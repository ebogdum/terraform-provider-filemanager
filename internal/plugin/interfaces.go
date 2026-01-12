// SPDX-License-Identifier: MIT

// Package plugin defines the core plugin interfaces for the filemanager provider.
// All plugin types (Backend, Format, App) implement these interfaces to provide
// extensible functionality for storage, content handling, and application configs.
package plugin

import (
	"context"
	"io"
	"os"
	"time"
)

// -----------------------------------------------------------------------------
// Backend Plugin Interface
// -----------------------------------------------------------------------------

// Backend defines the interface that all storage backend plugins must implement.
// Backends handle where files are stored and how they're accessed (local, SSH, S3, etc.)
type Backend interface {
	// Identity
	Name() string   // Human-readable name (e.g., "local", "ssh", "s3")
	Scheme() string // URI scheme (e.g., "file", "ssh", "s3", "azure", "gs")

	// Lifecycle
	Connect(ctx context.Context, config BackendConfig) error
	Close() error
	Ping(ctx context.Context) error

	// File operations
	Read(ctx context.Context, path string) (io.ReadCloser, error)
	Write(ctx context.Context, path string, r io.Reader, opts WriteOptions) error
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	Stat(ctx context.Context, path string) (*FileInfo, error)

	// Directory operations
	List(ctx context.Context, path string, opts ListOptions) ([]FileInfo, error)
	Mkdir(ctx context.Context, path string, opts MkdirOptions) error
	Rmdir(ctx context.Context, path string, recursive bool) error

	// Advanced operations (return ErrNotSupported if unavailable)
	Lock(ctx context.Context, path string, opts LockOptions) (Unlocker, error)
	Symlink(ctx context.Context, target, link string) error
	Chmod(ctx context.Context, path string, mode os.FileMode) error
	Chown(ctx context.Context, path string, uid, gid int) error

	// Capabilities
	Capabilities() BackendCapabilities
}

// BackendConfig contains configuration for backend connection.
type BackendConfig struct {
	// Common settings
	BasePath string
	Timeout  time.Duration

	// Authentication
	Username   string
	Password   string
	PrivateKey []byte
	Token      string

	// Connection settings
	Host     string
	Port     int
	Endpoint string

	// TLS settings
	TLSEnabled    bool
	TLSSkipVerify bool
	TLSCert       []byte
	TLSKey        []byte
	TLSCA         []byte

	// Backend-specific settings (S3 bucket, Azure container, etc.)
	Extra map[string]any
}

// BackendCapabilities describes what operations a backend supports.
type BackendCapabilities struct {
	// File operations
	SupportsRead   bool
	SupportsWrite  bool
	SupportsDelete bool
	SupportsStat   bool

	// Directory operations
	SupportsList  bool
	SupportsMkdir bool
	SupportsRmdir bool

	// Advanced operations
	SupportsLocking  bool
	SupportsSymlinks bool
	SupportsChmod    bool
	SupportsChown    bool

	// Transfer optimizations
	SupportsRangeRead        bool // Can read byte ranges
	SupportsMultipartUpload  bool // Can do multipart uploads
	SupportsDirectTransfer   bool // Can copy directly to another backend
	SupportsSendfile         bool // Supports sendfile() syscall
	SupportsSplice           bool // Supports splice() syscall
	SupportsCopyFileRange    bool // Supports copy_file_range() syscall
	SupportsMemoryMapping    bool // Supports mmap
	SupportsStreamingWrite   bool // Can write from a stream
	SupportsAtomicWrite      bool // Supports atomic write operations
	SupportsConcurrentAccess bool // Safe for concurrent access

	// Metadata
	SupportsMetadata   bool // Supports custom metadata/tags
	SupportsVersioning bool // Supports file versioning
	SupportsChecksum   bool // Returns checksums on operations

	// Maximum limits
	MaxFileSize      int64 // Maximum file size (0 = unlimited)
	MaxPathLength    int   // Maximum path length
	MaxFilenameBytes int   // Maximum filename length in bytes
}

// WriteOptions contains options for write operations.
type WriteOptions struct {
	Mode             os.FileMode
	Overwrite        bool
	CreateDirs       bool
	DirMode          os.FileMode
	Atomic           bool
	Checksum         string // Expected checksum for verification
	ChecksumAlgo     string // Checksum algorithm (md5, sha256, etc.)
	ContentType      string
	Metadata         map[string]string
	VerifyAfterWrite bool
}

// ListOptions contains options for directory listing.
type ListOptions struct {
	Recursive     bool
	IncludeHidden bool
	Pattern       string // Glob pattern filter
	MaxResults    int
	Offset        int
}

// MkdirOptions contains options for directory creation.
type MkdirOptions struct {
	Mode      os.FileMode
	Recursive bool
}

// LockOptions contains options for file locking.
type LockOptions struct {
	Exclusive bool          // Exclusive (write) lock vs shared (read) lock
	Timeout   time.Duration // How long to wait for lock
	Block     bool          // Whether to block waiting for lock
}

// Unlocker is returned by Lock() and used to release the lock.
type Unlocker interface {
	Unlock() error
}

// CommandExecutor is an optional interface for backends that support command execution.
// Backends like SSH can implement this to allow running remote commands.
type CommandExecutor interface {
	Execute(ctx context.Context, command string) ([]byte, error)
}

// FileInfo contains file metadata.
type FileInfo struct {
	Name    string
	Path    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool

	// Unix-specific
	UID int
	GID int

	// Checksums
	MD5    string
	SHA256 string
	SHA512 string
	CRC32  string
	ETag   string // S3/Azure ETag

	// Cloud-specific
	ContentType    string
	StorageClass   string
	VersionID      string
	Metadata       map[string]string
	Tags           map[string]string
	LastAccessTime time.Time
	CreationTime   time.Time

	// Link info
	IsSymlink  bool
	LinkTarget string
}

// -----------------------------------------------------------------------------
// Format Plugin Interface
// -----------------------------------------------------------------------------

// FormatPlugin defines the interface for content format handlers.
// Format plugins handle parsing, serialization, merging, and validation
// of structured content (JSON, YAML, TOML, etc.)
type FormatPlugin interface {
	// Identity
	Name() string         // Format name (e.g., "json", "yaml", "toml")
	Extensions() []string // File extensions (e.g., [".json"], [".yaml", ".yml"])
	MimeTypes() []string  // MIME types (e.g., ["application/json"])

	// Core operations
	Parse(data []byte) (any, error)
	Serialize(value any, opts SerializeOptions) ([]byte, error)

	// Structured operations
	Merge(base, overlay any, strategy MergeStrategy) (any, error)
	Query(data any, path string) (any, error) // JSONPath, YAMLPath, etc.
	Set(data any, path string, value any) (any, error)
	Delete(data any, path string) (any, error)

	// Validation
	Validate(data any, schema any) ([]ValidationError, error)

	// Schema generation for Terraform
	GetSchema() FormatSchema
}

// SerializeOptions contains options for content serialization.
type SerializeOptions struct {
	Indent           int    // Indentation level (spaces)
	IndentChar       string // Indent character (" " or "\t")
	SortKeys         bool   // Sort object keys
	PreserveComments bool   // Preserve comments (YAML/TOML)
	Compact          bool   // Minimize whitespace
	EscapeHTML       bool   // Escape HTML characters (JSON)
	TrailingNewline  bool   // Add trailing newline
}

// MergeStrategy defines how to merge two structured values.
type MergeStrategy string

const (
	MergeReplace MergeStrategy = "replace" // Overlay completely replaces base
	MergeDeep    MergeStrategy = "deep"    // Recursive deep merge
	MergeAppend  MergeStrategy = "append"  // Append arrays, merge objects
	MergeConcat  MergeStrategy = "concat"  // Concatenate arrays, merge objects
	MergeUnion   MergeStrategy = "union"   // Union of arrays, merge objects
)

// ValidationError represents a validation error.
type ValidationError struct {
	Path    string // JSON/YAML path to the error
	Message string // Error message
	Value   any    // The invalid value
	Schema  any    // The schema that was violated
}

// FormatSchema describes the schema for format-specific Terraform attributes.
type FormatSchema struct {
	// Attributes specific to this format
	Attributes map[string]SchemaAttribute
}

// SchemaAttribute describes a single schema attribute.
type SchemaAttribute struct {
	Type        string // "string", "number", "bool", "list", "map", "object"
	Required    bool
	Optional    bool
	Computed    bool
	Sensitive   bool
	Description string
	Default     any
}

// -----------------------------------------------------------------------------
// Application Plugin Interface
// -----------------------------------------------------------------------------

// AppPlugin defines the interface for application-specific config handlers.
// App plugins understand the structure, validation, and semantics of specific
// application configurations (nginx, consul, prometheus, etc.)
type AppPlugin interface {
	// Identity
	Name() string        // Application name (e.g., "nginx", "consul")
	Version() string     // Supported app version range
	Description() string // Human-readable description

	// Configuration structure
	Schema() AppSchema  // Define expected config structure
	DefaultConfig() any // Sensible defaults

	// Validation
	Validate(config any) ([]ValidationError, error)
	ValidateSemantic(config any) ([]ValidationError, error) // App-specific rules

	// Transformation
	Normalize(config any) (any, error)   // Normalize to canonical form
	ToNative(config any) ([]byte, error) // Convert to native config format
	FromNative(data []byte) (any, error) // Parse native config format

	// Operations
	Merge(base, overlay any) (any, error) // App-aware merge
	Diff(old, new any) ([]Change, error)  // Detect changes

	// Format info
	NativeFormat() string // The native format ("nginx", "yaml", "json", etc.)
}

// AppSchema describes the schema for an application's configuration.
type AppSchema struct {
	Sections   []SectionSchema   // Top-level sections
	Directives []DirectiveSchema // Top-level directives
	Validators []ValidatorFunc   // Custom validators
}

// SectionSchema describes a configuration section.
type SectionSchema struct {
	Name        string
	Required    bool
	Multiple    bool // Can appear multiple times
	Description string
	Directives  []DirectiveSchema
	Subsections []SectionSchema
}

// DirectiveSchema describes a configuration directive.
type DirectiveSchema struct {
	Name         string
	Required     bool
	Multiple     bool   // Can appear multiple times
	Type         string // "string", "int", "bool", "duration", "size", etc.
	Default      any
	ValidValues  []string // Allowed values (enum)
	Pattern      string   // Regex pattern for validation
	Description  string
	Deprecated   bool
	DeprecatedBy string
}

// ValidatorFunc is a custom validation function.
type ValidatorFunc func(config any) []ValidationError

// Change represents a detected change between two configs.
type Change struct {
	Type     ChangeType // Added, Removed, Modified
	Path     string     // Path to the changed element
	OldValue any
	NewValue any
}

// ChangeType represents the type of change.
type ChangeType string

const (
	ChangeAdded    ChangeType = "added"
	ChangeRemoved  ChangeType = "removed"
	ChangeModified ChangeType = "modified"
)

// -----------------------------------------------------------------------------
// Transfer Engine Interface
// -----------------------------------------------------------------------------

// TransferEngine defines the interface for optimized data transfer operations.
// It supports zero-copy operations, streaming, and parallel transfers.
type TransferEngine interface {
	// Single file operations
	Upload(ctx context.Context, src io.Reader, dst BackendPath, opts TransferOptions) (*TransferResult, error)
	Download(ctx context.Context, src BackendPath, dst io.Writer, opts TransferOptions) (*TransferResult, error)
	Copy(ctx context.Context, src, dst BackendPath, opts TransferOptions) (*TransferResult, error)

	// Batch operations
	Sync(ctx context.Context, src, dst BackendPath, opts SyncOptions) (*SyncResult, error)
	BatchCopy(ctx context.Context, operations []CopyOperation, opts BatchOptions) (*BatchResult, error)

	// Direct backend-to-backend (zero-copy where possible)
	Transfer(ctx context.Context, src, dst BackendPath, opts TransferOptions) (*TransferResult, error)
}

// BackendPath identifies a path on a specific backend.
type BackendPath struct {
	Backend string // Backend name/alias
	Path    string // Path within the backend
}

// TransferOptions contains options for transfer operations.
type TransferOptions struct {
	// Performance
	BufferSize  int64 // Buffer size in bytes (default: 32KB)
	Concurrency int   // Number of parallel workers (default: 4)
	PartSize    int64 // Multipart chunk size (default: 64MB)
	ZeroCopy    bool  // Attempt zero-copy transfer (default: true)
	Streaming   bool  // Stream without buffering (default: true)

	// Reliability
	Retries        int           // Retry attempts (default: 3)
	RetryDelay     time.Duration // Delay between retries
	ChecksumVerify bool          // Verify checksum after transfer
	ResumePartial  bool          // Resume partial transfers

	// Bandwidth
	RateLimit int64 // Bytes per second limit (0 = unlimited)

	// Metadata
	PreserveMetadata bool // Copy metadata/tags
	PreserveMtime    bool // Preserve modification time
	PreserveMode     bool // Preserve file permissions
}

// SyncOptions extends TransferOptions with sync-specific settings.
type SyncOptions struct {
	TransferOptions

	// Sync behavior
	DeleteOrphans   bool // Delete files not in source
	UpdateOnly      bool // Only update existing files
	SizeOnly        bool // Compare by size only (skip checksum)
	ChecksumCompare bool // Compare by checksum (slow but accurate)

	// Filters
	Include []string      // Glob patterns to include
	Exclude []string      // Glob patterns to exclude
	MinSize int64         // Minimum file size
	MaxSize int64         // Maximum file size (0 = unlimited)
	MinAge  time.Duration // Minimum file age
	MaxAge  time.Duration // Maximum file age
}

// BatchOptions contains options for batch operations.
type BatchOptions struct {
	TransferOptions
	ContinueOnError bool // Continue processing on error
	MaxErrors       int  // Maximum errors before stopping
}

// CopyOperation describes a single copy operation in a batch.
type CopyOperation struct {
	Source      BackendPath
	Destination BackendPath
	Options     TransferOptions
}

// TransferResult contains the result of a transfer operation.
type TransferResult struct {
	BytesTransferred int64
	Duration         time.Duration
	Checksum         string
	ChecksumAlgo     string
	ETag             string
	VersionID        string
	Method           TransferMethod // How the transfer was performed
}

// TransferMethod describes how a transfer was performed.
type TransferMethod string

const (
	TransferMethodDirect          TransferMethod = "direct"           // Backend-native copy
	TransferMethodParallelChunked TransferMethod = "parallel_chunked" // Parallel multipart
	TransferMethodKernelZeroCopy  TransferMethod = "kernel_zero_copy" // sendfile/splice
	TransferMethodStreamingBuffer TransferMethod = "streaming_buffer" // io.Pipe streaming
	TransferMethodStandard        TransferMethod = "standard"         // Standard read/write
)

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	FilesTransferred int
	FilesDeleted     int
	FilesSkipped     int
	FilesFailed      int
	BytesTransferred int64
	Duration         time.Duration
	Errors           []error
}

// BatchResult contains the result of a batch operation.
type BatchResult struct {
	Successful int
	Failed     int
	Skipped    int
	Results    []TransferResult
	Errors     []error
	Duration   time.Duration
}
