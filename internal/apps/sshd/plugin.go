// SPDX-License-Identifier: MIT

// Package sshd provides a plugin for OpenSSH server configuration management.
package sshd

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
)

// Plugin implements the OpenSSH server configuration plugin.
type Plugin struct{}

// New creates a new SSHD plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "sshd"
}

// Version returns the supported OpenSSH version range.
func (p *Plugin) Version() string {
	return ">=7.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "OpenSSH server (sshd_config) configuration management"
}

// NativeFormat returns the native configuration format.
func (p *Plugin) NativeFormat() string {
	return "sshd_config"
}

// Schema returns the configuration schema.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "Match",
				Description: "Conditional configuration blocks",
			},
		},
		Directives: []plugin.DirectiveSchema{
			// Network
			{Name: "Port", Description: "SSH port", Type: "int"},
			{Name: "ListenAddress", Description: "Address to listen on", Type: "string"},
			{Name: "AddressFamily", Description: "Address family", Type: "string"},
			// Authentication
			{Name: "PermitRootLogin", Description: "Allow root login", Type: "string"},
			{Name: "PubkeyAuthentication", Description: "Allow public key auth", Type: "bool"},
			{Name: "PasswordAuthentication", Description: "Allow password auth", Type: "bool"},
			{Name: "ChallengeResponseAuthentication", Description: "Challenge-response auth", Type: "bool"},
			{Name: "KbdInteractiveAuthentication", Description: "Keyboard-interactive auth", Type: "bool"},
			{Name: "UsePAM", Description: "Use PAM", Type: "bool"},
			{Name: "AuthenticationMethods", Description: "Required authentication methods", Type: "string"},
			{Name: "MaxAuthTries", Description: "Max authentication attempts", Type: "int"},
			{Name: "LoginGraceTime", Description: "Login grace time", Type: "string"},
			// Access Control
			{Name: "AllowUsers", Description: "Allowed users", Type: "string"},
			{Name: "DenyUsers", Description: "Denied users", Type: "string"},
			{Name: "AllowGroups", Description: "Allowed groups", Type: "string"},
			{Name: "DenyGroups", Description: "Denied groups", Type: "string"},
			// Security
			{Name: "Protocol", Description: "SSH protocol version", Type: "int"},
			{Name: "HostKey", Description: "Host key file", Type: "string"},
			{Name: "Ciphers", Description: "Allowed ciphers", Type: "string"},
			{Name: "MACs", Description: "Allowed MACs", Type: "string"},
			{Name: "KexAlgorithms", Description: "Key exchange algorithms", Type: "string"},
			{Name: "HostKeyAlgorithms", Description: "Host key algorithms", Type: "string"},
			// Forwarding
			{Name: "AllowTcpForwarding", Description: "Allow TCP forwarding", Type: "bool"},
			{Name: "AllowAgentForwarding", Description: "Allow agent forwarding", Type: "bool"},
			{Name: "X11Forwarding", Description: "Allow X11 forwarding", Type: "bool"},
			{Name: "GatewayPorts", Description: "Allow gateway ports", Type: "bool"},
			{Name: "PermitTunnel", Description: "Permit tunnel", Type: "string"},
			// Session
			{Name: "ClientAliveInterval", Description: "Client alive interval", Type: "int"},
			{Name: "ClientAliveCountMax", Description: "Client alive count max", Type: "int"},
			{Name: "MaxSessions", Description: "Max sessions", Type: "int"},
			{Name: "MaxStartups", Description: "Max startups", Type: "string"},
			// Logging
			{Name: "LogLevel", Description: "Log level", Type: "string"},
			{Name: "SyslogFacility", Description: "Syslog facility", Type: "string"},
			// Other
			{Name: "Subsystem", Description: "Subsystem configuration", Type: "string"},
			{Name: "Banner", Description: "Banner file", Type: "string"},
			{Name: "PrintMotd", Description: "Print MOTD", Type: "bool"},
			{Name: "PrintLastLog", Description: "Print last log", Type: "bool"},
			{Name: "TCPKeepAlive", Description: "TCP keepalive", Type: "bool"},
			{Name: "PermitUserEnvironment", Description: "Permit user environment", Type: "bool"},
			{Name: "StrictModes", Description: "Strict modes", Type: "bool"},
		},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"Port":                   22,
		"Protocol":               2,
		"PermitRootLogin":        "prohibit-password",
		"PubkeyAuthentication":   true,
		"PasswordAuthentication": false,
		"UsePAM":                 true,
		"X11Forwarding":          false,
		"PrintMotd":              false,
		"TCPKeepAlive":           true,
		"ClientAliveInterval":    300,
		"ClientAliveCountMax":    3,
		"MaxAuthTries":           3,
		"LogLevel":               "INFO",
		"Subsystem": map[string]string{
			"sftp": "/usr/lib/openssh/sftp-server",
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

	// Validate Port
	if port, ok := configMap["Port"]; ok {
		var portNum int
		switch v := port.(type) {
		case int:
			portNum = v
		case float64:
			portNum = int(v)
		}
		if portNum < 1 || portNum > 65535 {
			errors = append(errors, plugin.ValidationError{
				Path:    "Port",
				Message: fmt.Sprintf("port must be between 1 and 65535, got: %d", portNum),
			})
		}
	}

	// Validate PermitRootLogin
	if permitRoot, ok := configMap["PermitRootLogin"].(string); ok {
		validValues := []string{"yes", "no", "prohibit-password", "forced-commands-only", "without-password"}
		found := false
		for _, vv := range validValues {
			if permitRoot == vv {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "PermitRootLogin",
				Message: fmt.Sprintf("invalid value: %s (valid: yes, no, prohibit-password, forced-commands-only)", permitRoot),
			})
		}
	}

	// Validate LogLevel
	if logLevel, ok := configMap["LogLevel"].(string); ok {
		validLevels := []string{"QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3"}
		found := false
		for _, vl := range validLevels {
			if logLevel == vl {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "LogLevel",
				Message: fmt.Sprintf("invalid log level: %s", logLevel),
			})
		}
	}

	// Validate SyslogFacility
	if syslogFacility, ok := configMap["SyslogFacility"].(string); ok {
		validFacilities := []string{
			"DAEMON", "USER", "AUTH", "AUTHPRIV", "LOCAL0", "LOCAL1",
			"LOCAL2", "LOCAL3", "LOCAL4", "LOCAL5", "LOCAL6", "LOCAL7",
		}
		found := false
		for _, vf := range validFacilities {
			if syslogFacility == vf {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "SyslogFacility",
				Message: fmt.Sprintf("invalid syslog facility: %s", syslogFacility),
			})
		}
	}

	// Validate AddressFamily
	if addressFamily, ok := configMap["AddressFamily"].(string); ok {
		validFamilies := []string{"any", "inet", "inet6"}
		found := false
		for _, vf := range validFamilies {
			if addressFamily == vf {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "AddressFamily",
				Message: fmt.Sprintf("invalid address family: %s (valid: any, inet, inet6)", addressFamily),
			})
		}
	}

	// Validate ListenAddress
	if listenAddr, ok := configMap["ListenAddress"].(string); ok {
		if !isValidListenAddress(listenAddr) {
			errors = append(errors, plugin.ValidationError{
				Path:    "ListenAddress",
				Message: fmt.Sprintf("invalid listen address: %s", listenAddr),
			})
		}
	} else if listenAddrs, ok := configMap["ListenAddress"].([]any); ok {
		for i, addr := range listenAddrs {
			if addrStr, ok := addr.(string); ok {
				if !isValidListenAddress(addrStr) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("ListenAddress[%d]", i),
						Message: fmt.Sprintf("invalid listen address: %s", addrStr),
					})
				}
			}
		}
	}

	// Validate MaxStartups format (start:rate:full or just a number)
	if maxStartups, ok := configMap["MaxStartups"].(string); ok {
		if !isValidMaxStartups(maxStartups) {
			errors = append(errors, plugin.ValidationError{
				Path:    "MaxStartups",
				Message: fmt.Sprintf("invalid MaxStartups format: %s (expected: start:rate:full or just a number)", maxStartups),
			})
		}
	}

	// Validate Ciphers
	if ciphers, ok := configMap["Ciphers"].(string); ok {
		errors = append(errors, p.validateCiphers(ciphers)...)
	}

	// Validate MACs
	if macs, ok := configMap["MACs"].(string); ok {
		errors = append(errors, p.validateMACs(macs)...)
	}

	// Validate KexAlgorithms
	if kex, ok := configMap["KexAlgorithms"].(string); ok {
		errors = append(errors, p.validateKexAlgorithms(kex)...)
	}

	// Validate Match blocks
	if matches, ok := configMap["Match"].([]any); ok {
		errors = append(errors, p.validateMatchBlocks(matches)...)
	}

	// Validate integer fields
	intFields := map[string]struct{ min, max int }{
		"ClientAliveInterval": {0, 86400},
		"ClientAliveCountMax": {0, 100},
		"MaxAuthTries":        {1, 100},
		"MaxSessions":         {1, 100},
	}
	for field, limits := range intFields {
		if val, ok := configMap[field]; ok {
			var intVal int
			switch v := val.(type) {
			case int:
				intVal = v
			case float64:
				intVal = int(v)
			}
			if intVal < limits.min || intVal > limits.max {
				errors = append(errors, plugin.ValidationError{
					Path:    field,
					Message: fmt.Sprintf("%s must be between %d and %d, got: %d", field, limits.min, limits.max, intVal),
				})
			}
		}
	}

	return errors, nil
}

func (p *Plugin) validateCiphers(ciphers string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Weak ciphers that should be avoided
	weakCiphers := []string{
		"arcfour", "arcfour128", "arcfour256",
		"3des-cbc", "blowfish-cbc", "cast128-cbc",
		"aes128-cbc", "aes192-cbc", "aes256-cbc",
	}

	cipherList := strings.Split(ciphers, ",")
	for _, cipher := range cipherList {
		cipher = strings.TrimSpace(cipher)
		for _, weak := range weakCiphers {
			if cipher == weak {
				errors = append(errors, plugin.ValidationError{
					Path:    "Ciphers",
					Message: fmt.Sprintf("cipher '%s' is considered weak", cipher),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateMACs(macs string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Weak MACs that should be avoided
	weakMACs := []string{
		"hmac-md5", "hmac-md5-96",
		"hmac-sha1", "hmac-sha1-96",
		"umac-64@openssh.com",
	}

	macList := strings.Split(macs, ",")
	for _, mac := range macList {
		mac = strings.TrimSpace(mac)
		for _, weak := range weakMACs {
			if mac == weak {
				errors = append(errors, plugin.ValidationError{
					Path:    "MACs",
					Message: fmt.Sprintf("MAC '%s' is considered weak", mac),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateKexAlgorithms(kex string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Weak key exchange algorithms
	weakKex := []string{
		"diffie-hellman-group1-sha1",
		"diffie-hellman-group14-sha1",
		"diffie-hellman-group-exchange-sha1",
	}

	kexList := strings.Split(kex, ",")
	for _, k := range kexList {
		k = strings.TrimSpace(k)
		for _, weak := range weakKex {
			if k == weak {
				errors = append(errors, plugin.ValidationError{
					Path:    "KexAlgorithms",
					Message: fmt.Sprintf("key exchange algorithm '%s' is considered weak", k),
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateMatchBlocks(matches []any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for i, match := range matches {
		matchMap, ok := match.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("Match[%d]", i),
				Message: "Match block must be an object",
			})
			continue
		}

		// Validate condition
		if condition, ok := matchMap["condition"].(string); ok {
			if !isValidMatchCondition(condition) {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("Match[%d].condition", i),
					Message: fmt.Sprintf("invalid match condition: %s", condition),
				})
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("Match[%d].condition", i),
				Message: "Match block requires a condition",
			})
		}

		// Validate allowed directives in Match blocks
		validMatchDirectives := map[string]bool{
			"AllowAgentForwarding":         true,
			"AllowGroups":                  true,
			"AllowStreamLocalForwarding":   true,
			"AllowTcpForwarding":           true,
			"AllowUsers":                   true,
			"AuthenticationMethods":        true,
			"AuthorizedKeysCommand":        true,
			"AuthorizedKeysCommandUser":    true,
			"AuthorizedKeysFile":           true,
			"Banner":                       true,
			"ChrootDirectory":              true,
			"ClientAliveCountMax":          true,
			"ClientAliveInterval":          true,
			"DenyGroups":                   true,
			"DenyUsers":                    true,
			"ForceCommand":                 true,
			"GatewayPorts":                 true,
			"HostbasedAcceptedKeyTypes":    true,
			"HostbasedAuthentication":      true,
			"KbdInteractiveAuthentication": true,
			"MaxAuthTries":                 true,
			"MaxSessions":                  true,
			"PasswordAuthentication":       true,
			"PermitEmptyPasswords":         true,
			"PermitListen":                 true,
			"PermitOpen":                   true,
			"PermitRootLogin":              true,
			"PermitTTY":                    true,
			"PermitTunnel":                 true,
			"PermitUserRC":                 true,
			"PubkeyAcceptedKeyTypes":       true,
			"PubkeyAuthentication":         true,
			"RekeyLimit":                   true,
			"SetEnv":                       true,
			"StreamLocalBindMask":          true,
			"StreamLocalBindUnlink":        true,
			"X11DisplayOffset":             true,
			"X11Forwarding":                true,
			"X11UseLocalhost":              true,
		}

		for key := range matchMap {
			if key == "condition" {
				continue
			}
			if !validMatchDirectives[key] {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("Match[%d].%s", i, key),
					Message: fmt.Sprintf("directive '%s' is not valid in Match blocks", key),
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

	// Security warnings
	// PermitRootLogin
	if permitRoot, ok := configMap["PermitRootLogin"].(string); ok {
		if permitRoot == "yes" {
			errors = append(errors, plugin.ValidationError{
				Path:    "PermitRootLogin",
				Message: "allowing root login with password is a security risk",
			})
		}
	}

	// PasswordAuthentication
	if passwordAuth, ok := configMap["PasswordAuthentication"].(bool); ok && passwordAuth {
		// Check if public key auth is also enabled
		pubkeyAuth := true
		if pa, ok := configMap["PubkeyAuthentication"].(bool); ok {
			pubkeyAuth = pa
		}
		if !pubkeyAuth {
			errors = append(errors, plugin.ValidationError{
				Path:    "PasswordAuthentication",
				Message: "password-only authentication is less secure than public key authentication",
			})
		}
	}

	// PermitEmptyPasswords
	if permitEmpty, ok := configMap["PermitEmptyPasswords"].(bool); ok && permitEmpty {
		errors = append(errors, plugin.ValidationError{
			Path:    "PermitEmptyPasswords",
			Message: "allowing empty passwords is a serious security risk",
		})
	}

	// X11Forwarding
	if x11, ok := configMap["X11Forwarding"].(bool); ok && x11 {
		errors = append(errors, plugin.ValidationError{
			Path:    "X11Forwarding",
			Message: "X11 forwarding has security implications",
		})
	}

	// Protocol version
	if protocol, ok := configMap["Protocol"]; ok {
		var protocolNum int
		switch v := protocol.(type) {
		case int:
			protocolNum = v
		case float64:
			protocolNum = int(v)
		}
		if protocolNum == 1 {
			errors = append(errors, plugin.ValidationError{
				Path:    "Protocol",
				Message: "SSH protocol version 1 is deprecated and insecure",
			})
		}
	}

	// Check for UsePAM consistency
	if usePAM, ok := configMap["UsePAM"].(bool); ok {
		if challengeResponse, ok := configMap["ChallengeResponseAuthentication"].(bool); ok {
			if usePAM && !challengeResponse {
				errors = append(errors, plugin.ValidationError{
					Path:    "ChallengeResponseAuthentication",
					Message: "when UsePAM is enabled, ChallengeResponseAuthentication is typically also enabled",
				})
			}
		}
	}

	// Check for PermitUserEnvironment
	if permitUserEnv, ok := configMap["PermitUserEnvironment"].(bool); ok && permitUserEnv {
		errors = append(errors, plugin.ValidationError{
			Path:    "PermitUserEnvironment",
			Message: "permitting user environment may have security implications",
		})
	}

	// Check StrictModes
	if strictModes, ok := configMap["StrictModes"].(bool); ok && !strictModes {
		errors = append(errors, plugin.ValidationError{
			Path:    "StrictModes",
			Message: "disabling StrictModes may allow insecure file permissions",
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

	// Convert boolean values to "yes"/"no" for sshd_config
	// This is done in ToNative, not here

	return configMap, nil
}

// ToNative converts to native sshd_config format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var sb strings.Builder

	// Write header comment
	sb.WriteString("# sshd_config - Generated by terraform-provider-filemanager\n")
	sb.WriteString("# Do not edit manually\n\n")

	// Write main directives (excluding Match blocks)
	for key, value := range configMap {
		if key == "Match" {
			continue
		}

		// Handle Subsystem specially (it's a map)
		if key == "Subsystem" {
			if subsystems, ok := value.(map[string]any); ok {
				for name, cmd := range subsystems {
					sb.WriteString(fmt.Sprintf("Subsystem %s %v\n", name, cmd))
				}
			} else if subsystems, ok := value.(map[string]string); ok {
				for name, cmd := range subsystems {
					sb.WriteString(fmt.Sprintf("Subsystem %s %s\n", name, cmd))
				}
			}
			continue
		}

		// Handle HostKey (can be multiple)
		if key == "HostKey" {
			switch v := value.(type) {
			case string:
				sb.WriteString(fmt.Sprintf("HostKey %s\n", v))
			case []any:
				for _, hk := range v {
					sb.WriteString(fmt.Sprintf("HostKey %v\n", hk))
				}
			}
			continue
		}

		// Handle ListenAddress (can be multiple)
		if key == "ListenAddress" {
			switch v := value.(type) {
			case string:
				sb.WriteString(fmt.Sprintf("ListenAddress %s\n", v))
			case []any:
				for _, addr := range v {
					sb.WriteString(fmt.Sprintf("ListenAddress %v\n", addr))
				}
			}
			continue
		}

		// Convert value to sshd_config format
		sb.WriteString(fmt.Sprintf("%s %s\n", key, formatValue(value)))
	}

	// Write Match blocks
	if matches, ok := configMap["Match"].([]any); ok {
		for _, match := range matches {
			matchMap, ok := match.(map[string]any)
			if !ok {
				continue
			}

			sb.WriteString("\n")
			if condition, ok := matchMap["condition"].(string); ok {
				sb.WriteString(fmt.Sprintf("Match %s\n", condition))
			}

			for key, value := range matchMap {
				if key == "condition" {
					continue
				}
				sb.WriteString(fmt.Sprintf("    %s %s\n", key, formatValue(value)))
			}
		}
	}

	return []byte(sb.String()), nil
}

// FromNative parses native sshd_config format.
func (p *Plugin) FromNative(data []byte) (any, error) {
	config := make(map[string]any)
	var currentMatch map[string]any
	var matches []any

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split into key and value
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := strings.TrimSpace(parts[1])

		// Handle Match blocks
		if strings.EqualFold(key, "Match") {
			// Save previous match block
			if currentMatch != nil {
				matches = append(matches, currentMatch)
			}
			currentMatch = map[string]any{
				"condition": value,
			}
			continue
		}

		// If we're in a Match block, add to that
		if currentMatch != nil {
			currentMatch[key] = parseValue(value)
		} else {
			// Handle directives that can appear multiple times
			if key == "HostKey" || key == "ListenAddress" {
				if existing, ok := config[key].([]any); ok {
					config[key] = append(existing, value)
				} else if existing, ok := config[key].(string); ok {
					config[key] = []any{existing, value}
				} else {
					config[key] = value
				}
			} else if key == "Subsystem" {
				// Parse Subsystem name command
				subParts := strings.SplitN(value, " ", 2)
				if len(subParts) == 2 {
					if subsystems, ok := config["Subsystem"].(map[string]any); ok {
						subsystems[subParts[0]] = subParts[1]
					} else {
						config["Subsystem"] = map[string]any{
							subParts[0]: subParts[1],
						}
					}
				}
			} else {
				config[key] = parseValue(value)
			}
		}
	}

	// Save last match block
	if currentMatch != nil {
		matches = append(matches, currentMatch)
	}

	if len(matches) > 0 {
		config["Match"] = matches
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

func formatValue(value any) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "yes"
		}
		return "no"
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.Itoa(int(v))
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func parseValue(value string) any {
	// Try to parse as boolean
	if strings.EqualFold(value, "yes") {
		return true
	}
	if strings.EqualFold(value, "no") {
		return false
	}

	// Try to parse as integer
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}

	return value
}

func isValidListenAddress(addr string) bool {
	// Can be IP, IP:port, or hostname
	if addr == "*" || addr == "0.0.0.0" || addr == "::" {
		return true
	}

	// Check for IP:port format
	if strings.Contains(addr, ":") {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			// Might be IPv6 without port
			if net.ParseIP(addr) != nil {
				return true
			}
			return false
		}
		if net.ParseIP(host) == nil && host != "" {
			return false
		}
		if _, err := strconv.Atoi(port); err != nil {
			return false
		}
		return true
	}

	// Check for IP address
	if net.ParseIP(addr) != nil {
		return true
	}

	// Check for hostname
	pattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*$`)
	return pattern.MatchString(addr)
}

func isValidMaxStartups(value string) bool {
	// Can be just a number or start:rate:full
	if _, err := strconv.Atoi(value); err == nil {
		return true
	}

	pattern := regexp.MustCompile(`^\d+:\d+:\d+$`)
	return pattern.MatchString(value)
}

func isValidMatchCondition(condition string) bool {
	// Match condition format: criteria pattern [criteria pattern] ...
	// Valid criteria: User, Group, Host, LocalAddress, LocalPort, RDomain, Address
	validCriteria := []string{"User", "Group", "Host", "LocalAddress", "LocalPort", "RDomain", "Address", "All"}

	parts := strings.Fields(condition)
	if len(parts) < 2 && !strings.EqualFold(parts[0], "All") {
		return false
	}

	// Check first word is a valid criteria
	found := false
	for _, vc := range validCriteria {
		if strings.EqualFold(parts[0], vc) {
			found = true
			break
		}
	}

	return found
}
