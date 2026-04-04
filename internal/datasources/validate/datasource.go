// SPDX-License-Identifier: MIT

// Package validate implements the filemanager_validate data source.
package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ValidateDataSource{}

// NewValidateDataSource creates a new validate data source.
func NewValidateDataSource() datasource.DataSource {
	return &ValidateDataSource{}
}

// ValidateDataSource defines the data source implementation.
type ValidateDataSource struct {
	config *common.ProviderConfig
}

// ValidateDataSourceModel describes the data source data model.
type ValidateDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Path           types.String `tfsdk:"path"`
	Service        types.String `tfsdk:"service"`
	Format         types.String `tfsdk:"format"`
	ContextLines   types.Int64  `tfsdk:"context_lines"`
	IsValid        types.Bool   `tfsdk:"is_valid"`
	FormatDetected types.String `tfsdk:"format_detected"`
	ErrorMessage   types.String `tfsdk:"error_message"`
	ErrorLine      types.Int64  `tfsdk:"error_line"`
	ErrorColumn    types.Int64  `tfsdk:"error_column"`
	ErrorOffset    types.Int64  `tfsdk:"error_offset"`
	ErrorContext   types.String `tfsdk:"error_context"`
	Size           types.Int64  `tfsdk:"size"`
	Content        types.String `tfsdk:"content"`
}

// ValidationResult holds detailed validation information.
type ValidationResult struct {
	IsValid      bool
	ErrorMessage string
	ErrorLine    int
	ErrorColumn  int
	ErrorOffset  int64
}

// Metadata returns the data source type name.
func (d *ValidateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_validate"
}

// Schema defines the schema for the data source.
func (d *ValidateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Validates file content against expected formats with detailed error reporting.",
		MarkdownDescription: `Validates file content against expected formats (JSON, YAML, TOML, INI, ENV) and provides detailed error information including line/column position and context.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path to the file to validate.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"format": schema.StringAttribute{
				Description: "Format to validate against: json, yaml, toml, ini, env. Auto-detected from extension if omitted.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("json", "yaml", "toml", "ini", "env"),
				},
			},
			"context_lines": schema.Int64Attribute{
				Description: "Number of context lines to show above and below error location. Defaults to 3.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 10),
				},
			},
			"is_valid": schema.BoolAttribute{
				Description: "Whether the file content is valid for the detected/specified format.",
				Computed:    true,
			},
			"format_detected": schema.StringAttribute{
				Description: "The format that was used for validation (detected or specified).",
				Computed:    true,
			},
			"error_message": schema.StringAttribute{
				Description: "Human-readable error message. Null if valid.",
				Computed:    true,
			},
			"error_line": schema.Int64Attribute{
				Description: "Line number where the error occurred. Null if valid.",
				Computed:    true,
			},
			"error_column": schema.Int64Attribute{
				Description: "Column number where the error occurred. Null if valid.",
				Computed:    true,
			},
			"error_offset": schema.Int64Attribute{
				Description: "Byte offset where the error occurred. Null if valid.",
				Computed:    true,
			},
			"error_context": schema.StringAttribute{
				Description: "Lines around the error with line numbers and marker. Null if valid.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the file in bytes.",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "Raw file content.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *ValidateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*common.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	d.config = config
}

// Read reads the data source.
func (d *ValidateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ValidateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get backend
	backend, err := d.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Check if path exists
	exists, err := backend.Exists(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check file existence", err.Error())
		return
	}

	if !exists {
		resp.Diagnostics.AddError(
			"File not found",
			fmt.Sprintf("File %s does not exist", data.Path.ValueString()),
		)
		return
	}

	// Get file info
	info, err := backend.Stat(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to stat file", err.Error())
		return
	}

	if info.IsDir {
		resp.Diagnostics.AddError(
			"Path is a directory",
			fmt.Sprintf("Path %s is a directory, not a file", data.Path.ValueString()),
		)
		return
	}

	// Read file content
	reader, err := backend.Read(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file", err.Error())
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file content", err.Error())
		return
	}

	// Detect or use specified format
	format := data.Format.ValueString()
	if format == "" {
		format = detectFormat(data.Path.ValueString())
	}

	if format == "" {
		resp.Diagnostics.AddError(
			"Cannot detect format",
			fmt.Sprintf("Cannot detect format for %s. Please specify format explicitly.", data.Path.ValueString()),
		)
		return
	}

	// Get context lines setting
	contextLines := int(data.ContextLines.ValueInt64())
	if contextLines == 0 {
		contextLines = 3
	}

	// Validate content
	result := validateContent(content, format)

	// Set computed values
	data.ID = data.Path
	data.Size = types.Int64Value(int64(len(content)))
	data.Content = types.StringValue(string(content))
	data.FormatDetected = types.StringValue(format)
	data.IsValid = types.BoolValue(result.IsValid)

	if result.IsValid {
		data.ErrorMessage = types.StringNull()
		data.ErrorLine = types.Int64Null()
		data.ErrorColumn = types.Int64Null()
		data.ErrorOffset = types.Int64Null()
		data.ErrorContext = types.StringNull()
	} else {
		data.ErrorMessage = types.StringValue(result.ErrorMessage)

		if result.ErrorLine > 0 {
			data.ErrorLine = types.Int64Value(int64(result.ErrorLine))
		} else {
			data.ErrorLine = types.Int64Null()
		}

		if result.ErrorColumn > 0 {
			data.ErrorColumn = types.Int64Value(int64(result.ErrorColumn))
		} else {
			data.ErrorColumn = types.Int64Null()
		}

		if result.ErrorOffset > 0 {
			data.ErrorOffset = types.Int64Value(result.ErrorOffset)
		} else {
			data.ErrorOffset = types.Int64Null()
		}

		if result.ErrorLine > 0 {
			data.ErrorContext = types.StringValue(generateContext(string(content), result.ErrorLine, contextLines))
		} else {
			data.ErrorContext = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *ValidateDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// detectFormat detects the format from file extension.
func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".ini", ".cfg", ".conf":
		return "ini"
	case ".env":
		return "env"
	default:
		// Check if filename is .env (without extension)
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".env") {
			return "env"
		}
		return ""
	}
}

// validateContent validates content against the specified format.
func validateContent(content []byte, format string) ValidationResult {
	switch format {
	case "json":
		return validateJSON(content)
	case "yaml":
		return validateYAML(content)
	case "toml":
		return validateTOML(content)
	case "ini":
		return validateINI(content)
	case "env":
		return validateENV(content)
	default:
		return ValidationResult{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("unsupported format: %s", format),
		}
	}
}

// validateJSON validates JSON content.
func validateJSON(content []byte) ValidationResult {
	var v any
	err := json.Unmarshal(content, &v)
	if err == nil {
		return ValidationResult{IsValid: true}
	}

	result := ValidationResult{
		IsValid:      false,
		ErrorMessage: err.Error(),
	}

	// Extract position from json.SyntaxError
	if syntaxErr, ok := err.(*json.SyntaxError); ok {
		result.ErrorOffset = syntaxErr.Offset
		result.ErrorLine, result.ErrorColumn = offsetToLineCol(content, syntaxErr.Offset)
	}

	// Extract position from json.UnmarshalTypeError
	if typeErr, ok := err.(*json.UnmarshalTypeError); ok {
		result.ErrorOffset = typeErr.Offset
		result.ErrorLine, result.ErrorColumn = offsetToLineCol(content, typeErr.Offset)
	}

	return result
}

// validateYAML validates YAML content.
func validateYAML(content []byte) ValidationResult {
	var v any
	err := yaml.Unmarshal(content, &v)
	if err == nil {
		return ValidationResult{IsValid: true}
	}

	result := ValidationResult{
		IsValid:      false,
		ErrorMessage: err.Error(),
	}

	// yaml.v3 TypeError contains line information
	if typeErr, ok := err.(*yaml.TypeError); ok {
		result.ErrorMessage = strings.Join(typeErr.Errors, "; ")
		// Extract line from error message if possible (yaml.v3 format: "line X: ...")
		for _, e := range typeErr.Errors {
			var line int
			if _, scanErr := fmt.Sscanf(e, "line %d:", &line); scanErr == nil {
				result.ErrorLine = line
				break
			}
		}
	}

	return result
}

// validateTOML validates TOML content.
func validateTOML(content []byte) ValidationResult {
	var v any
	err := toml.Unmarshal(content, &v)
	if err == nil {
		return ValidationResult{IsValid: true}
	}

	result := ValidationResult{
		IsValid:      false,
		ErrorMessage: err.Error(),
	}

	// go-toml v2 DecodeError contains line/column via Position()
	if decodeErr, ok := err.(*toml.DecodeError); ok {
		result.ErrorLine, result.ErrorColumn = decodeErr.Position()
	}

	return result
}

// validateINI validates INI content.
func validateINI(content []byte) ValidationResult {
	_, err := ini.Load(content)
	if err == nil {
		return ValidationResult{IsValid: true}
	}

	result := ValidationResult{
		IsValid:      false,
		ErrorMessage: err.Error(),
	}

	// Try to extract line number from error message
	// go-ini errors often contain "line X"
	var line int
	if _, scanErr := fmt.Sscanf(err.Error(), "line %d:", &line); scanErr == nil {
		result.ErrorLine = line
	}

	return result
}

// validateENV validates ENV file content.
func validateENV(content []byte) ValidationResult {
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for valid KEY=VALUE format
		if !strings.Contains(trimmed, "=") {
			return ValidationResult{
				IsValid:      false,
				ErrorMessage: fmt.Sprintf("line %d: missing '=' in environment variable definition", lineNum),
				ErrorLine:    lineNum,
			}
		}

		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.TrimSpace(parts[0])

		// Validate key format (must start with letter or underscore, contain only alphanumeric and underscore)
		if key == "" {
			return ValidationResult{
				IsValid:      false,
				ErrorMessage: fmt.Sprintf("line %d: empty variable name", lineNum),
				ErrorLine:    lineNum,
			}
		}

		// Check first character
		firstChar := key[0]
		if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z') || firstChar == '_') {
			return ValidationResult{
				IsValid:      false,
				ErrorMessage: fmt.Sprintf("line %d: variable name must start with a letter or underscore: %s", lineNum, key),
				ErrorLine:    lineNum,
				ErrorColumn:  1,
			}
		}

		// Check remaining characters
		for j := 1; j < len(key); j++ {
			c := key[j]
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				return ValidationResult{
					IsValid:      false,
					ErrorMessage: fmt.Sprintf("line %d: invalid character '%c' in variable name: %s", lineNum, c, key),
					ErrorLine:    lineNum,
					ErrorColumn:  j + 1,
				}
			}
		}
	}

	return ValidationResult{IsValid: true}
}

// offsetToLineCol converts a byte offset to line and column numbers.
func offsetToLineCol(content []byte, offset int64) (line, col int) {
	if offset <= 0 {
		return 1, 1
	}

	line = 1
	col = 1
	for i := int64(0); i < offset && i < int64(len(content)); i++ {
		if content[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

// generateContext generates context lines around an error.
func generateContext(content string, errorLine int, contextLines int) string {
	lines := strings.Split(content, "\n")
	if errorLine < 1 || errorLine > len(lines) {
		return ""
	}

	startLine := errorLine - contextLines
	if startLine < 1 {
		startLine = 1
	}

	endLine := errorLine + contextLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var result strings.Builder
	maxLineNumWidth := len(fmt.Sprintf("%d", endLine))

	for i := startLine; i <= endLine; i++ {
		lineContent := lines[i-1]
		lineNum := fmt.Sprintf("%*d", maxLineNumWidth, i)

		if i == errorLine {
			result.WriteString(fmt.Sprintf("%s | %s  <-- ERROR\n", lineNum, lineContent))
		} else {
			result.WriteString(fmt.Sprintf("%s | %s\n", lineNum, lineContent))
		}
	}

	return result.String()
}
