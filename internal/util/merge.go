// SPDX-License-Identifier: MIT

// Package util provides common utility functions for the provider.
package util

import (
	"fmt"
	"reflect"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// DeepMerge performs a deep merge of two values, with overlay taking precedence.
func DeepMerge(base, overlay any) (any, error) {
	if overlay == nil {
		return base, nil
	}
	if base == nil {
		return overlay, nil
	}

	baseMap, baseOk := base.(map[string]any)
	overlayMap, overlayOk := overlay.(map[string]any)

	if !baseOk || !overlayOk {
		// If types don't match or aren't maps, overlay wins
		return overlay, nil
	}

	result := make(map[string]any)

	// Copy base
	for k, v := range baseMap {
		result[k] = v
	}

	// Merge overlay
	for k, v := range overlayMap {
		if baseVal, exists := result[k]; exists {
			// Both have the key, try to deep merge
			merged, err := DeepMerge(baseVal, v)
			if err != nil {
				return nil, err
			}
			result[k] = merged
		} else {
			result[k] = v
		}
	}

	return result, nil
}

// DiffConfigs compares two configurations and returns a list of changes.
func DiffConfigs(old, new any) ([]plugin.Change, error) {
	return diffValues(old, new, "")
}

func diffValues(old, new any, path string) ([]plugin.Change, error) {
	var changes []plugin.Change

	// Handle nil cases
	if old == nil && new == nil {
		return changes, nil
	}
	if old == nil {
		changes = append(changes, plugin.Change{
			Path:     path,
			Type:     "added",
			OldValue: nil,
			NewValue: new,
		})
		return changes, nil
	}
	if new == nil {
		changes = append(changes, plugin.Change{
			Path:     path,
			Type:     "removed",
			OldValue: old,
			NewValue: nil,
		})
		return changes, nil
	}

	// Check if types match
	oldType := reflect.TypeOf(old)
	newType := reflect.TypeOf(new)
	if oldType != newType {
		changes = append(changes, plugin.Change{
			Path:     path,
			Type:     "changed",
			OldValue: old,
			NewValue: new,
		})
		return changes, nil
	}

	// Handle different types
	switch oldVal := old.(type) {
	case map[string]any:
		newVal := new.(map[string]any)
		return diffMaps(oldVal, newVal, path)
	case []any:
		newVal := new.([]any)
		return diffSlices(oldVal, newVal, path)
	default:
		if !reflect.DeepEqual(old, new) {
			changes = append(changes, plugin.Change{
				Path:     path,
				Type:     "changed",
				OldValue: old,
				NewValue: new,
			})
		}
	}

	return changes, nil
}

func diffMaps(old, new map[string]any, path string) ([]plugin.Change, error) {
	var changes []plugin.Change

	// Check for removed and changed keys
	for k, oldVal := range old {
		keyPath := k
		if path != "" {
			keyPath = path + "." + k
		}

		if newVal, exists := new[k]; exists {
			subChanges, err := diffValues(oldVal, newVal, keyPath)
			if err != nil {
				return nil, err
			}
			changes = append(changes, subChanges...)
		} else {
			changes = append(changes, plugin.Change{
				Path:     keyPath,
				Type:     "removed",
				OldValue: oldVal,
				NewValue: nil,
			})
		}
	}

	// Check for added keys
	for k, newVal := range new {
		if _, exists := old[k]; !exists {
			keyPath := k
			if path != "" {
				keyPath = path + "." + k
			}
			changes = append(changes, plugin.Change{
				Path:     keyPath,
				Type:     "added",
				OldValue: nil,
				NewValue: newVal,
			})
		}
	}

	return changes, nil
}

func diffSlices(old, new []any, path string) ([]plugin.Change, error) {
	var changes []plugin.Change

	// Simple slice comparison - compare by index
	maxLen := len(old)
	if len(new) > maxLen {
		maxLen = len(new)
	}

	for i := 0; i < maxLen; i++ {
		indexPath := fmt.Sprintf("%s[%d]", path, i)
		if path == "" {
			indexPath = fmt.Sprintf("[%d]", i)
		}

		if i >= len(old) {
			changes = append(changes, plugin.Change{
				Path:     indexPath,
				Type:     "added",
				OldValue: nil,
				NewValue: new[i],
			})
		} else if i >= len(new) {
			changes = append(changes, plugin.Change{
				Path:     indexPath,
				Type:     "removed",
				OldValue: old[i],
				NewValue: nil,
			})
		} else {
			subChanges, err := diffValues(old[i], new[i], indexPath)
			if err != nil {
				return nil, err
			}
			changes = append(changes, subChanges...)
		}
	}

	return changes, nil
}
