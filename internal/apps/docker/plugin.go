// SPDX-License-Identifier: MIT

// Package docker provides a Docker daemon configuration management plugin.
package docker

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sort"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Docker.
type Plugin struct{}

// New creates a new Docker plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "docker"
}

// Version returns the supported Docker version range.
func (p *Plugin) Version() string {
	return ">=18.09"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Docker daemon configuration management (daemon.json)"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "json"
}

// Schema returns the configuration schema for Docker daemon.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "storage-driver",
				Required:    false,
				Description: "Storage driver to use",
				Directives: []plugin.DirectiveSchema{
					{Name: "storage-driver", Type: "string", ValidValues: []string{"overlay2", "fuse-overlayfs", "btrfs", "zfs", "vfs"}, Description: "Storage driver"},
					{Name: "storage-opts", Type: "list", Description: "Storage driver options"},
				},
			},
			{
				Name:        "log-driver",
				Required:    false,
				Description: "Default logging driver",
				Directives: []plugin.DirectiveSchema{
					{Name: "log-driver", Type: "string", ValidValues: []string{"json-file", "syslog", "journald", "gelf", "fluentd", "awslogs", "splunk", "local", "none"}, Description: "Logging driver"},
					{Name: "log-opts", Type: "map", Description: "Logging driver options"},
				},
			},
			{
				Name:        "registry-mirrors",
				Required:    false,
				Description: "Registry mirror URLs",
				Directives: []plugin.DirectiveSchema{
					{Name: "registry-mirrors", Type: "list", Description: "List of registry mirror URLs"},
				},
			},
			{
				Name:        "insecure-registries",
				Required:    false,
				Description: "Insecure registry addresses",
				Directives: []plugin.DirectiveSchema{
					{Name: "insecure-registries", Type: "list", Description: "List of insecure registry addresses"},
				},
			},
			{
				Name:        "exec-opts",
				Required:    false,
				Description: "Runtime execution options",
				Directives: []plugin.DirectiveSchema{
					{Name: "exec-opts", Type: "list", Description: "Execution options"},
				},
			},
			{
				Name:        "dns",
				Required:    false,
				Description: "DNS server addresses",
			},
			{
				Name:        "data-root",
				Required:    false,
				Description: "Root directory for Docker state",
			},
			{
				Name:        "live-restore",
				Required:    false,
				Description: "Enable live restore of containers",
			},
		},
	}
}

// DefaultConfig returns sensible default Docker daemon configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"storage-driver": "overlay2",
		"log-driver":     "json-file",
		"log-opts": map[string]any{
			"max-size": "10m",
			"max-file": "3",
		},
		"live-restore": true,
	}
}

// Validate validates the Docker daemon configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "", Message: "configuration must be a map"}}, nil
	}

	// Validate storage-driver
	if driver, ok := configMap["storage-driver"]; ok {
		validDrivers := []string{"overlay2", "fuse-overlayfs", "btrfs", "zfs", "vfs", "devicemapper", "aufs"}
		driverStr := fmt.Sprintf("%v", driver)
		found := false
		for _, d := range validDrivers {
			if d == driverStr {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "storage-driver",
				Message: fmt.Sprintf("invalid storage driver: %s (supported: %v)", driverStr, validDrivers),
			})
		}
	}

	// Validate log-driver
	if driver, ok := configMap["log-driver"]; ok {
		validDrivers := []string{"json-file", "syslog", "journald", "gelf", "fluentd", "awslogs", "splunk", "local", "none", "gcplogs", "etwlogs"}
		driverStr := fmt.Sprintf("%v", driver)
		found := false
		for _, d := range validDrivers {
			if d == driverStr {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "log-driver",
				Message: fmt.Sprintf("invalid log driver: %s (supported: %v)", driverStr, validDrivers),
			})
		}
	}

	// Validate registry-mirrors
	if mirrors, ok := configMap["registry-mirrors"]; ok {
		mirrorErrors := p.validateURLList(mirrors, "registry-mirrors")
		errors = append(errors, mirrorErrors...)
	}

	// Validate insecure-registries
	if registries, ok := configMap["insecure-registries"]; ok {
		registryErrors := p.validateRegistryList(registries, "insecure-registries")
		errors = append(errors, registryErrors...)
	}

	// Validate dns
	if dns, ok := configMap["dns"]; ok {
		dnsErrors := p.validateIPList(dns, "dns")
		errors = append(errors, dnsErrors...)
	}

	// Validate bip (bridge IP)
	if bip, ok := configMap["bip"]; ok {
		bipStr := fmt.Sprintf("%v", bip)
		if !isValidCIDR(bipStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    "bip",
				Message: fmt.Sprintf("invalid CIDR notation: %s", bipStr),
			})
		}
	}

	// Validate default-address-pools
	if pools, ok := configMap["default-address-pools"]; ok {
		poolErrors := p.validateAddressPools(pools)
		errors = append(errors, poolErrors...)
	}

	return errors, nil
}

func (p *Plugin) validateURLList(urls any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	urlList, ok := urls.([]any)
	if !ok {
		if urlStrList, ok := urls.([]string); ok {
			for i, u := range urlStrList {
				if !isValidURL(u) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("%s[%d]", path, i),
						Message: fmt.Sprintf("invalid URL: %s", u),
					})
				}
			}
			return errors
		}
		return []plugin.ValidationError{{Path: path, Message: "must be a list"}}
	}

	for i, u := range urlList {
		urlStr := fmt.Sprintf("%v", u)
		if !isValidURL(urlStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s[%d]", path, i),
				Message: fmt.Sprintf("invalid URL: %s", urlStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateRegistryList(registries any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	registryList, ok := registries.([]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "must be a list"}}
	}

	for i, r := range registryList {
		registryStr := fmt.Sprintf("%v", r)
		if !isValidRegistry(registryStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s[%d]", path, i),
				Message: fmt.Sprintf("invalid registry address: %s", registryStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateIPList(ips any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	ipList, ok := ips.([]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "must be a list"}}
	}

	for i, ip := range ipList {
		ipStr := fmt.Sprintf("%v", ip)
		if net.ParseIP(ipStr) == nil {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s[%d]", path, i),
				Message: fmt.Sprintf("invalid IP address: %s", ipStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateAddressPools(pools any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	poolList, ok := pools.([]any)
	if !ok {
		return []plugin.ValidationError{{Path: "default-address-pools", Message: "must be a list"}}
	}

	for i, pool := range poolList {
		path := fmt.Sprintf("default-address-pools[%d]", i)

		poolMap, ok := pool.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "pool must be a map with base and size",
			})
			continue
		}

		// Validate base
		if base, ok := poolMap["base"]; ok {
			baseStr := fmt.Sprintf("%v", base)
			if !isValidCIDR(baseStr) {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".base",
					Message: fmt.Sprintf("invalid CIDR notation: %s", baseStr),
				})
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "base is required",
			})
		}

		// Validate size
		if size, ok := poolMap["size"]; ok {
			switch s := size.(type) {
			case int:
				if s < 1 || s > 32 {
					errors = append(errors, plugin.ValidationError{
						Path:    path + ".size",
						Message: "size must be between 1 and 32",
					})
				}
			case float64:
				if s < 1 || s > 32 {
					errors = append(errors, plugin.ValidationError{
						Path:    path + ".size",
						Message: "size must be between 1 and 32",
					})
				}
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    path,
				Message: "size is required",
			})
		}
	}

	return errors
}

// ValidateSemantic performs Docker-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return errors, nil
	}

	// Warn about insecure registries
	if registries, ok := configMap["insecure-registries"].([]any); ok {
		if len(registries) > 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "insecure-registries",
				Message: fmt.Sprintf("insecure registries configured - traffic will not be encrypted: %v", registries),
			})
		}
	}

	// Warn about deprecated storage drivers
	if driver, ok := configMap["storage-driver"]; ok {
		driverStr := fmt.Sprintf("%v", driver)
		deprecated := []string{"devicemapper", "aufs"}
		for _, d := range deprecated {
			if d == driverStr {
				errors = append(errors, plugin.ValidationError{
					Path:    "storage-driver",
					Message: fmt.Sprintf("storage driver '%s' is deprecated, consider using overlay2", driverStr),
				})
			}
		}
	}

	// Warn about live-restore disabled
	if liveRestore, ok := configMap["live-restore"]; ok {
		if disabled, ok := liveRestore.(bool); ok && !disabled {
			errors = append(errors, plugin.ValidationError{
				Path:    "live-restore",
				Message: "live-restore is disabled - containers will stop when daemon restarts",
			})
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

	// Copy with canonical key order (most common options first)
	keyOrder := []string{
		"data-root",
		"storage-driver",
		"storage-opts",
		"log-driver",
		"log-opts",
		"registry-mirrors",
		"insecure-registries",
		"dns",
		"dns-opts",
		"dns-search",
		"bip",
		"default-address-pools",
		"exec-opts",
		"live-restore",
		"userland-proxy",
		"default-runtime",
		"runtimes",
	}

	for _, k := range keyOrder {
		if v, ok := configMap[k]; ok {
			result[k] = v
		}
	}

	// Copy any remaining keys
	for k, v := range configMap {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result, nil
}

// ToNative converts the configuration to Docker daemon.json format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// FromNative parses Docker daemon.json format into configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Docker config: %w", err)
	}
	return result, nil
}

// Merge merges two Docker daemon configurations.
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
		case "registry-mirrors", "insecure-registries", "dns", "dns-opts", "dns-search", "exec-opts", "storage-opts":
			// Append lists
			result[k] = appendLists(result[k], v)

		case "log-opts", "runtimes", "features":
			// Deep merge maps
			if existing, ok := result[k]; ok {
				merged, _ := mergeDeep(existing, v)
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

// Diff detects changes between two Docker daemon configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	return diffMaps(old, new, "")
}

// Helper functions

var urlRegex = regexp.MustCompile(`^https?://[^\s]+$`)

func isValidURL(s string) bool {
	return urlRegex.MatchString(s)
}

var registryRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*(:\d+)?(/.*)?$`)

func isValidRegistry(s string) bool {
	return registryRegex.MatchString(s)
}

func isValidCIDR(s string) bool {
	_, _, err := net.ParseCIDR(s)
	return err == nil
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

	// Deduplicate
	seen := make(map[string]bool)
	result := make([]any, 0, len(baseList)+len(overlayList))

	for _, v := range baseList {
		key := fmt.Sprintf("%v", v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}

	for _, v := range overlayList {
		key := fmt.Sprintf("%v", v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}

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
		if existing, ok := result[k]; ok {
			if _, isMap := existing.(map[string]any); isMap {
				if _, vIsMap := v.(map[string]any); vIsMap {
					merged, _ := mergeDeep(existing, v)
					result[k] = merged
					continue
				}
			}
		}
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
