// SPDX-License-Identifier: MIT

// Package traefik provides a plugin for Traefik proxy configuration management.
package traefik

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
	"gopkg.in/yaml.v3"
)

// Plugin implements the Traefik configuration plugin.
type Plugin struct{}

// New creates a new Traefik plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "traefik"
}

// Version returns the supported Traefik version range.
func (p *Plugin) Version() string {
	return ">=2.0.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "Traefik proxy configuration management"
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
				Name:        "entryPoints",
				Description: "Entry points configuration",
			},
			{
				Name:        "providers",
				Description: "Provider configurations (file, docker, kubernetes, etc.)",
			},
			{
				Name:        "api",
				Description: "API and dashboard configuration",
			},
			{
				Name:        "log",
				Description: "Logging configuration",
			},
			{
				Name:        "accessLog",
				Description: "Access logging configuration",
			},
			{
				Name:        "metrics",
				Description: "Metrics configuration",
			},
			{
				Name:        "tracing",
				Description: "Tracing configuration",
			},
			{
				Name:        "certificatesResolvers",
				Description: "ACME certificate resolvers",
			},
			{
				Name:        "tls",
				Description: "TLS configuration",
			},
			{
				Name:        "serversTransport",
				Description: "Servers transport configuration",
			},
		},
		Directives: []plugin.DirectiveSchema{
			{
				Name:        "global",
				Description: "Global configuration",
				Type:        "object",
			},
		},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"entryPoints": map[string]any{
			"web": map[string]any{
				"address": ":80",
			},
			"websecure": map[string]any{
				"address": ":443",
			},
		},
		"api": map[string]any{
			"dashboard": true,
			"insecure":  false,
		},
		"log": map[string]any{
			"level": "INFO",
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

	// Validate entryPoints
	if entryPoints, ok := configMap["entryPoints"].(map[string]any); ok {
		errors = append(errors, p.validateEntryPoints(entryPoints)...)
	}

	// Validate providers
	if providers, ok := configMap["providers"].(map[string]any); ok {
		errors = append(errors, p.validateProviders(providers)...)
	}

	// Validate api
	if api, ok := configMap["api"].(map[string]any); ok {
		errors = append(errors, p.validateAPI(api)...)
	}

	// Validate log
	if log, ok := configMap["log"].(map[string]any); ok {
		errors = append(errors, p.validateLog(log)...)
	}

	// Validate certificatesResolvers
	if resolvers, ok := configMap["certificatesResolvers"].(map[string]any); ok {
		errors = append(errors, p.validateCertResolvers(resolvers)...)
	}

	// Validate tls
	if tls, ok := configMap["tls"].(map[string]any); ok {
		errors = append(errors, p.validateTLS(tls)...)
	}

	// Validate metrics
	if metrics, ok := configMap["metrics"].(map[string]any); ok {
		errors = append(errors, p.validateMetrics(metrics)...)
	}

	return errors, nil
}

func (p *Plugin) validateEntryPoints(entryPoints map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedPorts := make(map[int]string)

	for name, ep := range entryPoints {
		epMap, ok := ep.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("entryPoints.%s", name),
				Message: "entryPoint must be an object",
			})
			continue
		}

		// Validate address
		if address, ok := epMap["address"].(string); ok {
			port, err := parsePort(address)
			if err != nil {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("entryPoints.%s.address", name),
					Message: fmt.Sprintf("invalid address format: %s", address),
				})
			} else {
				// Check for port conflicts
				if existing, exists := usedPorts[port]; exists {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("entryPoints.%s.address", name),
						Message: fmt.Sprintf("port %d is already used by entryPoint '%s'", port, existing),
					})
				}
				usedPorts[port] = name
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("entryPoints.%s.address", name),
				Message: "address is required",
			})
		}

		// Validate transport if present
		if transport, ok := epMap["transport"].(map[string]any); ok {
			errors = append(errors, p.validateTransport(transport, fmt.Sprintf("entryPoints.%s.transport", name))...)
		}

		// Validate http configuration if present
		if http, ok := epMap["http"].(map[string]any); ok {
			errors = append(errors, p.validateEntryPointHTTP(http, fmt.Sprintf("entryPoints.%s.http", name))...)
		}
	}

	return errors
}

func (p *Plugin) validateTransport(transport map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate respondingTimeouts
	if timeouts, ok := transport["respondingTimeouts"].(map[string]any); ok {
		for name, value := range timeouts {
			if valStr, ok := value.(string); ok {
				if !isValidDuration(valStr) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("%s.respondingTimeouts.%s", path, name),
						Message: fmt.Sprintf("invalid duration format: %s", valStr),
					})
				}
			}
		}
	}

	// Validate lifeCycle
	if lifecycle, ok := transport["lifeCycle"].(map[string]any); ok {
		if graceTimeout, ok := lifecycle["graceTimeOut"].(string); ok {
			if !isValidDuration(graceTimeout) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.lifeCycle.graceTimeOut", path),
					Message: fmt.Sprintf("invalid duration format: %s", graceTimeout),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateEntryPointHTTP(http map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate redirections
	if redirects, ok := http["redirections"].(map[string]any); ok {
		if entryPoint, ok := redirects["entryPoint"].(map[string]any); ok {
			if to, ok := entryPoint["to"].(string); ok {
				if to == "" {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("%s.redirections.entryPoint.to", path),
						Message: "redirect target cannot be empty",
					})
				}
			}
			if scheme, ok := entryPoint["scheme"].(string); ok {
				if scheme != "http" && scheme != "https" {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("%s.redirections.entryPoint.scheme", path),
						Message: fmt.Sprintf("invalid scheme: %s (valid: http, https)", scheme),
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateProviders(providers map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate file provider
	if file, ok := providers["file"].(map[string]any); ok {
		errors = append(errors, p.validateFileProvider(file)...)
	}

	// Validate docker provider
	if docker, ok := providers["docker"].(map[string]any); ok {
		errors = append(errors, p.validateDockerProvider(docker)...)
	}

	// Validate kubernetes provider
	if k8s, ok := providers["kubernetesIngress"].(map[string]any); ok {
		errors = append(errors, p.validateK8sProvider(k8s, "kubernetesIngress")...)
	}
	if k8s, ok := providers["kubernetesCRD"].(map[string]any); ok {
		errors = append(errors, p.validateK8sProvider(k8s, "kubernetesCRD")...)
	}

	// Validate consul provider
	if consul, ok := providers["consul"].(map[string]any); ok {
		errors = append(errors, p.validateConsulProvider(consul)...)
	}

	return errors
}

func (p *Plugin) validateFileProvider(file map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	hasFilename := false
	hasDirectory := false

	if _, ok := file["filename"].(string); ok {
		hasFilename = true
	}
	if _, ok := file["directory"].(string); ok {
		hasDirectory = true
	}

	if !hasFilename && !hasDirectory {
		errors = append(errors, plugin.ValidationError{
			Path:    "providers.file",
			Message: "either filename or directory is required",
		})
	}

	if hasFilename && hasDirectory {
		errors = append(errors, plugin.ValidationError{
			Path:    "providers.file",
			Message: "use either filename or directory, not both",
		})
	}

	return errors
}

func (p *Plugin) validateDockerProvider(docker map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate endpoint
	if endpoint, ok := docker["endpoint"].(string); ok {
		if !isValidDockerEndpoint(endpoint) {
			errors = append(errors, plugin.ValidationError{
				Path:    "providers.docker.endpoint",
				Message: fmt.Sprintf("invalid docker endpoint: %s", endpoint),
			})
		}
	}

	// Validate exposedByDefault
	if exposedByDefault, ok := docker["exposedByDefault"].(bool); ok && exposedByDefault {
		errors = append(errors, plugin.ValidationError{
			Path:    "providers.docker.exposedByDefault",
			Message: "exposedByDefault=true may expose unintended containers",
		})
	}

	return errors
}

func (p *Plugin) validateK8sProvider(k8s map[string]any, providerType string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate namespaces
	if namespaces, ok := k8s["namespaces"].([]any); ok {
		for i, ns := range namespaces {
			if nsStr, ok := ns.(string); ok {
				if !isValidK8sNamespace(nsStr) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("providers.%s.namespaces[%d]", providerType, i),
						Message: fmt.Sprintf("invalid namespace name: %s", nsStr),
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateConsulProvider(consul map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate endpoints
	if endpoints, ok := consul["endpoints"].([]any); ok {
		for i, ep := range endpoints {
			if epStr, ok := ep.(string); ok {
				if !isValidConsulEndpoint(epStr) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("providers.consul.endpoints[%d]", i),
						Message: fmt.Sprintf("invalid endpoint: %s", epStr),
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateAPI(api map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Warn if dashboard is exposed insecurely
	if dashboard, ok := api["dashboard"].(bool); ok && dashboard {
		if insecure, ok := api["insecure"].(bool); ok && insecure {
			errors = append(errors, plugin.ValidationError{
				Path:    "api.insecure",
				Message: "exposing dashboard insecurely is not recommended for production",
			})
		}
	}

	return errors
}

func (p *Plugin) validateLog(log map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate log level
	if level, ok := log["level"].(string); ok {
		validLevels := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PANIC"}
		found := false
		for _, vl := range validLevels {
			if strings.EqualFold(level, vl) {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "log.level",
				Message: fmt.Sprintf("invalid log level: %s", level),
			})
		}
	}

	// Validate format
	if format, ok := log["format"].(string); ok {
		validFormats := []string{"common", "json"}
		found := false
		for _, vf := range validFormats {
			if format == vf {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "log.format",
				Message: fmt.Sprintf("invalid log format: %s (valid: common, json)", format),
			})
		}
	}

	return errors
}

func (p *Plugin) validateCertResolvers(resolvers map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for name, resolver := range resolvers {
		resolverMap, ok := resolver.(map[string]any)
		if !ok {
			continue
		}

		// Validate ACME configuration
		if acme, ok := resolverMap["acme"].(map[string]any); ok {
			// Validate email
			if email, ok := acme["email"].(string); ok {
				if !isValidEmail(email) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("certificatesResolvers.%s.acme.email", name),
						Message: fmt.Sprintf("invalid email format: %s", email),
					})
				}
			} else {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("certificatesResolvers.%s.acme.email", name),
					Message: "email is required for ACME",
				})
			}

			// Validate storage
			if _, ok := acme["storage"].(string); !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("certificatesResolvers.%s.acme.storage", name),
					Message: "storage is required for ACME",
				})
			}

			// Validate challenge type
			hasChallenge := false
			if _, ok := acme["httpChallenge"]; ok {
				hasChallenge = true
			}
			if _, ok := acme["tlsChallenge"]; ok {
				hasChallenge = true
			}
			if _, ok := acme["dnsChallenge"]; ok {
				hasChallenge = true
			}
			if !hasChallenge {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("certificatesResolvers.%s.acme", name),
					Message: "at least one challenge type is required (httpChallenge, tlsChallenge, or dnsChallenge)",
				})
			}

			// Validate caServer if not using production
			if caServer, ok := acme["caServer"].(string); ok {
				if strings.Contains(caServer, "staging") {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("certificatesResolvers.%s.acme.caServer", name),
						Message: "using staging CA server - certificates won't be trusted",
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateTLS(tls map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate options
	if options, ok := tls["options"].(map[string]any); ok {
		for name, opt := range options {
			optMap, ok := opt.(map[string]any)
			if !ok {
				continue
			}

			// Validate minVersion
			if minVersion, ok := optMap["minVersion"].(string); ok {
				validVersions := []string{"VersionTLS10", "VersionTLS11", "VersionTLS12", "VersionTLS13"}
				found := false
				for _, vv := range validVersions {
					if minVersion == vv {
						found = true
						break
					}
				}
				if !found {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("tls.options.%s.minVersion", name),
						Message: fmt.Sprintf("invalid TLS version: %s", minVersion),
					})
				}

				// Warn about old TLS versions
				if minVersion == "VersionTLS10" || minVersion == "VersionTLS11" {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("tls.options.%s.minVersion", name),
						Message: "TLS 1.0 and 1.1 are deprecated, consider using TLS 1.2 or higher",
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateMetrics(metrics map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Check for multiple metrics backends (only one should be configured)
	count := 0
	if _, ok := metrics["prometheus"]; ok {
		count++
	}
	if _, ok := metrics["datadog"]; ok {
		count++
	}
	if _, ok := metrics["statsd"]; ok {
		count++
	}
	if _, ok := metrics["influxDB"]; ok {
		count++
	}

	if count > 1 {
		errors = append(errors, plugin.ValidationError{
			Path:    "metrics",
			Message: "only one metrics backend should be configured",
		})
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

	// Check for HTTP to HTTPS redirect without HTTPS entrypoint
	if entryPoints, ok := configMap["entryPoints"].(map[string]any); ok {
		hasHTTPS := false
		for _, ep := range entryPoints {
			if epMap, ok := ep.(map[string]any); ok {
				if address, ok := epMap["address"].(string); ok {
					if strings.Contains(address, "443") {
						hasHTTPS = true
						break
					}
				}
			}
		}

		// Check for redirects to HTTPS without HTTPS entrypoint
		for name, ep := range entryPoints {
			if epMap, ok := ep.(map[string]any); ok {
				if http, ok := epMap["http"].(map[string]any); ok {
					if redirects, ok := http["redirections"].(map[string]any); ok {
						if epRedirect, ok := redirects["entryPoint"].(map[string]any); ok {
							if scheme, ok := epRedirect["scheme"].(string); ok && scheme == "https" && !hasHTTPS {
								errors = append(errors, plugin.ValidationError{
									Path:    fmt.Sprintf("entryPoints.%s.http.redirections.entryPoint.scheme", name),
									Message: "redirecting to HTTPS but no HTTPS entryPoint is configured",
								})
							}
						}
					}
				}
			}
		}
	}

	// Check for providers
	if _, ok := configMap["providers"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "providers",
			Message: "no providers configured - Traefik won't discover any services",
		})
	}

	return errors, nil
}

// Normalize normalizes the configuration.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Normalize log level to uppercase
	if log, ok := configMap["log"].(map[string]any); ok {
		if level, ok := log["level"].(string); ok {
			log["level"] = strings.ToUpper(level)
		}
	}

	return configMap, nil
}

// ToNative converts to native Traefik configuration format (YAML).
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return yaml.Marshal(config)
}

// FromNative parses native Traefik configuration format.
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

func parsePort(address string) (int, error) {
	// Address format: :port or host:port
	if strings.HasPrefix(address, ":") {
		port, err := strconv.Atoi(strings.TrimPrefix(address, ":"))
		if err != nil || port < 1 || port > 65535 {
			return 0, fmt.Errorf("invalid port")
		}
		return port, nil
	}

	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port")
	}
	return port, nil
}

func isValidDuration(d string) bool {
	// Simple duration validation
	pattern := regexp.MustCompile(`^\d+(\.\d+)?(ns|us|µs|ms|s|m|h)?$`)
	return pattern.MatchString(d)
}

func isValidDockerEndpoint(endpoint string) bool {
	// Valid formats: unix:///var/run/docker.sock, tcp://host:port
	if strings.HasPrefix(endpoint, "unix://") {
		return true
	}
	if strings.HasPrefix(endpoint, "tcp://") {
		return true
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return true
	}
	return false
}

func isValidK8sNamespace(ns string) bool {
	// Kubernetes namespace naming rules
	pattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	return pattern.MatchString(ns) && len(ns) <= 63
}

func isValidConsulEndpoint(endpoint string) bool {
	// Consul endpoint format: host:port
	_, _, err := net.SplitHostPort(endpoint)
	return err == nil
}

func isValidEmail(email string) bool {
	// Simple email validation
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return pattern.MatchString(email)
}
