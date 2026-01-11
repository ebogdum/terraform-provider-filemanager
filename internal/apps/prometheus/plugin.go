// SPDX-License-Identifier: MIT

// Package prometheus provides a Prometheus configuration management plugin.
package prometheus

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Prometheus.
type Plugin struct{}

// New creates a new Prometheus plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "prometheus"
}

// Version returns the supported Prometheus version range.
func (p *Plugin) Version() string {
	return ">=2.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Prometheus monitoring system configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "yaml"
}

// Schema returns the configuration schema for Prometheus.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "global",
				Required:    false,
				Description: "Global configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "scrape_interval", Type: "duration", Default: "1m", Description: "Default scrape interval"},
					{Name: "scrape_timeout", Type: "duration", Default: "10s", Description: "Default scrape timeout"},
					{Name: "evaluation_interval", Type: "duration", Default: "1m", Description: "Rule evaluation interval"},
					{Name: "external_labels", Type: "map", Description: "Labels to add to metrics"},
				},
			},
			{
				Name:        "alerting",
				Required:    false,
				Description: "Alertmanager configuration",
				Subsections: []plugin.SectionSchema{
					{
						Name:        "alertmanagers",
						Multiple:    true,
						Description: "Alertmanager endpoints",
						Directives: []plugin.DirectiveSchema{
							{Name: "scheme", Type: "string", Default: "http", ValidValues: []string{"http", "https"}},
							{Name: "timeout", Type: "duration", Default: "10s"},
							{Name: "api_version", Type: "string", Default: "v2", ValidValues: []string{"v1", "v2"}},
						},
					},
				},
			},
			{
				Name:        "rule_files",
				Required:    false,
				Multiple:    true,
				Description: "Rule file paths (glob patterns supported)",
			},
			{
				Name:        "scrape_configs",
				Required:    false,
				Multiple:    true,
				Description: "Scrape configurations",
				Directives: []plugin.DirectiveSchema{
					{Name: "job_name", Type: "string", Required: true, Description: "Job name for scraped metrics"},
					{Name: "scrape_interval", Type: "duration", Description: "Override global scrape interval"},
					{Name: "scrape_timeout", Type: "duration", Description: "Override global scrape timeout"},
					{Name: "metrics_path", Type: "string", Default: "/metrics", Description: "Metrics endpoint path"},
					{Name: "scheme", Type: "string", Default: "http", ValidValues: []string{"http", "https"}},
					{Name: "honor_labels", Type: "bool", Default: false, Description: "Honor labels from scrape target"},
					{Name: "honor_timestamps", Type: "bool", Default: true, Description: "Honor timestamps from scrape target"},
					{Name: "params", Type: "map", Description: "HTTP URL parameters"},
				},
				Subsections: []plugin.SectionSchema{
					{
						Name:        "static_configs",
						Multiple:    true,
						Description: "Static target configuration",
						Directives: []plugin.DirectiveSchema{
							{Name: "targets", Type: "list", Required: true, Description: "List of targets (host:port)"},
							{Name: "labels", Type: "map", Description: "Labels for these targets"},
						},
					},
					{
						Name:        "file_sd_configs",
						Multiple:    true,
						Description: "File-based service discovery",
						Directives: []plugin.DirectiveSchema{
							{Name: "files", Type: "list", Required: true, Description: "File patterns to watch"},
							{Name: "refresh_interval", Type: "duration", Default: "5m", Description: "Refresh interval"},
						},
					},
					{
						Name:        "kubernetes_sd_configs",
						Multiple:    true,
						Description: "Kubernetes service discovery",
						Directives: []plugin.DirectiveSchema{
							{Name: "role", Type: "string", Required: true, ValidValues: []string{"node", "service", "pod", "endpoints", "endpointslice", "ingress"}},
							{Name: "namespaces", Type: "object", Description: "Namespace configuration"},
							{Name: "selectors", Type: "list", Description: "Label selectors"},
						},
					},
					{
						Name:        "relabel_configs",
						Multiple:    true,
						Description: "Relabeling configuration",
						Directives: []plugin.DirectiveSchema{
							{Name: "source_labels", Type: "list", Description: "Source labels"},
							{Name: "separator", Type: "string", Default: ";", Description: "Label separator"},
							{Name: "target_label", Type: "string", Description: "Target label"},
							{Name: "regex", Type: "string", Default: "(.*)", Description: "Regex for matching"},
							{Name: "modulus", Type: "int", Description: "Modulus for hashing"},
							{Name: "replacement", Type: "string", Default: "$1", Description: "Replacement value"},
							{Name: "action", Type: "string", Default: "replace", ValidValues: []string{"replace", "keep", "drop", "hashmod", "labelmap", "labeldrop", "labelkeep"}},
						},
					},
					{
						Name:        "metric_relabel_configs",
						Multiple:    true,
						Description: "Metric relabeling configuration",
						Directives: []plugin.DirectiveSchema{
							{Name: "source_labels", Type: "list"},
							{Name: "separator", Type: "string", Default: ";"},
							{Name: "target_label", Type: "string"},
							{Name: "regex", Type: "string", Default: "(.*)"},
							{Name: "replacement", Type: "string", Default: "$1"},
							{Name: "action", Type: "string", Default: "replace"},
						},
					},
				},
			},
			{
				Name:        "remote_write",
				Required:    false,
				Multiple:    true,
				Description: "Remote write configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "url", Type: "string", Required: true, Description: "Remote write endpoint URL"},
					{Name: "remote_timeout", Type: "duration", Default: "30s", Description: "Request timeout"},
					{Name: "name", Type: "string", Description: "Name for this remote write config"},
				},
			},
			{
				Name:        "remote_read",
				Required:    false,
				Multiple:    true,
				Description: "Remote read configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "url", Type: "string", Required: true, Description: "Remote read endpoint URL"},
					{Name: "remote_timeout", Type: "duration", Default: "1m", Description: "Request timeout"},
					{Name: "name", Type: "string", Description: "Name for this remote read config"},
					{Name: "read_recent", Type: "bool", Default: false, Description: "Read recent data from remote"},
				},
			},
		},
	}
}

// DefaultConfig returns sensible default Prometheus configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"global": map[string]any{
			"scrape_interval":     "15s",
			"evaluation_interval": "15s",
		},
		"scrape_configs": []any{
			map[string]any{
				"job_name": "prometheus",
				"static_configs": []any{
					map[string]any{
						"targets": []string{"localhost:9090"},
					},
				},
			},
		},
	}
}

// Validate validates the Prometheus configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "", Message: "configuration must be a map"}}, nil
	}

	// Validate global section
	if global, ok := configMap["global"]; ok {
		globalErrors := p.validateGlobal(global)
		errors = append(errors, globalErrors...)
	}

	// Validate scrape_configs
	if scrapeConfigs, ok := configMap["scrape_configs"]; ok {
		scrapeErrors := p.validateScrapeConfigs(scrapeConfigs)
		errors = append(errors, scrapeErrors...)
	}

	// Validate remote_write
	if remoteWrite, ok := configMap["remote_write"]; ok {
		rwErrors := p.validateRemoteWrite(remoteWrite)
		errors = append(errors, rwErrors...)
	}

	return errors, nil
}

func (p *Plugin) validateGlobal(global any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	globalMap, ok := global.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "global", Message: "must be a map"}}
	}

	// Validate duration fields
	durationFields := []string{"scrape_interval", "scrape_timeout", "evaluation_interval"}
	for _, field := range durationFields {
		if v, ok := globalMap[field]; ok {
			if !isValidDuration(v) {
				errors = append(errors, plugin.ValidationError{
					Path:    "global." + field,
					Message: fmt.Sprintf("invalid duration: %v", v),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateScrapeConfigs(scrapeConfigs any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	configs, ok := scrapeConfigs.([]any)
	if !ok {
		return []plugin.ValidationError{{Path: "scrape_configs", Message: "must be a list"}}
	}

	jobNames := make(map[string]bool)

	for i, config := range configs {
		path := fmt.Sprintf("scrape_configs[%d]", i)

		configMap, ok := config.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "scrape config must be a map",
			})
			continue
		}

		// Validate job_name (required and unique)
		jobName, ok := configMap["job_name"]
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "job_name is required",
			})
		} else {
			jobNameStr := fmt.Sprintf("%v", jobName)
			if jobNames[jobNameStr] {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".job_name",
					Message: fmt.Sprintf("duplicate job_name: %s", jobNameStr),
				})
			}
			jobNames[jobNameStr] = true

			// Validate job_name format
			if !isValidJobName(jobNameStr) {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".job_name",
					Message: fmt.Sprintf("invalid job_name format: %s (must match [a-zA-Z_:][a-zA-Z0-9_:]*)", jobNameStr),
				})
			}
		}

		// Validate static_configs
		if staticConfigs, ok := configMap["static_configs"]; ok {
			scErrors := p.validateStaticConfigs(staticConfigs, path)
			errors = append(errors, scErrors...)
		}

		// Validate relabel_configs
		if relabelConfigs, ok := configMap["relabel_configs"]; ok {
			rcErrors := p.validateRelabelConfigs(relabelConfigs, path+".relabel_configs")
			errors = append(errors, rcErrors...)
		}
	}

	return errors
}

func (p *Plugin) validateStaticConfigs(staticConfigs any, basePath string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	configs, ok := staticConfigs.([]any)
	if !ok {
		return []plugin.ValidationError{{Path: basePath + ".static_configs", Message: "must be a list"}}
	}

	for i, config := range configs {
		path := fmt.Sprintf("%s.static_configs[%d]", basePath, i)

		configMap, ok := config.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "static config must be a map",
			})
			continue
		}

		// Validate targets (required)
		targets, ok := configMap["targets"]
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "targets is required",
			})
		} else {
			targetErrors := p.validateTargets(targets, path+".targets")
			errors = append(errors, targetErrors...)
		}
	}

	return errors
}

func (p *Plugin) validateTargets(targets any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	targetList, ok := targets.([]any)
	if !ok {
		// Try string slice
		if _, ok := targets.([]string); ok {
			return errors
		}
		return []plugin.ValidationError{{Path: path, Message: "targets must be a list"}}
	}

	for i, target := range targetList {
		targetStr, ok := target.(string)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s[%d]", path, i),
				Message: "target must be a string",
			})
			continue
		}

		// Validate target format (host:port)
		if !isValidTarget(targetStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s[%d]", path, i),
				Message: fmt.Sprintf("invalid target format: %s (expected host:port)", targetStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateRelabelConfigs(relabelConfigs any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	configs, ok := relabelConfigs.([]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "must be a list"}}
	}

	validActions := map[string]bool{
		"replace": true, "keep": true, "drop": true, "hashmod": true,
		"labelmap": true, "labeldrop": true, "labelkeep": true,
	}

	for i, config := range configs {
		configPath := fmt.Sprintf("%s[%d]", path, i)

		configMap, ok := config.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    configPath,
				Message: "relabel config must be a map",
			})
			continue
		}

		// Validate action
		if action, ok := configMap["action"]; ok {
			actionStr := fmt.Sprintf("%v", action)
			if !validActions[actionStr] {
				errors = append(errors, plugin.ValidationError{
					Path:    configPath + ".action",
					Message: fmt.Sprintf("invalid action: %s", actionStr),
				})
			}
		}

		// Validate regex if present
		if regex, ok := configMap["regex"]; ok {
			regexStr := fmt.Sprintf("%v", regex)
			if _, err := regexp.Compile(regexStr); err != nil {
				errors = append(errors, plugin.ValidationError{
					Path:    configPath + ".regex",
					Message: fmt.Sprintf("invalid regex: %v", err),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateRemoteWrite(remoteWrite any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	configs, ok := remoteWrite.([]any)
	if !ok {
		return []plugin.ValidationError{{Path: "remote_write", Message: "must be a list"}}
	}

	for i, config := range configs {
		path := fmt.Sprintf("remote_write[%d]", i)

		configMap, ok := config.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "remote write config must be a map",
			})
			continue
		}

		// URL is required
		if _, ok := configMap["url"]; !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "url is required",
			})
		}
	}

	return errors
}

// ValidateSemantic performs Prometheus-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return errors, nil
	}

	// Check for common issues
	if scrapeConfigs, ok := configMap["scrape_configs"].([]any); ok {
		for i, sc := range scrapeConfigs {
			if scMap, ok := sc.(map[string]any); ok {
				// Warn if scrape_interval is less than global
				if interval, ok := scMap["scrape_interval"]; ok {
					globalInterval := "15s"
					if global, ok := configMap["global"].(map[string]any); ok {
						if gi, ok := global["scrape_interval"]; ok {
							globalInterval = fmt.Sprintf("%v", gi)
						}
					}

					if compareDurations(fmt.Sprintf("%v", interval), globalInterval) < 0 {
						errors = append(errors, plugin.ValidationError{
							Path:    fmt.Sprintf("scrape_configs[%d].scrape_interval", i),
							Message: fmt.Sprintf("scrape_interval (%v) is less than global (%s), this may cause high load", interval, globalInterval),
						})
					}
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

	// Process each section
	for k, v := range configMap {
		switch k {
		case "global":
			if globalMap, ok := v.(map[string]any); ok {
				normalized := make(map[string]any)
				for gk, gv := range globalMap {
					// Normalize duration fields
					if isDurationField(gk) {
						normalized[gk] = normalizeDuration(gv)
					} else {
						normalized[gk] = gv
					}
				}
				result[k] = normalized
			} else {
				result[k] = v
			}

		case "scrape_configs":
			if configs, ok := v.([]any); ok {
				normalizedConfigs := make([]any, len(configs))
				for i, c := range configs {
					normalizedConfigs[i] = p.normalizeScrapeConfig(c)
				}
				result[k] = normalizedConfigs
			} else {
				result[k] = v
			}

		default:
			result[k] = v
		}
	}

	return result, nil
}

func (p *Plugin) normalizeScrapeConfig(config any) any {
	configMap, ok := config.(map[string]any)
	if !ok {
		return config
	}

	result := make(map[string]any)
	for k, v := range configMap {
		if isDurationField(k) {
			result[k] = normalizeDuration(v)
		} else {
			result[k] = v
		}
	}
	return result
}

// ToNative converts the configuration to Prometheus native YAML format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return yaml.Marshal(config)
}

// FromNative parses Prometheus native YAML format into configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Prometheus config: %w", err)
	}
	return result, nil
}

// Merge merges two Prometheus configurations.
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

	// Merge overlay
	for k, v := range overlayMap {
		switch k {
		case "scrape_configs":
			// Merge scrape configs by job_name
			result[k] = p.mergeScrapeConfigs(result[k], v)

		case "rule_files":
			// Append rule files
			result[k] = appendLists(result[k], v)

		case "remote_write", "remote_read":
			// Append remote configs
			result[k] = appendLists(result[k], v)

		case "global", "alerting":
			// Deep merge these sections
			if existing, ok := result[k]; ok {
				merged, _ := p.mergeDeep(existing, v)
				result[k] = merged
			} else {
				result[k] = deepCopy(v)
			}

		default:
			result[k] = deepCopy(v)
		}
	}

	return result, nil
}

func (p *Plugin) mergeScrapeConfigs(base, overlay any) any {
	baseConfigs, baseOk := base.([]any)
	overlayConfigs, overlayOk := overlay.([]any)

	if !baseOk {
		return overlay
	}
	if !overlayOk {
		return base
	}

	// Index base configs by job_name
	jobIndex := make(map[string]int)
	result := make([]any, len(baseConfigs))
	for i, c := range baseConfigs {
		result[i] = deepCopy(c)
		if cMap, ok := c.(map[string]any); ok {
			if jobName, ok := cMap["job_name"].(string); ok {
				jobIndex[jobName] = i
			}
		}
	}

	// Merge or append overlay configs
	for _, c := range overlayConfigs {
		cMap, ok := c.(map[string]any)
		if !ok {
			result = append(result, c)
			continue
		}

		jobName, ok := cMap["job_name"].(string)
		if !ok {
			result = append(result, deepCopy(c))
			continue
		}

		if idx, exists := jobIndex[jobName]; exists {
			// Merge with existing config
			if baseMap, ok := result[idx].(map[string]any); ok {
				merged, _ := p.mergeDeep(baseMap, cMap)
				result[idx] = merged
			}
		} else {
			// Append new config
			result = append(result, deepCopy(c))
			jobIndex[jobName] = len(result) - 1
		}
	}

	return result
}

func (p *Plugin) mergeDeep(base, overlay any) (any, error) {
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
		if existing, ok := result[k]; ok {
			if _, isMap := existing.(map[string]any); isMap {
				if _, vIsMap := v.(map[string]any); vIsMap {
					merged, _ := p.mergeDeep(existing, v)
					result[k] = merged
					continue
				}
			}
		}
		result[k] = v
	}
	return result, nil
}

// Diff detects changes between two Prometheus configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	return diffMaps(old, new, "")
}

// Helper functions

func isValidDuration(v any) bool {
	s, ok := v.(string)
	if !ok {
		// Numbers are valid (interpreted as seconds)
		switch v.(type) {
		case int, int64, float64:
			return true
		default:
			return false
		}
	}
	_, err := time.ParseDuration(s)
	return err == nil
}

var jobNameRegex = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

func isValidJobName(name string) bool {
	return jobNameRegex.MatchString(name)
}

var targetRegex = regexp.MustCompile(`^[^:]+:\d+$`)

func isValidTarget(target string) bool {
	// Allow localhost:port or hostname:port or ip:port
	return targetRegex.MatchString(target) || strings.HasPrefix(target, "localhost")
}

func isDurationField(name string) bool {
	durationFields := map[string]bool{
		"scrape_interval":     true,
		"scrape_timeout":      true,
		"evaluation_interval": true,
		"timeout":             true,
		"remote_timeout":      true,
		"refresh_interval":    true,
	}
	return durationFields[name]
}

func normalizeDuration(v any) string {
	switch d := v.(type) {
	case string:
		// Parse and reformat for consistency
		if dur, err := time.ParseDuration(d); err == nil {
			return dur.String()
		}
		return d
	case int:
		return fmt.Sprintf("%ds", d)
	case int64:
		return fmt.Sprintf("%ds", d)
	case float64:
		return fmt.Sprintf("%gs", d)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func compareDurations(a, b string) int {
	durA, errA := time.ParseDuration(a)
	durB, errB := time.ParseDuration(b)

	if errA != nil || errB != nil {
		return 0
	}

	if durA < durB {
		return -1
	}
	if durA > durB {
		return 1
	}
	return 0
}

func appendLists(base, overlay any) any {
	baseList, baseOk := base.([]any)
	overlayList, overlayOk := overlay.([]any)

	if !baseOk {
		return overlay
	}
	if !overlayOk {
		return base
	}

	result := make([]any, 0, len(baseList)+len(overlayList))
	result = append(result, baseList...)
	result = append(result, overlayList...)
	return result
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

func diffMaps(old, new any, path string) ([]plugin.Change, error) {
	var changes []plugin.Change

	oldMap, oldOk := old.(map[string]any)
	newMap, newOk := new.(map[string]any)

	if !oldOk && !newOk {
		if old != new {
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

	// Both are maps - compare keys
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

// Helper types for Prometheus configuration

// ScrapeConfig represents a Prometheus scrape configuration.
type ScrapeConfig struct {
	JobName        string            `yaml:"job_name"`
	ScrapeInterval string            `yaml:"scrape_interval,omitempty"`
	ScrapeTimeout  string            `yaml:"scrape_timeout,omitempty"`
	MetricsPath    string            `yaml:"metrics_path,omitempty"`
	Scheme         string            `yaml:"scheme,omitempty"`
	StaticConfigs  []StaticConfig    `yaml:"static_configs,omitempty"`
	RelabelConfigs []RelabelConfig   `yaml:"relabel_configs,omitempty"`
	Params         map[string]string `yaml:"params,omitempty"`
}

// StaticConfig represents a static target configuration.
type StaticConfig struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels,omitempty"`
}

// RelabelConfig represents a relabeling configuration.
type RelabelConfig struct {
	SourceLabels []string `yaml:"source_labels,omitempty"`
	Separator    string   `yaml:"separator,omitempty"`
	TargetLabel  string   `yaml:"target_label,omitempty"`
	Regex        string   `yaml:"regex,omitempty"`
	Modulus      uint64   `yaml:"modulus,omitempty"`
	Replacement  string   `yaml:"replacement,omitempty"`
	Action       string   `yaml:"action,omitempty"`
}

// GlobalConfig represents the global Prometheus configuration.
type GlobalConfig struct {
	ScrapeInterval     string            `yaml:"scrape_interval,omitempty"`
	ScrapeTimeout      string            `yaml:"scrape_timeout,omitempty"`
	EvaluationInterval string            `yaml:"evaluation_interval,omitempty"`
	ExternalLabels     map[string]string `yaml:"external_labels,omitempty"`
}
