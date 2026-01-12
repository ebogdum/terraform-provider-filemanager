// SPDX-License-Identifier: MIT

// Package ssh_client provides a plugin for SSH client configuration management.
package ssh_client

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"github.com/ebogdum/filemanager/internal/util"
)

// Plugin implements the SSH client configuration plugin.
type Plugin struct{}

// New creates a new SSH client plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Ensure Plugin implements plugin.AppPlugin.
var _ plugin.AppPlugin = (*Plugin)(nil)

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "ssh_client"
}

// Version returns the supported OpenSSH version range.
func (p *Plugin) Version() string {
	return ">=7.0"
}

// Description returns a description of the plugin.
func (p *Plugin) Description() string {
	return "SSH client configuration (~/.ssh/config) management"
}

// NativeFormat returns the native configuration format.
func (p *Plugin) NativeFormat() string {
	return "ssh_config"
}

// Schema returns the configuration schema.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "Host",
				Description: "Host-specific configuration blocks",
			},
			{
				Name:        "Match",
				Description: "Conditional configuration blocks",
			},
		},
		Directives: []plugin.DirectiveSchema{
			// Connection
			{Name: "HostName", Description: "Real hostname to connect to", Type: "string"},
			{Name: "Port", Description: "SSH port", Type: "int"},
			{Name: "User", Description: "Username for connection", Type: "string"},
			{Name: "IdentityFile", Description: "Path to private key file", Type: "string"},
			{Name: "IdentitiesOnly", Description: "Only use configured identity files", Type: "bool"},
			// Proxy
			{Name: "ProxyJump", Description: "Jump host(s) for connection", Type: "string"},
			{Name: "ProxyCommand", Description: "Command to connect to server", Type: "string"},
			{Name: "ProxyUseFdpass", Description: "Use file descriptor passing for proxy", Type: "bool"},
			// Forwarding
			{Name: "ForwardAgent", Description: "Forward SSH agent", Type: "bool"},
			{Name: "ForwardX11", Description: "Forward X11 connections", Type: "bool"},
			{Name: "ForwardX11Trusted", Description: "Trust remote X11 clients", Type: "bool"},
			{Name: "LocalForward", Description: "Local port forwarding", Type: "string"},
			{Name: "RemoteForward", Description: "Remote port forwarding", Type: "string"},
			{Name: "DynamicForward", Description: "Dynamic port forwarding (SOCKS)", Type: "string"},
			// Authentication
			{Name: "PreferredAuthentications", Description: "Authentication methods to try", Type: "string"},
			{Name: "PubkeyAuthentication", Description: "Try public key authentication", Type: "bool"},
			{Name: "PasswordAuthentication", Description: "Try password authentication", Type: "bool"},
			{Name: "KbdInteractiveAuthentication", Description: "Try keyboard-interactive auth", Type: "bool"},
			{Name: "GSSAPIAuthentication", Description: "Try GSSAPI authentication", Type: "bool"},
			{Name: "BatchMode", Description: "Disable password/passphrase queries", Type: "bool"},
			// Host keys
			{Name: "StrictHostKeyChecking", Description: "Host key verification mode", Type: "string"},
			{Name: "UserKnownHostsFile", Description: "Known hosts file path", Type: "string"},
			{Name: "GlobalKnownHostsFile", Description: "Global known hosts file", Type: "string"},
			{Name: "HostKeyAlgorithms", Description: "Host key algorithms to use", Type: "string"},
			{Name: "HostKeyAlias", Description: "Alias for host key lookup", Type: "string"},
			// Connection options
			{Name: "AddKeysToAgent", Description: "Add keys to SSH agent", Type: "string"},
			{Name: "AddressFamily", Description: "Address family to use", Type: "string"},
			{Name: "BindAddress", Description: "Address to bind to", Type: "string"},
			{Name: "BindInterface", Description: "Interface to bind to", Type: "string"},
			{Name: "ConnectTimeout", Description: "Connection timeout in seconds", Type: "int"},
			{Name: "ConnectionAttempts", Description: "Number of connection attempts", Type: "int"},
			{Name: "TCPKeepAlive", Description: "Send TCP keepalive", Type: "bool"},
			{Name: "ServerAliveInterval", Description: "Server alive check interval", Type: "int"},
			{Name: "ServerAliveCountMax", Description: "Max server alive checks", Type: "int"},
			// Cryptography
			{Name: "Ciphers", Description: "Allowed ciphers", Type: "string"},
			{Name: "MACs", Description: "Allowed MACs", Type: "string"},
			{Name: "KexAlgorithms", Description: "Key exchange algorithms", Type: "string"},
			{Name: "PubkeyAcceptedAlgorithms", Description: "Accepted public key algorithms", Type: "string"},
			// Multiplexing
			{Name: "ControlMaster", Description: "Connection multiplexing mode", Type: "string"},
			{Name: "ControlPath", Description: "Path for control socket", Type: "string"},
			{Name: "ControlPersist", Description: "Keep master connection open", Type: "string"},
			// Session
			{Name: "RequestTTY", Description: "Request TTY allocation", Type: "string"},
			{Name: "RemoteCommand", Description: "Command to execute on remote", Type: "string"},
			{Name: "SendEnv", Description: "Environment variables to send", Type: "string"},
			{Name: "SetEnv", Description: "Set environment variable", Type: "string"},
			{Name: "LogLevel", Description: "Logging verbosity", Type: "string"},
			{Name: "Compression", Description: "Enable compression", Type: "bool"},
			// Other
			{Name: "Include", Description: "Include config file(s)", Type: "string"},
			{Name: "HashKnownHosts", Description: "Hash known hosts entries", Type: "bool"},
			{Name: "VisualHostKey", Description: "Display ASCII art host key", Type: "bool"},
			{Name: "CheckHostIP", Description: "Check host IP in known_hosts", Type: "bool"},
			{Name: "NoHostAuthenticationForLocalhost", Description: "Skip host auth for localhost", Type: "bool"},
			{Name: "NumberOfPasswordPrompts", Description: "Max password prompts", Type: "int"},
			{Name: "PermitLocalCommand", Description: "Allow local command execution", Type: "bool"},
			{Name: "LocalCommand", Description: "Command to run locally after connect", Type: "string"},
			{Name: "EscapeChar", Description: "Escape character", Type: "string"},
		},
	}
}

// DefaultConfig returns sensible default configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"Host": []any{
			map[string]any{
				"pattern":                "*",
				"ServerAliveInterval":   60,
				"ServerAliveCountMax":   3,
				"AddKeysToAgent":        "yes",
				"IdentitiesOnly":        false,
				"StrictHostKeyChecking": "ask",
			},
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

	// Validate Host blocks
	if hosts, ok := configMap["Host"].([]any); ok {
		errors = append(errors, p.validateHostBlocks(hosts)...)
	}

	// Validate global directives
	errors = append(errors, p.validateDirectives(configMap, "")...)

	return errors, nil
}

func (p *Plugin) validateHostBlocks(hosts []any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	for i, host := range hosts {
		hostMap, ok := host.(map[string]any)
		if !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("Host[%d]", i),
				Message: "Host block must be an object",
			})
			continue
		}

		// Validate pattern exists
		if _, ok := hostMap["pattern"].(string); !ok {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("Host[%d].pattern", i),
				Message: "Host block requires a pattern",
			})
		}

		// Validate directives in this block
		errors = append(errors, p.validateDirectives(hostMap, fmt.Sprintf("Host[%d].", i))...)
	}

	return errors
}

func (p *Plugin) validateDirectives(directives map[string]any, prefix string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate Port
	if port, ok := directives["Port"]; ok {
		var portNum int
		switch v := port.(type) {
		case int:
			portNum = v
		case float64:
			portNum = int(v)
		}
		if portNum < 1 || portNum > 65535 {
			errors = append(errors, plugin.ValidationError{
				Path:    prefix + "Port",
				Message: fmt.Sprintf("port must be between 1 and 65535, got: %d", portNum),
			})
		}
	}

	// Validate StrictHostKeyChecking
	if strictHost, ok := directives["StrictHostKeyChecking"].(string); ok {
		validValues := []string{"yes", "no", "ask", "accept-new", "off"}
		if !containsString(validValues, strictHost) {
			errors = append(errors, plugin.ValidationError{
				Path:    prefix + "StrictHostKeyChecking",
				Message: fmt.Sprintf("invalid value: %s (valid: yes, no, ask, accept-new, off)", strictHost),
			})
		}
	}

	// Validate AddKeysToAgent
	if addKeys, ok := directives["AddKeysToAgent"].(string); ok {
		validValues := []string{"yes", "no", "ask", "confirm"}
		// Also supports time specs like "1h"
		if !containsString(validValues, addKeys) && !isTimeSpec(addKeys) {
			errors = append(errors, plugin.ValidationError{
				Path:    prefix + "AddKeysToAgent",
				Message: fmt.Sprintf("invalid value: %s (valid: yes, no, ask, confirm, or time spec)", addKeys),
			})
		}
	}

	// Validate ControlMaster
	if controlMaster, ok := directives["ControlMaster"].(string); ok {
		validValues := []string{"yes", "no", "ask", "auto", "autoask"}
		if !containsString(validValues, controlMaster) {
			errors = append(errors, plugin.ValidationError{
				Path:    prefix + "ControlMaster",
				Message: fmt.Sprintf("invalid value: %s (valid: yes, no, ask, auto, autoask)", controlMaster),
			})
		}
	}

	// Validate RequestTTY
	if requestTTY, ok := directives["RequestTTY"].(string); ok {
		validValues := []string{"yes", "no", "auto", "force"}
		if !containsString(validValues, requestTTY) {
			errors = append(errors, plugin.ValidationError{
				Path:    prefix + "RequestTTY",
				Message: fmt.Sprintf("invalid value: %s (valid: yes, no, auto, force)", requestTTY),
			})
		}
	}

	// Validate LogLevel
	if logLevel, ok := directives["LogLevel"].(string); ok {
		validLevels := []string{"QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3"}
		if !containsString(validLevels, logLevel) {
			errors = append(errors, plugin.ValidationError{
				Path:    prefix + "LogLevel",
				Message: fmt.Sprintf("invalid log level: %s", logLevel),
			})
		}
	}

	// Validate AddressFamily
	if addressFamily, ok := directives["AddressFamily"].(string); ok {
		validFamilies := []string{"any", "inet", "inet6"}
		if !containsString(validFamilies, addressFamily) {
			errors = append(errors, plugin.ValidationError{
				Path:    prefix + "AddressFamily",
				Message: fmt.Sprintf("invalid address family: %s (valid: any, inet, inet6)", addressFamily),
			})
		}
	}

	// Validate integer fields
	intFields := map[string]struct{ min, max int }{
		"ConnectTimeout":      {0, 3600},
		"ConnectionAttempts":  {1, 100},
		"ServerAliveInterval": {0, 86400},
		"ServerAliveCountMax": {0, 100},
	}
	for field, limits := range intFields {
		if val, ok := directives[field]; ok {
			var intVal int
			switch v := val.(type) {
			case int:
				intVal = v
			case float64:
				intVal = int(v)
			}
			if intVal < limits.min || intVal > limits.max {
				errors = append(errors, plugin.ValidationError{
					Path:    prefix + field,
					Message: fmt.Sprintf("%s must be between %d and %d, got: %d", field, limits.min, limits.max, intVal),
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

	// Check Host blocks for security concerns
	if hosts, ok := configMap["Host"].([]any); ok {
		for i, host := range hosts {
			hostMap, ok := host.(map[string]any)
			if !ok {
				continue
			}

			prefix := fmt.Sprintf("Host[%d].", i)

			// Warn about disabling host key checking
			if strictHost, ok := hostMap["StrictHostKeyChecking"].(string); ok {
				if strictHost == "no" || strictHost == "off" {
					errors = append(errors, plugin.ValidationError{
						Path:    prefix + "StrictHostKeyChecking",
						Message: "disabling host key checking is a security risk",
					})
				}
			}

			// Warn about forwarding agent with wildcard
			if pattern, ok := hostMap["pattern"].(string); ok {
				if pattern == "*" {
					if forward, ok := hostMap["ForwardAgent"].(bool); ok && forward {
						errors = append(errors, plugin.ValidationError{
							Path:    prefix + "ForwardAgent",
							Message: "forwarding agent to all hosts (*) is a security risk",
						})
					}
				}
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

	return configMap, nil
}

// ToNative converts to native ssh_config format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var sb strings.Builder

	// Write header comment
	sb.WriteString("# SSH config - Generated by terraform-provider-filemanager\n")
	sb.WriteString("# Do not edit manually\n\n")

	// Write global directives first (non-Host, non-Match)
	for key, value := range configMap {
		if key == "Host" || key == "Match" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", key, formatValue(value)))
	}

	// Write Host blocks
	if hosts, ok := configMap["Host"].([]any); ok {
		for _, host := range hosts {
			hostMap, ok := host.(map[string]any)
			if !ok {
				continue
			}

			sb.WriteString("\n")
			if pattern, ok := hostMap["pattern"].(string); ok {
				sb.WriteString(fmt.Sprintf("Host %s\n", pattern))
			}

			for key, value := range hostMap {
				if key == "pattern" {
					continue
				}
				sb.WriteString(fmt.Sprintf("    %s %s\n", key, formatValue(value)))
			}
		}
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

// FromNative parses native ssh_config format.
func (p *Plugin) FromNative(data []byte) (any, error) {
	config := make(map[string]any)
	var currentHost map[string]any
	var currentMatch map[string]any
	var hosts []any
	var matches []any
	inBlock := ""

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split into key and value
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			parts = strings.SplitN(line, "\t", 2)
			if len(parts) < 2 {
				continue
			}
		}

		key := parts[0]
		value := strings.TrimSpace(parts[1])

		// Handle Host blocks
		if strings.EqualFold(key, "Host") {
			// Save previous block
			if currentHost != nil {
				hosts = append(hosts, currentHost)
			}
			if currentMatch != nil {
				matches = append(matches, currentMatch)
				currentMatch = nil
			}

			currentHost = map[string]any{
				"pattern": value,
			}
			inBlock = "Host"
			continue
		}

		// Handle Match blocks
		if strings.EqualFold(key, "Match") {
			// Save previous block
			if currentHost != nil {
				hosts = append(hosts, currentHost)
				currentHost = nil
			}
			if currentMatch != nil {
				matches = append(matches, currentMatch)
			}

			currentMatch = map[string]any{
				"condition": value,
			}
			inBlock = "Match"
			continue
		}

		// Add to appropriate block
		if inBlock == "Host" && currentHost != nil {
			currentHost[key] = parseValue(value)
		} else if inBlock == "Match" && currentMatch != nil {
			currentMatch[key] = parseValue(value)
		} else {
			// Global directive
			config[key] = parseValue(value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SSH config: %w", err)
	}

	// Save last block
	if currentHost != nil {
		hosts = append(hosts, currentHost)
	}
	if currentMatch != nil {
		matches = append(matches, currentMatch)
	}

	if len(hosts) > 0 {
		config["Host"] = hosts
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

// SSHConfigHost holds parsed SSH config for a specific host.
type SSHConfigHost struct {
	Hostname     string
	Port         int
	User         string
	IdentityFile string
}

// ParseSSHConfigFile parses ~/.ssh/config for a specific host alias.
func (p *Plugin) ParseSSHConfigFile(data []byte, hostAlias string) (*SSHConfigHost, error) {
	config, err := p.FromNative(data)
	if err != nil {
		return nil, err
	}

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid config format")
	}

	result := &SSHConfigHost{}

	// Check Host blocks
	if hosts, ok := configMap["Host"].([]any); ok {
		for _, host := range hosts {
			hostMap, ok := host.(map[string]any)
			if !ok {
				continue
			}

			pattern, ok := hostMap["pattern"].(string)
			if !ok {
				continue
			}

			// Check if pattern matches
			patterns := strings.Fields(pattern)
			matched := false
			for _, p := range patterns {
				if matchHost(hostAlias, p) {
					matched = true
					break
				}
			}

			if !matched {
				continue
			}

			// Apply matching config
			if hostname, ok := hostMap["HostName"].(string); ok && result.Hostname == "" {
				result.Hostname = hostname
			}
			if port, ok := hostMap["Port"]; ok && result.Port == 0 {
				switch v := port.(type) {
				case int:
					result.Port = v
				case float64:
					result.Port = int(v)
				}
			}
			if user, ok := hostMap["User"].(string); ok && result.User == "" {
				result.User = user
			}
			if identity, ok := hostMap["IdentityFile"].(string); ok && result.IdentityFile == "" {
				result.IdentityFile = identity
			}
		}
	}

	// Use alias as hostname if not specified
	if result.Hostname == "" {
		result.Hostname = hostAlias
	}

	return result, nil
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

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}

func isTimeSpec(s string) bool {
	// Time specs like "1h", "30m", "1d"
	pattern := regexp.MustCompile(`^\d+[smhdw]$`)
	return pattern.MatchString(s)
}

// matchHost checks if a hostname matches a pattern (supports * and ?).
func matchHost(hostname, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple pattern matching
	if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
		return hostname == pattern
	}

	// Convert glob pattern to simple matching
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		return strings.HasPrefix(hostname, parts[0]) && strings.HasSuffix(hostname, parts[1])
	}

	return hostname == pattern
}
