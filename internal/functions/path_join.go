// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = &PathJoinFunction{}

// NewPathJoinFunction creates a new path_join function.
func NewPathJoinFunction() function.Function {
	return &PathJoinFunction{}
}

// PathJoinFunction implements the path_join function.
type PathJoinFunction struct{}

// Metadata returns the function metadata.
func (f *PathJoinFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "path_join"
}

// Definition returns the function definition.
func (f *PathJoinFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Joins path elements",
		Description:         "Joins path elements into a single path using the operating system's path separator.",
		MarkdownDescription: "Joins path elements into a single path using the operating system's path separator.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:                "parts",
				Description:         "The path parts to join.",
				MarkdownDescription: "The path parts to join.",
				ElementType:         types.StringType,
			},
		},
		Return: function.StringReturn{},
	}
}

// Run executes the function.
func (f *PathJoinFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var parts []string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &parts))
	if resp.Error != nil {
		return
	}

	result := filepath.Join(parts...)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
