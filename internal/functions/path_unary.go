// SPDX-License-Identifier: MIT

package functions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

type pathUnaryFunction struct {
	name                string
	summary             string
	description         string
	markdownDescription string
	paramDescription    string
	eval                func(string) string
}

func newPathUnaryFunction(
	name, summary, description, markdownDescription, paramDescription string,
	eval func(string) string,
) *pathUnaryFunction {
	return &pathUnaryFunction{
		name:                name,
		summary:             summary,
		description:         description,
		markdownDescription: markdownDescription,
		paramDescription:    paramDescription,
		eval:                eval,
	}
}

func (f *pathUnaryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = f.name
}

func (f *pathUnaryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             f.summary,
		Description:         f.description,
		MarkdownDescription: f.markdownDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "path",
				Description:         f.paramDescription,
				MarkdownDescription: f.paramDescription,
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *pathUnaryFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var path string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &path))
	if resp.Error != nil {
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, f.eval(path)))
}
