// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &PathExtFunction{}

// NewPathExtFunction creates a new path_ext function.
func NewPathExtFunction() function.Function {
	return &PathExtFunction{}
}

// PathExtFunction implements the path_ext function.
type PathExtFunction struct{}

// Metadata returns the function metadata.
func (f *PathExtFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "path_ext"
}

// Definition returns the function definition.
func (f *PathExtFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Gets file extension",
		Description:         "Returns the file extension of a path.",
		MarkdownDescription: "Returns the file extension of a path (including the leading dot, e.g., `.txt`).",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "path",
				Description:         "The path to get the extension from.",
				MarkdownDescription: "The path to get the extension from.",
			},
		},
		Return: function.StringReturn{},
	}
}

// Run executes the function.
func (f *PathExtFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var path string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &path))
	if resp.Error != nil {
		return
	}

	result := filepath.Ext(path)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
