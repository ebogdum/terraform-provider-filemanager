// SPDX-License-Identifier: MIT

// Package grafana provides a plugin for Grafana dashboard configuration management.
package grafana

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
)

// Plugin implements the Grafana dashboard configuration plugin.
type Plugin struct{}

// New creates a new Grafana plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "grafana"
}

// Version returns the supported Grafana version range.
func (p *Plugin) Version() string {
	return ">=8.0.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "Grafana dashboard JSON configuration management"
}

// NativeFormat returns the native configuration format.
func (p *Plugin) NativeFormat() string {
	return "json"
}

// Schema returns the configuration schema.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "panels",
				Description: "Dashboard panels",
			},
			{
				Name:        "templating",
				Description: "Dashboard variables/templating",
			},
			{
				Name:        "annotations",
				Description: "Dashboard annotations",
			},
			{
				Name:        "time",
				Description: "Time range settings",
			},
			{
				Name:        "timepicker",
				Description: "Time picker settings",
			},
			{
				Name:        "links",
				Description: "Dashboard links",
			},
		},
		Directives: []plugin.DirectiveSchema{
			{
				Name:        "title",
				Description: "Dashboard title",
				Required:    true,
				Type:        "string",
			},
			{
				Name:        "uid",
				Description: "Dashboard unique identifier",
				Type:        "string",
			},
			{
				Name:        "description",
				Description: "Dashboard description",
				Type:        "string",
			},
			{
				Name:        "tags",
				Description: "Dashboard tags",
				Type:        "array",
			},
			{
				Name:        "editable",
				Description: "Allow editing",
				Type:        "bool",
			},
			{
				Name:        "refresh",
				Description: "Auto-refresh interval",
				Type:        "string",
			},
			{
				Name:        "schemaVersion",
				Description: "Dashboard schema version",
				Type:        "number",
			},
		},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"title":         "New Dashboard",
		"editable":      true,
		"schemaVersion": 39,
		"panels":        []any{},
		"templating": map[string]any{
			"list": []any{},
		},
		"annotations": map[string]any{
			"list": []any{},
		},
		"time": map[string]any{
			"from": "now-6h",
			"to":   "now",
		},
		"timepicker": map[string]any{
			"refresh_intervals": []string{"5s", "10s", "30s", "1m", "5m", "15m", "30m", "1h", "2h", "1d"},
		},
		"timezone": "browser",
		"tags":     []any{},
	}
}

// Validate validates the configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Validate required fields
	if _, ok := configMap["title"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "title",
			Message: "title is required",
		})
	}

	// Validate uid format if present
	if uid, ok := configMap["uid"].(string); ok {
		if !isValidUID(uid) {
			errors = append(errors, plugin.ValidationError{
				Path:    "uid",
				Message: "uid must contain only alphanumeric characters, dashes, and underscores",
			})
		}
	}

	// Validate panels
	if panels, ok := configMap["panels"].([]any); ok {
		errors = append(errors, p.validatePanels(panels)...)
	}

	// Validate templating
	if templating, ok := configMap["templating"].(map[string]any); ok {
		errors = append(errors, p.validateTemplating(templating)...)
	}

	// Validate time range
	if timeRange, ok := configMap["time"].(map[string]any); ok {
		errors = append(errors, p.validateTimeRange(timeRange)...)
	}

	// Validate refresh interval
	if refresh, ok := configMap["refresh"].(string); ok && refresh != "" {
		if !isValidRefreshInterval(refresh) {
			errors = append(errors, plugin.ValidationError{
				Path:    "refresh",
				Message: fmt.Sprintf("invalid refresh interval: %s", refresh),
			})
		}
	}

	// Validate schemaVersion
	if schemaVersion, ok := configMap["schemaVersion"]; ok {
		var version int
		switch v := schemaVersion.(type) {
		case int:
			version = v
		case float64:
			version = int(v)
		}
		if version < 1 {
			errors = append(errors, plugin.ValidationError{
				Path:    "schemaVersion",
				Message: "schemaVersion must be a positive integer",
			})
		}
	}

	return errors, nil
}

func (p *Plugin) validatePanels(panels []any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedIDs := make(map[int]bool)
	gridPositions := make(map[string]int) // Track grid positions for overlap detection

	for i, panel := range panels {
		panelMap, ok := panel.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("panels[%d]", i),
				Message: "panel must be an object",
			})
			continue
		}

		// Validate panel ID
		if id, ok := panelMap["id"]; ok {
			var panelID int
			switch v := id.(type) {
			case int:
				panelID = v
			case float64:
				panelID = int(v)
			}
			if usedIDs[panelID] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("panels[%d].id", i),
					Message: fmt.Sprintf("duplicate panel ID: %d", panelID),
				})
			}
			usedIDs[panelID] = true
		}

		// Validate panel type
		if panelType, ok := panelMap["type"].(string); ok {
			if !isValidPanelType(panelType) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("panels[%d].type", i),
					Message: fmt.Sprintf("unknown panel type: %s", panelType),
				})
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("panels[%d].type", i),
				Message: "panel type is required",
			})
		}

		// Validate panel title
		if _, ok := panelMap["title"]; !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("panels[%d].title", i),
				Message: "panel title is recommended",
			})
		}

		// Validate gridPos
		if gridPos, ok := panelMap["gridPos"].(map[string]any); ok {
			errors = append(errors, p.validateGridPos(gridPos, i, gridPositions)...)
		}

		// Validate targets (data queries)
		if targets, ok := panelMap["targets"].([]any); ok {
			errors = append(errors, p.validateTargets(targets, i)...)
		}
	}

	return errors
}

func (p *Plugin) validateGridPos(gridPos map[string]any, panelIndex int, positions map[string]int) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Extract grid position values
	var x, y, w, h int

	if v, ok := gridPos["x"]; ok {
		switch val := v.(type) {
		case int:
			x = val
		case float64:
			x = int(val)
		}
	}
	if v, ok := gridPos["y"]; ok {
		switch val := v.(type) {
		case int:
			y = val
		case float64:
			y = int(val)
		}
	}
	if v, ok := gridPos["w"]; ok {
		switch val := v.(type) {
		case int:
			w = val
		case float64:
			w = int(val)
		}
	}
	if v, ok := gridPos["h"]; ok {
		switch val := v.(type) {
		case int:
			h = val
		case float64:
			h = int(val)
		}
	}

	// Validate dimensions
	if w <= 0 || w > 24 {
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("panels[%d].gridPos.w", panelIndex),
			Message: fmt.Sprintf("width must be between 1 and 24, got: %d", w),
		})
	}

	if h <= 0 {
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("panels[%d].gridPos.h", panelIndex),
			Message: fmt.Sprintf("height must be positive, got: %d", h),
		})
	}

	if x < 0 || x >= 24 {
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("panels[%d].gridPos.x", panelIndex),
			Message: fmt.Sprintf("x position must be between 0 and 23, got: %d", x),
		})
	}

	if y < 0 {
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("panels[%d].gridPos.y", panelIndex),
			Message: fmt.Sprintf("y position must be non-negative, got: %d", y),
		})
	}

	// Check for panel overflow
	if x+w > 24 {
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("panels[%d].gridPos", panelIndex),
			Message: fmt.Sprintf("panel extends beyond grid (x=%d + w=%d > 24)", x, w),
		})
	}

	// Check for overlapping panels (simplified check)
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			key := fmt.Sprintf("%d,%d", px, py)
			if existingPanel, exists := positions[key]; exists {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("panels[%d].gridPos", panelIndex),
					Message: fmt.Sprintf("panel overlaps with panel at index %d at position (%d, %d)", existingPanel, px, py),
				})
			}
			positions[key] = panelIndex
		}
	}

	return errors
}

func (p *Plugin) validateTargets(targets []any, panelIndex int) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for i, target := range targets {
		targetMap, ok := target.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("panels[%d].targets[%d]", panelIndex, i),
				Message: "target must be an object",
			})
			continue
		}

		// Validate refId
		if refId, ok := targetMap["refId"].(string); ok {
			if refId == "" {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("panels[%d].targets[%d].refId", panelIndex, i),
					Message: "refId should not be empty",
				})
			}
		}

		// Validate datasource if present
		if ds, ok := targetMap["datasource"].(map[string]any); ok {
			if dsType, ok := ds["type"].(string); ok && dsType == "" {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("panels[%d].targets[%d].datasource.type", panelIndex, i),
					Message: "datasource type should not be empty",
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateTemplating(templating map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	if list, ok := templating["list"].([]any); ok {
		usedNames := make(map[string]bool)

		for i, variable := range list {
			varMap, ok := variable.(map[string]any)
			if !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("templating.list[%d]", i),
					Message: "variable must be an object",
				})
				continue
			}

			// Validate name
			name, ok := varMap["name"].(string)
			if !ok || name == "" {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("templating.list[%d].name", i),
					Message: "variable name is required",
				})
			} else {
				if usedNames[name] {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("templating.list[%d].name", i),
						Message: fmt.Sprintf("duplicate variable name: %s", name),
					})
				}
				usedNames[name] = true

				// Validate name format
				if !isValidVariableName(name) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("templating.list[%d].name", i),
						Message: "variable name must contain only alphanumeric characters and underscores",
					})
				}
			}

			// Validate type
			if varType, ok := varMap["type"].(string); ok {
				validTypes := []string{"query", "interval", "datasource", "custom", "constant", "textbox", "adhoc"}
				found := false
				for _, vt := range validTypes {
					if varType == vt {
						found = true
						break
					}
				}
				if !found {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("templating.list[%d].type", i),
						Message: fmt.Sprintf("invalid variable type: %s", varType),
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateTimeRange(timeRange map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate from
	if from, ok := timeRange["from"].(string); ok {
		if !isValidTimeValue(from) {
			errors = append(errors, plugin.ValidationError{
				Path:    "time.from",
				Message: fmt.Sprintf("invalid time value: %s", from),
			})
		}
	}

	// Validate to
	if to, ok := timeRange["to"].(string); ok {
		if !isValidTimeValue(to) {
			errors = append(errors, plugin.ValidationError{
				Path:    "time.to",
				Message: fmt.Sprintf("invalid time value: %s", to),
			})
		}
	}

	return errors
}

// ValidateSemantic performs semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Check for empty dashboard
	if panels, ok := configMap["panels"].([]any); ok && len(panels) == 0 {
		errors = append(errors, plugin.ValidationError{
			Path:    "panels",
			Message: "dashboard has no panels",
		})
	}

	// Check for missing uid
	if _, ok := configMap["uid"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "uid",
			Message: "uid is recommended for dashboard identification",
		})
	}

	// Check for panels without datasource
	if panels, ok := configMap["panels"].([]any); ok {
		for i, panel := range panels {
			panelMap, ok := panel.(map[string]any)
			if !ok {
				continue
			}

			// Check if panel has targets but no datasource
			if targets, ok := panelMap["targets"].([]any); ok && len(targets) > 0 {
				hasGlobalDatasource := false
				if _, ok := panelMap["datasource"]; ok {
					hasGlobalDatasource = true
				}

				for j, target := range targets {
					targetMap, ok := target.(map[string]any)
					if !ok {
						continue
					}
					if _, ok := targetMap["datasource"]; !ok && !hasGlobalDatasource {
						errors = append(errors, plugin.ValidationError{
							Path:    fmt.Sprintf("panels[%d].targets[%d]", i, j),
							Message: "target has no datasource specified",
						})
					}
				}
			}
		}
	}

	return errors, nil
}

// Normalize normalizes the configuration.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Ensure schemaVersion is set
	if _, ok := configMap["schemaVersion"]; !ok {
		configMap["schemaVersion"] = 39
	}

	// Ensure required sections exist
	if _, ok := configMap["panels"]; !ok {
		configMap["panels"] = []any{}
	}

	if _, ok := configMap["templating"]; !ok {
		configMap["templating"] = map[string]any{"list": []any{}}
	}

	if _, ok := configMap["annotations"]; !ok {
		configMap["annotations"] = map[string]any{"list": []any{}}
	}

	return configMap, nil
}

// ToNative converts to native Grafana dashboard format (JSON).
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// FromNative parses native Grafana dashboard format.
func (p *Plugin) FromNative(data []byte) (any, error) {
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return config, nil
}

// Merge merges two configurations.
func (p *Plugin) Merge(base, overlay any) (any, error) {
	return util.DeepMerge(base, overlay)
}

// Diff calculates the difference between two configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	return util.DiffConfigs(old, new)
}

// Helper functions

func isValidUID(uid string) bool {
	// UID should contain only alphanumeric characters, dashes, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, uid)
	return matched
}

func isValidPanelType(panelType string) bool {
	validTypes := []string{
		"graph", "timeseries", "stat", "gauge", "bargauge", "table",
		"text", "heatmap", "histogram", "piechart", "barchart",
		"news", "alertlist", "dashlist", "logs", "nodeGraph",
		"geomap", "canvas", "candlestick", "state-timeline", "status-history",
		"row", // Row is also a valid panel type for grouping
	}
	for _, vt := range validTypes {
		if panelType == vt {
			return true
		}
	}
	return false
}

func isValidRefreshInterval(refresh string) bool {
	// Valid refresh intervals: 5s, 10s, 30s, 1m, 5m, 15m, 30m, 1h, 2h, 1d, or empty string
	if refresh == "" || refresh == "false" || refresh == "off" {
		return true
	}

	validIntervals := []string{"5s", "10s", "30s", "1m", "5m", "15m", "30m", "1h", "2h", "1d"}
	for _, vi := range validIntervals {
		if refresh == vi {
			return true
		}
	}

	// Also accept custom intervals in duration format
	matched, _ := regexp.MatchString(`^\d+[smhd]$`, refresh)
	return matched
}

func isValidVariableName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

func isValidTimeValue(value string) bool {
	// Check for "now" or relative time values like "now-6h", "now-1d"
	if value == "now" {
		return true
	}

	// Relative time patterns
	relativePattern := regexp.MustCompile(`^now[-+]\d+[smhdwMy](/[smhdwMy])?$`)
	if relativePattern.MatchString(value) {
		return true
	}

	// Absolute time (ISO format or epoch)
	isoPattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?)?$`)
	if isoPattern.MatchString(value) {
		return true
	}

	// Epoch milliseconds
	epochPattern := regexp.MustCompile(`^\d{13}$`)
	if epochPattern.MatchString(value) {
		return true
	}

	// Check for valid time keywords
	validKeywords := []string{"now/d", "now/w", "now/M", "now/y"}
	for _, keyword := range validKeywords {
		if strings.HasPrefix(value, keyword) {
			return true
		}
	}

	return false
}
