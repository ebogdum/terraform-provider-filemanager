// SPDX-License-Identifier: MIT

// Package nomad provides a plugin for HashiCorp Nomad configuration management.
package nomad

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
)

// Plugin implements the Nomad configuration plugin.
type Plugin struct{}

// New creates a new Nomad plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "nomad"
}

// Version returns the supported Nomad version range.
func (p *Plugin) Version() string {
	return ">=1.0.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "HashiCorp Nomad agent configuration management"
}

// NativeFormat returns the native configuration format.
func (p *Plugin) NativeFormat() string {
	return "json" // HCL is also supported, but JSON is easier to work with programmatically
}

// Schema returns the configuration schema.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "server",
				Description: "Server-specific configuration",
			},
			{
				Name:        "client",
				Description: "Client-specific configuration",
			},
			{
				Name:        "ports",
				Description: "Network port configuration",
			},
			{
				Name:        "addresses",
				Description: "Bind addresses configuration",
			},
			{
				Name:        "advertise",
				Description: "Advertised addresses configuration",
			},
			{
				Name:        "consul",
				Description: "Consul integration configuration",
			},
			{
				Name:        "vault",
				Description: "Vault integration configuration",
			},
			{
				Name:        "acl",
				Description: "ACL configuration",
			},
			{
				Name:        "tls",
				Description: "TLS configuration",
			},
			{
				Name:        "telemetry",
				Description: "Telemetry configuration",
			},
			{
				Name:        "autopilot",
				Description: "Autopilot configuration for servers",
			},
			{
				Name:        "plugin",
				Description: "Plugin configuration",
			},
		},
		Directives: []plugin.DirectiveSchema{
			{
				Name:        "datacenter",
				Description: "Datacenter name",
				Required:    true,
				Type:        "string",
			},
			{
				Name:        "data_dir",
				Description: "Data directory path",
				Required:    true,
				Type:        "string",
			},
			{
				Name:        "name",
				Description: "Agent name",
				Type:        "string",
			},
			{
				Name:        "region",
				Description: "Region name",
				Type:        "string",
			},
			{
				Name:        "bind_addr",
				Description: "Bind address for all services",
				Type:        "string",
			},
			{
				Name:        "log_level",
				Description: "Log level",
				Type:        "string",
			},
			{
				Name:        "log_file",
				Description: "Log file path",
				Type:        "string",
			},
			{
				Name:        "enable_debug",
				Description: "Enable debug mode",
				Type:        "bool",
			},
			{
				Name:        "leave_on_interrupt",
				Description: "Leave cluster on interrupt signal",
				Type:        "bool",
			},
			{
				Name:        "leave_on_terminate",
				Description: "Leave cluster on terminate signal",
				Type:        "bool",
			},
		},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"datacenter": "dc1",
		"data_dir":   "/opt/nomad/data",
		"log_level":  "INFO",
		"bind_addr":  "0.0.0.0",
		"ports": map[string]any{
			"http": 4646,
			"rpc":  4647,
			"serf": 4648,
		},
		"leave_on_interrupt": false,
		"leave_on_terminate": false,
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
	if _, ok := configMap["datacenter"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "datacenter",
			Message: "datacenter is required",
		})
	}

	if _, ok := configMap["data_dir"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "data_dir",
			Message: "data_dir is required",
		})
	}

	// Validate log_level if present
	if logLevel, ok := configMap["log_level"].(string); ok {
		validLevels := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}
		found := false
		for _, v := range validLevels {
			if strings.EqualFold(logLevel, v) {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "log_level",
				Message: fmt.Sprintf("log_level must be one of: %s", strings.Join(validLevels, ", ")),
			})
		}
	}

	// Validate server configuration
	if server, ok := configMap["server"].(map[string]any); ok {
		errors = append(errors, p.validateServer(server)...)
	}

	// Validate client configuration
	if client, ok := configMap["client"].(map[string]any); ok {
		errors = append(errors, p.validateClient(client)...)
	}

	// Validate ports configuration
	if ports, ok := configMap["ports"].(map[string]any); ok {
		errors = append(errors, p.validatePorts(ports)...)
	}

	// Validate addresses
	if addresses, ok := configMap["addresses"].(map[string]any); ok {
		errors = append(errors, p.validateAddresses(addresses)...)
	}

	// Validate bind_addr
	if bindAddr, ok := configMap["bind_addr"].(string); ok {
		if !isValidBindAddress(bindAddr) {
			errors = append(errors, plugin.ValidationError{
				Path:    "bind_addr",
				Message: fmt.Sprintf("invalid bind address: %s", bindAddr),
			})
		}
	}

	// Validate TLS configuration
	if tls, ok := configMap["tls"].(map[string]any); ok {
		errors = append(errors, p.validateTLS(tls)...)
	}

	// Validate ACL configuration
	if acl, ok := configMap["acl"].(map[string]any); ok {
		errors = append(errors, p.validateACL(acl)...)
	}

	return errors, nil
}

func (p *Plugin) validateServer(server map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate bootstrap_expect
	if bootstrapExpect, ok := server["bootstrap_expect"]; ok {
		var expect int
		switch v := bootstrapExpect.(type) {
		case int:
			expect = v
		case float64:
			expect = int(v)
		}
		if expect < 1 {
			errors = append(errors, plugin.ValidationError{
				Path:    "server.bootstrap_expect",
				Message: "bootstrap_expect must be at least 1",
			})
		}
		// Odd numbers recommended for consensus
		if expect > 1 && expect%2 == 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "server.bootstrap_expect",
				Message: "bootstrap_expect should be an odd number for proper consensus",
			})
		}
	}

	// Validate raft_multiplier
	if raftMult, ok := server["raft_multiplier"]; ok {
		var mult int
		switch v := raftMult.(type) {
		case int:
			mult = v
		case float64:
			mult = int(v)
		}
		if mult < 1 || mult > 10 {
			errors = append(errors, plugin.ValidationError{
				Path:    "server.raft_multiplier",
				Message: "raft_multiplier must be between 1 and 10",
			})
		}
	}

	// Validate retry_join
	if retryJoin, ok := server["retry_join"].([]any); ok {
		for i, addr := range retryJoin {
			if addrStr, ok := addr.(string); ok {
				if addrStr == "" {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("server.retry_join[%d]", i),
						Message: "retry_join address cannot be empty",
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateClient(client map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate node_class if present
	if nodeClass, ok := client["node_class"].(string); ok {
		if nodeClass == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    "client.node_class",
				Message: "node_class cannot be empty if specified",
			})
		}
	}

	// Validate servers
	if servers, ok := client["servers"].([]any); ok {
		if len(servers) == 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "client.servers",
				Message: "at least one server should be specified for client mode",
			})
		}
	}

	// Validate max_kill_timeout
	if timeout, ok := client["max_kill_timeout"].(string); ok {
		if !isValidDuration(timeout) {
			errors = append(errors, plugin.ValidationError{
				Path:    "client.max_kill_timeout",
				Message: fmt.Sprintf("invalid duration format: %s", timeout),
			})
		}
	}

	// Validate network_speed if present
	if speed, ok := client["network_speed"]; ok {
		var netSpeed int
		switch v := speed.(type) {
		case int:
			netSpeed = v
		case float64:
			netSpeed = int(v)
		}
		if netSpeed < 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "client.network_speed",
				Message: "network_speed cannot be negative",
			})
		}
	}

	return errors
}

func (p *Plugin) validatePorts(ports map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	validPorts := map[string]bool{
		"http": true,
		"rpc":  true,
		"serf": true,
	}

	for name, value := range ports {
		if !validPorts[name] {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("ports.%s", name),
				Message: fmt.Sprintf("unknown port: %s (valid ports: http, rpc, serf)", name),
			})
		}

		var port int
		switch v := value.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		}

		if port < 1 || port > 65535 {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("ports.%s", name),
				Message: fmt.Sprintf("port must be between 1 and 65535, got: %d", port),
			})
		}
	}

	return errors
}

func (p *Plugin) validateAddresses(addresses map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	validAddresses := map[string]bool{
		"http": true,
		"rpc":  true,
		"serf": true,
	}

	for name, value := range addresses {
		if !validAddresses[name] {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("addresses.%s", name),
				Message: fmt.Sprintf("unknown address type: %s", name),
			})
		}

		if addr, ok := value.(string); ok {
			if !isValidBindAddress(addr) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("addresses.%s", name),
					Message: fmt.Sprintf("invalid address: %s", addr),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateTLS(tls map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// If TLS is enabled, validate required fields
	if enabled, ok := tls["http"].(bool); ok && enabled {
		if _, ok := tls["ca_file"].(string); !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    "tls.ca_file",
				Message: "ca_file is recommended when TLS is enabled",
			})
		}
		if _, ok := tls["cert_file"].(string); !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    "tls.cert_file",
				Message: "cert_file is required when TLS is enabled",
			})
		}
		if _, ok := tls["key_file"].(string); !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    "tls.key_file",
				Message: "key_file is required when TLS is enabled",
			})
		}
	}

	// Validate verify_https_client
	if verifyClient, ok := tls["verify_https_client"].(bool); ok && verifyClient {
		if _, ok := tls["ca_file"].(string); !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    "tls.ca_file",
				Message: "ca_file is required when verify_https_client is enabled",
			})
		}
	}

	return errors
}

func (p *Plugin) validateACL(acl map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	if enabled, ok := acl["enabled"].(bool); ok && enabled {
		// Validate token_ttl if present
		if tokenTTL, ok := acl["token_ttl"].(string); ok {
			if !isValidDuration(tokenTTL) {
				errors = append(errors, plugin.ValidationError{
					Path:    "acl.token_ttl",
					Message: fmt.Sprintf("invalid duration format: %s", tokenTTL),
				})
			}
		}

		// Validate policy_ttl if present
		if policyTTL, ok := acl["policy_ttl"].(string); ok {
			if !isValidDuration(policyTTL) {
				errors = append(errors, plugin.ValidationError{
					Path:    "acl.policy_ttl",
					Message: fmt.Sprintf("invalid duration format: %s", policyTTL),
				})
			}
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

	// Check if both server and client modes are enabled
	serverEnabled := false
	clientEnabled := false

	if server, ok := configMap["server"].(map[string]any); ok {
		if enabled, ok := server["enabled"].(bool); ok && enabled {
			serverEnabled = true
		}
	}

	if client, ok := configMap["client"].(map[string]any); ok {
		if enabled, ok := client["enabled"].(bool); ok && enabled {
			clientEnabled = true
		}
	}

	// Neither mode enabled
	if !serverEnabled && !clientEnabled {
		errors = append(errors, plugin.ValidationError{
			Path:    "",
			Message: "either server.enabled or client.enabled must be true",
		})
	}

	// Check for production warnings
	// TLS should be enabled in production
	if tls, ok := configMap["tls"].(map[string]any); ok {
		httpTLS := false
		if enabled, ok := tls["http"].(bool); ok && enabled {
			httpTLS = true
		}
		if !httpTLS {
			errors = append(errors, plugin.ValidationError{
				Path:    "tls.http",
				Message: "TLS is recommended for production environments",
			})
		}
	} else {
		errors = append(errors, plugin.ValidationError{
			Path:    "tls",
			Message: "TLS configuration is recommended for production environments",
		})
	}

	// ACL should be enabled in production
	if acl, ok := configMap["acl"].(map[string]any); ok {
		if enabled, ok := acl["enabled"].(bool); !ok || !enabled {
			errors = append(errors, plugin.ValidationError{
				Path:    "acl.enabled",
				Message: "ACL is recommended for production environments",
			})
		}
	} else {
		errors = append(errors, plugin.ValidationError{
			Path:    "acl",
			Message: "ACL configuration is recommended for production environments",
		})
	}

	// Check Consul integration for service discovery
	if client, ok := configMap["client"].(map[string]any); ok {
		if enabled, ok := client["enabled"].(bool); ok && enabled {
			if _, hasConsul := configMap["consul"]; !hasConsul {
				errors = append(errors, plugin.ValidationError{
					Path:    "consul",
					Message: "Consul integration is recommended for service discovery",
				})
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

	// Normalize log_level to uppercase
	if logLevel, ok := configMap["log_level"].(string); ok {
		configMap["log_level"] = strings.ToUpper(logLevel)
	}

	return configMap, nil
}

// ToNative converts to native Nomad configuration format (JSON).
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// FromNative parses native Nomad configuration format.
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

func isValidBindAddress(addr string) bool {
	// Check for special addresses
	if addr == "0.0.0.0" || addr == "::" || addr == "localhost" {
		return true
	}

	// Check for Go template syntax (e.g., {{ GetInterfaceIP "eth0" }})
	if strings.HasPrefix(addr, "{{") && strings.HasSuffix(addr, "}}") {
		return true
	}

	// Try parsing as IP
	if ip := net.ParseIP(addr); ip != nil {
		return true
	}

	// Try parsing as hostname
	if _, err := net.LookupHost(addr); err == nil {
		return true
	}

	return false
}

func isValidDuration(d string) bool {
	// Simple duration validation (e.g., "30s", "5m", "1h")
	if d == "" {
		return false
	}

	// Check for valid duration suffixes
	validSuffixes := []string{"ns", "us", "µs", "ms", "s", "m", "h"}
	for _, suffix := range validSuffixes {
		if strings.HasSuffix(d, suffix) {
			return true
		}
	}

	return false
}
