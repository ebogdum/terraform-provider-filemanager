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

		errors = append(errors, validateFrontendName(feMap, i, usedNames)...)
		errors = append(errors, validateFrontendBind(feMap, i, usedBinds)...)
		errors = append(errors, validateFrontendDefaultBackend(feMap, i)...)
		errors = append(errors, validateFrontendMode(feMap, i)...)
		errors = append(errors, p.validateFrontendACLs(feMap, i)...)
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
		errors = append(errors, validateServerEntry(server, i, parentPath, usedNames)...)
	}

	return errors
}

func validateFrontendName(feMap map[string]any, index int, usedNames map[string]bool) []plugin.ValidationError {
	path := fmt.Sprintf("frontend[%d].name", index)
	name, ok := feMap["name"].(string)
	if !ok || name == "" {
		return []plugin.ValidationError{{
			Path:    path,
			Message: "frontend name is required",
		}}
	}

	if usedNames[name] {
		return []plugin.ValidationError{{
			Path:    path,
			Message: fmt.Sprintf("duplicate frontend name: %s", name),
		}}
	}

	usedNames[name] = true
	return nil
}

func validateFrontendBind(feMap map[string]any, index int, usedBinds map[string]bool) []plugin.ValidationError {
	if bind, ok := feMap["bind"].(string); ok {
		return validateFrontendBindString(bind, index, usedBinds)
	}

	if binds, ok := feMap["bind"].([]any); ok {
		return validateFrontendBindList(binds, index)
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("frontend[%d].bind", index),
		Message: "frontend bind is required",
	}}
}

func validateFrontendBindString(bind string, index int, usedBinds map[string]bool) []plugin.ValidationError {
	var errors []plugin.ValidationError
	path := fmt.Sprintf("frontend[%d].bind", index)

	if !isValidBind(bind) {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: fmt.Sprintf("invalid bind address: %s", bind),
		})
	}
	if usedBinds[bind] {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: fmt.Sprintf("duplicate bind address: %s", bind),
		})
	}
	usedBinds[bind] = true
	return errors
}

func validateFrontendBindList(binds []any, index int) []plugin.ValidationError {
	var errors []plugin.ValidationError
	for j, bind := range binds {
		bindStr, ok := bind.(string)
		if !ok || isValidBind(bindStr) {
			continue
		}
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("frontend[%d].bind[%d]", index, j),
			Message: fmt.Sprintf("invalid bind address: %s", bindStr),
		})
	}
	return errors
}

func validateFrontendDefaultBackend(feMap map[string]any, index int) []plugin.ValidationError {
	if _, hasDefaultBackend := feMap["default_backend"]; hasDefaultBackend {
		return nil
	}
	if _, hasUseBackend := feMap["use_backend"]; hasUseBackend {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("frontend[%d].default_backend", index),
		Message: "frontend should have default_backend or use_backend rules",
	}}
}

func validateFrontendMode(feMap map[string]any, index int) []plugin.ValidationError {
	mode, ok := feMap["mode"].(string)
	if !ok {
		return nil
	}

	validModes := []string{"tcp", "http"}
	for _, validMode := range validModes {
		if mode == validMode {
			return nil
		}
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("frontend[%d].mode", index),
		Message: fmt.Sprintf("invalid mode: %s", mode),
	}}
}

func (p *Plugin) validateFrontendACLs(feMap map[string]any, index int) []plugin.ValidationError {
	acls, ok := feMap["acl"].([]any)
	if !ok {
		return nil
	}
	return p.validateACLs(acls, fmt.Sprintf("frontend[%d]", index))
}

func validateServerEntry(server any, index int, parentPath string, usedNames map[string]bool) []plugin.ValidationError {
	serverMap, ok := server.(map[string]any)
	if !ok {
		if serverStr, ok := server.(string); ok && !isValidServerLine(serverStr) {
			return []plugin.ValidationError{{
				Path:    fmt.Sprintf("%s.server[%d]", parentPath, index),
				Message: fmt.Sprintf("invalid server definition: %s", serverStr),
			}}
		}
		return nil
	}

	var errors []plugin.ValidationError
	errors = append(errors, validateServerName(serverMap, index, parentPath, usedNames)...)
	errors = append(errors, validateServerAddress(serverMap, index, parentPath)...)
	errors = append(errors, validateServerWeight(serverMap, index, parentPath)...)
	return errors
}

func validateServerName(serverMap map[string]any, index int, parentPath string, usedNames map[string]bool) []plugin.ValidationError {
	path := fmt.Sprintf("%s.server[%d].name", parentPath, index)
	name, ok := serverMap["name"].(string)
	if !ok || name == "" {
		return []plugin.ValidationError{{
			Path:    path,
			Message: "server name is required",
		}}
	}

	if usedNames[name] {
		return []plugin.ValidationError{{
			Path:    path,
			Message: fmt.Sprintf("duplicate server name: %s", name),
		}}
	}

	usedNames[name] = true
	return nil
}

func validateServerAddress(serverMap map[string]any, index int, parentPath string) []plugin.ValidationError {
	path := fmt.Sprintf("%s.server[%d].address", parentPath, index)
	address, ok := serverMap["address"].(string)
	if !ok {
		return []plugin.ValidationError{{
			Path:    path,
			Message: "server address is required",
		}}
	}
	if isValidServerAddress(address) {
		return nil
	}
	return []plugin.ValidationError{{
		Path:    path,
		Message: fmt.Sprintf("invalid server address: %s", address),
	}}
}

func validateServerWeight(serverMap map[string]any, index int, parentPath string) []plugin.ValidationError {
	weight, ok := serverMap["weight"]
	if !ok {
		return nil
	}

	w, ok := haproxyAnyToInt(weight)
	if !ok {
		w = 0
	}
	if w >= 0 && w <= 256 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("%s.server[%d].weight", parentPath, index),
		Message: "server weight must be between 0 and 256",
	}}
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
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	backendNames := collectBackendNames(configMap)

	var errors []plugin.ValidationError
	errors = append(errors, validateFrontendBackendReferences(configMap, backendNames)...)
	errors = append(errors, validateHAProxyGlobalSecurity(configMap)...)
	return errors, nil
}

func collectBackendNames(configMap map[string]any) map[string]bool {
	backendNames := make(map[string]bool)

	backends, ok := configMap["backend"].([]any)
	if ok {
		for _, backend := range backends {
			if backendMap, ok := backend.(map[string]any); ok {
				if name, ok := backendMap["name"].(string); ok {
					backendNames[name] = true
				}
			}
		}
	}

	listenSections, ok := configMap["listen"].([]any)
	if ok {
		for _, section := range listenSections {
			if listenMap, ok := section.(map[string]any); ok {
				if name, ok := listenMap["name"].(string); ok {
					backendNames[name] = true
				}
			}
		}
	}

	return backendNames
}

func validateFrontendBackendReferences(configMap map[string]any, backendNames map[string]bool) []plugin.ValidationError {
	frontends, ok := configMap["frontend"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for i, frontend := range frontends {
		feMap, ok := frontend.(map[string]any)
		if !ok {
			continue
		}
		errors = append(errors, validateFrontendDefaultBackendReference(feMap, i, backendNames)...)
		errors = append(errors, validateFrontendUseBackendReferences(feMap, i, backendNames)...)
	}
	return errors
}

func validateFrontendDefaultBackendReference(feMap map[string]any, index int, backendNames map[string]bool) []plugin.ValidationError {
	defaultBackend, ok := feMap["default_backend"].(string)
	if !ok || backendNames[defaultBackend] {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    fmt.Sprintf("frontend[%d].default_backend", index),
		Message: fmt.Sprintf("backend '%s' does not exist", defaultBackend),
	}}
}

func validateFrontendUseBackendReferences(feMap map[string]any, index int, backendNames map[string]bool) []plugin.ValidationError {
	useBackends, ok := feMap["use_backend"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for j, useBackend := range useBackends {
		useBackendMap, ok := useBackend.(map[string]any)
		if !ok {
			continue
		}
		backend, ok := useBackendMap["backend"].(string)
		if !ok || backendNames[backend] {
			continue
		}
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("frontend[%d].use_backend[%d].backend", index, j),
			Message: fmt.Sprintf("backend '%s' does not exist", backend),
		})
	}
	return errors
}

func validateHAProxyGlobalSecurity(configMap map[string]any) []plugin.ValidationError {
	global, ok := configMap["global"].(map[string]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	if user, ok := global["user"].(string); ok && user == "root" {
		errors = append(errors, plugin.ValidationError{
			Path:    "global.user",
			Message: "running HAProxy as root is not recommended",
		})
	}

	stats, ok := global["stats"].(map[string]any)
	if !ok {
		return errors
	}

	socket, ok := stats["socket"].(string)
	if ok && !strings.Contains(socket, "level") {
		errors = append(errors, plugin.ValidationError{
			Path:    "global.stats.socket",
			Message: "consider setting access level for stats socket",
		})
	}

	return errors
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
			continue
		}
		writeSectionEntry(sb, prefix, key, value)
	}
}

func writeSectionEntry(sb *strings.Builder, prefix, key string, value any) {
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
		writeSectionList(sb, prefix, key, v)
	case map[string]any:
		writeSectionMap(sb, prefix, key, v)
	}
}

func writeSectionList(sb *strings.Builder, prefix, key string, items []any) {
	for _, item := range items {
		writeSectionListItem(sb, prefix, key, item)
	}
}

func writeSectionListItem(sb *strings.Builder, prefix, key string, item any) {
	switch typedItem := item.(type) {
	case string:
		sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, key, typedItem))
	case map[string]any:
		if key == "server" {
			writeSectionServerItem(sb, prefix, typedItem)
		}
	}
}

func writeSectionServerItem(sb *strings.Builder, prefix string, server map[string]any) {
	name, hasName := server["name"].(string)
	address, hasAddress := server["address"].(string)
	if !hasName || !hasAddress {
		return
	}

	line := fmt.Sprintf("%sserver %s %s", prefix, name, address)
	if check, ok := server["check"].(bool); ok && check {
		line += " check"
	}
	if weight, ok := server["weight"]; ok {
		line += fmt.Sprintf(" weight %v", weight)
	}
	sb.WriteString(line + "\n")
}

func writeSectionMap(sb *strings.Builder, prefix, key string, values map[string]any) {
	switch key {
	case "timeout":
		writeSectionNamedMap(sb, prefix, "timeout", values)
	case "option":
		writeSectionOptions(sb, prefix, values)
	case "stats":
		writeSectionNamedMap(sb, prefix, "stats", values)
	default:
		writeSectionGenericMap(sb, prefix, key, values)
	}
}

func writeSectionNamedMap(sb *strings.Builder, prefix, directive string, values map[string]any) {
	for nestedKey, nestedValue := range values {
		sb.WriteString(fmt.Sprintf("%s%s %s %v\n", prefix, directive, nestedKey, nestedValue))
	}
}

func writeSectionOptions(sb *strings.Builder, prefix string, values map[string]any) {
	for optionKey, optionValue := range values {
		if enabled, isBool := optionValue.(bool); isBool && enabled {
			sb.WriteString(fmt.Sprintf("%soption %s\n", prefix, optionKey))
			continue
		}
		sb.WriteString(fmt.Sprintf("%soption %s %v\n", prefix, optionKey, optionValue))
	}
}

func writeSectionGenericMap(sb *strings.Builder, prefix, key string, values map[string]any) {
	for nestedKey, nestedValue := range values {
		sb.WriteString(fmt.Sprintf("%s%s %s %v\n", prefix, key, nestedKey, nestedValue))
	}
}

// FromNative parses native HAProxy configuration format.
func (p *Plugin) FromNative(data []byte) (any, error) {
	config := make(map[string]any)
	var currentSection string
	var currentName string
	var currentData map[string]any
	var frontends []any
	var backends []any
	var listen []any

	lines := strings.Split(string(data), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if shouldSkipHAProxyLine(line) {
			continue
		}

		section, name, isHeader := parseHAProxySectionHeader(line)
		if isHeader {
			p.persistSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
			currentSection, currentName, currentData = startHAProxySection(section, name)
			continue
		}

		if currentData != nil {
			parseHAProxyDirective(line, currentData)
		}
	}

	p.persistSection(config, currentSection, currentName, currentData, &frontends, &backends, &listen)
	setHAProxyParsedSections(config, frontends, backends, listen)
	return config, nil
}

func shouldSkipHAProxyLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

func parseHAProxySectionHeader(line string) (section string, name string, ok bool) {
	switch {
	case strings.HasPrefix(line, "global"):
		return "global", "", true
	case strings.HasPrefix(line, "defaults"):
		return "defaults", "", true
	case strings.HasPrefix(line, "frontend "):
		return "frontend", strings.TrimPrefix(line, "frontend "), true
	case strings.HasPrefix(line, "backend "):
		return "backend", strings.TrimPrefix(line, "backend "), true
	case strings.HasPrefix(line, "listen "):
		return "listen", strings.TrimPrefix(line, "listen "), true
	default:
		return "", "", false
	}
}

func startHAProxySection(section, name string) (string, string, map[string]any) {
	switch section {
	case "frontend", "backend", "listen":
		return section, name, map[string]any{"name": name}
	default:
		return section, name, make(map[string]any)
	}
}

func parseHAProxyDirective(line string, currentData map[string]any) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 1 {
		currentData[parts[0]] = true
		return
	}
	currentData[parts[0]] = parts[1]
}

func (p *Plugin) persistSection(
	config map[string]any,
	currentSection string,
	currentName string,
	currentData map[string]any,
	frontends *[]any,
	backends *[]any,
	listen *[]any,
) {
	if currentSection == "" || currentData == nil {
		return
	}
	p.saveSection(config, currentSection, currentName, currentData, frontends, backends, listen)
}

func setHAProxyParsedSections(config map[string]any, frontends []any, backends []any, listen []any) {
	if len(frontends) > 0 {
		config["frontend"] = frontends
	}
	if len(backends) > 0 {
		config["backend"] = backends
	}
	if len(listen) > 0 {
		config["listen"] = listen
	}
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

func haproxyAnyToInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
