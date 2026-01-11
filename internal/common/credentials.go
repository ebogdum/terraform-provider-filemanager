// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"os"
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
		return "", fmt.Errorf("failed to read credential file %s: %w", valuePath, err)
	}

	return strings.TrimSpace(string(data)), nil
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand home directory: %w", err)
		}
		return home + path[1:], nil
	}
	return path, nil
}
