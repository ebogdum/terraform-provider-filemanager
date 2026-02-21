// SPDX-License-Identifier: MIT

// Package json provides JSON format plugin implementation.
package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Format implements the FormatPlugin interface for JSON.
type Format struct{}

// New creates a new JSON format plugin.
func New() *Format {
	return &Format{}
}

// Name returns the format name.
func (f *Format) Name() string {
	return "json"
}

// Extensions returns the supported file extensions.
func (f *Format) Extensions() []string {
	return []string{".json"}
}

// MimeTypes returns the supported MIME types.
func (f *Format) MimeTypes() []string {
	return []string{"application/json", "text/json"}
}

// Parse parses JSON data into a Go value.
func (f *Format) Parse(data []byte) (any, error) {
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}
	return result, nil
}

// Serialize serializes a Go value to JSON.
func (f *Format) Serialize(value any, opts plugin.SerializeOptions) ([]byte, error) {
	// Handle sorting keys if requested
	if opts.SortKeys {
		value = sortKeys(value)
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	if opts.Compact {
		encoder.SetIndent("", "")
	} else {
		indent := "  "
		if opts.Indent > 0 {
			if opts.IndentChar != "" {
				indent = strings.Repeat(opts.IndentChar, opts.Indent)
			} else {
				indent = strings.Repeat(" ", opts.Indent)
			}
		}
		encoder.SetIndent("", indent)
	}

	if !opts.EscapeHTML {
		encoder.SetEscapeHTML(false)
	}

	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("JSON serialize error: %w", err)
	}

	result := buf.Bytes()

	// Remove trailing newline if not wanted
	if !opts.TrailingNewline && len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// Merge merges two JSON values according to the strategy.
func (f *Format) Merge(base, overlay any, strategy plugin.MergeStrategy) (any, error) {
	switch strategy {
	case plugin.MergeReplace:
		return overlay, nil

	case plugin.MergeDeep:
		return deepMerge(base, overlay), nil

	case plugin.MergeAppend:
		return appendMerge(base, overlay), nil

	case plugin.MergeConcat:
		return concatMerge(base, overlay), nil

	case plugin.MergeUnion:
		return unionMerge(base, overlay), nil

	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// Query queries a JSON value using JSONPath-like syntax.
func (f *Format) Query(data any, path string) (any, error) {
	return jsonQuery(data, path)
}

// Set sets a value at the specified path.
func (f *Format) Set(data any, path string, value any) (any, error) {
	return jsonSet(data, path, value)
}

// Delete deletes a value at the specified path.
func (f *Format) Delete(data any, path string) (any, error) {
	return jsonDelete(data, path)
}

// Validate validates JSON data structure.
// Note: Full JSON Schema validation is not implemented.
// This performs basic structural validation.
func (f *Format) Validate(data any, schema any) ([]plugin.ValidationError, error) {
	// If data is nil, it's valid (empty JSON)
	if data == nil {
		return nil, nil
	}

	// JSON data should be a map, slice, or primitive type
	switch v := data.(type) {
	case map[string]any:
		// Valid JSON object - recursively validate nested values
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
		// Valid JSON array - recursively validate elements
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
	case string, float64, bool:
		// Valid JSON primitives
		return nil, nil
	case int, int64, float32:
		// Also valid (from Go type system)
		return nil, nil
	default:
		// Unknown type - might be invalid
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("unexpected JSON type: %T", v),
				Value:   v,
			},
		}, nil
	}
}

// GetSchema returns the Terraform schema for JSON-specific attributes.
func (f *Format) GetSchema() plugin.FormatSchema {
	return plugin.FormatSchema{
		Attributes: map[string]plugin.SchemaAttribute{
			"sort_keys": {
				Type:        "bool",
				Optional:    true,
				Description: "Sort object keys alphabetically",
			},
			"indent": {
				Type:        "number",
				Optional:    true,
				Default:     2,
				Description: "Indentation spaces",
			},
			"compact": {
				Type:        "bool",
				Optional:    true,
				Description: "Output compact JSON without whitespace",
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
		// Copy base
		for k, v := range baseMap {
			result[k] = v
		}
		// Merge overlay
		for k, v := range overlayMap {
			if existing, ok := result[k]; ok {
				result[k] = deepMerge(existing, v)
			} else {
				result[k] = v
			}
		}
		return result
	}

	// For non-maps, overlay wins
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

// concatMerge concatenates arrays and deep merges objects.
func concatMerge(base, overlay any) any {
	return appendMerge(base, overlay)
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
		// Create union (unique values)
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

// jsonQuery implements a simple JSONPath-like query.
// Supports: .key, [index], .key1.key2
func jsonQuery(data any, path string) (any, error) {
	if path == "" || path == "." || path == "$" {
		return data, nil
	}

	// Remove leading $ or .
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")

	parts := splitPath(path)
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

		case []any:
			// Parse array index
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				return nil, fmt.Errorf("invalid array index: %s", part)
			}
			if idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("array index out of bounds: %d", idx)
			}
			current = v[idx]

		default:
			return nil, fmt.Errorf("cannot traverse %T with key %s", current, part)
		}
	}

	return current, nil
}

// jsonSet sets a value at the specified path.
func jsonSet(data any, path string, value any) (any, error) {
	if path == "" || path == "." || path == "$" {
		return value, nil
	}

	// Remove leading $ or .
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")

	parts := splitPath(path)

	// Deep copy the data
	result := deepCopy(data)
	if result == nil {
		result = make(map[string]any)
	}

	// Navigate to parent and set value
	current := result
	for i, part := range parts[:len(parts)-1] {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				// Create intermediate objects
				v[part] = make(map[string]any)
				next = v[part]
			}
			current = next

		case []any:
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				return nil, fmt.Errorf("invalid array index at %d: %s", i, part)
			}
			if idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("array index out of bounds at %d: %d", i, idx)
			}
			current = v[idx]

		default:
			return nil, fmt.Errorf("cannot set in %T at %s", current, part)
		}
	}

	// Set the final value
	lastPart := parts[len(parts)-1]
	switch v := current.(type) {
	case map[string]any:
		v[lastPart] = value

	case []any:
		var idx int
		if _, err := fmt.Sscanf(lastPart, "%d", &idx); err != nil {
			return nil, fmt.Errorf("invalid array index: %s", lastPart)
		}
		if idx < 0 || idx >= len(v) {
			return nil, fmt.Errorf("array index out of bounds: %d", idx)
		}
		v[idx] = value

	default:
		return nil, fmt.Errorf("cannot set in %T", current)
	}

	return result, nil
}

// jsonDelete deletes a value at the specified path.
func jsonDelete(data any, path string) (any, error) {
	if path == "" || path == "." || path == "$" {
		return nil, nil
	}

	// Remove leading $ or .
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")

	parts := splitPath(path)

	// Deep copy the data
	result := deepCopy(data)
	if result == nil {
		return nil, nil
	}

	current, parents, err := navigateToDeleteParent(result, parts)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return result, nil
	}

	// Delete the final key
	lastPart := parts[len(parts)-1]
	switch v := current.(type) {
	case map[string]any:
		delete(v, lastPart)

	case []any:
		idx, err := parseArrayIndex(lastPart)
		if err != nil {
			return nil, fmt.Errorf("invalid array index: %s", lastPart)
		}
		if idx >= 0 && idx < len(v) {
			// Remove element at index
			newSlice := append(v[:idx], v[idx+1:]...)
			// Update parent reference to the new slice
			if len(parents) == 0 {
				// The array is at root level
				return newSlice, nil
			}
			if err := assignArrayToParent(parents[len(parents)-1], newSlice); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

type jsonParentInfo struct {
	container any
	key       string
}

func navigateToDeleteParent(root any, parts []string) (any, []jsonParentInfo, error) {
	var parents []jsonParentInfo
	current := root

	for i, part := range parts[:len(parts)-1] {
		if part == "" {
			continue
		}

		next, ok, err := nextJSONDeleteNode(current, part, i)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, parents, nil
		}

		parents = append(parents, jsonParentInfo{container: current, key: part})
		current = next
	}
	return current, parents, nil
}

func nextJSONDeleteNode(current any, part string, idx int) (any, bool, error) {
	switch v := current.(type) {
	case map[string]any:
		next, ok := v[part]
		return next, ok, nil
	case []any:
		arrIdx, err := parseArrayIndex(part)
		if err != nil {
			return nil, false, fmt.Errorf("invalid array index at %d: %s", idx, part)
		}
		if arrIdx < 0 || arrIdx >= len(v) {
			return nil, false, nil
		}
		return v[arrIdx], true, nil
	default:
		return nil, false, nil
	}
}

func parseArrayIndex(raw string) (int, error) {
	var idx int
	if _, err := fmt.Sscanf(raw, "%d", &idx); err != nil {
		return 0, err
	}
	return idx, nil
}

func assignArrayToParent(parent jsonParentInfo, newSlice []any) error {
	switch p := parent.container.(type) {
	case map[string]any:
		p[parent.key] = newSlice
		return nil
	case []any:
		pIdx, err := parseArrayIndex(parent.key)
		if err != nil {
			return fmt.Errorf("invalid parent array index: %s", parent.key)
		}
		p[pIdx] = newSlice
		return nil
	default:
		return nil
	}
}

// splitPath splits a path like "foo.bar[0].baz" into parts.
func splitPath(path string) []string {
	var parts []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		c := path[i]
		switch c {
		case '.':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case ']':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
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
		// For primitive types, just return as-is (they're immutable)
		return v
	}
}
