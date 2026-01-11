// SPDX-License-Identifier: MIT

// Package consul provides a HashiCorp Consul configuration management plugin.
package consul

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Consul.
type Plugin struct{}

// New creates a new Consul plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "consul"
}

// Version returns the supported Consul version range.
func (p *Plugin) Version() string {
	return ">=1.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "HashiCorp Consul service mesh configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "hcl"
}

// Schema returns the configuration schema for Consul.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "root",
				Required:    false,
				Multiple:    false,
				Description: "Root configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "datacenter", Type: "string", Description: "Datacenter name"},
					{Name: "data_dir", Type: "string", Required: true, Description: "Data directory path"},
					{Name: "log_level", Type: "string", ValidValues: []string{"TRACE", "DEBUG", "INFO", "WARN", "ERR"}, Description: "Log level"},
					{Name: "log_file", Type: "string", Description: "Log file path"},
					{Name: "node_name", Type: "string", Description: "Node name"},
					{Name: "server", Type: "bool", Description: "Run as server"},
					{Name: "bootstrap", Type: "bool", Description: "Bootstrap mode"},
					{Name: "bootstrap_expect", Type: "int", Description: "Expected number of servers"},
					{Name: "bind_addr", Type: "string", Description: "Bind address"},
					{Name: "client_addr", Type: "string", Description: "Client address"},
					{Name: "advertise_addr", Type: "string", Description: "Advertise address"},
					{Name: "encrypt", Type: "string", Description: "Gossip encryption key"},
					{Name: "retry_join", Type: "string", Multiple: true, Description: "Addresses to join"},
					{Name: "retry_interval", Type: "duration", Description: "Retry interval"},
					{Name: "rejoin_after_leave", Type: "bool", Description: "Rejoin after leave"},
					{Name: "leave_on_terminate", Type: "bool", Description: "Leave on terminate"},
					{Name: "skip_leave_on_interrupt", Type: "bool", Description: "Skip leave on interrupt"},
					{Name: "enable_syslog", Type: "bool", Description: "Enable syslog"},
					{Name: "syslog_facility", Type: "string", Description: "Syslog facility"},
					{Name: "disable_update_check", Type: "bool", Description: "Disable update check"},
					{Name: "disable_anonymous_signature", Type: "bool", Description: "Disable anonymous signature"},
				},
			},
			{
				Name:        "ports",
				Required:    false,
				Multiple:    false,
				Description: "Port configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "dns", Type: "int", Description: "DNS port (default: 8600)"},
					{Name: "http", Type: "int", Description: "HTTP port (default: 8500)"},
					{Name: "https", Type: "int", Description: "HTTPS port (default: -1)"},
					{Name: "grpc", Type: "int", Description: "gRPC port (default: -1)"},
					{Name: "grpc_tls", Type: "int", Description: "gRPC TLS port (default: -1)"},
					{Name: "serf_lan", Type: "int", Description: "Serf LAN port (default: 8301)"},
					{Name: "serf_wan", Type: "int", Description: "Serf WAN port (default: 8302)"},
					{Name: "server", Type: "int", Description: "Server RPC port (default: 8300)"},
				},
			},
			{
				Name:        "ui_config",
				Required:    false,
				Multiple:    false,
				Description: "UI configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "enabled", Type: "bool", Description: "Enable UI"},
					{Name: "dir", Type: "string", Description: "UI directory"},
					{Name: "content_path", Type: "string", Description: "Content path prefix"},
				},
			},
			{
				Name:        "connect",
				Required:    false,
				Multiple:    false,
				Description: "Consul Connect configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "enabled", Type: "bool", Description: "Enable Connect"},
					{Name: "ca_provider", Type: "string", ValidValues: []string{"consul", "vault", "aws-pca"}, Description: "CA provider"},
				},
			},
			{
				Name:        "acl",
				Required:    false,
				Multiple:    false,
				Description: "ACL configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "enabled", Type: "bool", Description: "Enable ACLs"},
					{Name: "default_policy", Type: "string", ValidValues: []string{"allow", "deny"}, Description: "Default policy"},
					{Name: "down_policy", Type: "string", ValidValues: []string{"allow", "deny", "extend-cache", "async-cache"}, Description: "Down policy"},
					{Name: "enable_token_persistence", Type: "bool", Description: "Enable token persistence"},
				},
				Subsections: []plugin.SectionSchema{
					{
						Name:        "tokens",
						Required:    false,
						Multiple:    false,
						Description: "ACL tokens",
						Directives: []plugin.DirectiveSchema{
							{Name: "initial_management", Type: "string", Description: "Initial management token"},
							{Name: "agent", Type: "string", Description: "Agent token"},
							{Name: "agent_recovery", Type: "string", Description: "Agent recovery token"},
							{Name: "default", Type: "string", Description: "Default token"},
							{Name: "replication", Type: "string", Description: "Replication token"},
						},
					},
				},
			},
			{
				Name:        "tls",
				Required:    false,
				Multiple:    false,
				Description: "TLS configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "defaults", Type: "object", Description: "Default TLS settings"},
					{Name: "internal_rpc", Type: "object", Description: "Internal RPC TLS settings"},
					{Name: "https", Type: "object", Description: "HTTPS TLS settings"},
					{Name: "grpc", Type: "object", Description: "gRPC TLS settings"},
				},
			},
			{
				Name:        "performance",
				Required:    false,
				Multiple:    false,
				Description: "Performance tuning",
				Directives: []plugin.DirectiveSchema{
					{Name: "raft_multiplier", Type: "int", Description: "Raft multiplier (1-10)"},
					{Name: "leave_drain_time", Type: "duration", Description: "Leave drain time"},
					{Name: "rpc_hold_timeout", Type: "duration", Description: "RPC hold timeout"},
				},
			},
			{
				Name:        "autopilot",
				Required:    false,
				Multiple:    false,
				Description: "Autopilot configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "cleanup_dead_servers", Type: "bool", Description: "Cleanup dead servers"},
					{Name: "last_contact_threshold", Type: "duration", Description: "Last contact threshold"},
					{Name: "max_trailing_logs", Type: "int", Description: "Max trailing logs"},
					{Name: "min_quorum", Type: "int", Description: "Minimum quorum"},
					{Name: "server_stabilization_time", Type: "duration", Description: "Server stabilization time"},
				},
			},
			{
				Name:        "telemetry",
				Required:    false,
				Multiple:    false,
				Description: "Telemetry configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "disable_hostname", Type: "bool", Description: "Disable hostname in metrics"},
					{Name: "prometheus_retention_time", Type: "duration", Description: "Prometheus retention time"},
					{Name: "statsd_address", Type: "string", Description: "StatsD address"},
					{Name: "statsite_address", Type: "string", Description: "Statsite address"},
					{Name: "dogstatsd_addr", Type: "string", Description: "DogStatsD address"},
					{Name: "dogstatsd_tags", Type: "string", Multiple: true, Description: "DogStatsD tags"},
				},
			},
		},
	}
}

// DefaultConfig returns sensible defaults for Consul.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"datacenter":         "dc1",
		"data_dir":           "/opt/consul",
		"log_level":          "INFO",
		"server":             false,
		"bind_addr":          "0.0.0.0",
		"client_addr":        "127.0.0.1",
		"leave_on_terminate": true,
		"ui_config": map[string]any{
			"enabled": false,
		},
	}
}

// Validate validates the Consul configuration.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Check required fields
	if _, ok := configMap["data_dir"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "data_dir",
			Message: "data_dir is required",
		})
	}

	// Validate log_level
	if logLevel, ok := configMap["log_level"].(string); ok {
		validLevels := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERR"}
		valid := false
		for _, v := range validLevels {
			if strings.EqualFold(logLevel, v) {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, plugin.ValidationError{
				Path:    "log_level",
				Message: fmt.Sprintf("invalid log_level: %s (must be one of: %s)", logLevel, strings.Join(validLevels, ", ")),
			})
		}
	}

	// Validate bootstrap_expect
	if bootstrapExpect, ok := configMap["bootstrap_expect"]; ok {
		var be int
		switch v := bootstrapExpect.(type) {
		case int:
			be = v
		case int64:
			be = int(v)
		case float64:
			be = int(v)
		}
		if be > 0 && be%2 == 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "bootstrap_expect",
				Message: fmt.Sprintf("bootstrap_expect should be an odd number for quorum, got: %d", be),
			})
		}
	}

	// Validate ports
	if ports, ok := configMap["ports"].(map[string]any); ok {
		for name, port := range ports {
			var portNum int
			switch v := port.(type) {
			case int:
				portNum = v
			case int64:
				portNum = int(v)
			case float64:
				portNum = int(v)
			default:
				continue
			}
			if portNum > 0 && (portNum < 1 || portNum > 65535) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("ports.%s", name),
					Message: fmt.Sprintf("invalid port number: %d (must be 1-65535)", portNum),
				})
			}
		}
	}

	return errors, nil
}

// ValidateSemantic performs Consul-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	isServer := false
	if server, ok := configMap["server"].(bool); ok {
		isServer = server
	}

	// Server-specific validations
	if isServer {
		// Server should have bootstrap or bootstrap_expect
		hasBootstrap := false
		if bootstrap, ok := configMap["bootstrap"].(bool); ok && bootstrap {
			hasBootstrap = true
		}
		if bootstrapExpect, ok := configMap["bootstrap_expect"]; ok {
			var be int
			switch v := bootstrapExpect.(type) {
			case int:
				be = v
			case int64:
				be = int(v)
			case float64:
				be = int(v)
			}
			if be > 0 {
				hasBootstrap = true
			}
		}
		if !hasBootstrap {
			errors = append(errors, plugin.ValidationError{
				Path:    "server",
				Message: "server mode requires either 'bootstrap: true' or 'bootstrap_expect' to be set",
			})
		}
	}

	// Validate retry_join addresses
	if retryJoin, ok := configMap["retry_join"].([]any); ok {
		for i, addr := range retryJoin {
			addrStr, ok := addr.(string)
			if !ok {
				continue
			}
			if err := validateJoinAddress(addrStr); err != nil {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("retry_join[%d]", i),
					Message: err.Error(),
				})
			}
		}
	}

	// Validate bind_addr
	if bindAddr, ok := configMap["bind_addr"].(string); ok {
		if err := validateAddress(bindAddr); err != nil {
			errors = append(errors, plugin.ValidationError{
				Path:    "bind_addr",
				Message: err.Error(),
			})
		}
	}

	// Validate client_addr
	if clientAddr, ok := configMap["client_addr"].(string); ok {
		if err := validateAddress(clientAddr); err != nil {
			errors = append(errors, plugin.ValidationError{
				Path:    "client_addr",
				Message: err.Error(),
			})
		}
	}

	// ACL validations
	if acl, ok := configMap["acl"].(map[string]any); ok {
		if enabled, ok := acl["enabled"].(bool); ok && enabled {
			// If ACLs are enabled, default_policy should be set
			if _, ok := acl["default_policy"]; !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    "acl",
					Message: "when ACLs are enabled, 'default_policy' should be explicitly set",
				})
			}
		}
	}

	// Connect validations
	if connect, ok := configMap["connect"].(map[string]any); ok {
		if enabled, ok := connect["enabled"].(bool); ok && enabled {
			// Connect requires server mode or client with configured servers
			if !isServer {
				if _, hasRetryJoin := configMap["retry_join"]; !hasRetryJoin {
					errors = append(errors, plugin.ValidationError{
						Path:    "connect",
						Message: "Connect enabled on client requires 'retry_join' to connect to servers",
					})
				}
			}
		}
	}

	return errors, nil
}

// validateJoinAddress validates a retry_join address.
func validateJoinAddress(addr string) error {
	// Could be IP, IP:port, hostname, hostname:port, or cloud auto-join
	if strings.HasPrefix(addr, "provider=") {
		// Cloud auto-join format
		return nil
	}

	// Check for Go template (used in Consul for dynamic addresses)
	if strings.Contains(addr, "{{") {
		return nil
	}

	// Try to parse as host:port
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// No port, just validate the host
		host = addr
	}

	// Validate IP or hostname
	if ip := net.ParseIP(host); ip == nil {
		// Not an IP, check if it's a valid hostname
		if !isValidHostname(host) {
			return fmt.Errorf("invalid address: %s", addr)
		}
	}

	return nil
}

// validateAddress validates a bind/client address.
func validateAddress(addr string) error {
	// Could be IP, 0.0.0.0, or Go template
	if strings.Contains(addr, "{{") {
		return nil
	}

	if ip := net.ParseIP(addr); ip == nil {
		return fmt.Errorf("invalid IP address: %s", addr)
	}

	return nil
}

// isValidHostname checks if a string is a valid hostname.
func isValidHostname(hostname string) bool {
	if len(hostname) > 253 {
		return false
	}
	// Simple hostname validation
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)
	return hostnameRegex.MatchString(hostname)
}

// Normalize normalizes the Consul configuration to canonical form.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Ensure consistent casing for log_level
	if logLevel, ok := configMap["log_level"].(string); ok {
		configMap["log_level"] = strings.ToUpper(logLevel)
	}

	return configMap, nil
}

// ToNative converts the configuration to native Consul HCL format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	// Consul accepts JSON format, which is simpler to generate
	// and fully compatible with HCL
	return json.MarshalIndent(config, "", "  ")
}

// FromNative parses native Consul configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	// Try JSON first
	var config any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Consul config: %w", err)
	}
	return config, nil
}

// Merge merges two Consul configurations.
func (p *Plugin) Merge(base, overlay any) (any, error) {
	baseMap, ok := base.(map[string]any)
	if !ok {
		return overlay, nil
	}

	overlayMap, ok := overlay.(map[string]any)
	if !ok {
		return base, nil
	}

	return deepMerge(baseMap, overlayMap), nil
}

// Diff computes the differences between two configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	var changes []plugin.Change

	oldMap, _ := old.(map[string]any)
	newMap, _ := new.(map[string]any)

	changes = diffMaps("", oldMap, newMap, changes)

	return changes, nil
}

// diffMaps recursively diffs two maps.
func diffMaps(prefix string, old, new map[string]any, changes []plugin.Change) []plugin.Change {
	// Check for removed and modified keys
	for k, oldVal := range old {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		newVal, exists := new[k]
		if !exists {
			changes = append(changes, plugin.Change{
				Path:     path,
				Type:     "remove",
				OldValue: oldVal,
			})
			continue
		}

		// Check if both are maps for recursive diff
		oldMap, oldIsMap := oldVal.(map[string]any)
		newMap, newIsMap := newVal.(map[string]any)
		if oldIsMap && newIsMap {
			changes = diffMaps(path, oldMap, newMap, changes)
			continue
		}

		// Compare values
		if !equalValues(oldVal, newVal) {
			changes = append(changes, plugin.Change{
				Path:     path,
				Type:     "modify",
				OldValue: oldVal,
				NewValue: newVal,
			})
		}
	}

	// Check for added keys
	for k, newVal := range new {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		if _, exists := old[k]; !exists {
			changes = append(changes, plugin.Change{
				Path:     path,
				Type:     "add",
				NewValue: newVal,
			})
		}
	}

	return changes
}

// equalValues compares two values for equality.
func equalValues(a, b any) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// deepMerge performs a deep merge of two maps.
func deepMerge(base, overlay map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Merge overlay
	for k, v := range overlay {
		if baseVal, exists := result[k]; exists {
			// Both have this key
			baseMap, baseIsMap := baseVal.(map[string]any)
			overlayMap, overlayIsMap := v.(map[string]any)
			if baseIsMap && overlayIsMap {
				result[k] = deepMerge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = v
	}

	return result
}

// Ensure Plugin implements AppPlugin interface.
var _ plugin.AppPlugin = (*Plugin)(nil)
