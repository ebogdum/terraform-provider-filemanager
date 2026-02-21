// SPDX-License-Identifier: MIT

// Package systemd provides a systemd unit file configuration management plugin.
package systemd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/ini.v1"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for systemd.
type Plugin struct{}

// New creates a new systemd plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "systemd"
}

// Version returns the supported systemd version range.
func (p *Plugin) Version() string {
	return ">=219"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Systemd unit file configuration management (.service, .timer, .socket, etc.)"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "ini"
}

// Schema returns the configuration schema for systemd units.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "Unit",
				Required:    false,
				Description: "Unit metadata and dependencies",
				Directives: []plugin.DirectiveSchema{
					{Name: "Description", Type: "string", Description: "Human-readable description"},
					{Name: "Documentation", Type: "string", Description: "Documentation URL"},
					{Name: "After", Type: "string", Description: "Units to start after"},
					{Name: "Before", Type: "string", Description: "Units to start before"},
					{Name: "Requires", Type: "string", Description: "Required units"},
					{Name: "Wants", Type: "string", Description: "Wanted units (weak dependency)"},
					{Name: "BindsTo", Type: "string", Description: "Strongly bound units"},
					{Name: "Conflicts", Type: "string", Description: "Conflicting units"},
					{Name: "ConditionPathExists", Type: "string", Description: "Condition: path must exist"},
					{Name: "ConditionPathIsDirectory", Type: "string", Description: "Condition: path must be directory"},
				},
			},
			{
				Name:        "Service",
				Required:    false,
				Description: "Service-specific configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "Type", Type: "string", ValidValues: []string{"simple", "exec", "forking", "oneshot", "dbus", "notify", "idle"}, Description: "Service type"},
					{Name: "ExecStart", Type: "string", Required: true, Description: "Command to start the service"},
					{Name: "ExecStartPre", Type: "string", Description: "Commands to run before starting"},
					{Name: "ExecStartPost", Type: "string", Description: "Commands to run after starting"},
					{Name: "ExecStop", Type: "string", Description: "Command to stop the service"},
					{Name: "ExecReload", Type: "string", Description: "Command to reload the service"},
					{Name: "Restart", Type: "string", ValidValues: []string{"no", "on-success", "on-failure", "on-abnormal", "on-watchdog", "on-abort", "always"}, Description: "Restart policy"},
					{Name: "RestartSec", Type: "string", Description: "Time to wait before restarting"},
					{Name: "User", Type: "string", Description: "User to run as"},
					{Name: "Group", Type: "string", Description: "Group to run as"},
					{Name: "WorkingDirectory", Type: "string", Description: "Working directory"},
					{Name: "Environment", Type: "string", Description: "Environment variables"},
					{Name: "EnvironmentFile", Type: "string", Description: "Environment file"},
					{Name: "PIDFile", Type: "string", Description: "PID file path (for forking type)"},
					{Name: "TimeoutStartSec", Type: "string", Description: "Startup timeout"},
					{Name: "TimeoutStopSec", Type: "string", Description: "Stop timeout"},
					{Name: "KillMode", Type: "string", ValidValues: []string{"control-group", "mixed", "process", "none"}, Description: "Kill mode"},
					{Name: "KillSignal", Type: "string", Description: "Signal to use for stopping"},
				},
			},
			{
				Name:        "Timer",
				Required:    false,
				Description: "Timer-specific configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "OnCalendar", Type: "string", Description: "Calendar-based schedule"},
					{Name: "OnBootSec", Type: "string", Description: "Time after boot"},
					{Name: "OnUnitActiveSec", Type: "string", Description: "Time after unit activation"},
					{Name: "OnUnitInactiveSec", Type: "string", Description: "Time after unit deactivation"},
					{Name: "Unit", Type: "string", Description: "Unit to activate"},
					{Name: "Persistent", Type: "bool", Description: "Run missed runs on boot"},
					{Name: "AccuracySec", Type: "string", Description: "Timer accuracy"},
				},
			},
			{
				Name:        "Socket",
				Required:    false,
				Description: "Socket-specific configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "ListenStream", Type: "string", Description: "Stream socket address"},
					{Name: "ListenDatagram", Type: "string", Description: "Datagram socket address"},
					{Name: "ListenSequentialPacket", Type: "string", Description: "Sequential packet socket"},
					{Name: "ListenFIFO", Type: "string", Description: "FIFO path"},
					{Name: "Accept", Type: "bool", Description: "Accept connections"},
					{Name: "Service", Type: "string", Description: "Service to activate"},
				},
			},
			{
				Name:        "Install",
				Required:    false,
				Description: "Installation and enablement",
				Directives: []plugin.DirectiveSchema{
					{Name: "WantedBy", Type: "string", Description: "Target units that want this unit"},
					{Name: "RequiredBy", Type: "string", Description: "Target units that require this unit"},
					{Name: "Alias", Type: "string", Description: "Alias names"},
					{Name: "Also", Type: "string", Description: "Additional units to enable/disable"},
				},
			},
		},
	}
}

// DefaultConfig returns a sample systemd service configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"Unit": map[string]any{
			"Description": "Example Service",
			"After":       "network.target",
		},
		"Service": map[string]any{
			"Type":             "simple",
			"ExecStart":        "/usr/bin/example",
			"Restart":          "on-failure",
			"RestartSec":       "5s",
			"User":             "nobody",
			"WorkingDirectory": "/opt/example",
		},
		"Install": map[string]any{
			"WantedBy": "multi-user.target",
		},
	}
}

// Validate validates the systemd unit configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "", Message: "configuration must be a map"}}, nil
	}

	// Validate sections
	validSections := map[string]bool{
		"Unit": true, "Service": true, "Timer": true, "Socket": true,
		"Install": true, "Mount": true, "Automount": true, "Swap": true,
		"Path": true, "Slice": true, "Scope": true,
	}

	for section := range configMap {
		if !validSections[section] {
			errors = append(errors, plugin.ValidationError{
				Path:    section,
				Message: fmt.Sprintf("unknown section: %s", section),
			})
		}
	}

	// Validate Service section
	if service, ok := configMap["Service"].(map[string]any); ok {
		serviceErrors := p.validateService(service)
		errors = append(errors, serviceErrors...)
	}

	// Validate Timer section
	if timer, ok := configMap["Timer"].(map[string]any); ok {
		timerErrors := p.validateTimer(timer)
		errors = append(errors, timerErrors...)
	}

	// Validate Socket section
	if socket, ok := configMap["Socket"].(map[string]any); ok {
		socketErrors := p.validateSocket(socket)
		errors = append(errors, socketErrors...)
	}

	// Validate Install section
	if install, ok := configMap["Install"].(map[string]any); ok {
		installErrors := p.validateInstall(install)
		errors = append(errors, installErrors...)
	}

	return errors, nil
}

func (p *Plugin) validateService(service map[string]any) []plugin.ValidationError {
	errors := make([]plugin.ValidationError, 0)
	errors = append(errors, validateServiceEnumField(service, "Type", "service type", []string{"simple", "exec", "forking", "oneshot", "dbus", "notify", "idle"})...)
	errors = append(errors, validateServiceEnumField(service, "Restart", "Restart value", []string{"no", "on-success", "on-failure", "on-abnormal", "on-watchdog", "on-abort", "always"})...)
	errors = append(errors, validateServiceEnumField(service, "KillMode", "KillMode", []string{"control-group", "mixed", "process", "none"})...)
	errors = append(errors, validateServiceTimeFields(service)...)
	errors = append(errors, validateServiceExecStart(service)...)
	return errors
}

func validateServiceEnumField(service map[string]any, field, label string, valid []string) []plugin.ValidationError {
	value, ok := service[field]
	if !ok {
		return nil
	}

	valueStr := fmt.Sprintf("%v", value)
	if containsString(valid, valueStr) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "Service." + field,
		Message: fmt.Sprintf("invalid %s: %s (valid: %v)", label, valueStr, valid),
	}}
}

func validateServiceTimeFields(service map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for _, field := range []string{"RestartSec", "TimeoutStartSec", "TimeoutStopSec", "WatchdogSec"} {
		value, ok := service[field]
		if !ok {
			continue
		}
		if isValidTimeSpec(fmt.Sprintf("%v", value)) {
			continue
		}

		errors = append(errors, plugin.ValidationError{
			Path:    "Service." + field,
			Message: fmt.Sprintf("invalid time specification: %v", value),
		})
	}

	return errors
}

func validateServiceExecStart(service map[string]any) []plugin.ValidationError {
	svcType := fmt.Sprintf("%v", service["Type"])
	if svcType == "oneshot" {
		return nil
	}

	if _, hasExecStart := service["ExecStart"]; hasExecStart {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "Service.ExecStart",
		Message: "ExecStart is required for non-oneshot service types",
	}}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (p *Plugin) validateTimer(timer map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// At least one trigger is required
	triggers := []string{"OnCalendar", "OnBootSec", "OnStartupSec", "OnUnitActiveSec", "OnUnitInactiveSec", "OnActiveSec"}
	hasTrigger := false
	for _, t := range triggers {
		if _, ok := timer[t]; ok {
			hasTrigger = true
			break
		}
	}

	if !hasTrigger {
		errors = append(errors, plugin.ValidationError{
			Path:    "Timer",
			Message: fmt.Sprintf("at least one timer trigger is required (one of: %v)", triggers),
		})
	}

	// Validate OnCalendar format if present
	if calendar, ok := timer["OnCalendar"]; ok {
		calStr := fmt.Sprintf("%v", calendar)
		if !isValidCalendarSpec(calStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    "Timer.OnCalendar",
				Message: fmt.Sprintf("invalid calendar specification: %s", calStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateSocket(socket map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// At least one Listen* directive is required
	listeners := []string{"ListenStream", "ListenDatagram", "ListenSequentialPacket", "ListenFIFO", "ListenSpecial", "ListenNetlink", "ListenMessageQueue", "ListenUSBFunction"}
	hasListener := false
	for _, l := range listeners {
		if _, ok := socket[l]; ok {
			hasListener = true
			break
		}
	}

	if !hasListener {
		errors = append(errors, plugin.ValidationError{
			Path:    "Socket",
			Message: "at least one Listen* directive is required",
		})
	}

	return errors
}

func (p *Plugin) validateInstall(install map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate target references
	targetFields := []string{"WantedBy", "RequiredBy"}
	for _, field := range targetFields {
		if val, ok := install[field]; ok {
			valStr := fmt.Sprintf("%v", val)
			targets := strings.Fields(valStr)
			for _, target := range targets {
				if !isValidUnitName(target) {
					errors = append(errors, plugin.ValidationError{
						Path:    "Install." + field,
						Message: fmt.Sprintf("invalid unit name: %s", target),
					})
				}
			}
		}
	}

	return errors
}

// ValidateSemantic performs systemd-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return errors, nil
	}

	// Check for common issues
	if service, ok := configMap["Service"].(map[string]any); ok {
		svcType := fmt.Sprintf("%v", service["Type"])

		// Warn if forking type without PIDFile
		if svcType == "forking" {
			if _, hasPID := service["PIDFile"]; !hasPID {
				errors = append(errors, plugin.ValidationError{
					Path:    "Service",
					Message: "forking service type usually requires PIDFile",
				})
			}
		}

		// Warn if running as root
		if user, ok := service["User"]; ok {
			userStr := fmt.Sprintf("%v", user)
			if userStr == "root" || userStr == "0" {
				errors = append(errors, plugin.ValidationError{
					Path:    "Service.User",
					Message: "service runs as root - consider using a dedicated user",
				})
			}
		} else {
			// No user specified means root
			errors = append(errors, plugin.ValidationError{
				Path:    "Service",
				Message: "no User specified - service will run as root",
			})
		}

		// Warn about Restart=always without rate limiting
		if restart, ok := service["Restart"]; ok {
			if fmt.Sprintf("%v", restart) == "always" {
				if _, hasStartLimit := service["StartLimitIntervalSec"]; !hasStartLimit {
					errors = append(errors, plugin.ValidationError{
						Path:    "Service.Restart",
						Message: "Restart=always without StartLimitIntervalSec may cause rapid restart loops",
					})
				}
			}
		}
	}

	return errors, nil
}

// Normalize normalizes the configuration to a canonical form.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return config, nil
	}

	result := make(map[string]any)

	// Copy with canonical section order
	sectionOrder := []string{"Unit", "Service", "Timer", "Socket", "Mount", "Automount", "Swap", "Path", "Slice", "Scope", "Install"}
	for _, section := range sectionOrder {
		if v, ok := configMap[section]; ok {
			result[section] = v
		}
	}

	// Copy any remaining sections
	for k, v := range configMap {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result, nil
}

// ToNative converts the configuration to systemd unit file format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("configuration must be a map")
	}

	cfg := ini.Empty()

	// Section order
	sectionOrder := []string{"Unit", "Service", "Timer", "Socket", "Mount", "Automount", "Swap", "Path", "Slice", "Scope", "Install"}

	for _, sectionName := range sectionOrder {
		sectionData, ok := configMap[sectionName]
		if !ok {
			continue
		}

		sectionMap, ok := sectionData.(map[string]any)
		if !ok {
			continue
		}

		section, err := cfg.NewSection(sectionName)
		if err != nil {
			return nil, fmt.Errorf("failed to create section %s: %w", sectionName, err)
		}

		// Sort keys for consistent output
		keys := make([]string, 0, len(sectionMap))
		for k := range sectionMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := sectionMap[key]
			if _, err := section.NewKey(key, fmt.Sprintf("%v", value)); err != nil {
				return nil, fmt.Errorf("failed to set key %s in section %s: %w", key, sectionName, err)
			}
		}
	}

	var buf strings.Builder
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("failed to render systemd unit: %w", err)
	}
	return []byte(buf.String()), nil
}

// FromNative parses systemd unit file format into configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse systemd unit: %w", err)
	}

	result := make(map[string]any)

	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		sectionMap := make(map[string]any)
		for _, key := range section.Keys() {
			sectionMap[key.Name()] = key.Value()
		}

		if len(sectionMap) > 0 {
			result[section.Name()] = sectionMap
		}
	}

	return result, nil
}

// Merge merges two systemd unit configurations.
func (p *Plugin) Merge(base, overlay any) (any, error) {
	baseMap, baseOk := base.(map[string]any)
	overlayMap, overlayOk := overlay.(map[string]any)

	if !baseOk || !overlayOk {
		return overlay, nil
	}

	result := make(map[string]any)

	// Copy base
	for k, v := range baseMap {
		result[k] = deepCopy(v)
	}

	// Merge overlay sections
	for k, v := range overlayMap {
		if existing, ok := result[k]; ok {
			// Deep merge sections
			merged, _ := mergeDeep(existing, v)
			result[k] = merged
		} else {
			result[k] = deepCopy(v)
		}
	}

	return result, nil
}

// Diff detects changes between two systemd unit configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	return diffMaps(old, new, "")
}

// Helper functions

var timeSpecRegex = regexp.MustCompile(`^(\d+(\.\d+)?)(us|ms|s|sec|seconds|m|min|minutes|h|hr|hours|d|days|w|weeks|M|months|y|years)?$`)

func isValidTimeSpec(s string) bool {
	// Handle special values
	if s == "infinity" || s == "" {
		return true
	}
	return timeSpecRegex.MatchString(s)
}

func isValidCalendarSpec(s string) bool {
	// Handle common shortcuts
	shortcuts := []string{"hourly", "daily", "weekly", "monthly", "yearly", "quarterly", "semiannually", "minutely"}
	for _, shortcut := range shortcuts {
		if s == shortcut {
			return true
		}
	}
	// Handle complex specs (simplified validation)
	return len(s) > 0
}

var unitNameRegex = regexp.MustCompile(`^[a-zA-Z0-9:._@-]+\.(service|timer|socket|mount|automount|swap|target|path|slice|scope)$`)

func isValidUnitName(s string) bool {
	return unitNameRegex.MatchString(s)
}

func deepCopy(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = deepCopy(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = deepCopy(v)
		}
		return result
	default:
		return v
	}
}

func mergeDeep(base, overlay any) (any, error) {
	baseMap, baseOk := base.(map[string]any)
	overlayMap, overlayOk := overlay.(map[string]any)

	if !baseOk || !overlayOk {
		return overlay, nil
	}

	result := make(map[string]any)
	for k, v := range baseMap {
		result[k] = v
	}
	for k, v := range overlayMap {
		result[k] = v
	}
	return result, nil
}

func diffMaps(old, new any, path string) ([]plugin.Change, error) {
	var changes []plugin.Change

	oldMap, oldOk := old.(map[string]any)
	newMap, newOk := new.(map[string]any)

	if !oldOk && !newOk {
		if fmt.Sprintf("%v", old) != fmt.Sprintf("%v", new) {
			changes = append(changes, plugin.Change{
				Type:     plugin.ChangeModified,
				Path:     path,
				OldValue: old,
				NewValue: new,
			})
		}
		return changes, nil
	}

	if !oldOk || !newOk {
		changes = append(changes, plugin.Change{
			Type:     plugin.ChangeModified,
			Path:     path,
			OldValue: old,
			NewValue: new,
		})
		return changes, nil
	}

	allKeys := make(map[string]bool)
	for k := range oldMap {
		allKeys[k] = true
	}
	for k := range newMap {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		keyPath := k
		if path != "" {
			keyPath = path + "." + k
		}

		oldVal, oldHas := oldMap[k]
		newVal, newHas := newMap[k]

		if !oldHas {
			changes = append(changes, plugin.Change{
				Type:     plugin.ChangeAdded,
				Path:     keyPath,
				NewValue: newVal,
			})
		} else if !newHas {
			changes = append(changes, plugin.Change{
				Type:     plugin.ChangeRemoved,
				Path:     keyPath,
				OldValue: oldVal,
			})
		} else {
			subChanges, _ := diffMaps(oldVal, newVal, keyPath)
			changes = append(changes, subChanges...)
		}
	}

	return changes, nil
}
