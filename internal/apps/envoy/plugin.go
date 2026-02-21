// SPDX-License-Identifier: MIT

// Package envoy provides a plugin for Envoy proxy configuration management.
package envoy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
	"gopkg.in/yaml.v3"
)

// Plugin implements the Envoy configuration plugin.
type Plugin struct{}

// New creates a new Envoy plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "envoy"
}

// Version returns the supported Envoy version range.
func (p *Plugin) Version() string {
	return ">=1.20.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "Envoy proxy configuration management"
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
				Name:        "admin",
				Description: "Admin interface configuration",
			},
			{
				Name:        "static_resources",
				Description: "Static resource configuration",
			},
			{
				Name:        "dynamic_resources",
				Description: "Dynamic resource configuration (xDS)",
			},
			{
				Name:        "cluster_manager",
				Description: "Cluster manager configuration",
			},
			{
				Name:        "hds_config",
				Description: "Health discovery service configuration",
			},
			{
				Name:        "stats_config",
				Description: "Statistics configuration",
			},
			{
				Name:        "tracing",
				Description: "Distributed tracing configuration",
			},
			{
				Name:        "layered_runtime",
				Description: "Runtime configuration",
			},
			{
				Name:        "overload_manager",
				Description: "Overload manager configuration",
			},
		},
		Directives: []plugin.DirectiveSchema{
			{
				Name:        "node",
				Description: "Node identification",
				Type:        "object",
			},
		},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"admin": map[string]any{
			"address": map[string]any{
				"socket_address": map[string]any{
					"address":    "0.0.0.0",
					"port_value": 9901,
				},
			},
		},
		"static_resources": map[string]any{
			"listeners": []any{},
			"clusters":  []any{},
		},
	}
}

// Validate validates the configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Validate admin section
	if admin, ok := configMap["admin"].(map[string]any); ok {
		errors = append(errors, p.validateAdmin(admin)...)
	}

	// Validate node if present
	if node, ok := configMap["node"].(map[string]any); ok {
		errors = append(errors, p.validateNode(node)...)
	}

	// Validate static_resources
	if staticRes, ok := configMap["static_resources"].(map[string]any); ok {
		errors = append(errors, p.validateStaticResources(staticRes)...)
	}

	// Validate dynamic_resources
	if dynamicRes, ok := configMap["dynamic_resources"].(map[string]any); ok {
		errors = append(errors, p.validateDynamicResources(dynamicRes)...)
	}

	// Validate overload_manager
	if overloadMgr, ok := configMap["overload_manager"].(map[string]any); ok {
		errors = append(errors, p.validateOverloadManager(overloadMgr)...)
	}

	return errors, nil
}

func (p *Plugin) validateAdmin(admin map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate address
	if address, ok := admin["address"].(map[string]any); ok {
		errors = append(errors, p.validateAddress(address, "admin.address")...)
	}

	// Validate access_log if present
	if accessLog, ok := admin["access_log"].([]any); ok {
		for i, log := range accessLog {
			if logMap, ok := log.(map[string]any); ok {
				errors = append(errors, p.validateAccessLog(logMap, fmt.Sprintf("admin.access_log[%d]", i))...)
			}
		}
	}

	return errors
}

func (p *Plugin) validateNode(node map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate cluster
	if cluster, ok := node["cluster"].(string); ok {
		if cluster == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    "node.cluster",
				Message: "cluster name should not be empty",
			})
		}
	}

	// Validate id
	if id, ok := node["id"].(string); ok {
		if id == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    "node.id",
				Message: "node id should not be empty",
			})
		}
	}

	return errors
}

func (p *Plugin) validateStaticResources(staticRes map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate listeners
	if listeners, ok := staticRes["listeners"].([]any); ok {
		errors = append(errors, p.validateListeners(listeners)...)
	}

	// Validate clusters
	if clusters, ok := staticRes["clusters"].([]any); ok {
		errors = append(errors, p.validateClusters(clusters)...)
	}

	// Validate secrets
	if secrets, ok := staticRes["secrets"].([]any); ok {
		errors = append(errors, p.validateSecrets(secrets)...)
	}

	return errors
}

func (p *Plugin) validateListeners(listeners []any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)
	usedAddresses := make(map[string]bool)

	for i, listener := range listeners {
		listenerMap, ok := listener.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("static_resources.listeners[%d]", i),
				Message: "listener must be an object",
			})
			continue
		}

		// Validate name
		if name, ok := listenerMap["name"].(string); ok {
			if usedNames[name] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("static_resources.listeners[%d].name", i),
					Message: fmt.Sprintf("duplicate listener name: %s", name),
				})
			}
			usedNames[name] = true
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("static_resources.listeners[%d].name", i),
				Message: "listener name is required",
			})
		}

		// Validate address
		if address, ok := listenerMap["address"].(map[string]any); ok {
			addressKey := addressToKey(address)
			if addressKey != "" {
				if usedAddresses[addressKey] {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("static_resources.listeners[%d].address", i),
						Message: fmt.Sprintf("duplicate listener address: %s", addressKey),
					})
				}
				usedAddresses[addressKey] = true
			}
			errors = append(errors, p.validateAddress(address, fmt.Sprintf("static_resources.listeners[%d].address", i))...)
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("static_resources.listeners[%d].address", i),
				Message: "listener address is required",
			})
		}

		// Validate filter_chains
		if filterChains, ok := listenerMap["filter_chains"].([]any); ok {
			errors = append(errors, p.validateFilterChains(filterChains, fmt.Sprintf("static_resources.listeners[%d]", i))...)
		}
	}

	return errors
}

func (p *Plugin) validateFilterChains(filterChains []any, parentPath string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for i, fc := range filterChains {
		fcMap, ok := fc.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.filter_chains[%d]", parentPath, i),
				Message: "filter_chain must be an object",
			})
			continue
		}

		// Validate filters
		if filters, ok := fcMap["filters"].([]any); ok {
			errors = append(errors, p.validateFilters(filters, fmt.Sprintf("%s.filter_chains[%d]", parentPath, i))...)
		}

		// Validate transport_socket if present
		if transportSocket, ok := fcMap["transport_socket"].(map[string]any); ok {
			errors = append(errors, p.validateTransportSocket(transportSocket, fmt.Sprintf("%s.filter_chains[%d].transport_socket", parentPath, i))...)
		}
	}

	return errors
}

func (p *Plugin) validateFilters(filters []any, parentPath string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for i, filter := range filters {
		filterMap, ok := filter.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.filters[%d]", parentPath, i),
				Message: "filter must be an object",
			})
			continue
		}

		// Validate name
		if name, ok := filterMap["name"].(string); ok {
			if !isValidFilterName(name) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.filters[%d].name", parentPath, i),
					Message: fmt.Sprintf("unknown filter name: %s", name),
				})
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.filters[%d].name", parentPath, i),
				Message: "filter name is required",
			})
		}

		// Validate typed_config
		if typedConfig, ok := filterMap["typed_config"].(map[string]any); ok {
			if typeURL, ok := typedConfig["@type"].(string); ok {
				if !isValidTypeURL(typeURL) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("%s.filters[%d].typed_config.@type", parentPath, i),
						Message: fmt.Sprintf("unknown type URL: %s", typeURL),
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateTransportSocket(ts map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate name
	if name, ok := ts["name"].(string); ok {
		validNames := []string{
			"envoy.transport_sockets.tls",
			"envoy.transport_sockets.raw_buffer",
			"envoy.transport_sockets.quic",
		}
		found := false
		for _, vn := range validNames {
			if name == vn {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    path + ".name",
				Message: fmt.Sprintf("unknown transport socket name: %s", name),
			})
		}
	}

	return errors
}

func (p *Plugin) validateClusters(clusters []any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)

	for i, cluster := range clusters {
		clusterMap, ok := cluster.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("static_resources.clusters[%d]", i),
				Message: "cluster must be an object",
			})
			continue
		}

		errors = append(errors, validateClusterName(clusterMap, i, usedNames)...)
		errors = append(errors, validateClusterType(clusterMap, i)...)
		errors = append(errors, validateClusterLBPolicy(clusterMap, i)...)
		errors = append(errors, p.validateClusterLoadAssignment(clusterMap, i)...)
		errors = append(errors, p.validateClusterHealthChecks(clusterMap, i)...)
	}
	return errors
}

func (p *Plugin) validateLoadAssignment(la map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError
	errors = append(errors, validateLoadAssignmentClusterName(la, path)...)
	errors = append(errors, p.validateLoadAssignmentEndpoints(la, path)...)
	return errors
}

func validateClusterName(clusterMap map[string]any, index int, usedNames map[string]bool) []plugin.ValidationError {
	path := fmt.Sprintf("static_resources.clusters[%d].name", index)
	name, ok := clusterMap["name"].(string)
	if !ok {
		return []plugin.ValidationError{{
			Path:    path,
			Message: "cluster name is required",
		}}
	}

	if usedNames[name] {
		return []plugin.ValidationError{{
			Path:    path,
			Message: fmt.Sprintf("duplicate cluster name: %s", name),
		}}
	}

	usedNames[name] = true
	return nil
}

func validateClusterType(clusterMap map[string]any, index int) []plugin.ValidationError {
	clusterType, ok := clusterMap["type"].(string)
	if !ok {
		return nil
	}

	validTypes := []string{"STATIC", "STRICT_DNS", "LOGICAL_DNS", "EDS", "ORIGINAL_DST"}
	if envoyContains(validTypes, clusterType) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("static_resources.clusters[%d].type", index),
		Message: fmt.Sprintf("invalid cluster type: %s", clusterType),
	}}
}

func validateClusterLBPolicy(clusterMap map[string]any, index int) []plugin.ValidationError {
	lbPolicy, ok := clusterMap["lb_policy"].(string)
	if !ok {
		return nil
	}

	validPolicies := []string{
		"ROUND_ROBIN", "LEAST_REQUEST", "RING_HASH", "RANDOM",
		"MAGLEV", "CLUSTER_PROVIDED", "LOAD_BALANCING_POLICY_CONFIG",
	}
	if envoyContains(validPolicies, lbPolicy) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("static_resources.clusters[%d].lb_policy", index),
		Message: fmt.Sprintf("invalid lb_policy: %s", lbPolicy),
	}}
}

func (p *Plugin) validateClusterLoadAssignment(clusterMap map[string]any, index int) []plugin.ValidationError {
	loadAssignment, ok := clusterMap["load_assignment"].(map[string]any)
	if !ok {
		return nil
	}
	return p.validateLoadAssignment(loadAssignment, fmt.Sprintf("static_resources.clusters[%d].load_assignment", index))
}

func (p *Plugin) validateClusterHealthChecks(clusterMap map[string]any, index int) []plugin.ValidationError {
	healthChecks, ok := clusterMap["health_checks"].([]any)
	if !ok {
		return nil
	}
	return p.validateHealthChecks(healthChecks, fmt.Sprintf("static_resources.clusters[%d]", index))
}

func validateLoadAssignmentClusterName(la map[string]any, path string) []plugin.ValidationError {
	clusterName, ok := la["cluster_name"].(string)
	if !ok || clusterName != "" {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    path + ".cluster_name",
		Message: "cluster_name should not be empty",
	}}
}

func (p *Plugin) validateLoadAssignmentEndpoints(la map[string]any, path string) []plugin.ValidationError {
	endpoints, ok := la["endpoints"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for i, endpoint := range endpoints {
		errors = append(errors, p.validateLBEndpoints(endpoint, path, i)...)
	}
	return errors
}

func (p *Plugin) validateLBEndpoints(endpoint any, path string, endpointIndex int) []plugin.ValidationError {
	epMap, ok := endpoint.(map[string]any)
	if !ok {
		return nil
	}

	lbEndpoints, ok := epMap["lb_endpoints"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for j, lbEndpoint := range lbEndpoints {
		errors = append(errors, p.validateLBEndpointAddress(lbEndpoint, path, endpointIndex, j)...)
	}
	return errors
}

func (p *Plugin) validateLBEndpointAddress(lbEndpoint any, path string, endpointIndex, lbIndex int) []plugin.ValidationError {
	lbEndpointMap, ok := lbEndpoint.(map[string]any)
	if !ok {
		return nil
	}

	endpoint, ok := lbEndpointMap["endpoint"].(map[string]any)
	if !ok {
		return nil
	}

	address, ok := endpoint["address"].(map[string]any)
	if !ok {
		return nil
	}

	addressPath := fmt.Sprintf("%s.endpoints[%d].lb_endpoints[%d].endpoint.address", path, endpointIndex, lbIndex)
	return p.validateAddress(address, addressPath)
}

func (p *Plugin) validateHealthChecks(hc []any, parentPath string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for i, check := range hc {
		checkMap, ok := check.(map[string]any)
		if !ok {
			continue
		}

		// Validate timeout
		if timeout, ok := checkMap["timeout"].(string); ok {
			if !isValidDuration(timeout) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.health_checks[%d].timeout", parentPath, i),
					Message: fmt.Sprintf("invalid duration format: %s", timeout),
				})
			}
		}

		// Validate interval
		if interval, ok := checkMap["interval"].(string); ok {
			if !isValidDuration(interval) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.health_checks[%d].interval", parentPath, i),
					Message: fmt.Sprintf("invalid duration format: %s", interval),
				})
			}
		}

		// Check for health check type
		hasType := false
		checkTypes := []string{"tcp_health_check", "http_health_check", "grpc_health_check", "custom_health_check"}
		for _, ct := range checkTypes {
			if _, ok := checkMap[ct]; ok {
				hasType = true
				break
			}
		}
		if !hasType {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.health_checks[%d]", parentPath, i),
				Message: "health check type is required (tcp_health_check, http_health_check, grpc_health_check, or custom_health_check)",
			})
		}
	}

	return errors
}

func (p *Plugin) validateSecrets(secrets []any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)

	for i, secret := range secrets {
		secretMap, ok := secret.(map[string]any)
		if !ok {
			continue
		}

		// Validate name
		if name, ok := secretMap["name"].(string); ok {
			if usedNames[name] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("static_resources.secrets[%d].name", i),
					Message: fmt.Sprintf("duplicate secret name: %s", name),
				})
			}
			usedNames[name] = true
		}
	}

	return errors
}

func (p *Plugin) validateDynamicResources(dr map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate cds_config
	if cdsConfig, ok := dr["cds_config"].(map[string]any); ok {
		errors = append(errors, p.validateXdsConfig(cdsConfig, "dynamic_resources.cds_config")...)
	}

	// Validate lds_config
	if ldsConfig, ok := dr["lds_config"].(map[string]any); ok {
		errors = append(errors, p.validateXdsConfig(ldsConfig, "dynamic_resources.lds_config")...)
	}

	// Validate ads_config
	if adsConfig, ok := dr["ads_config"].(map[string]any); ok {
		errors = append(errors, p.validateAdsConfig(adsConfig, "dynamic_resources.ads_config")...)
	}

	return errors
}

func (p *Plugin) validateXdsConfig(xds map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Check for resource source
	hasSource := false
	if _, ok := xds["path"].(string); ok {
		hasSource = true
	}
	if _, ok := xds["api_config_source"]; ok {
		hasSource = true
	}
	if _, ok := xds["ads"]; ok {
		hasSource = true
	}

	if !hasSource {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: "xDS config must specify a resource source (path, api_config_source, or ads)",
		})
	}

	return errors
}

func (p *Plugin) validateAdsConfig(ads map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate api_type
	if apiType, ok := ads["api_type"].(string); ok {
		validTypes := []string{"REST", "GRPC", "DELTA_GRPC"}
		found := false
		for _, vt := range validTypes {
			if apiType == vt {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    path + ".api_type",
				Message: fmt.Sprintf("invalid api_type: %s", apiType),
			})
		}
	}

	// Validate grpc_services
	if grpcServices, ok := ads["grpc_services"].([]any); ok {
		if len(grpcServices) == 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    path + ".grpc_services",
				Message: "at least one gRPC service is required for ADS",
			})
		}
	}

	return errors
}

func (p *Plugin) validateOverloadManager(om map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate refresh_interval
	if refreshInterval, ok := om["refresh_interval"].(string); ok {
		if !isValidDuration(refreshInterval) {
			errors = append(errors, plugin.ValidationError{
				Path:    "overload_manager.refresh_interval",
				Message: fmt.Sprintf("invalid duration format: %s", refreshInterval),
			})
		}
	}

	return errors
}

func (p *Plugin) validateAddress(address map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	if socketAddr, ok := address["socket_address"].(map[string]any); ok {
		// Validate address
		if addr, ok := socketAddr["address"].(string); ok {
			if addr == "" {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".socket_address.address",
					Message: "address should not be empty",
				})
			}
		}

		// Validate port_value
		if portValue, ok := socketAddr["port_value"]; ok {
			var port int
			switch v := portValue.(type) {
			case int:
				port = v
			case float64:
				port = int(v)
			}
			if port < 0 || port > 65535 {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".socket_address.port_value",
					Message: fmt.Sprintf("port must be between 0 and 65535, got: %d", port),
				})
			}
		}
	}

	if pipe, ok := address["pipe"].(map[string]any); ok {
		if pipePath, ok := pipe["path"].(string); ok {
			if pipePath == "" {
				errors = append(errors, plugin.ValidationError{
					Path:    path + ".pipe.path",
					Message: "pipe path should not be empty",
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateAccessLog(al map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate name
	if name, ok := al["name"].(string); ok {
		validNames := []string{
			"envoy.access_loggers.file",
			"envoy.access_loggers.stdout",
			"envoy.access_loggers.stderr",
		}
		found := false
		for _, vn := range validNames {
			if name == vn {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    path + ".name",
				Message: fmt.Sprintf("unknown access logger: %s", name),
			})
		}
	}

	return errors
}

// ValidateSemantic performs semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var errors []plugin.ValidationError
	errors = append(errors, validateEnvoyResourcePresence(configMap)...)
	errors = append(errors, p.validateEnvoyClusterReferences(configMap)...)
	errors = append(errors, validateEnvoyAdminBinding(configMap)...)
	return errors, nil
}

func validateEnvoyResourcePresence(configMap map[string]any) []plugin.ValidationError {
	hasStatic := false
	if staticRes, ok := configMap["static_resources"].(map[string]any); ok {
		hasStatic = hasStaticResources(staticRes)
	}

	_, hasDynamic := configMap["dynamic_resources"]
	if hasStatic || hasDynamic {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "",
		Message: "configuration should have either static_resources or dynamic_resources",
	}}
}

func hasStaticResources(staticRes map[string]any) bool {
	if listeners, ok := staticRes["listeners"].([]any); ok && len(listeners) > 0 {
		return true
	}
	if clusters, ok := staticRes["clusters"].([]any); ok && len(clusters) > 0 {
		return true
	}
	return false
}

func (p *Plugin) validateEnvoyClusterReferences(configMap map[string]any) []plugin.ValidationError {
	staticRes, ok := configMap["static_resources"].(map[string]any)
	if !ok {
		return nil
	}

	clusterNames := collectEnvoyClusterNames(staticRes["clusters"])
	listeners, ok := staticRes["listeners"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for i, listener := range listeners {
		errors = append(errors, validateListenerClusterRefs(listener, i, clusterNames)...)
	}
	return errors
}

func collectEnvoyClusterNames(clustersAny any) map[string]bool {
	clusterNames := make(map[string]bool)
	clusters, ok := clustersAny.([]any)
	if !ok {
		return clusterNames
	}

	for _, cluster := range clusters {
		clusterMap, ok := cluster.(map[string]any)
		if !ok {
			continue
		}
		name, ok := clusterMap["name"].(string)
		if ok {
			clusterNames[name] = true
		}
	}
	return clusterNames
}

func validateListenerClusterRefs(listener any, listenerIndex int, clusterNames map[string]bool) []plugin.ValidationError {
	listenerMap, ok := listener.(map[string]any)
	if !ok {
		return nil
	}

	filterChains, ok := listenerMap["filter_chains"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for _, filterChain := range filterChains {
		errors = append(errors, validateFilterChainClusterRefs(filterChain, listenerIndex, clusterNames)...)
	}
	return errors
}

func validateFilterChainClusterRefs(filterChain any, listenerIndex int, clusterNames map[string]bool) []plugin.ValidationError {
	filterChainMap, ok := filterChain.(map[string]any)
	if !ok {
		return nil
	}

	filters, ok := filterChainMap["filters"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for _, filter := range filters {
		errors = append(errors, validateFilterClusterRefs(filter, listenerIndex, clusterNames)...)
	}
	return errors
}

func validateFilterClusterRefs(filter any, listenerIndex int, clusterNames map[string]bool) []plugin.ValidationError {
	filterMap, ok := filter.(map[string]any)
	if !ok {
		return nil
	}

	typedConfig, ok := filterMap["typed_config"].(map[string]any)
	if !ok {
		return nil
	}

	routeConfig, ok := typedConfig["route_config"].(map[string]any)
	if !ok {
		return nil
	}

	virtualHosts, ok := routeConfig["virtual_hosts"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for _, virtualHost := range virtualHosts {
		errors = append(errors, validateVirtualHostClusterRefs(virtualHost, listenerIndex, clusterNames)...)
	}
	return errors
}

func validateVirtualHostClusterRefs(virtualHost any, listenerIndex int, clusterNames map[string]bool) []plugin.ValidationError {
	virtualHostMap, ok := virtualHost.(map[string]any)
	if !ok {
		return nil
	}

	routes, ok := virtualHostMap["routes"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for _, route := range routes {
		errors = append(errors, validateRouteClusterRef(route, listenerIndex, clusterNames)...)
	}
	return errors
}

func validateRouteClusterRef(route any, listenerIndex int, clusterNames map[string]bool) []plugin.ValidationError {
	routeMap, ok := route.(map[string]any)
	if !ok {
		return nil
	}

	routeAction, ok := routeMap["route"].(map[string]any)
	if !ok {
		return nil
	}

	cluster, ok := routeAction["cluster"].(string)
	if !ok || clusterNames[cluster] {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("static_resources.listeners[%d]", listenerIndex),
		Message: fmt.Sprintf("route references unknown cluster: %s", cluster),
	}}
}

func validateEnvoyAdminBinding(configMap map[string]any) []plugin.ValidationError {
	admin, ok := configMap["admin"].(map[string]any)
	if !ok {
		return nil
	}

	address, ok := admin["address"].(map[string]any)
	if !ok {
		return nil
	}

	socketAddr, ok := address["socket_address"].(map[string]any)
	if !ok {
		return nil
	}

	addr, ok := socketAddr["address"].(string)
	if !ok || addr != "0.0.0.0" {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "admin.address.socket_address.address",
		Message: "admin interface bound to all interfaces (0.0.0.0) - consider restricting access",
	}}
}

// Normalize normalizes the configuration.
func (p *Plugin) Normalize(config any) (any, error) {
	// Envoy config is already well-structured
	return config, nil
}

// ToNative converts to native Envoy configuration format (YAML).
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return yaml.Marshal(config)
}

// FromNative parses native Envoy configuration format.
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

func addressToKey(address map[string]any) string {
	if socketAddr, ok := address["socket_address"].(map[string]any); ok {
		addr := socketAddr["address"]
		port := socketAddr["port_value"]
		return fmt.Sprintf("%v:%v", addr, port)
	}
	if pipe, ok := address["pipe"].(map[string]any); ok {
		if path, ok := pipe["path"].(string); ok {
			return "unix:" + path
		}
	}
	return ""
}

func isValidFilterName(name string) bool {
	// Common filter names
	validFilters := []string{
		"envoy.filters.network.http_connection_manager",
		"envoy.filters.network.tcp_proxy",
		"envoy.filters.network.redis_proxy",
		"envoy.filters.network.mongo_proxy",
		"envoy.filters.network.mysql_proxy",
		"envoy.filters.http.router",
		"envoy.filters.http.cors",
		"envoy.filters.http.grpc_web",
		"envoy.filters.http.health_check",
		"envoy.filters.http.jwt_authn",
		"envoy.filters.http.lua",
		"envoy.filters.http.rbac",
		"envoy.filters.http.ext_authz",
	}
	for _, vf := range validFilters {
		if name == vf {
			return true
		}
	}
	// Allow custom filters
	return strings.HasPrefix(name, "envoy.")
}

func isValidTypeURL(typeURL string) bool {
	// Type URLs should start with type.googleapis.com/
	return strings.HasPrefix(typeURL, "type.googleapis.com/")
}

func isValidDuration(d string) bool {
	// Envoy duration format: number + s (seconds)
	pattern := regexp.MustCompile(`^\d+(\.\d+)?s$`)
	return pattern.MatchString(d)
}

func envoyContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
