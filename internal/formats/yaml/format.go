// SPDX-License-Identifier: MIT

// Package yaml provides YAML format plugin implementation with comment preservation.
package yaml

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"gopkg.in/yaml.v3"
)

// Format implements the FormatPlugin interface for YAML.
type Format struct{}

// New creates a new YAML format plugin.
func New() *Format {
	return &Format{}
}

// Name returns the format name.
func (f *Format) Name() string {
	return "yaml"
}

// Extensions returns the supported file extensions.
func (f *Format) Extensions() []string {
	return []string{".yaml", ".yml"}
}

// MimeTypes returns the supported MIME types.
func (f *Format) MimeTypes() []string {
	return []string{"application/yaml", "application/x-yaml", "text/yaml", "text/x-yaml"}
}

const maxYAMLInputBytes = 10 * 1024 * 1024 // 10 MB

// Parse parses YAML data into a Go value.
func (f *Format) Parse(data []byte) (any, error) {
	if len(data) > maxYAMLInputBytes {
		return nil, fmt.Errorf("YAML input exceeds maximum size of %d bytes", maxYAMLInputBytes)
	}
	var result any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("YAML parse error: %w", err)
	}
	return normalizeYAML(result), nil
}

// ParseWithComments parses YAML data and preserves the document structure including comments.
func (f *Format) ParseWithComments(data []byte) (*yaml.Node, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("YAML parse error: %w", err)
	}
	return &node, nil
}

// Serialize serializes a Go value to YAML.
func (f *Format) Serialize(value any, opts plugin.SerializeOptions) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)

	indent := 2
	if opts.Indent > 0 {
		indent = opts.Indent
	}
	encoder.SetIndent(indent)

	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("YAML serialize error: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("YAML encoder close error: %w", err)
	}

	result := buf.Bytes()

	// Remove trailing newline if not wanted
	if !opts.TrailingNewline && len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// SerializeNode serializes a YAML node, preserving comments and structure.
func (f *Format) SerializeNode(node *yaml.Node, opts plugin.SerializeOptions) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)

	indent := 2
	if opts.Indent > 0 {
		indent = opts.Indent
	}
	encoder.SetIndent(indent)

	if err := encoder.Encode(node); err != nil {
		return nil, fmt.Errorf("YAML serialize error: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("YAML encoder close error: %w", err)
	}

	return buf.Bytes(), nil
}

// Merge merges two YAML values according to the strategy.
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

// Query queries a YAML value using a path expression.
func (f *Format) Query(data any, path string) (any, error) {
	return yamlQuery(data, path)
}

// Set sets a value at the specified path.
func (f *Format) Set(data any, path string, value any) (any, error) {
	return yamlSet(data, path, value)
}

// Delete deletes a value at the specified path.
func (f *Format) Delete(data any, path string) (any, error) {
	return yamlDelete(data, path)
}

// Validate validates YAML data structure.
// Note: Full YAML schema validation is not implemented.
// This performs basic structural validation.
func (f *Format) Validate(data any, schema any) ([]plugin.ValidationError, error) {
	// If data is nil, it's valid (empty YAML)
	if data == nil {
		return nil, nil
	}

	// YAML data should be a map, slice, or primitive type
	switch v := data.(type) {
	case map[string]any:
		// Valid YAML mapping - recursively validate nested values
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
		// Valid YAML sequence - recursively validate elements
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
	case string, float64, int, int64, bool:
		// Valid YAML scalar types
		return nil, nil
	default:
		// Unknown type - might be invalid
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("unexpected YAML type: %T", v),
				Value:   v,
			},
		}, nil
	}
}

// GetSchema returns the Terraform schema for YAML-specific attributes.
func (f *Format) GetSchema() plugin.FormatSchema {
	return plugin.FormatSchema{
		Attributes: map[string]plugin.SchemaAttribute{
			"indent": {
				Type:        "number",
				Optional:    true,
				Default:     2,
				Description: "Indentation spaces",
			},
			"preserve_comments": {
				Type:        "bool",
				Optional:    true,
				Description: "Preserve YAML comments in output",
			},
		},
	}
}

// normalizeYAML converts yaml.v3 types to standard Go types.
// yaml.v3 returns map[string]interface{} which is what we want,
// but we need to handle nested structures.
func normalizeYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = normalizeYAML(v)
		}
		return result

	case map[any]any:
		// Convert map[any]any to map[string]any
		result := make(map[string]any, len(val))
		for k, v := range val {
			key := fmt.Sprintf("%v", k)
			result[key] = normalizeYAML(v)
		}
		return result

	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = normalizeYAML(v)
		}
		return result

	default:
		return v
	}
}

// deepMerge performs a recursive deep merge of two YAML values.
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

// yamlQuery implements a simple path query for YAML data.
func yamlQuery(data any, path string) (any, error) {
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

		case []any:
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

// yamlSet sets a value at the specified path.
func yamlSet(data any, path string, value any) (any, error) {
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

// yamlDelete deletes a value at the specified path.
func yamlDelete(data any, path string) (any, error) {
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
	for i, part := range parts[:len(parts)-1] {
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

		case []any:
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				return nil, fmt.Errorf("invalid array index at %d: %s", i, part)
			}
			if idx < 0 || idx >= len(v) {
				return result, nil
			}
			current = v[idx]

		default:
			return result, nil
		}
	}

	lastPart := parts[len(parts)-1]
	if m, ok := current.(map[string]any); ok {
		delete(m, lastPart)
	}

	return result, nil
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
