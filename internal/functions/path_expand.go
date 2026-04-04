// SPDX-License-Identifier: MIT

package functions

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &PathExpandFunction{}

// NewPathExpandFunction creates a new path_expand function.
func NewPathExpandFunction() function.Function {
	return &PathExpandFunction{}
}

// PathExpandFunction implements the path_expand function.
type PathExpandFunction struct{}

// Metadata returns the function metadata.
func (f *PathExpandFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "path_expand"
}

// Definition returns the function definition.
func (f *PathExpandFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Expands path",
		Description:         "Expands ~ to the user's home directory in a path.",
		MarkdownDescription: "Expands `~` to the user's home directory in a path.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "path",
				Description:         "The path to expand.",
				MarkdownDescription: "The path to expand (may contain `~` for home directory).",
			},
		},
		Return: function.StringReturn{},
	}
}

// Run executes the function.
func (f *PathExpandFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var path string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &path))
	if resp.Error != nil {
		return
	}

	result := expandPath(path)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}

// expandPath expands ~ to the user's home directory.
// It intentionally does NOT expand environment variables to prevent
// information disclosure through $VAR expansion in Terraform plans.
func expandPath(path string) string {
	return safeExpandPath(path)
}

// safeExpandPath performs home-directory-only expansion without expanding
// environment variables.
func safeExpandPath(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		if p == "~" {
			return home
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
