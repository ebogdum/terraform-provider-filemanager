// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &PathBasenameFunction{}

// NewPathBasenameFunction creates a new path_basename function.
func NewPathBasenameFunction() function.Function {
	return &PathBasenameFunction{}
}

// PathBasenameFunction implements the path_basename function.
type PathBasenameFunction struct{}

// Metadata returns the function metadata.
func (f *PathBasenameFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "path_basename"
}

// Definition returns the function definition.
func (f *PathBasenameFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Gets base name",
		Description:         "Returns the last element of a path.",
		MarkdownDescription: "Returns the last element of a path (the file or directory name).",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "path",
				Description:         "The path to get the base name from.",
				MarkdownDescription: "The path to get the base name from.",
			},
		},
		Return: function.StringReturn{},
	}
}

// Run executes the function.
func (f *PathBasenameFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var path string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &path))
	if resp.Error != nil {
		return
	}

	result := filepath.Base(path)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
