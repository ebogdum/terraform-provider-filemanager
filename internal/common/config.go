// SPDX-License-Identifier: MIT

// Package common provides shared types and utilities for the filemanager provider.
package common

import (
	"time"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// ProviderConfig holds the provider configuration for use by resources.
type ProviderConfig struct {
	Registry                   *plugin.Registry
	LocalBackend               plugin.Backend
	BasePath                   string
	DefaultFilePermission      string
	DefaultDirectoryPermission string
	AtomicWrites               bool
	VerifyChecksum             bool
	EnableLocking              bool
	LockTimeout                time.Duration
	BackupEnabled              bool
	BackupRetention            int
	BackupDir                  string
}
