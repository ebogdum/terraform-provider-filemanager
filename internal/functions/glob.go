// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = &GlobFunction{}

// NewGlobFunction creates a new glob function.
func NewGlobFunction() function.Function {
	return &GlobFunction{}
}

// GlobFunction implements the glob function.
type GlobFunction struct{}

// Metadata returns the function metadata.
func (f *GlobFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "glob"
}

// Definition returns the function definition.
func (f *GlobFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Finds files by pattern",
		Description:         "Returns a list of files matching the glob pattern.",
		MarkdownDescription: "Returns a list of files matching the glob pattern. Supports `*`, `?`, and `[...]` wildcards.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "pattern",
				Description:         "The glob pattern to match files against.",
				MarkdownDescription: "The glob pattern to match files against (e.g., `/etc/*.conf`, `*.txt`).",
			},
		},
		Return: function.ListReturn{
			ElementType: types.StringType,
		},
	}
}

// Run executes the function.
func (f *GlobFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var pattern string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &pattern))
	if resp.Error != nil {
		return
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to evaluate glob pattern: "+err.Error()))
		return
	}

	// Convert to list of strings (empty list if no matches)
	if matches == nil {
		matches = []string{}
	}

	result, diags := types.ListValueFrom(ctx, types.StringType, matches)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
