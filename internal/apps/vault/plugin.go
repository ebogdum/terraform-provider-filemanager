// SPDX-License-Identifier: MIT

// Package vault provides a HashiCorp Vault configuration management plugin.
package vault

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Vault.
type Plugin struct{}

// New creates a new Vault plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "vault"
}

// Version returns the supported Vault version range.
func (p *Plugin) Version() string {
	return ">=1.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "HashiCorp Vault server configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "hcl"
}

// Schema returns the configuration schema for Vault.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "storage",
				Required:    true,
				Description: "Storage backend configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "type", Type: "string", Required: true, Description: "Storage backend type (file, raft, consul, etc.)"},
					{Name: "path", Type: "string", Description: "Storage path (for file/raft backends)"},
					{Name: "address", Type: "string", Description: "Storage address (for consul/etcd backends)"},
					{Name: "node_id", Type: "string", Description: "Raft node ID"},
				},
			},
			{
				Name:        "listener",
				Required:    true,
				Multiple:    true,
				Description: "Network listener configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "type", Type: "string", Required: true, ValidValues: []string{"tcp", "unix"}, Description: "Listener type"},
					{Name: "address", Type: "string", Required: true, Description: "Listen address (e.g., 0.0.0.0:8200)"},
					{Name: "tls_disable", Type: "bool", Default: false, Description: "Disable TLS"},
					{Name: "tls_cert_file", Type: "string", Description: "Path to TLS certificate"},
					{Name: "tls_key_file", Type: "string", Description: "Path to TLS private key"},
					{Name: "tls_min_version", Type: "string", Default: "tls12", ValidValues: []string{"tls10", "tls11", "tls12", "tls13"}},
				},
			},
			{
				Name:        "seal",
				Required:    false,
				Description: "Auto-unseal configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "type", Type: "string", Description: "Seal type (awskms, azurekeyvault, gcpckms, etc.)"},
					{Name: "region", Type: "string", Description: "Cloud region"},
					{Name: "kms_key_id", Type: "string", Description: "KMS key ID"},
				},
			},
			{
				Name:        "telemetry",
				Required:    false,
				Description: "Telemetry configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "disable_hostname", Type: "bool", Default: false},
					{Name: "prometheus_retention_time", Type: "string", Default: "24h"},
					{Name: "statsite_address", Type: "string"},
					{Name: "statsd_address", Type: "string"},
				},
			},
			{
				Name:        "api_addr",
				Required:    false,
				Description: "API address for client redirects",
			},
			{
				Name:        "cluster_addr",
				Required:    false,
				Description: "Cluster address for node communication",
			},
			{
				Name:        "ui",
				Required:    false,
				Description: "Enable UI",
			},
			{
				Name:        "log_level",
				Required:    false,
				Description: "Log level (trace, debug, info, warn, error)",
			},
			{
				Name:        "disable_mlock",
				Required:    false,
				Description: "Disable mlock (not recommended for production)",
			},
		},
	}
}

// DefaultConfig returns sensible default Vault configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"storage": map[string]any{
			"raft": map[string]any{
				"path": "/opt/vault/data",
			},
		},
		"listener": []any{
			map[string]any{
				"tcp": map[string]any{
					"address":     "0.0.0.0:8200",
					"tls_disable": false,
				},
			},
		},
		"ui":        true,
		"log_level": "info",
	}
}

// Validate validates the Vault configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "", Message: "configuration must be a map"}}, nil
	}

	// Validate storage (required)
	if _, ok := configMap["storage"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "storage",
			Message: "storage backend configuration is required",
		})
	} else {
		storageErrors := p.validateStorage(configMap["storage"])
		errors = append(errors, storageErrors...)
	}

	// Validate listener (required)
	if _, ok := configMap["listener"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "listener",
			Message: "at least one listener is required",
		})
	} else {
		listenerErrors := p.validateListeners(configMap["listener"])
		errors = append(errors, listenerErrors...)
	}

	// Validate seal if present
	if seal, ok := configMap["seal"]; ok {
		sealErrors := p.validateSeal(seal)
		errors = append(errors, sealErrors...)
	}

	// Validate log_level if present
	if logLevel, ok := configMap["log_level"]; ok {
		validLevels := []string{"trace", "debug", "info", "warn", "error"}
		levelStr := strings.ToLower(fmt.Sprintf("%v", logLevel))
		found := false
		for _, l := range validLevels {
			if l == levelStr {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "log_level",
				Message: fmt.Sprintf("invalid log_level: %s (must be one of: %v)", logLevel, validLevels),
			})
		}
	}

	return errors, nil
}

func (p *Plugin) validateStorage(storage any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	storageMap, ok := storage.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "storage", Message: "storage must be a map"}}
	}

	// Check that exactly one storage backend is configured
	validBackends := []string{"file", "raft", "consul", "etcd", "s3", "azure", "gcs", "inmem"}
	foundBackends := 0
	for k := range storageMap {
		for _, backend := range validBackends {
			if k == backend {
				foundBackends++
			}
		}
	}

	if foundBackends == 0 {
		errors = append(errors, plugin.ValidationError{
			Path:    "storage",
			Message: fmt.Sprintf("storage backend must be one of: %v", validBackends),
		})
	} else if foundBackends > 1 {
		errors = append(errors, plugin.ValidationError{
			Path:    "storage",
			Message: "only one storage backend can be configured",
		})
	}

	// Validate raft storage specifics
	if raft, ok := storageMap["raft"].(map[string]any); ok {
		if _, ok := raft["path"]; !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    "storage.raft.path",
				Message: "path is required for raft storage",
			})
		}
	}

	// Validate consul storage specifics
	if consul, ok := storageMap["consul"].(map[string]any); ok {
		if _, ok := consul["address"]; !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    "storage.consul.address",
				Message: "address is required for consul storage",
			})
		}
	}

	return errors
}

func (p *Plugin) validateListeners(listener any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	listeners, ok := listener.([]any)
	if !ok {
		// Could be a single listener map
		if listenerMap, ok := listener.(map[string]any); ok {
			return p.validateSingleListener(listenerMap, "listener")
		}
		return []plugin.ValidationError{{Path: "listener", Message: "listener must be a list or map"}}
	}

	if len(listeners) == 0 {
		return []plugin.ValidationError{{Path: "listener", Message: "at least one listener is required"}}
	}

	for i, l := range listeners {
		listenerMap, ok := l.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("listener[%d]", i),
				Message: "listener must be a map",
			})
			continue
		}
		listenerErrors := p.validateSingleListener(listenerMap, fmt.Sprintf("listener[%d]", i))
		errors = append(errors, listenerErrors...)
	}

	return errors
}

func (p *Plugin) validateSingleListener(listener map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Check for tcp or unix listener
	tcp, hasTCP := listener["tcp"].(map[string]any)
	unix, hasUnix := listener["unix"].(map[string]any)

	if !hasTCP && !hasUnix {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: "listener must specify tcp or unix type",
		})
		return errors
	}

	if hasTCP {
		// Validate TCP listener
		if address, ok := tcp["address"]; ok {
			addrStr := fmt.Sprintf("%v", address)
			if !isValidAddress(addrStr) {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".tcp.address",
					Message: fmt.Sprintf("invalid address format: %s", addrStr),
				})
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    path + ".tcp.address",
				Message: "address is required for tcp listener",
			})
		}

		// Check TLS configuration
		tlsDisable, _ := tcp["tls_disable"].(bool)
		if !tlsDisable {
			if _, ok := tcp["tls_cert_file"]; !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".tcp.tls_cert_file",
					Message: "tls_cert_file is required when TLS is enabled",
				})
			}
			if _, ok := tcp["tls_key_file"]; !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".tcp.tls_key_file",
					Message: "tls_key_file is required when TLS is enabled",
				})
			}
		}
	}

	if hasUnix {
		// Validate Unix socket listener
		if _, ok := unix["address"]; !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    path + ".unix.address",
				Message: "address (socket path) is required for unix listener",
			})
		}
	}

	return errors
}

func (p *Plugin) validateSeal(seal any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	sealMap, ok := seal.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "seal", Message: "seal must be a map"}}
	}

	validSealTypes := []string{"awskms", "azurekeyvault", "gcpckms", "ocikms", "pkcs11", "transit", "shamir"}

	foundSeal := false
	for k := range sealMap {
		for _, t := range validSealTypes {
			if k == t {
				foundSeal = true
				break
			}
		}
	}

	if !foundSeal {
		errors = append(errors, plugin.ValidationError{
			Path:    "seal",
			Message: fmt.Sprintf("seal type must be one of: %v", validSealTypes),
		})
	}

	return errors
}

// ValidateSemantic performs Vault-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return errors, nil
	}

	// Warn about TLS disabled
	if listener, ok := configMap["listener"]; ok {
		if listeners, ok := listener.([]any); ok {
			for i, l := range listeners {
				if lMap, ok := l.(map[string]any); ok {
					if tcp, ok := lMap["tcp"].(map[string]any); ok {
						if tlsDisable, ok := tcp["tls_disable"].(bool); ok && tlsDisable {
							errors = append(errors, plugin.ValidationError{
								Path:    fmt.Sprintf("listener[%d].tcp.tls_disable", i),
								Message: "TLS is disabled - this is not recommended for production",
							})
						}
					}
				}
			}
		}
	}

	// Warn about disable_mlock
	if disableMlock, ok := configMap["disable_mlock"]; ok {
		if disabled, ok := disableMlock.(bool); ok && disabled {
			errors = append(errors, plugin.ValidationError{
				Path:    "disable_mlock",
				Message: "mlock is disabled - sensitive data may be swapped to disk",
			})
		}
	}

	// Warn about HA without cluster_addr
	if storage, ok := configMap["storage"].(map[string]any); ok {
		if _, hasRaft := storage["raft"]; hasRaft {
			if _, hasClusterAddr := configMap["cluster_addr"]; !hasClusterAddr {
				errors = append(errors, plugin.ValidationError{
					Path:    "cluster_addr",
					Message: "cluster_addr should be set for raft storage in HA mode",
				})
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

	// Copy with canonical key order
	keyOrder := []string{"storage", "listener", "seal", "api_addr", "cluster_addr", "ui", "log_level", "disable_mlock", "telemetry"}
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

// ToNative converts the configuration to Vault native HCL/JSON format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	// Vault accepts JSON, so we output JSON which is valid HCL
	return json.MarshalIndent(config, "", "  ")
}

// FromNative parses Vault native format into configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Vault config: %w", err)
	}
	return result, nil
}

// Merge merges two Vault configurations.
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
		case "listener":
			// Append listeners
			result[k] = appendLists(result[k], v)

		case "storage", "seal", "telemetry":
			// Deep merge these sections
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

// Diff detects changes between two Vault configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	return diffMaps(old, new, "")
}

// Helper functions

var addressRegex = regexp.MustCompile(`^[^:]+:\d+$|^:\d+$`)

func isValidAddress(s string) bool {
	return addressRegex.MatchString(s)
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
