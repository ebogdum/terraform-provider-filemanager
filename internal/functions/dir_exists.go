// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &DirExistsFunction{}

// NewDirExistsFunction creates a new dir_exists function.
func NewDirExistsFunction() function.Function {
	return &DirExistsFunction{}
}

// DirExistsFunction implements the dir_exists function.
type DirExistsFunction struct{}

// Metadata returns the function metadata.
func (f *DirExistsFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dir_exists"
}

// Definition returns the function definition.
func (f *DirExistsFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Checks if a directory exists",
		Description:         "Returns true if a directory exists at the specified path, false otherwise.",
		MarkdownDescription: "Returns `true` if a directory exists at the specified path, `false` otherwise.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "path",
				Description:         "The path to check.",
				MarkdownDescription: "The path to check.",
			},
		},
		Return: function.BoolReturn{},
	}
}

// Run executes the function.
func (f *DirExistsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var path string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &path))
	if resp.Error != nil {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, false))
			return
		}
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to check directory: "+err.Error()))
		return
	}

	// Return true only if it's a directory
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, info.IsDir()))
}
