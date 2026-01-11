// SPDX-License-Identifier: MIT

//go:build darwin

package local

import (
	"syscall"
	"time"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// fillStatTimes fills Darwin-specific time fields from stat.
func fillStatTimes(stat *syscall.Stat_t, fileInfo *plugin.FileInfo) {
	fileInfo.LastAccessTime = time.Unix(int64(stat.Atimespec.Sec), int64(stat.Atimespec.Nsec))
	fileInfo.CreationTime = time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec))
}
