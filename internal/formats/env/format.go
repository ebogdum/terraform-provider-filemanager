// SPDX-License-Identifier: MIT

// Package env provides .env format plugin implementation.
package env

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Format implements the FormatPlugin interface for .env files.
type Format struct{}

// New creates a new ENV format plugin.
func New() *Format {
	return &Format{}
}

// Name returns the format name.
func (f *Format) Name() string {
	return "env"
}

// Extensions returns the supported file extensions.
func (f *Format) Extensions() []string {
	return []string{".env"}
}

// MimeTypes returns the supported MIME types.
func (f *Format) MimeTypes() []string {
	return []string{"text/plain"}
}

// Parse parses .env data into a Go value.
// Returns map[string]any where keys are variable names and values are strings.
func (f *Format) Parse(data []byte) (any, error) {
	result := make(map[string]any)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Find the = sign
		idx := strings.Index(line, "=")
		if idx == -1 {
			// Line without = is invalid but we'll skip it
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove export prefix if present
		key = strings.TrimPrefix(key, "export ")
		key = strings.TrimSpace(key)

		// Handle quoted values
		value = unquote(value)

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ENV parse error: %w", err)
	}

	return result, nil
}

// Serialize serializes a Go value to .env format.
// Expects map[string]any where keys are variable names.
func (f *Format) Serialize(value any, opts plugin.SerializeOptions) ([]byte, error) {
	data, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ENV serialize: expected map[string]any, got %T", value)
	}

	var buf bytes.Buffer

	// Get keys and optionally sort them
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	if opts.SortKeys {
		sort.Strings(keys)
	}

	for _, key := range keys {
		val := data[key]
		strVal := fmt.Sprintf("%v", val)

		// Quote value if it contains spaces, special chars, or is empty
		if needsQuoting(strVal) {
			strVal = quote(strVal)
		}

		buf.WriteString(key)
		buf.WriteString("=")
		buf.WriteString(strVal)
		buf.WriteString("\n")
	}

	result := buf.Bytes()

	// Handle trailing newline
	if !opts.TrailingNewline && len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// Merge merges two ENV values according to the strategy.
func (f *Format) Merge(base, overlay any, strategy plugin.MergeStrategy) (any, error) {
	switch strategy {
	case plugin.MergeReplace:
		return overlay, nil

	case plugin.MergeDeep, plugin.MergeAppend, plugin.MergeConcat, plugin.MergeUnion:
		// For ENV files, all strategies behave as merge (overlay wins)
		return merge(base, overlay), nil

	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// Query queries an ENV value by variable name.
func (f *Format) Query(data any, path string) (any, error) {
	if path == "" || path == "." {
		return data, nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ENV query: expected map[string]any, got %T", data)
	}

	path = strings.TrimPrefix(path, ".")
	value, ok := dataMap[path]
	if !ok {
		return nil, fmt.Errorf("variable not found: %s", path)
	}

	return value, nil
}

// Set sets a value at the specified path (variable name).
func (f *Format) Set(data any, path string, value any) (any, error) {
	if path == "" || path == "." {
		return value, nil
	}

	result := deepCopy(data)
	if result == nil {
		result = make(map[string]any)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ENV set: expected map[string]any, got %T", result)
	}

	path = strings.TrimPrefix(path, ".")
	resultMap[path] = value

	return resultMap, nil
}

// Delete deletes a value at the specified path.
func (f *Format) Delete(data any, path string) (any, error) {
	if path == "" || path == "." {
		return nil, nil
	}

	result := deepCopy(data)
	if result == nil {
		return nil, nil
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return result, nil
	}

	path = strings.TrimPrefix(path, ".")
	delete(resultMap, path)

	return resultMap, nil
}

// Validate validates ENV data structure.
// ENV files should have a map[key]value structure with string values.
func (f *Format) Validate(data any, schema any) ([]plugin.ValidationError, error) {
	// If data is nil, it's valid (empty ENV)
	if data == nil {
		return nil, nil
	}

	// ENV data should be a flat map of string key-value pairs
	envMap, ok := data.(map[string]any)
	if !ok {
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("ENV data must be a flat map, got %T", data),
				Value:   data,
			},
		}, nil
	}

	var errors []plugin.ValidationError
	for key, value := range envMap {
		// Validate key format (should be valid env var name)
		if !isValidEnvKey(key) {
			errors = append(errors, plugin.ValidationError{
				Path:    key,
				Message: "ENV key should start with a letter or underscore and contain only alphanumeric characters and underscores",
				Value:   key,
			})
		}

		// Validate value type
		switch value.(type) {
		case string:
			// Valid ENV value type
		case int, int64, float64, bool:
			// Will be converted to strings
		default:
			errors = append(errors, plugin.ValidationError{
				Path:    key,
				Message: fmt.Sprintf("ENV values must be scalars, got %T", value),
				Value:   value,
			})
		}
	}

	return errors, nil
}

// isValidEnvKey checks if a key is a valid environment variable name.
func isValidEnvKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for i, r := range key {
		if i == 0 {
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}

// GetSchema returns the Terraform schema for ENV-specific attributes.
func (f *Format) GetSchema() plugin.FormatSchema {
	return plugin.FormatSchema{
		Attributes: map[string]plugin.SchemaAttribute{
			"sort_keys": {
				Type:        "bool",
				Optional:    true,
				Description: "Sort variable names alphabetically",
			},
		},
	}
}

// merge combines two ENV maps with overlay taking precedence.
func merge(base, overlay any) any {
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
			result[k] = v
		}
		return result
	}

	return overlay
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

	default:
		return v
	}
}

// unquote removes surrounding quotes from a value.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}

	// Double quotes
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
		// Unescape escaped characters
		s = strings.ReplaceAll(s, `\"`, `"`)
		s = strings.ReplaceAll(s, `\\`, `\`)
		s = strings.ReplaceAll(s, `\n`, "\n")
		s = strings.ReplaceAll(s, `\t`, "\t")
		return s
	}

	// Single quotes
	if s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}

	return s
}

// needsQuoting returns true if a value needs to be quoted.
func needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	for _, c := range s {
		switch c {
		case ' ', '\t', '\n', '\r', '"', '\'', '\\', '#', '$', '`':
			return true
		}
	}

	return false
}

// quote wraps a string in double quotes and escapes special characters.
func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}
