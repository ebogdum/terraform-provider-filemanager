// SPDX-License-Identifier: MIT

// Package ini provides INI format plugin implementation.
package ini

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"gopkg.in/ini.v1"
)

// Format implements the FormatPlugin interface for INI.
type Format struct{}

// New creates a new INI format plugin.
func New() *Format {
	return &Format{}
}

// Name returns the format name.
func (f *Format) Name() string {
	return "ini"
}

// Extensions returns the supported file extensions.
func (f *Format) Extensions() []string {
	return []string{".ini", ".cfg", ".conf"}
}

// MimeTypes returns the supported MIME types.
func (f *Format) MimeTypes() []string {
	return []string{"text/plain"}
}

// Parse parses INI data into a Go value.
// Returns map[string]any where keys are section names and values are
// map[string]string of key-value pairs. The default section uses "" as key.
func (f *Format) Parse(data []byte) (any, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("INI parse error: %w", err)
	}

	result := make(map[string]any)

	for _, section := range cfg.Sections() {
		sectionName := section.Name()
		sectionData := make(map[string]any)

		for _, key := range section.Keys() {
			sectionData[key.Name()] = key.String()
		}

		// Use empty string for default section
		if sectionName == "DEFAULT" {
			sectionName = ""
		}

		if len(sectionData) > 0 {
			result[sectionName] = sectionData
		}
	}

	return result, nil
}

// Serialize serializes a Go value to INI format.
// Expects map[string]any where keys are section names and values are
// map[string]any of key-value pairs.
func (f *Format) Serialize(value any, opts plugin.SerializeOptions) ([]byte, error) {
	data, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("INI serialize: expected map[string]any, got %T", value)
	}

	cfg := ini.Empty()

	// Get sections and optionally sort them
	sections := make([]string, 0, len(data))
	for section := range data {
		sections = append(sections, section)
	}
	if opts.SortKeys {
		sort.Strings(sections)
	}

	for _, sectionName := range sections {
		sectionData := data[sectionName]
		sectionMap, ok := sectionData.(map[string]any)
		if !ok {
			continue
		}

		// Use DEFAULT for empty section name
		iniSectionName := sectionName
		if sectionName == "" {
			iniSectionName = "DEFAULT"
		}

		section, err := cfg.NewSection(iniSectionName)
		if err != nil {
			return nil, fmt.Errorf("INI serialize: failed to create section %s: %w", sectionName, err)
		}

		// Get keys and optionally sort them
		keys := make([]string, 0, len(sectionMap))
		for key := range sectionMap {
			keys = append(keys, key)
		}
		if opts.SortKeys {
			sort.Strings(keys)
		}

		for _, key := range keys {
			val := sectionMap[key]
			_, err := section.NewKey(key, fmt.Sprintf("%v", val))
			if err != nil {
				return nil, fmt.Errorf("INI serialize: failed to create key %s: %w", key, err)
			}
		}
	}

	var buf bytes.Buffer
	_, err := cfg.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("INI serialize error: %w", err)
	}

	result := buf.Bytes()

	// Handle trailing newline
	if !opts.TrailingNewline && len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// Merge merges two INI values according to the strategy.
func (f *Format) Merge(base, overlay any, strategy plugin.MergeStrategy) (any, error) {
	switch strategy {
	case plugin.MergeReplace:
		return overlay, nil

	case plugin.MergeDeep:
		return deepMerge(base, overlay), nil

	case plugin.MergeAppend, plugin.MergeConcat, plugin.MergeUnion:
		// For INI, these all behave as deep merge
		return deepMerge(base, overlay), nil

	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// Query queries an INI value using section.key notation.
func (f *Format) Query(data any, path string) (any, error) {
	if path == "" || path == "." {
		return data, nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("INI query: expected map[string]any, got %T", data)
	}

	path = strings.TrimPrefix(path, ".")
	parts := strings.SplitN(path, ".", 2)

	// First part is section
	section := parts[0]
	sectionData, ok := dataMap[section]
	if !ok {
		return nil, fmt.Errorf("section not found: %s", section)
	}

	// If only section specified, return entire section
	if len(parts) == 1 {
		return sectionData, nil
	}

	// Second part is key
	sectionMap, ok := sectionData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("INI query: section %s is not a map", section)
	}

	key := parts[1]
	value, ok := sectionMap[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s in section %s", key, section)
	}

	return value, nil
}

// Set sets a value at the specified path (section.key).
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
		return nil, fmt.Errorf("INI set: expected map[string]any, got %T", result)
	}

	path = strings.TrimPrefix(path, ".")
	parts := strings.SplitN(path, ".", 2)

	section := parts[0]

	// If only section specified, set entire section
	if len(parts) == 1 {
		resultMap[section] = value
		return resultMap, nil
	}

	// Ensure section exists
	if _, ok := resultMap[section]; !ok {
		resultMap[section] = make(map[string]any)
	}

	sectionMap, ok := resultMap[section].(map[string]any)
	if !ok {
		resultMap[section] = make(map[string]any)
		sectionMap = resultMap[section].(map[string]any)
	}

	key := parts[1]
	sectionMap[key] = value

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
	parts := strings.SplitN(path, ".", 2)

	section := parts[0]

	// If only section specified, delete entire section
	if len(parts) == 1 {
		delete(resultMap, section)
		return resultMap, nil
	}

	sectionData, ok := resultMap[section]
	if !ok {
		return resultMap, nil
	}

	sectionMap, ok := sectionData.(map[string]any)
	if !ok {
		return resultMap, nil
	}

	key := parts[1]
	delete(sectionMap, key)

	return resultMap, nil
}

// Validate validates INI data structure.
// INI files should have a map[section][key]value structure.
func (f *Format) Validate(data any, schema any) ([]plugin.ValidationError, error) {
	// If data is nil, it's valid (empty INI)
	if data == nil {
		return nil, nil
	}

	// INI data should be a map of sections
	sectionsMap, ok := data.(map[string]any)
	if !ok {
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("INI data must be a map of sections, got %T", data),
				Value:   data,
			},
		}, nil
	}

	var errors []plugin.ValidationError
	for section, sectionData := range sectionsMap {
		// Each section should be a map of string key-value pairs
		sectionMap, ok := sectionData.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    section,
				Message: fmt.Sprintf("INI section must be a map of key-value pairs, got %T", sectionData),
				Value:   sectionData,
			})
			continue
		}

		// Validate that values are strings or can be converted to strings
		for key, value := range sectionMap {
			switch value.(type) {
			case string, int, int64, float64, bool:
				// Valid INI value types (will be converted to strings)
			default:
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.%s", section, key),
					Message: fmt.Sprintf("INI values must be scalars, got %T", value),
					Value:   value,
				})
			}
		}
	}

	return errors, nil
}

// GetSchema returns the Terraform schema for INI-specific attributes.
func (f *Format) GetSchema() plugin.FormatSchema {
	return plugin.FormatSchema{
		Attributes: map[string]plugin.SchemaAttribute{
			"sort_keys": {
				Type:        "bool",
				Optional:    true,
				Description: "Sort section and key names alphabetically",
			},
		},
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
