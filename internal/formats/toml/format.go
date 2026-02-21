// SPDX-License-Identifier: MIT

// Package toml provides TOML format plugin implementation.
package toml

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/pelletier/go-toml/v2"
)

// Format implements the FormatPlugin interface for TOML.
type Format struct{}

// New creates a new TOML format plugin.
func New() *Format {
	return &Format{}
}

// Name returns the format name.
func (f *Format) Name() string {
	return "toml"
}

// Extensions returns the supported file extensions.
func (f *Format) Extensions() []string {
	return []string{".toml"}
}

// MimeTypes returns the supported MIME types.
func (f *Format) MimeTypes() []string {
	return []string{"application/toml", "text/toml"}
}

// Parse parses TOML data into a Go value.
func (f *Format) Parse(data []byte) (any, error) {
	var result any
	if err := toml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("TOML parse error: %w", err)
	}
	return result, nil
}

// Serialize serializes a Go value to TOML.
func (f *Format) Serialize(value any, opts plugin.SerializeOptions) ([]byte, error) {
	// Handle sorting keys if requested
	if opts.SortKeys {
		value = sortKeys(value)
	}

	var buf bytes.Buffer

	encoder := toml.NewEncoder(&buf)
	if opts.Indent > 0 {
		if opts.IndentChar == "\t" {
			encoder.SetIndentTables(true)
		}
	}

	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("TOML serialize error: %w", err)
	}

	result := buf.Bytes()

	// Handle trailing newline
	if !opts.TrailingNewline && len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// Merge merges two TOML values according to the strategy.
func (f *Format) Merge(base, overlay any, strategy plugin.MergeStrategy) (any, error) {
	switch strategy {
	case plugin.MergeReplace:
		return overlay, nil

	case plugin.MergeDeep:
		return deepMerge(base, overlay), nil

	case plugin.MergeAppend:
		return appendMerge(base, overlay), nil

	case plugin.MergeConcat:
		return appendMerge(base, overlay), nil

	case plugin.MergeUnion:
		return unionMerge(base, overlay), nil

	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// Query queries a TOML value using dot notation.
func (f *Format) Query(data any, path string) (any, error) {
	return query(data, path)
}

// Set sets a value at the specified path.
func (f *Format) Set(data any, path string, value any) (any, error) {
	return set(data, path, value)
}

// Delete deletes a value at the specified path.
func (f *Format) Delete(data any, path string) (any, error) {
	return deletePath(data, path)
}

// Validate validates TOML data structure.
// Note: Full TOML schema validation is not implemented.
// This performs basic structural validation.
func (f *Format) Validate(data any, schema any) ([]plugin.ValidationError, error) {
	// If data is nil, it's valid (empty TOML)
	if data == nil {
		return nil, nil
	}

	// TOML data should be a map (tables) at the root level
	switch v := data.(type) {
	case map[string]any:
		// Valid TOML table - recursively validate nested values
		var errors []plugin.ValidationError
		for key, value := range v {
			if nested, _ := f.Validate(value, nil); len(nested) > 0 {
				for _, e := range nested {
					e.Path = key + "." + e.Path
					errors = append(errors, e)
				}
			}
		}
		return errors, nil
	case []any:
		// Valid TOML array - recursively validate elements
		var errors []plugin.ValidationError
		for i, item := range v {
			if nested, _ := f.Validate(item, nil); len(nested) > 0 {
				for _, e := range nested {
					e.Path = fmt.Sprintf("[%d].%s", i, e.Path)
					errors = append(errors, e)
				}
			}
		}
		return errors, nil
	case string, float64, int, int64, bool, time.Time:
		// Valid TOML value types
		return nil, nil
	default:
		// Unknown type - might be invalid
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("unexpected TOML type: %T", v),
				Value:   v,
			},
		}, nil
	}
}

// GetSchema returns the Terraform schema for TOML-specific attributes.
func (f *Format) GetSchema() plugin.FormatSchema {
	return plugin.FormatSchema{
		Attributes: map[string]plugin.SchemaAttribute{
			"sort_keys": {
				Type:        "bool",
				Optional:    true,
				Description: "Sort table keys alphabetically",
			},
		},
	}
}

// sortKeys recursively sorts map keys.
func sortKeys(v any) any {
	switch val := v.(type) {
	case map[string]any:
		sorted := make(map[string]any)
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sorted[k] = sortKeys(val[k])
		}
		return sorted
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = sortKeys(item)
		}
		return result
	default:
		return v
	}
}

// deepMerge performs a recursive deep merge of two values.
func deepMerge(base, overlay any) any {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	baseMap, baseIsMap := base.(map[string]any)
	overlayMap, overlayIsMap := overlay.(map[string]any)

	if baseIsMap && overlayIsMap {
		result := make(map[string]any)
		for k, v := range baseMap {
			result[k] = v
		}
		for k, v := range overlayMap {
			if existing, ok := result[k]; ok {
				result[k] = deepMerge(existing, v)
			} else {
				result[k] = v
			}
		}
		return result
	}

	return overlay
}

// appendMerge appends arrays and deep merges objects.
func appendMerge(base, overlay any) any {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	baseArr, baseIsArr := base.([]any)
	overlayArr, overlayIsArr := overlay.([]any)

	if baseIsArr && overlayIsArr {
		result := make([]any, len(baseArr)+len(overlayArr))
		copy(result, baseArr)
		copy(result[len(baseArr):], overlayArr)
		return result
	}

	return deepMerge(base, overlay)
}

// unionMerge creates a union of arrays and deep merges objects.
func unionMerge(base, overlay any) any {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	baseArr, baseIsArr := base.([]any)
	overlayArr, overlayIsArr := overlay.([]any)

	if baseIsArr && overlayIsArr {
		seen := make(map[string]bool)
		result := make([]any, 0)

		for _, v := range baseArr {
			key := fmt.Sprintf("%v", v)
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		}
		for _, v := range overlayArr {
			key := fmt.Sprintf("%v", v)
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		}
		return result
	}

	return deepMerge(base, overlay)
}

// query implements dot notation path query.
func query(data any, path string) (any, error) {
	if path == "" || path == "." {
		return data, nil
	}

	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("key not found: %s", part)
			}

		default:
			return nil, fmt.Errorf("cannot traverse %T with key %s", current, part)
		}
	}

	return current, nil
}

// set sets a value at the specified path.
func set(data any, path string, value any) (any, error) {
	if path == "" || path == "." {
		return value, nil
	}

	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")

	result := deepCopy(data)
	if result == nil {
		result = make(map[string]any)
	}

	current := result
	for i, part := range parts[:len(parts)-1] {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				v[part] = make(map[string]any)
				next = v[part]
			}
			current = next

		default:
			return nil, fmt.Errorf("cannot set in %T at %d: %s", current, i, part)
		}
	}

	lastPart := parts[len(parts)-1]
	switch v := current.(type) {
	case map[string]any:
		v[lastPart] = value
	default:
		return nil, fmt.Errorf("cannot set in %T", current)
	}

	return result, nil
}

// deletePath deletes a value at the specified path.
func deletePath(data any, path string) (any, error) {
	if path == "" || path == "." {
		return nil, nil
	}

	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")

	result := deepCopy(data)
	if result == nil {
		return nil, nil
	}

	current := result
	for _, part := range parts[:len(parts)-1] {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return result, nil
			}
			current = next
		default:
			return result, nil
		}
	}

	lastPart := parts[len(parts)-1]
	if v, ok := current.(map[string]any); ok {
		builtinDelete(v, lastPart)
	}

	return result, nil
}

func builtinDelete(v map[string]any, key string) {
	delete(v, key)
}

// deepCopy creates a deep copy of a value.
func deepCopy(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = deepCopy(v)
		}
		return result

	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = deepCopy(v)
		}
		return result

	default:
		return v
	}
}
