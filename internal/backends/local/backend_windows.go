// SPDX-License-Identifier: MIT

//go:build windows

package local

import (
	"os"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// fillUnixInfo is a no-op on Windows as it uses different mechanisms.
func (b *Backend) fillUnixInfo(info os.FileInfo, fileInfo *plugin.FileInfo) {
	// Windows uses different structures for file information
}
