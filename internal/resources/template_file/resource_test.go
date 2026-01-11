// SPDX-License-Identifier: MIT

package template_file

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRenderGoTemplate_BasicVariables(t *testing.T) {
	r := &TemplateFileResource{}

	templateContent := `Hello, {{ .name }}! You are {{ .age }} years old.`
	vars := map[string]string{
		"name": "World",
		"age":  "42",
	}

	data := &TemplateFileResourceModel{}
	data.LeftDelim = stringValue("{{")
	data.RightDelim = stringValue("}}")
	data.MissingKey = stringValue("error")

	result, err := r.renderGoTemplate(templateContent, vars, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello, World! You are 42 years old."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderGoTemplate_CustomDelimiters(t *testing.T) {
	r := &TemplateFileResource{}

	templateContent := `Hello, [[ .name ]]!`
	vars := map[string]string{
		"name": "World",
	}

	data := &TemplateFileResourceModel{}
	data.LeftDelim = stringValue("[[")
	data.RightDelim = stringValue("]]")
	data.MissingKey = stringValue("error")

	result, err := r.renderGoTemplate(templateContent, vars, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello, World!"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenderGoTemplate_BuiltinFunctions(t *testing.T) {
	r := &TemplateFileResource{}

	tests := []struct {
		name     string
		template string
		vars     map[string]string
		expected string
	}{
		{
			name:     "upper",
			template: `{{ .text | upper }}`,
			vars:     map[string]string{"text": "hello"},
			expected: "HELLO",
		},
		{
			name:     "lower",
			template: `{{ .text | lower }}`,
			vars:     map[string]string{"text": "HELLO"},
			expected: "hello",
		},
		{
			name:     "trim",
			template: `{{ .text | trim }}`,
			vars:     map[string]string{"text": "  hello  "},
			expected: "hello",
		},
		{
			name:     "default with empty",
			template: `{{ default "fallback" .text }}`,
			vars:     map[string]string{"text": ""},
			expected: "fallback",
		},
		{
			name:     "default with value",
			template: `{{ default "fallback" .text }}`,
			vars:     map[string]string{"text": "actual"},
			expected: "actual",
		},
		{
			name:     "quote",
			template: `{{ .text | quote }}`,
			vars:     map[string]string{"text": "hello world"},
			expected: `"hello world"`,
		},
		{
			name:     "contains",
			template: `{{ if contains "ell" .text }}yes{{ else }}no{{ end }}`,
			vars:     map[string]string{"text": "hello"},
			expected: "yes",
		},
		{
			name:     "hasPrefix",
			template: `{{ if hasPrefix "he" .text }}yes{{ else }}no{{ end }}`,
			vars:     map[string]string{"text": "hello"},
			expected: "yes",
		},
		{
			name:     "hasSuffix",
			template: `{{ if hasSuffix "lo" .text }}yes{{ else }}no{{ end }}`,
			vars:     map[string]string{"text": "hello"},
			expected: "yes",
		},
		{
			name:     "trimPrefix",
			template: `{{ trimPrefix "hello-" .text }}`,
			vars:     map[string]string{"text": "hello-world"},
			expected: "world",
		},
		{
			name:     "trimSuffix",
			template: `{{ trimSuffix "-world" .text }}`,
			vars:     map[string]string{"text": "hello-world"},
			expected: "hello",
		},
		{
			name:     "replace",
			template: `{{ replace "o" "0" .text }}`,
			vars:     map[string]string{"text": "hello world"},
			expected: "hell0 w0rld",
		},
		{
			name:     "base64",
			template: `{{ .text | base64 }}`,
			vars:     map[string]string{"text": "hello"},
			expected: "aGVsbG8=",
		},
		{
			name:     "toBool true",
			template: `{{ if toBool .flag }}yes{{ else }}no{{ end }}`,
			vars:     map[string]string{"flag": "true"},
			expected: "yes",
		},
		{
			name:     "toBool false",
			template: `{{ if toBool .flag }}yes{{ else }}no{{ end }}`,
			vars:     map[string]string{"flag": "false"},
			expected: "no",
		},
		{
			name:     "toInt",
			template: `{{ $n := toInt .num }}{{ if eq $n 42 }}yes{{ else }}no{{ end }}`,
			vars:     map[string]string{"num": "42"},
			expected: "yes",
		},
		{
			name:     "coalesce",
			template: `{{ coalesce .a .b .c }}`,
			vars:     map[string]string{"a": "", "b": "", "c": "third"},
			expected: "third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &TemplateFileResourceModel{}
			data.LeftDelim = stringValue("{{")
			data.RightDelim = stringValue("}}")
			data.MissingKey = stringValue("zero")

			result, err := r.renderGoTemplate(tt.template, tt.vars, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRenderGoTemplate_MissingKey(t *testing.T) {
	r := &TemplateFileResource{}

	templateContent := `Hello, {{ .name }}!`
	vars := map[string]string{} // No "name" variable

	tests := []struct {
		name        string
		missingKey  string
		expectError bool
		expected    string
	}{
		{
			name:        "error mode",
			missingKey:  "error",
			expectError: true,
		},
		{
			name:        "zero mode",
			missingKey:  "zero",
			expectError: false,
			expected:    "Hello, !",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &TemplateFileResourceModel{}
			data.LeftDelim = stringValue("{{")
			data.RightDelim = stringValue("}}")
			data.MissingKey = stringValue(tt.missingKey)

			result, err := r.renderGoTemplate(templateContent, vars, data)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("got %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestRenderGoTemplate_Conditionals(t *testing.T) {
	r := &TemplateFileResource{}

	templateContent := `{{ if .enabled }}Feature is ON{{ else }}Feature is OFF{{ end }}`

	tests := []struct {
		name     string
		vars     map[string]string
		expected string
	}{
		{
			name:     "condition true",
			vars:     map[string]string{"enabled": "true"},
			expected: "Feature is ON",
		},
		{
			name:     "condition false (empty string is falsy in Go templates)",
			vars:     map[string]string{"enabled": ""},
			expected: "Feature is OFF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &TemplateFileResourceModel{}
			data.LeftDelim = stringValue("{{")
			data.RightDelim = stringValue("}}")
			data.MissingKey = stringValue("zero")

			result, err := r.renderGoTemplate(templateContent, tt.vars, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRenderMustacheTemplate(t *testing.T) {
	r := &TemplateFileResource{}

	tests := []struct {
		name     string
		template string
		vars     map[string]string
		expected string
	}{
		{
			name:     "simple variable",
			template: "Hello, {{name}}!",
			vars:     map[string]string{"name": "World"},
			expected: "Hello, World!",
		},
		{
			name:     "multiple variables",
			template: "{{greeting}}, {{name}}!",
			vars:     map[string]string{"greeting": "Hi", "name": "Alice"},
			expected: "Hi, Alice!",
		},
		{
			name:     "unescaped triple braces",
			template: "Value: {{{value}}}",
			vars:     map[string]string{"value": "test"},
			expected: "Value: test",
		},
		{
			name:     "missing variable leaves placeholder",
			template: "Hello, {{name}}!",
			vars:     map[string]string{},
			expected: "Hello, {{name}}!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.renderMustacheTemplate(tt.template, tt.vars)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIndentFunction(t *testing.T) {
	funcs := templateFuncs()
	indentFunc := funcs["indent"].(func(int, string) string)

	tests := []struct {
		name     string
		spaces   int
		input    string
		expected string
	}{
		{
			name:     "single line",
			spaces:   2,
			input:    "hello",
			expected: "  hello",
		},
		{
			name:     "multiple lines",
			spaces:   4,
			input:    "line1\nline2\nline3",
			expected: "    line1\n    line2\n    line3",
		},
		{
			name:     "empty lines preserved",
			spaces:   2,
			input:    "line1\n\nline3",
			expected: "  line1\n\n  line3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indentFunc(tt.spaces, tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRenderGoTemplate_Multiline(t *testing.T) {
	r := &TemplateFileResource{}

	templateContent := `server {
    listen {{ .port }};
    server_name {{ .server_name }};

    location / {
        proxy_pass http://{{ .upstream_host }}:{{ .upstream_port }};
    }
}`

	vars := map[string]string{
		"port":          "80",
		"server_name":   "example.com",
		"upstream_host": "127.0.0.1",
		"upstream_port": "3000",
	}

	data := &TemplateFileResourceModel{}
	data.LeftDelim = stringValue("{{")
	data.RightDelim = stringValue("}}")
	data.MissingKey = stringValue("error")

	result, err := r.renderGoTemplate(templateContent, vars, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `server {
    listen 80;
    server_name example.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}`

	if result != expected {
		t.Errorf("got:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestComputeChecksums(t *testing.T) {
	r := &TemplateFileResource{}
	data := &TemplateFileResourceModel{}

	content := []byte("hello world")
	r.computeChecksums(data, content)

	// Known checksums for "hello world"
	expectedMD5 := "5eb63bbbe01eeed093cb22bb8f5acdc3"
	expectedSHA256 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	if data.MD5.ValueString() != expectedMD5 {
		t.Errorf("MD5: got %q, want %q", data.MD5.ValueString(), expectedMD5)
	}

	if data.SHA256.ValueString() != expectedSHA256 {
		t.Errorf("SHA256: got %q, want %q", data.SHA256.ValueString(), expectedSHA256)
	}
}

func TestParseFileMode(t *testing.T) {
	tests := []struct {
		input    string
		expected os.FileMode
	}{
		{"0644", 0644},
		{"644", 0644},
		{"0755", 0755},
		{"0600", 0600},
		{"", 0644}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := common.ParseFileMode(tt.input)
			if result != tt.expected {
				t.Errorf("got %o, want %o", result, tt.expected)
			}
		})
	}
}

func TestRenderTemplateFromFile(t *testing.T) {
	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tpl")

	templateContent := `Hello, {{ .name }}!`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Read it back (simulating what renderTemplate does)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("failed to read template file: %v", err)
	}

	if string(content) != templateContent {
		t.Errorf("got %q, want %q", string(content), templateContent)
	}
}

func TestRenderTemplateWithSourceAttribute(t *testing.T) {
	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "nginx.conf.tpl")

	templateContent := `server {
    listen {{ .port }};
    server_name {{ .server_name }};

    location / {
        proxy_pass http://{{ .upstream }};
    }
}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Create resource model with source attribute
	r := &TemplateFileResource{}
	data := &TemplateFileResourceModel{}
	data.Template = types.StringNull()
	data.TemplateFile = types.StringNull()
	data.Source = types.StringValue(templatePath)
	data.LeftDelim = types.StringValue("{{")
	data.RightDelim = types.StringValue("}}")
	data.MissingKey = types.StringValue("error")
	data.Engine = types.StringValue("go")

	// Create vars
	varsMap := map[string]string{
		"port":        "80",
		"server_name": "example.com",
		"upstream":    "127.0.0.1:3000",
	}
	vars, _ := types.MapValueFrom(context.Background(), types.StringType, varsMap)
	data.Vars = vars

	// Render template
	result, err := r.renderTemplate(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `server {
    listen 80;
    server_name example.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}`

	if result != expected {
		t.Errorf("got:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestRenderTemplateSourceFileNotFound(t *testing.T) {
	r := &TemplateFileResource{}
	data := &TemplateFileResourceModel{}
	data.Template = types.StringNull()
	data.TemplateFile = types.StringNull()
	data.Source = types.StringValue("/nonexistent/path/template.tpl")
	data.LeftDelim = types.StringValue("{{")
	data.RightDelim = types.StringValue("}}")
	data.MissingKey = types.StringValue("error")
	data.Engine = types.StringValue("go")
	data.Vars = types.MapNull(types.StringType)

	_, err := r.renderTemplate(context.Background(), data)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

// Helper function to create types.String values for testing
func stringValue(s string) types.String {
	return types.StringValue(s)
}
