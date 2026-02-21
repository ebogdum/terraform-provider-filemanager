// SPDX-License-Identifier: MIT

package functions

import (
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &PathExtFunction{}

// NewPathExtFunction creates a new path_ext function.
func NewPathExtFunction() function.Function {
	return &PathExtFunction{
		pathUnaryFunction: newPathUnaryFunction(
			"path_ext",
			"Gets file extension",
			"Returns the file extension of a path.",
			"Returns the file extension of a path (including the leading dot, e.g., `.txt`).",
			"The path to get the extension from.",
			filepath.Ext,
		),
	}
}

// PathExtFunction implements the path_ext function.
type PathExtFunction struct{ *pathUnaryFunction }
