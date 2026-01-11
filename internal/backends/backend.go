// SPDX-License-Identifier: MIT

// Package backends provides storage backend implementations for the filemanager provider.
package backends

import (
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Re-export types from plugin package for convenience.
type (
	Backend             = plugin.Backend
	BackendConfig       = plugin.BackendConfig
	BackendCapabilities = plugin.BackendCapabilities
	WriteOptions        = plugin.WriteOptions
	ListOptions         = plugin.ListOptions
	MkdirOptions        = plugin.MkdirOptions
	LockOptions         = plugin.LockOptions
	Unlocker            = plugin.Unlocker
	FileInfo            = plugin.FileInfo
)
