// SPDX-License-Identifier: MIT

package functions

import (
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &PathBasenameFunction{}

// NewPathBasenameFunction creates a new path_basename function.
func NewPathBasenameFunction() function.Function {
	return &PathBasenameFunction{
		pathUnaryFunction: newPathUnaryFunction(
			"path_basename",
			"Gets base name",
			"Returns the last element of a path.",
			"Returns the last element of a path (the file or directory name).",
			"The path to get the base name from.",
			filepath.Base,
		),
	}
}

// PathBasenameFunction implements the path_basename function.
type PathBasenameFunction struct{ *pathUnaryFunction }
