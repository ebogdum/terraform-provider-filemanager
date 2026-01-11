// SPDX-License-Identifier: MIT

// Package formats provides format plugin implementations for structured content handling.
package formats

import (
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Re-export types from plugin package for convenience.
type (
	FormatPlugin     = plugin.FormatPlugin
	SerializeOptions = plugin.SerializeOptions
	MergeStrategy    = plugin.MergeStrategy
	ValidationError  = plugin.ValidationError
	FormatSchema     = plugin.FormatSchema
	SchemaAttribute  = plugin.SchemaAttribute
)

// Merge strategy constants.
const (
	MergeReplace = plugin.MergeReplace
	MergeDeep    = plugin.MergeDeep
	MergeAppend  = plugin.MergeAppend
	MergeConcat  = plugin.MergeConcat
	MergeUnion   = plugin.MergeUnion
)
