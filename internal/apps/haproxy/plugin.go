// SPDX-License-Identifier: MIT

// Package haproxy provides a plugin for HAProxy configuration management.
package haproxy

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
)

// Plugin implements the HAProxy configuration plugin.
type Plugin struct{}

// New creates a new HAProxy plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "haproxy"
}

// Version returns the supported HAProxy version range.
func (p *Plugin) Version() string {
	return ">=2.0.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "HAProxy load balancer configuration management"
}

// NativeFormat returns the native configuration format.
func (p *Plugin) NativeFormat() string {
	return "haproxy"
}

// Schema returns the configuration schema.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "global",
				Description: "Global settings",
				Required:    true,
			},
			{
				Name:        "defaults",
				Description: "Default settings for all sections",
			},
			{
				Name:        "frontend",
				Description: "Frontend definitions",
			},
			{
				Name:        "backend",
				Description: "Backend definitions",
			},
			{
				Name:        "listen",
				Description: "Combined frontend/backend definitions",
			},
			{
				Name:        "resolvers",
				Description: "DNS resolver configurations",
			},
			{
				Name:        "peers",
				Description: "Peer definitions for stick-table replication",
			},
			{
				Name:        "mailers",
				Description: "Mailer definitions for alerts",
			},
		},
		Directives: []plugin.DirectiveSchema{},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"global": map[string]any{
			"log":     "/dev/log local0",
			"maxconn": 4096,
			"user":    "haproxy",
			"group":   "haproxy",
			"daemon":  true,
			"pidfile": "/var/run/haproxy.pid",
			"stats":   map[string]any{"socket": "/var/run/haproxy.sock mode 660 level admin"},
		},
		"defaults": map[string]any{
			"log":    "global",
			"mode":   "http",
			"option": []string{"httplog", "dontlognull"},
			"timeout": map[string]any{
				"connect": "5000ms",
				"client":  "50000ms",
				"server":  "50000ms",
			},
			"retries": 3,
		},
		"frontend": []any{},
		"backend":  []any{},
	}
}

// Validate validates the configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Validate global section (required)
	if global, ok := configMap["global"].(map[string]any); ok {
		errors = append(errors, p.validateGlobal(global)...)
	} else {
		errors = append(errors, plugin.ValidationError{
			Path:    "global",
			Message: "global section is required",
		})
	}

	// Validate defaults section
	if defaults, ok := configMap["defaults"].(map[string]any); ok {
		errors = append(errors, p.validateDefaults(defaults)...)
	}

	// Validate frontends
	if frontends, ok := configMap["frontend"].([]any); ok {
		errors = append(errors, p.validateFrontends(frontends)...)
	}

	// Validate backends
	if backends, ok := configMap["backend"].([]any); ok {
		errors = append(errors, p.validateBackends(backends)...)
	}

	// Validate listen sections
	if listen, ok := configMap["listen"].([]any); ok {
		errors = append(errors, p.validateListenSections(listen)...)
	}

	return errors, nil
}

func (p *Plugin) validateGlobal(global map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate maxconn
	if maxconn, ok := global["maxconn"]; ok {
		var conn int
		switch v := maxconn.(type) {
		case int:
			conn = v
		case float64:
			conn = int(v)
		}
		if conn <= 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "global.maxconn",
				Message: "maxconn must be a positive integer",
			})
		}
	}

	// Validate nbproc/nbthread
	if nbproc, ok := global["nbproc"]; ok {
		var procs int
		switch v := nbproc.(type) {
		case int:
			procs = v
		case float64:
			procs = int(v)
		}
		if procs < 1 {
			errors = append(errors, plugin.ValidationError{
				Path:    "global.nbproc",
				Message: "nbproc must be at least 1",
			})
		}
		// nbproc is deprecated in favor of nbthread
		errors = append(errors, plugin.ValidationError{
			Path:    "global.nbproc",
			Message: "nbproc is deprecated, consider using nbthread instead",
		})
	}

	// Validate ssl-default-bind-options
	if sslOpts, ok := global["ssl-default-bind-options"].(string); ok {
		if strings.Contains(sslOpts, "no-sslv3") || strings.Contains(sslOpts, "no-tlsv10") {
			// Good practice
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    "global.ssl-default-bind-options",
				Message: "consider disabling SSLv3 and TLSv1.0 for security",
			})
		}
	}

	return errors
}

func (p *Plugin) validateDefaults(defaults map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate mode
	if mode, ok := defaults["mode"].(string); ok {
		validModes := []string{"tcp", "http", "health"}
		found := false
		for _, vm := range validModes {
			if mode == vm {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "defaults.mode",
				Message: fmt.Sprintf("invalid mode: %s (valid: tcp, http, health)", mode),
			})
		}
	}

	// Validate timeouts
	if timeout, ok := defaults["timeout"].(map[string]any); ok {
		for name, value := range timeout {
			if valStr, ok := value.(string); ok {
				if !isValidTimeout(valStr) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("defaults.timeout.%s", name),
						Message: fmt.Sprintf("invalid timeout format: %s", valStr),
					})
				}
			}
		}

		// Check for required timeouts
		requiredTimeouts := []string{"connect", "client", "server"}
		for _, rt := range requiredTimeouts {
			if _, ok := timeout[rt]; !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("defaults.timeout.%s", rt),
					Message: fmt.Sprintf("timeout %s is recommended", rt),
				})
			}
		}
	}

	// Validate balance algorithm
	if balance, ok := defaults["balance"].(string); ok {
		errors = append(errors, p.validateBalanceAlgorithm(balance, "defaults.balance")...)
	}

	return errors
}

func (p *Plugin) validateFrontends(frontends []any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)
	usedBinds := make(map[string]bool)

	for i, frontend := range frontends {
		feMap, ok := frontend.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("frontend[%d]", i),
				Message: "frontend must be an object",
			})
			continue
		}

		// Validate name
		name, ok := feMap["name"].(string)
		if !ok || name == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("frontend[%d].name", i),
				Message: "frontend name is required",
			})
		} else {
			if usedNames[name] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("frontend[%d].name", i),
					Message: fmt.Sprintf("duplicate frontend name: %s", name),
				})
			}
			usedNames[name] = true
		}

		// Validate bind
		if bind, ok := feMap["bind"].(string); ok {
			if !isValidBind(bind) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("frontend[%d].bind", i),
					Message: fmt.Sprintf("invalid bind address: %s", bind),
				})
			}
			if usedBinds[bind] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("frontend[%d].bind", i),
					Message: fmt.Sprintf("duplicate bind address: %s", bind),
				})
			}
			usedBinds[bind] = true
		} else if binds, ok := feMap["bind"].([]any); ok {
			for j, b := range binds {
				if bindStr, ok := b.(string); ok {
					if !isValidBind(bindStr) {
						errors = append(errors, plugin.ValidationError{
							Path:    fmt.Sprintf("frontend[%d].bind[%d]", i, j),
							Message: fmt.Sprintf("invalid bind address: %s", bindStr),
						})
					}
				}
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("frontend[%d].bind", i),
				Message: "frontend bind is required",
			})
		}

		// Validate default_backend
		if _, ok := feMap["default_backend"]; !ok {
			// Check if there are use_backend rules
			if _, hasUseBackend := feMap["use_backend"]; !hasUseBackend {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("frontend[%d].default_backend", i),
					Message: "frontend should have default_backend or use_backend rules",
				})
			}
		}

		// Validate mode if specified
		if mode, ok := feMap["mode"].(string); ok {
			validModes := []string{"tcp", "http"}
			found := false
			for _, vm := range validModes {
				if mode == vm {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("frontend[%d].mode", i),
					Message: fmt.Sprintf("invalid mode: %s", mode),
				})
			}
		}

		// Validate ACLs
		if acls, ok := feMap["acl"].([]any); ok {
			errors = append(errors, p.validateACLs(acls, fmt.Sprintf("frontend[%d]", i))...)
		}
	}

	return errors
}

func (p *Plugin) validateBackends(backends []any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)

	for i, backend := range backends {
		beMap, ok := backend.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("backend[%d]", i),
				Message: "backend must be an object",
			})
			continue
		}

		// Validate name
		name, ok := beMap["name"].(string)
		if !ok || name == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("backend[%d].name", i),
				Message: "backend name is required",
			})
		} else {
			if usedNames[name] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("backend[%d].name", i),
					Message: fmt.Sprintf("duplicate backend name: %s", name),
				})
			}
			usedNames[name] = true
		}

		// Validate balance
		if balance, ok := beMap["balance"].(string); ok {
			errors = append(errors, p.validateBalanceAlgorithm(balance, fmt.Sprintf("backend[%d].balance", i))...)
		}

		// Validate servers
		if servers, ok := beMap["server"].([]any); ok {
			errors = append(errors, p.validateServers(servers, fmt.Sprintf("backend[%d]", i))...)
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("backend[%d].server", i),
				Message: "backend should have at least one server",
			})
		}

		// Validate health check
		if _, ok := beMap["option"]; ok {
			// Check for health check options
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("backend[%d]", i),
				Message: "consider adding health checks to backend",
			})
		}
	}

	return errors
}

func (p *Plugin) validateListenSections(listen []any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)

	for i, section := range listen {
		listenMap, ok := section.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("listen[%d]", i),
				Message: "listen section must be an object",
			})
			continue
		}

		// Validate name
		name, ok := listenMap["name"].(string)
		if !ok || name == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("listen[%d].name", i),
				Message: "listen section name is required",
			})
		} else {
			if usedNames[name] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("listen[%d].name", i),
					Message: fmt.Sprintf("duplicate listen section name: %s", name),
				})
			}
			usedNames[name] = true
		}

		// Validate bind
		if _, ok := listenMap["bind"]; !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("listen[%d].bind", i),
				Message: "listen section bind is required",
			})
		}

		// Validate servers
		if servers, ok := listenMap["server"].([]any); ok {
			errors = append(errors, p.validateServers(servers, fmt.Sprintf("listen[%d]", i))...)
		}
	}

	return errors
}

func (p *Plugin) validateBalanceAlgorithm(balance string, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Extract algorithm name (may have parameters)
	parts := strings.Fields(balance)
	if len(parts) == 0 {
		return errors
	}

	algorithm := parts[0]
	validAlgorithms := []string{
		"roundrobin", "static-rr", "leastconn", "first",
		"source", "uri", "url_param", "hdr", "random",
		"rdp-cookie",
	}

	found := false
	for _, va := range validAlgorithms {
		if algorithm == va {
			found = true
			break
		}
	}

	if !found {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: fmt.Sprintf("unknown balance algorithm: %s", algorithm),
		})
	}

	return errors
}

func (p *Plugin) validateServers(servers []any, parentPath string) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)

	for i, server := range servers {
		serverMap, ok := server.(map[string]any)
		if !ok {
			// Server might be a string in simple format
			if serverStr, ok := server.(string); ok {
				if !isValidServerLine(serverStr) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("%s.server[%d]", parentPath, i),
						Message: fmt.Sprintf("invalid server definition: %s", serverStr),
					})
				}
			}
			continue
		}

		// Validate name
		name, ok := serverMap["name"].(string)
		if !ok || name == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.server[%d].name", parentPath, i),
				Message: "server name is required",
			})
		} else {
			if usedNames[name] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.server[%d].name", parentPath, i),
					Message: fmt.Sprintf("duplicate server name: %s", name),
				})
			}
			usedNames[name] = true
		}

		// Validate address
		if address, ok := serverMap["address"].(string); ok {
			if !isValidServerAddress(address) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.server[%d].address", parentPath, i),
					Message: fmt.Sprintf("invalid server address: %s", address),
				})
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.server[%d].address", parentPath, i),
				Message: "server address is required",
			})
		}

		// Validate weight if present
		if weight, ok := serverMap["weight"]; ok {
			var w int
			switch v := weight.(type) {
			case int:
				w = v
			case float64:
				w = int(v)
			}
			if w < 0 || w > 256 {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.server[%d].weight", parentPath, i),
					Message: "server weight must be between 0 and 256",
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateACLs(acls []any, parentPath string) []plugin.ValidationError {
	var errors []plugin.ValidationError
	usedNames := make(map[string]bool)

	for i, acl := range acls {
		aclMap, ok := acl.(map[string]any)
		if !ok {
			continue
		}

		// Validate name
		name, ok := aclMap["name"].(string)
		if !ok || name == "" {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.acl[%d].name", parentPath, i),
				Message: "ACL name is required",
			})
		} else {
			if usedNames[name] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("%s.acl[%d].name", parentPath, i),
					Message: fmt.Sprintf("duplicate ACL name: %s", name),
				})
			}
			usedNames[name] = true
		}

		// Validate criterion
		if _, ok := aclMap["criterion"].(string); !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.acl[%d].criterion", parentPath, i),
				Message: "ACL criterion is required",
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

	// Collect all backend names
	backendNames := make(map[string]bool)
	if backends, ok := configMap["backend"].([]any); ok {
		for _, backend := range backends {
			if beMap, ok := backend.(map[string]any); ok {
				if name, ok := beMap["name"].(string); ok {
					backendNames[name] = true
				}
			}
		}
	}

	// Also collect listen section names (they can be used as backends)
	if listen, ok := configMap["listen"].([]any); ok {
		for _, section := range listen {
			if listenMap, ok := section.(map[string]any); ok {
				if name, ok := listenMap["name"].(string); ok {
					backendNames[name] = true
				}
			}
		}
	}

	// Check that all referenced backends exist
	if frontends, ok := configMap["frontend"].([]any); ok {
		for i, frontend := range frontends {
			feMap, ok := frontend.(map[string]any)
			if !ok {
				continue
			}

			// Check default_backend
			if defaultBackend, ok := feMap["default_backend"].(string); ok {
				if !backendNames[defaultBackend] {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("frontend[%d].default_backend", i),
						Message: fmt.Sprintf("backend '%s' does not exist", defaultBackend),
					})
				}
			}

			// Check use_backend rules
			if useBackends, ok := feMap["use_backend"].([]any); ok {
				for j, ub := range useBackends {
					if ubMap, ok := ub.(map[string]any); ok {
						if backend, ok := ubMap["backend"].(string); ok {
							if !backendNames[backend] {
								errors = append(errors, plugin.ValidationError{
									Path:    fmt.Sprintf("frontend[%d].use_backend[%d].backend", i, j),
									Message: fmt.Sprintf("backend '%s' does not exist", backend),
								})
							}
						}
					}
				}
			}
		}
	}

	// Security warnings
	if global, ok := configMap["global"].(map[string]any); ok {
		// Check if running as root
		if user, ok := global["user"].(string); ok && user == "root" {
			errors = append(errors, plugin.ValidationError{
				Path:    "global.user",
				Message: "running HAProxy as root is not recommended",
			})
		}

		// Check for stats socket security
		if stats, ok := global["stats"].(map[string]any); ok {
			if socket, ok := stats["socket"].(string); ok {
				if !strings.Contains(socket, "level") {
					errors = append(errors, plugin.ValidationError{
						Path:    "global.stats.socket",
						Message: "consider setting access level for stats socket",
					})
				}
			}
		}
	}

	return errors, nil
}

// Normalize normalizes the configuration.
func (p *Plugin) Normalize(config any) (any, error) {
	// HAProxy config doesn't need much normalization
	return config, nil
}

// ToNative converts to native HAProxy configuration format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var sb strings.Builder

	// Write global section
	if global, ok := configMap["global"].(map[string]any); ok {
		sb.WriteString("global\n")
		writeSection(&sb, global, 1)
		sb.WriteString("\n")
	}

	// Write defaults section
	if defaults, ok := configMap["defaults"].(map[string]any); ok {
		sb.WriteString("defaults\n")
		writeSection(&sb, defaults, 1)
		sb.WriteString("\n")
	}

	// Write frontend sections
	if frontends, ok := configMap["frontend"].([]any); ok {
		for _, frontend := range frontends {
			if feMap, ok := frontend.(map[string]any); ok {
				name := feMap["name"].(string)
				sb.WriteString(fmt.Sprintf("frontend %s\n", name))
				writeSection(&sb, feMap, 1)
				sb.WriteString("\n")
			}
		}
	}

	// Write backend sections
	if backends, ok := configMap["backend"].([]any); ok {
		for _, backend := range backends {
			if beMap, ok := backend.(map[string]any); ok {
				name := beMap["name"].(string)
				sb.WriteString(fmt.Sprintf("backend %s\n", name))
				writeSection(&sb, beMap, 1)
				sb.WriteString("\n")
			}
		}
	}

	// Write listen sections
	if listen, ok := configMap["listen"].([]any); ok {
		for _, section := range listen {
			if listenMap, ok := section.(map[string]any); ok {
				name := listenMap["name"].(string)
				sb.WriteString(fmt.Sprintf("listen %s\n", name))
				writeSection(&sb, listenMap, 1)
				sb.WriteString("\n")
			}
		}
	}

	return []byte(sb.String()), nil
}

func writeSection(sb *strings.Builder, section map[string]any, indent int) {
	prefix := strings.Repeat("    ", indent)

	for key, value := range section {
		if key == "name" {
			continue // Skip name, it's already in the section header
		}

		switch v := value.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, key, v))
		case bool:
			if v {
				sb.WriteString(fmt.Sprintf("%s%s\n", prefix, key))
			}
		case int, float64:
			sb.WriteString(fmt.Sprintf("%s%s %v\n", prefix, key, v))
		case []any:
			for _, item := range v {
				switch it := item.(type) {
				case string:
					sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, key, it))
				case map[string]any:
					// Handle structured items like servers
					if key == "server" {
						if name, ok := it["name"].(string); ok {
							if addr, ok := it["address"].(string); ok {
								line := fmt.Sprintf("%sserver %s %s", prefix, name, addr)
								// Add optional parameters
								if check, ok := it["check"].(bool); ok && check {
									line += " check"
								}
								if weight, ok := it["weight"]; ok {
									line += fmt.Sprintf(" weight %v", weight)
								}
								sb.WriteString(line + "\n")
							}
						}
					}
				}
			}
		case map[string]any:
			// Handle nested maps like timeout
			if key == "timeout" {
				for tk, tv := range v {
					sb.WriteString(fmt.Sprintf("%stimeout %s %v\n", prefix, tk, tv))
				}
			} else if key == "option" {
				for ok, ov := range v {
					if ob, isBool := ov.(bool); isBool && ob {
						sb.WriteString(fmt.Sprintf("%soption %s\n", prefix, ok))
					} else {
						sb.WriteString(fmt.Sprintf("%soption %s %v\n", prefix, ok, ov))
					}
				}
			} else if key == "stats" {
				for sk, sv := range v {
					sb.WriteString(fmt.Sprintf("%sstats %s %v\n", prefix, sk, sv))
				}
			} else {
				// Generic nested map
				for nk, nv := range v {
					sb.WriteString(fmt.Sprintf("%s%s %s %v\n", prefix, key, nk, nv))
				}
			}
		}
	}
}

// FromNative parses native HAProxy configuration format.
func (p *Plugin) FromNative(data []byte) (any, error) {
	// This is a simplified parser - a full parser would be more complex
	config := make(map[string]any)
	lines := strings.Split(string(data), "\n")

	var currentSection string
	var currentName string
	var currentData map[string]any

	frontends := []any{}
	backends := []any{}
	listen := []any{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "global") {
			if currentSection != "" && currentData != nil {
				p.saveSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
			}
			currentSection = "global"
			currentName = ""
			currentData = make(map[string]any)
		} else if strings.HasPrefix(line, "defaults") {
			if currentSection != "" && currentData != nil {
				p.saveSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
			}
			currentSection = "defaults"
			currentName = ""
			currentData = make(map[string]any)
		} else if strings.HasPrefix(line, "frontend ") {
			if currentSection != "" && currentData != nil {
				p.saveSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
			}
			currentSection = "frontend"
			currentName = strings.TrimPrefix(line, "frontend ")
			currentData = map[string]any{"name": currentName}
		} else if strings.HasPrefix(line, "backend ") {
			if currentSection != "" && currentData != nil {
				p.saveSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
			}
			currentSection = "backend"
			currentName = strings.TrimPrefix(line, "backend ")
			currentData = map[string]any{"name": currentName}
		} else if strings.HasPrefix(line, "listen ") {
			if currentSection != "" && currentData != nil {
				p.saveSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
			}
			currentSection = "listen"
			currentName = strings.TrimPrefix(line, "listen ")
			currentData = map[string]any{"name": currentName}
		} else if currentData != nil {
			// Parse directive
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 1 {
				currentData[parts[0]] = true
			} else {
				currentData[parts[0]] = parts[1]
			}
		}
	}

	// Save last section
	if currentSection != "" && currentData != nil {
		p.saveSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
	}

	if len(frontends) > 0 {
		config["frontend"] = frontends
	}
	if len(backends) > 0 {
		config["backend"] = backends
	}
	if len(listen) > 0 {
		config["listen"] = listen
	}

	return config, nil
}

func (p *Plugin) saveSection(config map[string]any, section, name string, data map[string]any, frontends, backends, listen *[]any) {
	switch section {
	case "global":
		config["global"] = data
	case "defaults":
		config["defaults"] = data
	case "frontend":
		*frontends = append(*frontends, data)
	case "backend":
		*backends = append(*backends, data)
	case "listen":
		*listen = append(*listen, data)
	}
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

func isValidTimeout(timeout string) bool {
	// HAProxy timeout format: number + optional unit (us, ms, s, m, h, d)
	pattern := regexp.MustCompile(`^\d+(\.\d+)?(us|ms|s|m|h|d)?$`)
	return pattern.MatchString(timeout)
}

func isValidBind(bind string) bool {
	// Bind can be: IP:port, :port, unix@/path, or *:port
	if strings.HasPrefix(bind, "unix@") {
		return true
	}
	if strings.HasPrefix(bind, "*:") {
		_, err := strconv.Atoi(strings.TrimPrefix(bind, "*:"))
		return err == nil
	}
	if strings.HasPrefix(bind, ":") {
		_, err := strconv.Atoi(strings.TrimPrefix(bind, ":"))
		return err == nil
	}

	// IP:port format
	host, portStr, err := net.SplitHostPort(bind)
	if err != nil {
		return false
	}
	if host != "" && net.ParseIP(host) == nil {
		// Not a valid IP, might be a hostname
		// Accept it anyway
	}
	_, err = strconv.Atoi(portStr)
	return err == nil
}

func isValidServerLine(server string) bool {
	// Simple validation: should have at least name and address
	parts := strings.Fields(server)
	return len(parts) >= 2
}

func isValidServerAddress(address string) bool {
	// Address can be IP:port or hostname:port
	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	_, err = strconv.Atoi(portStr)
	return err == nil
}
