// SPDX-License-Identifier: MIT

//go:build !windows

package local

import (
	"os"
	"syscall"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// fillUnixInfo fills Unix-specific file information.
func (b *Backend) fillUnixInfo(info os.FileInfo, fileInfo *plugin.FileInfo) {
	if sys := info.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			fileInfo.UID = int(stat.Uid)
			fileInfo.GID = int(stat.Gid)
			fillStatTimes(stat, fileInfo)
		}
	}
}
