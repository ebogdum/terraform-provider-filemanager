// SPDX-License-Identifier: MIT

//go:build linux

package local

import (
	"syscall"
	"time"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// fillStatTimes fills Linux-specific time fields from stat.
func fillStatTimes(stat *syscall.Stat_t, fileInfo *plugin.FileInfo) {
	fileInfo.LastAccessTime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	fileInfo.CreationTime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
}
