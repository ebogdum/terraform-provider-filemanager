// SPDX-License-Identifier: MIT

package functions

import (
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &PathDirnameFunction{}

// NewPathDirnameFunction creates a new path_dirname function.
func NewPathDirnameFunction() function.Function {
	return &PathDirnameFunction{
		pathUnaryFunction: newPathUnaryFunction(
			"path_dirname",
			"Gets directory name",
			"Returns the directory portion of a path.",
			"Returns the directory portion of a path (everything except the last element).",
			"The path to get the directory from.",
			filepath.Dir,
		),
	}
}

// PathDirnameFunction implements the path_dirname function.
type PathDirnameFunction struct{ *pathUnaryFunction }
