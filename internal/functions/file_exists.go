// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &FileExistsFunction{}

// NewFileExistsFunction creates a new file_exists function.
func NewFileExistsFunction() function.Function {
	return &FileExistsFunction{}
}

// FileExistsFunction implements the file_exists function.
type FileExistsFunction struct{}

// Metadata returns the function metadata.
func (f *FileExistsFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "file_exists"
}

// Definition returns the function definition.
func (f *FileExistsFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Checks if a file exists",
		Description:         "Returns true if a file exists at the specified path, false otherwise.",
		MarkdownDescription: "Returns `true` if a file exists at the specified path, `false` otherwise.",
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
func (f *FileExistsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
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
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to check file: "+err.Error()))
		return
	}

	// Return true only if it's a file (not a directory)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, !info.IsDir()))
}
