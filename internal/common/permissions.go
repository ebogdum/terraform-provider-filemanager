// SPDX-License-Identifier: MIT

package common

import (
	"os"
	"strconv"
	"strings"
)

// DefaultFileMode is the default permission for files.
const DefaultFileMode os.FileMode = 0644

// DefaultDirMode is the default permission for directories.
const DefaultDirMode os.FileMode = 0755

// ParseOctalMode parses an octal permission string (e.g., "0644") into os.FileMode.
// Returns the provided default if the string is empty or invalid.
func ParseOctalMode(s string, defaultMode os.FileMode) os.FileMode {
	if s == "" {
		return defaultMode
	}
	s = strings.TrimPrefix(s, "0")
	mode, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return defaultMode
	}
	return os.FileMode(mode)
}

// ParseFileMode parses an octal permission string with default file permissions (0644).
func ParseFileMode(s string) os.FileMode {
	return ParseOctalMode(s, DefaultFileMode)
}

// ParseDirMode parses an octal permission string with default directory permissions (0755).
func ParseDirMode(s string) os.FileMode {
	return ParseOctalMode(s, DefaultDirMode)
}

// FormatOctalMode formats an os.FileMode as an octal string (e.g., "0644").
func FormatOctalMode(mode os.FileMode) string {
	return "0" + strconv.FormatUint(uint64(mode.Perm()), 8)
}
