// SPDX-License-Identifier: MIT

// Package common provides shared types and utilities for the filemanager provider.
package common

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// TerraformDynamicToGoValue converts a Terraform types.Dynamic value back to Go native types.
func TerraformDynamicToGoValue(ctx context.Context, val types.Dynamic) (any, error) {
	if val.IsNull() || val.IsUnknown() {
		return nil, nil
	}
	return attrValueToGo(ctx, val.UnderlyingValue())
}

// attrValueToGo converts an attr.Value to a Go value.
func attrValueToGo(ctx context.Context, val attr.Value) (any, error) {
	if val == nil || val.IsNull() || val.IsUnknown() {
		return nil, nil
	}

	switch v := val.(type) {
	case basetypes.StringValue:
		return v.ValueString(), nil

	case basetypes.BoolValue:
		return v.ValueBool(), nil

	case basetypes.NumberValue:
		return numberValueToGo(v), nil

	case basetypes.TupleValue:
		return attrSliceToGo(ctx, v.Elements(), "tuple")

	case basetypes.ObjectValue:
		return attrMapToGo(ctx, v.Attributes(), "object")

	case basetypes.ListValue:
		return attrSliceToGo(ctx, v.Elements(), "list")

	case basetypes.MapValue:
		return attrMapToGo(ctx, v.Elements(), "map")

	case basetypes.SetValue:
		return attrSliceToGo(ctx, v.Elements(), "set")

	case basetypes.DynamicValue:
		return attrValueToGo(ctx, v.UnderlyingValue())

	default:
		return nil, fmt.Errorf("unsupported attr.Value type: %T", val)
	}
}

func numberValueToGo(v basetypes.NumberValue) any {
	bf := v.ValueBigFloat()
	if nil == bf {
		return nil
	}
	if bf.IsInt() {
		i, accuracy := bf.Int64()
		if accuracy != big.Exact {
			// Value doesn't fit in int64; return as float64 instead
			f, _ := bf.Float64()
			return f
		}
		return i
	}
	f, _ := bf.Float64()
	return f
}

func attrSliceToGo(ctx context.Context, elems []attr.Value, kind string) ([]any, error) {
	result := make([]any, len(elems))
	for i, elem := range elems {
		goVal, err := attrValueToGo(ctx, elem)
		if err != nil {
			return nil, fmt.Errorf("%s element %d: %w", kind, i, err)
		}
		result[i] = goVal
	}
	return result, nil
}

func attrMapToGo(ctx context.Context, elems map[string]attr.Value, kind string) (map[string]any, error) {
	result := make(map[string]any, len(elems))
	for key, elem := range elems {
		goVal, err := attrValueToGo(ctx, elem)
		if err != nil {
			return nil, fmt.Errorf("%s key %q: %w", kind, key, err)
		}
		result[key] = goVal
	}
	return result, nil
}

// GoValueToTerraformDynamic converts a Go value (from JSON/YAML/etc parsing)
// to a Terraform basetypes.DynamicValue that can be used in data source outputs.
func GoValueToTerraformDynamic(ctx context.Context, value any) (basetypes.DynamicValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrValue, d := goValueToAttrValue(ctx, value)
	diags.Append(d...)
	if diags.HasError() {
		return basetypes.NewDynamicNull(), diags
	}

	return basetypes.NewDynamicValue(attrValue), diags
}

// goValueToAttrValue converts a Go value to a Terraform attr.Value.
func goValueToAttrValue(ctx context.Context, value any) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value == nil {
		return types.StringNull(), diags
	}

	switch v := value.(type) {
	case bool:
		return types.BoolValue(v), diags

	case string:
		return types.StringValue(v), diags

	case float64:
		// JSON numbers are always float64
		return types.NumberValue(big.NewFloat(v)), diags

	case float32:
		return types.NumberValue(big.NewFloat(float64(v))), diags

	case int:
		return types.NumberValue(big.NewFloat(float64(v))), diags

	case int64:
		return types.NumberValue(big.NewFloat(float64(v))), diags

	case int32:
		return types.NumberValue(big.NewFloat(float64(v))), diags

	case uint:
		return types.NumberValue(big.NewFloat(float64(v))), diags

	case uint64:
		return types.NumberValue(new(big.Float).SetUint64(v)), diags

	case uint32:
		return types.NumberValue(big.NewFloat(float64(v))), diags

	case []any:
		return goSliceToTerraformTuple(ctx, v)

	case []string:
		// Convert []string to []any for compatibility
		anySlice := make([]any, len(v))
		for i, s := range v {
			anySlice[i] = s
		}
		return goSliceToTerraformTuple(ctx, anySlice)

	case map[string]any:
		return goMapToTerraformObject(ctx, v)

	default:
		diags.AddError(
			"Unsupported value type",
			fmt.Sprintf("Cannot convert Go type %T to Terraform value", value),
		)
		return nil, diags
	}
}

// goSliceToTerraformTuple converts a Go slice to a Terraform tuple value.
// We use tuple instead of list because JSON/YAML arrays can contain mixed types.
func goSliceToTerraformTuple(ctx context.Context, slice []any) (basetypes.TupleValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(slice) == 0 {
		return types.TupleValueMust([]attr.Type{}, []attr.Value{}), diags
	}

	attrValues := make([]attr.Value, len(slice))
	attrTypes := make([]attr.Type, len(slice))

	for i, item := range slice {
		attrValue, d := goValueToAttrValue(ctx, item)
		diags.Append(d...)
		if diags.HasError() {
			return types.TupleNull(nil), diags
		}
		attrValues[i] = attrValue
		attrTypes[i] = attrValue.Type(ctx)
	}

	tupleVal, d := types.TupleValue(attrTypes, attrValues)
	diags.Append(d...)

	return tupleVal, diags
}

// goMapToTerraformObject converts a Go map to a Terraform object value.
// We use object instead of map because JSON/YAML objects can have mixed-type values.
func goMapToTerraformObject(ctx context.Context, m map[string]any) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(m) == 0 {
		return types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}), diags
	}

	attrValues := make(map[string]attr.Value, len(m))
	attrTypes := make(map[string]attr.Type, len(m))

	for k, v := range m {
		attrValue, d := goValueToAttrValue(ctx, v)
		diags.Append(d...)
		if diags.HasError() {
			return types.ObjectNull(nil), diags
		}
		attrValues[k] = attrValue
		attrTypes[k] = attrValue.Type(ctx)
	}

	objectVal, d := types.ObjectValue(attrTypes, attrValues)
	diags.Append(d...)

	return objectVal, diags
}
