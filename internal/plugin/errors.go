// SPDX-License-Identifier: MIT

package plugin

import "errors"

// Plugin-related errors.
var (
	// ErrPluginNotFound is returned when a requested plugin is not registered.
	ErrPluginNotFound = errors.New("plugin not found")

	// ErrPluginExists is returned when attempting to register a duplicate plugin.
	ErrPluginExists = errors.New("plugin already exists")

	// ErrNotSupported is returned when an operation is not supported by a plugin.
	ErrNotSupported = errors.New("operation not supported")

	// ErrNotConnected is returned when a backend operation is attempted without connection.
	ErrNotConnected = errors.New("backend not connected")

	// ErrAlreadyConnected is returned when Connect is called on an already connected backend.
	ErrAlreadyConnected = errors.New("backend already connected")

	// ErrInvalidConfig is returned when plugin configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timed out")

	// ErrLockFailed is returned when a lock cannot be acquired.
	ErrLockFailed = errors.New("failed to acquire lock")

	// ErrNotLocked is returned when unlock is called without holding the lock.
	ErrNotLocked = errors.New("not locked")

	// ErrPathNotFound is returned when a file or directory does not exist.
	ErrPathNotFound = errors.New("path not found")

	// ErrPathExists is returned when a file or directory already exists.
	ErrPathExists = errors.New("path already exists")

	// ErrNotADirectory is returned when a path is expected to be a directory but is not.
	ErrNotADirectory = errors.New("not a directory")

	// ErrNotAFile is returned when a path is expected to be a file but is not.
	ErrNotAFile = errors.New("not a file")

	// ErrPermissionDenied is returned when the operation is not allowed.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrChecksumMismatch is returned when checksum verification fails.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrInvalidPath is returned when a path is malformed or invalid.
	ErrInvalidPath = errors.New("invalid path")

	// ErrReadOnly is returned when attempting to write to a read-only backend.
	ErrReadOnly = errors.New("backend is read-only")

	// ErrDirNotEmpty is returned when attempting to remove a non-empty directory.
	ErrDirNotEmpty = errors.New("directory not empty")

	// ErrTransferFailed is returned when a file transfer operation fails.
	ErrTransferFailed = errors.New("transfer failed")

	// ErrParseError is returned when content parsing fails.
	ErrParseError = errors.New("parse error")

	// ErrSerializeError is returned when content serialization fails.
	ErrSerializeError = errors.New("serialization error")

	// ErrValidationFailed is returned when validation fails.
	ErrValidationFailed = errors.New("validation failed")

	// ErrMergeFailed is returned when content merge fails.
	ErrMergeFailed = errors.New("merge failed")

	// ErrQueryFailed is returned when a path query fails.
	ErrQueryFailed = errors.New("query failed")

	// ErrSchemaViolation is returned when data violates a schema.
	ErrSchemaViolation = errors.New("schema violation")
)
