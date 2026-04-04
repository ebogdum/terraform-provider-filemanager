// SPDX-License-Identifier: MIT

package common

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PathOutputs holds computed path-related outputs for resources.
type PathOutputs struct {
	Path         types.String
	Directory    types.String
	Filename     types.String
	Extension    types.String
	AbsolutePath types.String
}

// ComputePathOutputs computes all path-related outputs from a file path.
func ComputePathOutputs(path string) PathOutputs {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	ext := filepath.Ext(absPath)

	return PathOutputs{
		Path:         types.StringValue(path),
		Directory:    types.StringValue(filepath.Dir(absPath)),
		Filename:     types.StringValue(filepath.Base(absPath)),
		Extension:    types.StringValue(strings.TrimPrefix(ext, ".")),
		AbsolutePath: types.StringValue(absPath),
	}
}

// PathOutputSchema returns the common schema attributes for path outputs.
// These should be added to resources that manage files/directories.
func PathOutputSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"directory": schema.StringAttribute{
			Description: "The parent directory of the path.",
			Computed:    true,
		},
		"filename": schema.StringAttribute{
			Description: "The base name of the file (e.g., 'config.json').",
			Computed:    true,
		},
		"extension": schema.StringAttribute{
			Description: "The file extension without the leading dot (e.g., 'json').",
			Computed:    true,
		},
		"absolute_path": schema.StringAttribute{
			Description: "The absolute resolved path.",
			Computed:    true,
		},
	}
}
