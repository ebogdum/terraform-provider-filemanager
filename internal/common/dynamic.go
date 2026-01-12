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
