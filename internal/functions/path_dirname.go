// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &PathDirnameFunction{}

// NewPathDirnameFunction creates a new path_dirname function.
func NewPathDirnameFunction() function.Function {
	return &PathDirnameFunction{}
}

// PathDirnameFunction implements the path_dirname function.
type PathDirnameFunction struct{}

// Metadata returns the function metadata.
func (f *PathDirnameFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "path_dirname"
}

// Definition returns the function definition.
func (f *PathDirnameFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Gets directory name",
		Description:         "Returns the directory portion of a path.",
		MarkdownDescription: "Returns the directory portion of a path (everything except the last element).",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "path",
				Description:         "The path to get the directory from.",
				MarkdownDescription: "The path to get the directory from.",
			},
		},
		Return: function.StringReturn{},
	}
}

// Run executes the function.
func (f *PathDirnameFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var path string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &path))
	if resp.Error != nil {
		return
	}

	result := filepath.Dir(path)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
