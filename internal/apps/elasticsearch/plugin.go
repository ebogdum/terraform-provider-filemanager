// SPDX-License-Identifier: MIT

// Package elasticsearch provides a plugin for Elasticsearch configuration management.
package elasticsearch

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
	"gopkg.in/yaml.v3"
)

// Plugin implements the Elasticsearch configuration plugin.
type Plugin struct{}

// New creates a new Elasticsearch plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "elasticsearch"
}

// Version returns the supported Elasticsearch version range.
func (p *Plugin) Version() string {
	return ">=7.0.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "Elasticsearch configuration management"
}

// NativeFormat returns the native configuration format.
func (p *Plugin) NativeFormat() string {
	return "yaml"
}

// Schema returns the configuration schema.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "cluster",
				Description: "Cluster settings",
			},
			{
				Name:        "node",
				Description: "Node settings",
			},
			{
				Name:        "path",
				Description: "Path settings",
			},
			{
				Name:        "network",
				Description: "Network settings",
			},
			{
				Name:        "discovery",
				Description: "Discovery settings",
			},
			{
				Name:        "gateway",
				Description: "Gateway settings",
			},
			{
				Name:        "action",
				Description: "Action settings",
			},
			{
				Name:        "xpack",
				Description: "X-Pack settings",
			},
		},
		Directives: []plugin.DirectiveSchema{},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"cluster.name": "elasticsearch",
		"node.name":    "${HOSTNAME}",
		"path.data":    "/var/lib/elasticsearch",
		"path.logs":    "/var/log/elasticsearch",
		"network.host": "0.0.0.0",
		"http.port":    9200,
	}
}

// Validate validates the configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Elasticsearch uses flat key structure with dots
	flatConfig := flattenConfig(configMap)

	// Validate cluster settings
	errors = append(errors, p.validateClusterSettings(flatConfig)...)

	// Validate node settings
	errors = append(errors, p.validateNodeSettings(flatConfig)...)

	// Validate path settings
	errors = append(errors, p.validatePathSettings(flatConfig)...)

	// Validate network settings
	errors = append(errors, p.validateNetworkSettings(flatConfig)...)

	// Validate discovery settings
	errors = append(errors, p.validateDiscoverySettings(flatConfig)...)

	// Validate xpack settings
	errors = append(errors, p.validateXPackSettings(flatConfig)...)

	return errors, nil
}

func (p *Plugin) validateClusterSettings(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate cluster.name
	if clusterName, ok := config["cluster.name"].(string); ok {
		if clusterName == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    "cluster.name",
				Message: "cluster.name should not be empty",
			})
		}
		if !isValidClusterName(clusterName) {
			errors = append(errors, plugin.ValidationError{
				Path:    "cluster.name",
				Message: fmt.Sprintf("invalid cluster name: %s", clusterName),
			})
		}
	}

	// Validate cluster.initial_master_nodes
	if initialMasters, ok := config["cluster.initial_master_nodes"].([]any); ok {
		if len(initialMasters) == 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "cluster.initial_master_nodes",
				Message: "cluster.initial_master_nodes should not be empty",
			})
		}
		// Check for odd number (recommended for quorum)
		if len(initialMasters) > 1 && len(initialMasters)%2 == 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "cluster.initial_master_nodes",
				Message: "odd number of master nodes is recommended for proper quorum",
			})
		}
	}

	// Validate cluster.routing.allocation settings
	if diskThreshold, ok := config["cluster.routing.allocation.disk.threshold_enabled"].(bool); ok && !diskThreshold {
		errors = append(errors, plugin.ValidationError{
			Path:    "cluster.routing.allocation.disk.threshold_enabled",
			Message: "disabling disk threshold can lead to disk exhaustion",
		})
	}

	return errors
}

func (p *Plugin) validateNodeSettings(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate node.name
	if nodeName, ok := config["node.name"].(string); ok {
		if nodeName == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    "node.name",
				Message: "node.name should not be empty",
			})
		}
	}

	// Validate node roles (ES 8.x+)
	if nodeRoles, ok := config["node.roles"].([]any); ok {
		validRoles := map[string]bool{
			"master":                true,
			"data":                  true,
			"data_content":          true,
			"data_hot":              true,
			"data_warm":             true,
			"data_cold":             true,
			"data_frozen":           true,
			"ingest":                true,
			"ml":                    true,
			"remote_cluster_client": true,
			"transform":             true,
			"voting_only":           true,
			"coordinating_only":     true,
		}
		for i, role := range nodeRoles {
			if roleStr, ok := role.(string); ok {
				if !validRoles[roleStr] {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("node.roles[%d]", i),
						Message: fmt.Sprintf("unknown node role: %s", roleStr),
					})
				}
			}
		}
	}

	// Deprecated node.master, node.data, node.ingest (ES < 8.x)
	deprecatedNodeSettings := []string{"node.master", "node.data", "node.ingest", "node.ml"}
	for _, setting := range deprecatedNodeSettings {
		if _, ok := config[setting]; ok {
			errors = append(errors, plugin.ValidationError{
				Path:    setting,
				Message: fmt.Sprintf("%s is deprecated, use node.roles instead", setting),
			})
		}
	}

	// Validate node.attr.*
	for key := range config {
		if strings.HasPrefix(key, "node.attr.") {
			attrName := strings.TrimPrefix(key, "node.attr.")
			if !isValidAttributeName(attrName) {
				errors = append(errors, plugin.ValidationError{
					Path:    key,
					Message: fmt.Sprintf("invalid attribute name: %s", attrName),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validatePathSettings(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate path.data
	if pathData, ok := config["path.data"].(string); ok {
		if pathData == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    "path.data",
				Message: "path.data should not be empty",
			})
		}
	} else if pathDataList, ok := config["path.data"].([]any); ok {
		if len(pathDataList) == 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "path.data",
				Message: "path.data should not be empty",
			})
		}
	}

	// Validate path.logs
	if pathLogs, ok := config["path.logs"].(string); ok {
		if pathLogs == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    "path.logs",
				Message: "path.logs should not be empty",
			})
		}
	}

	return errors
}

func (p *Plugin) validateNetworkSettings(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate network.host
	if networkHost, ok := config["network.host"].(string); ok {
		if !isValidNetworkHost(networkHost) {
			errors = append(errors, plugin.ValidationError{
				Path:    "network.host",
				Message: fmt.Sprintf("invalid network host: %s", networkHost),
			})
		}
	}

	// Validate http.port
	if httpPort, ok := config["http.port"]; ok {
		var port int
		switch v := httpPort.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		case string:
			// Port ranges like "9200-9300" are valid
			if !isValidPortRange(v) {
				errors = append(errors, plugin.ValidationError{
					Path:    "http.port",
					Message: fmt.Sprintf("invalid port or port range: %s", v),
				})
			}
		}
		if port > 0 && (port < 1 || port > 65535) {
			errors = append(errors, plugin.ValidationError{
				Path:    "http.port",
				Message: fmt.Sprintf("port must be between 1 and 65535, got: %d", port),
			})
		}
	}

	// Validate transport.port
	if transportPort, ok := config["transport.port"]; ok {
		var port int
		switch v := transportPort.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		case string:
			if !isValidPortRange(v) {
				errors = append(errors, plugin.ValidationError{
					Path:    "transport.port",
					Message: fmt.Sprintf("invalid port or port range: %s", v),
				})
			}
		}
		if port > 0 && (port < 1 || port > 65535) {
			errors = append(errors, plugin.ValidationError{
				Path:    "transport.port",
				Message: fmt.Sprintf("port must be between 1 and 65535, got: %d", port),
			})
		}
	}

	// Warn if network.host is binding to all interfaces
	if networkHost, ok := config["network.host"].(string); ok {
		if networkHost == "0.0.0.0" || networkHost == "_site_" || networkHost == "_global_" {
			errors = append(errors, plugin.ValidationError{
				Path:    "network.host",
				Message: "binding to all interfaces may expose Elasticsearch to the network",
			})
		}
	}

	return errors
}

func (p *Plugin) validateDiscoverySettings(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate discovery.seed_hosts
	if seedHosts, ok := config["discovery.seed_hosts"].([]any); ok {
		for i, host := range seedHosts {
			if hostStr, ok := host.(string); ok {
				if !isValidSeedHost(hostStr) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("discovery.seed_hosts[%d]", i),
						Message: fmt.Sprintf("invalid seed host: %s", hostStr),
					})
				}
			}
		}
	}

	// Validate discovery.type
	if discoveryType, ok := config["discovery.type"].(string); ok {
		validTypes := []string{"single-node", "multi-node", "zen"}
		found := false
		for _, vt := range validTypes {
			if discoveryType == vt {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "discovery.type",
				Message: fmt.Sprintf("unknown discovery type: %s", discoveryType),
			})
		}

		// Warn about single-node mode
		if discoveryType == "single-node" {
			errors = append(errors, plugin.ValidationError{
				Path:    "discovery.type",
				Message: "single-node mode is not suitable for production",
			})
		}
	}

	// Check for discovery settings consistency
	hasSeedHosts := false
	hasInitialMasters := false
	if _, ok := config["discovery.seed_hosts"]; ok {
		hasSeedHosts = true
	}
	if _, ok := config["cluster.initial_master_nodes"]; ok {
		hasInitialMasters = true
	}

	// In production, both should be set
	if hasSeedHosts && !hasInitialMasters {
		errors = append(errors, plugin.ValidationError{
			Path:    "cluster.initial_master_nodes",
			Message: "cluster.initial_master_nodes should be set for multi-node clusters",
		})
	}

	return errors
}

func (p *Plugin) validateXPackSettings(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate xpack.security.enabled
	if securityEnabled, ok := config["xpack.security.enabled"].(bool); ok {
		if !securityEnabled {
			errors = append(errors, plugin.ValidationError{
				Path:    "xpack.security.enabled",
				Message: "security should be enabled for production environments",
			})
		}
	} else {
		// Security is enabled by default in ES 8.x, but check for ES 7.x
		errors = append(errors, plugin.ValidationError{
			Path:    "xpack.security.enabled",
			Message: "consider explicitly enabling security",
		})
	}

	// Validate transport SSL
	if transportSSL, ok := config["xpack.security.transport.ssl.enabled"].(bool); ok && transportSSL {
		// Check for required certificates
		if _, ok := config["xpack.security.transport.ssl.keystore.path"]; !ok {
			if _, ok := config["xpack.security.transport.ssl.key"]; !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    "xpack.security.transport.ssl",
					Message: "transport SSL enabled but no keystore or key configured",
				})
			}
		}
	}

	// Validate HTTP SSL
	if httpSSL, ok := config["xpack.security.http.ssl.enabled"].(bool); ok && httpSSL {
		// Check for required certificates
		if _, ok := config["xpack.security.http.ssl.keystore.path"]; !ok {
			if _, ok := config["xpack.security.http.ssl.key"]; !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    "xpack.security.http.ssl",
					Message: "HTTP SSL enabled but no keystore or key configured",
				})
			}
		}
	}

	// Validate xpack.monitoring settings
	if monitoringEnabled, ok := config["xpack.monitoring.enabled"].(bool); ok && monitoringEnabled {
		// Check for monitoring export configuration
		if _, ok := config["xpack.monitoring.collection.enabled"]; !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    "xpack.monitoring.collection.enabled",
				Message: "monitoring enabled but collection not explicitly configured",
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

	flatConfig := flattenConfig(configMap)

	// Check for production readiness
	isProduction := true

	// Check discovery type
	if discoveryType, ok := flatConfig["discovery.type"].(string); ok && discoveryType == "single-node" {
		isProduction = false
	}

	// Check security
	if securityEnabled, ok := flatConfig["xpack.security.enabled"].(bool); ok && !securityEnabled {
		isProduction = false
	}

	// Check network binding
	if networkHost, ok := flatConfig["network.host"].(string); ok && networkHost == "localhost" {
		isProduction = false
	}

	if !isProduction {
		errors = append(errors, plugin.ValidationError{
			Path:    "",
			Message: "configuration may not be suitable for production use",
		})
	}

	// Check memory settings
	if heapSize, ok := flatConfig["bootstrap.memory_lock"].(bool); ok && !heapSize {
		errors = append(errors, plugin.ValidationError{
			Path:    "bootstrap.memory_lock",
			Message: "memory lock is recommended for production to prevent swapping",
		})
	}

	return errors, nil
}

// Normalize normalizes the configuration.
func (p *Plugin) Normalize(config any) (any, error) {
	// Elasticsearch config doesn't need much normalization
	return config, nil
}

// ToNative converts to native Elasticsearch configuration format (YAML).
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return yaml.Marshal(config)
}

// FromNative parses native Elasticsearch configuration format.
func (p *Plugin) FromNative(data []byte) (any, error) {
	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
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

// flattenConfig flattens a nested map into dot-notation keys
func flattenConfig(config map[string]any) map[string]any {
	result := make(map[string]any)
	flattenConfigRecursive("", config, result)
	return result
}

func flattenConfigRecursive(prefix string, config map[string]any, result map[string]any) {
	for key, value := range config {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			flattenConfigRecursive(fullKey, v, result)
		default:
			result[fullKey] = v
		}
	}
}

func isValidClusterName(name string) bool {
	// Cluster name can contain alphanumeric characters, dashes, underscores
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return pattern.MatchString(name)
}

func isValidAttributeName(name string) bool {
	// Attribute names should be alphanumeric with underscores
	pattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return pattern.MatchString(name)
}

func isValidNetworkHost(host string) bool {
	// Check for special values
	specialHosts := []string{
		"_local_", "_site_", "_global_",
		"_[networkInterface]_", "0.0.0.0", "localhost",
	}
	for _, sh := range specialHosts {
		if host == sh {
			return true
		}
	}

	// Check for interface patterns
	if strings.HasPrefix(host, "_") && strings.HasSuffix(host, "_") {
		return true
	}

	// Check for IP address
	if net.ParseIP(host) != nil {
		return true
	}

	// Check for hostname
	return isValidHostname(host)
}

func isValidHostname(hostname string) bool {
	// Simple hostname validation
	pattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)
	return pattern.MatchString(hostname)
}

func isValidPortRange(portRange string) bool {
	// Port range format: port or port-port
	pattern := regexp.MustCompile(`^\d+(-\d+)?$`)
	return pattern.MatchString(portRange)
}

func isValidSeedHost(host string) bool {
	// Seed host can be IP, hostname, or IP:port
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			return false
		}
		if !isValidNetworkHost(h) {
			return false
		}
		return isValidPortRange(p)
	}
	return isValidNetworkHost(host)
}
