// SPDX-License-Identifier: MIT

package tfvars_file

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// SerializeTfvars converts map[string]any to HCL tfvars format.
// Only writes top-level attribute assignments (no blocks).
func SerializeTfvars(vars map[string]any, sortKeys bool) ([]byte, error) {
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	if sortKeys {
		sort.Strings(keys)
	}

	for _, k := range keys {
		body.SetAttributeValue(k, goToCty(vars[k]))
	}

	result := file.Bytes()
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}

	return result, nil
}

// SerializeTfvarsJSON converts map[string]any to JSON format (.tfvars.json).
func SerializeTfvarsJSON(vars map[string]any, indent string) ([]byte, error) {
	if "" == indent {
		indent = "  "
	}
	data, err := json.MarshalIndent(vars, "", indent)
	if err != nil {
		return nil, fmt.Errorf("JSON marshal error: %w", err)
	}
	// Append trailing newline
	data = append(data, '\n')
	return data, nil
}

// ParseTfvarsHCL parses a .tfvars file (HCL format) into map[string]any.
func ParseTfvarsHCL(data []byte) (map[string]any, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, "input.tfvars")
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type: %T", file.Body)
	}

	result := make(map[string]any, len(body.Attributes))
	for name, attr := range body.Attributes {
		val, valDiags := attr.Expr.Value(nil)
		if valDiags.HasErrors() {
			return nil, fmt.Errorf("error evaluating attribute %q: %s", name, valDiags.Error())
		}
		result[name] = ctyToGo(val)
	}

	return result, nil
}

// ParseTfvarsJSON parses a .tfvars.json file into map[string]any.
func ParseTfvarsJSON(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}
	return result, nil
}

// ValidateTfvarsHCL validates that data is valid HCL tfvars syntax.
func ValidateTfvarsHCL(data []byte) error {
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL(data, "validate.tfvars")
	if diags.HasErrors() {
		return fmt.Errorf("invalid tfvars syntax: %s", diags.Error())
	}
	return nil
}

// goToCty converts a Go value to cty.Value for hclwrite.
func goToCty(v any) cty.Value {
	if nil == v {
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
		if 0 == len(val) {
			return cty.ListValEmpty(cty.DynamicPseudoType)
		}
		vals := make([]cty.Value, len(val))
		for i, item := range val {
			vals[i] = goToCty(item)
		}
		return cty.TupleVal(vals)

	case []string:
		if 0 == len(val) {
			return cty.ListValEmpty(cty.String)
		}
		vals := make([]cty.Value, len(val))
		for i, item := range val {
			vals[i] = cty.StringVal(item)
		}
		return cty.ListVal(vals)

	case map[string]any:
		if 0 == len(val) {
			return cty.MapValEmpty(cty.DynamicPseudoType)
		}
		vals := make(map[string]cty.Value, len(val))
		for k, v := range val {
			vals[k] = goToCty(v)
		}
		return cty.ObjectVal(vals)

	default:
		return cty.StringVal(fmt.Sprintf("%v", v))
	}
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
		return val.GoString()
	}
}

// detectFormat determines if a file is HCL or JSON based on extension.
func detectFormat(path string) string {
	if len(path) > 11 && ".tfvars.json" == path[len(path)-12:] {
		return "json"
	}
	return "hcl"
}
