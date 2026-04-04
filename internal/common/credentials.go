// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"os"
	"os/user"
	"strings"
)

// ReadCredential returns the credential value, reading from file if path is provided.
// If value is non-empty, it's returned directly.
// If valuePath is non-empty, the file is read and its contents returned (trimmed).
// Returns empty string if both are empty.
func ReadCredential(value, valuePath string) (string, error) {
	if value != "" {
		return value, nil
	}
	if valuePath == "" {
		return "", nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(valuePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand home directory: %w", err)
		}
		valuePath = home + valuePath[1:]
	}

	data, err := os.ReadFile(valuePath)
	if err != nil {
		return "", fmt.Errorf("failed to read credential file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// ExpandPath expands ~ and ~username to the appropriate home directory.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	// ~/... → current user's home
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return home + path[1:], nil
	}
	// ~username/... → lookup that user's home
	slashIdx := strings.IndexByte(path, '/')
	var username string
	if slashIdx < 0 {
		username = path[1:]
	} else {
		username = path[1:slashIdx]
	}
	u, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("failed to lookup user %s: %w", username, err)
	}
	if slashIdx < 0 {
		return u.HomeDir, nil
	}
	return u.HomeDir + path[slashIdx:], nil
}
