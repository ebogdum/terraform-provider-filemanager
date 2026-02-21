// SPDX-License-Identifier: MIT

// Package hcl provides HCL format plugin implementation.
package hcl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Format implements the FormatPlugin interface for HCL.
type Format struct{}

// New creates a new HCL format plugin.
func New() *Format {
	return &Format{}
}

// Name returns the format name.
func (f *Format) Name() string {
	return "hcl"
}

// Extensions returns the supported file extensions.
func (f *Format) Extensions() []string {
	return []string{".hcl", ".tf", ".tfvars"}
}

// MimeTypes returns the supported MIME types.
func (f *Format) MimeTypes() []string {
	return []string{"application/hcl", "text/hcl"}
}

// Parse parses HCL data into a Go value.
// Returns a map[string]any representation of the HCL.
func (f *Format) Parse(data []byte) (any, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, "input.hcl")
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}

	return hclFileToMap(file)
}

// hclFileToMap converts an HCL file to a map representation.
func hclFileToMap(file *hcl.File) (map[string]any, error) {
	result := make(map[string]any)

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type: %T", file.Body)
	}

	// Process attributes
	for name, attr := range body.Attributes {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			// If we can't evaluate, store as string representation
			result[name] = exprToString(attr.Expr)
			continue
		}
		result[name] = ctyToGo(val)
	}

	// Process blocks
	for _, block := range body.Blocks {
		blockValue, err := blockToMap(block)
		if err != nil {
			return nil, err
		}

		// Build block key from type and labels
		blockKey := block.Type
		if len(block.Labels) > 0 {
			blockKey = block.Type + "." + strings.Join(block.Labels, ".")
		}

		// Handle multiple blocks with same key
		if existing, ok := result[blockKey]; ok {
			switch v := existing.(type) {
			case []any:
				result[blockKey] = append(v, blockValue)
			default:
				result[blockKey] = []any{v, blockValue}
			}
		} else {
			result[blockKey] = blockValue
		}
	}

	return result, nil
}

// blockToMap converts an HCL block to a map.
func blockToMap(block *hclsyntax.Block) (map[string]any, error) {
	result := make(map[string]any)

	// Add labels as metadata
	if len(block.Labels) > 0 {
		result["__labels__"] = block.Labels
	}

	// Process attributes
	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			result[name] = exprToString(attr.Expr)
			continue
		}
		result[name] = ctyToGo(val)
	}

	// Process nested blocks
	for _, nestedBlock := range block.Body.Blocks {
		nestedValue, err := blockToMap(nestedBlock)
		if err != nil {
			return nil, err
		}

		nestedKey := nestedBlock.Type
		if len(nestedBlock.Labels) > 0 {
			nestedKey = nestedBlock.Type + "." + strings.Join(nestedBlock.Labels, ".")
		}

		if existing, ok := result[nestedKey]; ok {
			switch v := existing.(type) {
			case []any:
				result[nestedKey] = append(v, nestedValue)
			default:
				result[nestedKey] = []any{v, nestedValue}
			}
		} else {
			result[nestedKey] = nestedValue
		}
	}

	return result, nil
}

// exprToString converts an HCL expression to a string representation.
func exprToString(expr hclsyntax.Expression) string {
	// This is a fallback for expressions that can't be evaluated
	return fmt.Sprintf("%v", expr)
}

// ctyToGo converts a cty.Value to a Go value.
func ctyToGo(val cty.Value) any {
	if val.IsNull() {
		return nil
	}

	ty := val.Type()

	switch {
	case ty == cty.String:
		return val.AsString()

	case ty == cty.Number:
		bf := val.AsBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return i
		}
		f, _ := bf.Float64()
		return f

	case ty == cty.Bool:
		return val.True()

	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		var result []any
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			result = append(result, ctyToGo(v))
		}
		return result

	case ty.IsMapType() || ty.IsObjectType():
		result := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			result[k.AsString()] = ctyToGo(v)
		}
		return result

	default:
		// Fallback: try JSON marshaling
		jsonBytes, err := ctyjson.Marshal(val, ty)
		if err == nil {
			var result any
			if json.Unmarshal(jsonBytes, &result) == nil {
				return result
			}
		}
		return val.GoString()
	}
}

// goToCty converts a Go value to a cty.Value.
func goToCty(v any) cty.Value {
	if v == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}

	switch val := v.(type) {
	case string:
		return cty.StringVal(val)

	case bool:
		return cty.BoolVal(val)

	case int:
		return cty.NumberIntVal(int64(val))

	case int64:
		return cty.NumberIntVal(val)

	case float64:
		return cty.NumberFloatVal(val)

	case []any:
		if len(val) == 0 {
			return cty.ListValEmpty(cty.DynamicPseudoType)
		}
		vals := make([]cty.Value, len(val))
		for i, item := range val {
			vals[i] = goToCty(item)
		}
		return cty.TupleVal(vals)

	case map[string]any:
		if len(val) == 0 {
			return cty.MapValEmpty(cty.DynamicPseudoType)
		}
		vals := make(map[string]cty.Value)
		for k, v := range val {
			vals[k] = goToCty(v)
		}
		return cty.ObjectVal(vals)

	default:
		// Try JSON as fallback
		jsonBytes, err := json.Marshal(v)
		if err == nil {
			return cty.StringVal(string(jsonBytes))
		}
		return cty.StringVal(fmt.Sprintf("%v", v))
	}
}

// Serialize serializes a Go value to HCL.
func (f *Format) Serialize(value any, opts plugin.SerializeOptions) ([]byte, error) {
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	valueMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("HCL serialization requires a map, got %T", value)
	}

	if err := writeMapToBody(body, valueMap, opts); err != nil {
		return nil, err
	}

	result := file.Bytes()

	if opts.TrailingNewline && len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}

	return result, nil
}

// writeMapToBody writes a map to an HCL body.
func writeMapToBody(body *hclwrite.Body, m map[string]any, opts plugin.SerializeOptions) error {
	// Get sorted keys if requested
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if opts.SortKeys {
		sort.Strings(keys)
	}

	for _, k := range keys {
		v := m[k]

		// Skip internal metadata
		if k == "__labels__" {
			continue
		}

		// Check if this is a block (contains a map with nested content)
		if isBlock(k, v) {
			if err := writeBlock(body, k, v, opts); err != nil {
				return err
			}
		} else {
			// Regular attribute
			body.SetAttributeValue(k, goToCty(v))
		}
	}

	return nil
}

// isBlock determines if a key-value pair should be written as a block.
func isBlock(key string, value any) bool {
	// If key contains dots, it's a labeled block
	if strings.Contains(key, ".") {
		return true
	}

	// If value is a map with nested maps/blocks, it's a block
	switch v := value.(type) {
	case map[string]any:
		// Check for block-like content
		for _, val := range v {
			switch val.(type) {
			case map[string]any:
				return true
			}
		}
	case []any:
		// Multiple blocks
		if len(v) > 0 {
			if _, ok := v[0].(map[string]any); ok {
				return true
			}
		}
	}

	return false
}

// writeBlock writes an HCL block.
func writeBlock(body *hclwrite.Body, key string, value any, opts plugin.SerializeOptions) error {
	blockType, labels := parseBlockKey(key)

	switch v := value.(type) {
	case map[string]any:
		return appendBlock(body, blockType, labelsFromValue(v, labels), v, opts)

	case []any:
		for _, item := range v {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if err := appendBlock(body, blockType, labelsFromValue(itemMap, labels), itemMap, opts); err != nil {
				return err
			}
		}
		return nil

	default:
		// Simple value - write as attribute instead
		body.SetAttributeValue(key, goToCty(v))
		return nil
	}
}

func parseBlockKey(key string) (string, []string) {
	parts := strings.Split(key, ".")
	return parts[0], parts[1:]
}

func labelsFromValue(m map[string]any, fallback []string) []string {
	if len(fallback) > 0 {
		return fallback
	}

	lbls, ok := m["__labels__"].([]any)
	if !ok {
		return fallback
	}

	labels := make([]string, 0, len(lbls))
	for _, l := range lbls {
		s, ok := l.(string)
		if ok {
			labels = append(labels, s)
		}
	}
	return labels
}

func appendBlock(body *hclwrite.Body, blockType string, labels []string, blockMap map[string]any, opts plugin.SerializeOptions) error {
	block := body.AppendNewBlock(blockType, labels)
	return writeMapToBody(block.Body(), blockMap, opts)
}

// Merge merges two HCL values according to the strategy.
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

// Query queries an HCL value using a dot-separated path.
func (f *Format) Query(data any, path string) (any, error) {
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
			idx, err := parseArrayIndex(part)
			if err != nil {
				return nil, err
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

// Set sets a value at the specified path.
func (f *Format) Set(data any, path string, value any) (any, error) {
	if path == "" || path == "." {
		return value, nil
	}

	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")

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
				v[part] = make(map[string]any)
				next = v[part]
			}
			current = next

		case []any:
			idx, err := parseArrayIndex(part)
			if err != nil {
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
		idx, err := parseArrayIndex(lastPart)
		if err != nil {
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

// Delete deletes a value at the specified path.
func (f *Format) Delete(data any, path string) (any, error) {
	if path == "" || path == "." {
		return nil, nil
	}

	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")

	result := deepCopy(data)
	if result == nil {
		return nil, nil
	}

	// Navigate to parent
	current := result
	var parent any
	var parentMapKey string
	parentSliceIdx := -1

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
			parent = v
			parentMapKey = part
			parentSliceIdx = -1
			current = next

		case []any:
			idx, err := parseArrayIndex(part)
			if err != nil {
				return nil, fmt.Errorf("invalid array index at %d: %s", i, part)
			}
			if idx < 0 || idx >= len(v) {
				return result, nil
			}
			parent = v
			parentSliceIdx = idx
			parentMapKey = ""
			current = v[idx]

		default:
			return result, nil
		}
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
			result = replaceParentSlice(result, parent, parentMapKey, parentSliceIdx, v, idx)
		}
	}

	return result, nil
}

// Validate validates HCL data.
func (f *Format) Validate(data any, schema any) ([]plugin.ValidationError, error) {
	// Try to serialize and re-parse to validate structure
	hclBytes, err := f.Serialize(data, plugin.SerializeOptions{})
	if err != nil {
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("invalid HCL structure: %v", err),
			},
		}, nil
	}

	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL(hclBytes, "validate.hcl")
	if diags.HasErrors() {
		var errors []plugin.ValidationError
		for _, diag := range diags {
			errors = append(errors, plugin.ValidationError{
				Path:    diag.Subject.String(),
				Message: diag.Detail,
			})
		}
		return errors, nil
	}

	return nil, nil
}

// GetSchema returns the Terraform schema for HCL-specific attributes.
func (f *Format) GetSchema() plugin.FormatSchema {
	return plugin.FormatSchema{
		Attributes: map[string]plugin.SchemaAttribute{
			"sort_keys": {
				Type:        "bool",
				Optional:    true,
				Description: "Sort attribute names alphabetically",
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

// ParseFile parses an HCL file from bytes.
func ParseFile(data []byte, filename string) (*hcl.File, hcl.Diagnostics) {
	parser := hclparse.NewParser()
	return parser.ParseHCL(data, filename)
}

func parseArrayIndex(part string) (int, error) {
	idx, err := strconv.Atoi(part)
	if err != nil {
		return 0, fmt.Errorf("invalid array index: %s", part)
	}
	return idx, nil
}

func replaceParentSlice(result, parent any, parentMapKey string, parentSliceIdx int, source []any, removeIdx int) any {
	trimmed := append(source[:removeIdx], source[removeIdx+1:]...)
	switch p := parent.(type) {
	case map[string]any:
		p[parentMapKey] = trimmed
	case []any:
		if parentSliceIdx >= 0 && parentSliceIdx < len(p) {
			p[parentSliceIdx] = trimmed
		}
	default:
		result = trimmed
	}
	return result
}

// FormatFile formats HCL data.
func FormatFile(data []byte) ([]byte, error) {
	file, diags := hclwrite.ParseConfig(data, "format.hcl", hcl.Pos{})
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}
	return file.Bytes(), nil
}

// ValidateSyntax validates HCL syntax without evaluating.
func ValidateSyntax(data []byte) error {
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL(data, "validate.hcl")
	if diags.HasErrors() {
		return fmt.Errorf("HCL syntax error: %s", diags.Error())
	}
	return nil
}

// GetBlocks extracts all blocks of a specific type from HCL.
func GetBlocks(data []byte, blockType string) ([]map[string]any, error) {
	f := New()
	parsed, err := f.Parse(data)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0)

	parsedMap, ok := parsed.(map[string]any)
	if !ok {
		return result, nil
	}

	// Find blocks matching the type
	for key, value := range parsedMap {
		// Check if key starts with blockType
		if key == blockType || strings.HasPrefix(key, blockType+".") {
			switch v := value.(type) {
			case map[string]any:
				result = append(result, v)
			case []any:
				for _, item := range v {
					if itemMap, ok := item.(map[string]any); ok {
						result = append(result, itemMap)
					}
				}
			}
		}
	}

	return result, nil
}

// GetAttribute gets an attribute value from HCL by path.
func GetAttribute(data []byte, path string) (any, error) {
	f := New()
	parsed, err := f.Parse(data)
	if err != nil {
		return nil, err
	}
	return f.Query(parsed, path)
}

// SetAttribute sets an attribute value in HCL.
func SetAttribute(data []byte, path string, value any) ([]byte, error) {
	f := New()
	parsed, err := f.Parse(data)
	if err != nil {
		return nil, err
	}

	modified, err := f.Set(parsed, path, value)
	if err != nil {
		return nil, err
	}

	return f.Serialize(modified, plugin.SerializeOptions{})
}

// MergeFiles merges multiple HCL files.
func MergeFiles(files [][]byte, strategy plugin.MergeStrategy) ([]byte, error) {
	if len(files) == 0 {
		return nil, nil
	}

	f := New()

	var result any
	for i, fileData := range files {
		parsed, err := f.Parse(fileData)
		if err != nil {
			return nil, fmt.Errorf("error parsing file %d: %w", i, err)
		}

		if i == 0 {
			result = parsed
		} else {
			result, err = f.Merge(result, parsed, strategy)
			if err != nil {
				return nil, fmt.Errorf("error merging file %d: %w", i, err)
			}
		}
	}

	return f.Serialize(result, plugin.SerializeOptions{})
}

// TokenizeHCL returns the tokens from HCL data (useful for syntax highlighting).
func TokenizeHCL(data []byte) ([]hclsyntax.Token, error) {
	tokens, diags := hclsyntax.LexConfig(data, "input.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL lex error: %s", diags.Error())
	}
	return tokens, nil
}

// WriteAttribute writes a single attribute to an HCL buffer.
func WriteAttribute(buf *bytes.Buffer, name string, value any) {
	file := hclwrite.NewEmptyFile()
	body := file.Body()
	body.SetAttributeValue(name, goToCty(value))
	buf.Write(file.Bytes())
}
