// SPDX-License-Identifier: MIT

// Package redis provides a Redis configuration management plugin.
package redis

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Redis.
type Plugin struct{}

// New creates a new Redis plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "redis"
}

// Version returns the supported Redis version range.
func (p *Plugin) Version() string {
	return ">=5.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Redis in-memory data store configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "redis"
}

// Schema returns the configuration schema for Redis.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "network",
				Required:    false,
				Multiple:    false,
				Description: "Network configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "bind", Type: "string", Multiple: true, Description: "Bind addresses"},
					{Name: "port", Type: "int", Description: "Listen port (default: 6379)"},
					{Name: "unixsocket", Type: "string", Description: "Unix socket path"},
					{Name: "unixsocketperm", Type: "string", Description: "Unix socket permissions"},
					{Name: "timeout", Type: "int", Description: "Client timeout in seconds"},
					{Name: "tcp-backlog", Type: "int", Description: "TCP backlog"},
					{Name: "tcp-keepalive", Type: "int", Description: "TCP keepalive in seconds"},
				},
			},
			{
				Name:        "general",
				Required:    false,
				Multiple:    false,
				Description: "General configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "daemonize", Type: "bool", Description: "Run as daemon"},
					{Name: "pidfile", Type: "string", Description: "PID file path"},
					{Name: "loglevel", Type: "string", ValidValues: []string{"debug", "verbose", "notice", "warning"}, Description: "Log level"},
					{Name: "logfile", Type: "string", Description: "Log file path"},
					{Name: "databases", Type: "int", Description: "Number of databases"},
					{Name: "always-show-logo", Type: "bool", Description: "Show logo on startup"},
				},
			},
			{
				Name:        "security",
				Required:    false,
				Multiple:    false,
				Description: "Security configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "requirepass", Type: "string", Description: "Password for AUTH"},
					{Name: "aclfile", Type: "string", Description: "ACL file path"},
					{Name: "acllog-max-len", Type: "int", Description: "ACL log max length"},
				},
			},
			{
				Name:        "snapshotting",
				Required:    false,
				Multiple:    false,
				Description: "RDB persistence configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "save", Type: "string", Multiple: true, Description: "Save intervals (seconds changes)"},
					{Name: "stop-writes-on-bgsave-error", Type: "bool", Description: "Stop writes on BGSAVE error"},
					{Name: "rdbcompression", Type: "bool", Description: "Compress RDB files"},
					{Name: "rdbchecksum", Type: "bool", Description: "RDB checksum"},
					{Name: "dbfilename", Type: "string", Description: "RDB filename"},
					{Name: "dir", Type: "string", Description: "Working directory"},
				},
			},
			{
				Name:        "replication",
				Required:    false,
				Multiple:    false,
				Description: "Replication configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "replicaof", Type: "string", Description: "Master host port"},
					{Name: "masterauth", Type: "string", Description: "Master password"},
					{Name: "masteruser", Type: "string", Description: "Master username"},
					{Name: "replica-serve-stale-data", Type: "bool", Description: "Serve stale data"},
					{Name: "replica-read-only", Type: "bool", Description: "Read-only replica"},
					{Name: "repl-diskless-sync", Type: "bool", Description: "Diskless sync"},
					{Name: "repl-diskless-sync-delay", Type: "int", Description: "Diskless sync delay"},
					{Name: "repl-ping-replica-period", Type: "int", Description: "Ping period"},
					{Name: "repl-timeout", Type: "int", Description: "Replication timeout"},
					{Name: "repl-backlog-size", Type: "string", Description: "Backlog size"},
					{Name: "repl-backlog-ttl", Type: "int", Description: "Backlog TTL"},
					{Name: "replica-priority", Type: "int", Description: "Replica priority"},
				},
			},
			{
				Name:        "memory",
				Required:    false,
				Multiple:    false,
				Description: "Memory management configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "maxmemory", Type: "string", Description: "Maximum memory (e.g., 100mb, 1gb)"},
					{Name: "maxmemory-policy", Type: "string", ValidValues: []string{"volatile-lru", "allkeys-lru", "volatile-lfu", "allkeys-lfu", "volatile-random", "allkeys-random", "volatile-ttl", "noeviction"}, Description: "Eviction policy"},
					{Name: "maxmemory-samples", Type: "int", Description: "LRU/LFU samples"},
					{Name: "replica-ignore-maxmemory", Type: "bool", Description: "Replica ignores maxmemory"},
				},
			},
			{
				Name:        "append-only",
				Required:    false,
				Multiple:    false,
				Description: "AOF persistence configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "appendonly", Type: "bool", Description: "Enable AOF"},
					{Name: "appendfilename", Type: "string", Description: "AOF filename"},
					{Name: "appendfsync", Type: "string", ValidValues: []string{"always", "everysec", "no"}, Description: "Fsync policy"},
					{Name: "no-appendfsync-on-rewrite", Type: "bool", Description: "No fsync on rewrite"},
					{Name: "auto-aof-rewrite-percentage", Type: "int", Description: "Rewrite percentage"},
					{Name: "auto-aof-rewrite-min-size", Type: "string", Description: "Rewrite min size"},
					{Name: "aof-load-truncated", Type: "bool", Description: "Load truncated AOF"},
					{Name: "aof-use-rdb-preamble", Type: "bool", Description: "Use RDB preamble"},
				},
			},
			{
				Name:        "cluster",
				Required:    false,
				Multiple:    false,
				Description: "Cluster configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "cluster-enabled", Type: "bool", Description: "Enable cluster mode"},
					{Name: "cluster-config-file", Type: "string", Description: "Cluster config file"},
					{Name: "cluster-node-timeout", Type: "int", Description: "Node timeout in ms"},
					{Name: "cluster-replica-validity-factor", Type: "int", Description: "Replica validity factor"},
					{Name: "cluster-migration-barrier", Type: "int", Description: "Migration barrier"},
					{Name: "cluster-require-full-coverage", Type: "bool", Description: "Require full coverage"},
					{Name: "cluster-replica-no-failover", Type: "bool", Description: "Disable replica failover"},
				},
			},
			{
				Name:        "slowlog",
				Required:    false,
				Multiple:    false,
				Description: "Slow log configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "slowlog-log-slower-than", Type: "int", Description: "Slowlog threshold in microseconds"},
					{Name: "slowlog-max-len", Type: "int", Description: "Slowlog max entries"},
				},
			},
			{
				Name:        "clients",
				Required:    false,
				Multiple:    false,
				Description: "Client configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "maxclients", Type: "int", Description: "Max clients"},
				},
			},
		},
	}
}

// DefaultConfig returns sensible defaults for Redis.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"bind":                        []string{"127.0.0.1"},
		"port":                        6379,
		"timeout":                     0,
		"tcp-keepalive":               300,
		"daemonize":                   false,
		"loglevel":                    "notice",
		"logfile":                     "",
		"databases":                   16,
		"save":                        []string{"900 1", "300 10", "60 10000"},
		"stop-writes-on-bgsave-error": true,
		"rdbcompression":              true,
		"rdbchecksum":                 true,
		"dbfilename":                  "dump.rdb",
		"dir":                         "./",
		"replica-serve-stale-data":    true,
		"replica-read-only":           true,
		"repl-diskless-sync":          false,
		"repl-diskless-sync-delay":    5,
		"appendonly":                  false,
		"appendfilename":              "appendonly.aof",
		"appendfsync":                 "everysec",
		"no-appendfsync-on-rewrite":   false,
		"auto-aof-rewrite-percentage": 100,
		"auto-aof-rewrite-min-size":   "64mb",
		"aof-load-truncated":          true,
		"aof-use-rdb-preamble":        true,
		"slowlog-log-slower-than":     10000,
		"slowlog-max-len":             128,
		"maxmemory-policy":            "noeviction",
	}
}

// Validate validates the Redis configuration.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Validate port
	if port, ok := configMap["port"]; ok {
		var portNum int
		switch v := port.(type) {
		case int:
			portNum = v
		case int64:
			portNum = int(v)
		case float64:
			portNum = int(v)
		}
		if portNum < 0 || portNum > 65535 {
			errors = append(errors, plugin.ValidationError{
				Path:    "port",
				Message: fmt.Sprintf("invalid port number: %d (must be 0-65535)", portNum),
			})
		}
	}

	// Validate loglevel
	if loglevel, ok := configMap["loglevel"].(string); ok {
		validLevels := []string{"debug", "verbose", "notice", "warning"}
		valid := false
		for _, v := range validLevels {
			if strings.EqualFold(loglevel, v) {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, plugin.ValidationError{
				Path:    "loglevel",
				Message: fmt.Sprintf("invalid loglevel: %s (must be one of: %s)", loglevel, strings.Join(validLevels, ", ")),
			})
		}
	}

	// Validate maxmemory-policy
	if policy, ok := configMap["maxmemory-policy"].(string); ok {
		validPolicies := []string{"volatile-lru", "allkeys-lru", "volatile-lfu", "allkeys-lfu", "volatile-random", "allkeys-random", "volatile-ttl", "noeviction"}
		valid := false
		for _, v := range validPolicies {
			if policy == v {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, plugin.ValidationError{
				Path:    "maxmemory-policy",
				Message: fmt.Sprintf("invalid maxmemory-policy: %s", policy),
			})
		}
	}

	// Validate appendfsync
	if fsync, ok := configMap["appendfsync"].(string); ok {
		validOptions := []string{"always", "everysec", "no"}
		valid := false
		for _, v := range validOptions {
			if fsync == v {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, plugin.ValidationError{
				Path:    "appendfsync",
				Message: fmt.Sprintf("invalid appendfsync: %s (must be one of: %s)", fsync, strings.Join(validOptions, ", ")),
			})
		}
	}

	// Validate save format
	if saves, ok := configMap["save"]; ok {
		var saveList []any
		switch v := saves.(type) {
		case []any:
			saveList = v
		case []string:
			for _, s := range v {
				saveList = append(saveList, s)
			}
		}
		for i, save := range saveList {
			saveStr, ok := save.(string)
			if !ok {
				continue
			}
			if saveStr == "" {
				continue
			}
			parts := strings.Fields(saveStr)
			if len(parts) != 2 {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("save[%d]", i),
					Message: fmt.Sprintf("invalid save format: %s (expected 'seconds changes')", saveStr),
				})
			}
		}
	}

	// Validate maxmemory format
	if maxmem, ok := configMap["maxmemory"].(string); ok && maxmem != "" {
		if !isValidMemorySize(maxmem) {
			errors = append(errors, plugin.ValidationError{
				Path:    "maxmemory",
				Message: fmt.Sprintf("invalid maxmemory format: %s (expected: 100mb, 1gb, etc.)", maxmem),
			})
		}
	}

	return errors, nil
}

// ValidateSemantic performs Redis-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Check for security issues
	if binds, ok := configMap["bind"]; ok {
		var bindList []string
		switch v := binds.(type) {
		case []any:
			for _, b := range v {
				if s, ok := b.(string); ok {
					bindList = append(bindList, s)
				}
			}
		case []string:
			bindList = v
		case string:
			bindList = []string{v}
		}

		for _, bind := range bindList {
			if bind == "0.0.0.0" || bind == "*" {
				// Check if password is set
				if _, hasPass := configMap["requirepass"]; !hasPass {
					errors = append(errors, plugin.ValidationError{
						Path:    "bind",
						Message: "binding to 0.0.0.0 without 'requirepass' is a security risk",
					})
				}
				break
			}
		}
	}

	// Check cluster configuration consistency
	if clusterEnabled, ok := configMap["cluster-enabled"].(bool); ok && clusterEnabled {
		if _, hasConfigFile := configMap["cluster-config-file"]; !hasConfigFile {
			errors = append(errors, plugin.ValidationError{
				Path:    "cluster-enabled",
				Message: "cluster mode enabled but 'cluster-config-file' not set",
			})
		}
	}

	// Check persistence configuration
	appendonly := false
	if ao, ok := configMap["appendonly"].(bool); ok {
		appendonly = ao
	}

	hasSave := false
	if saves, ok := configMap["save"]; ok {
		switch v := saves.(type) {
		case []any:
			hasSave = len(v) > 0
		case []string:
			hasSave = len(v) > 0
		case string:
			hasSave = v != ""
		}
	}

	if !appendonly && !hasSave {
		errors = append(errors, plugin.ValidationError{
			Path:    "persistence",
			Message: "neither AOF nor RDB persistence is enabled - data will be lost on restart",
		})
	}

	// Check replication configuration
	if replicaof, ok := configMap["replicaof"].(string); ok && replicaof != "" {
		// If this is a replica, masterauth should be set if master requires password
		if _, hasMasterAuth := configMap["masterauth"]; !hasMasterAuth {
			errors = append(errors, plugin.ValidationError{
				Path:    "replicaof",
				Message: "replica configured but 'masterauth' not set - ensure master doesn't require authentication",
			})
		}
	}

	return errors, nil
}

// isValidMemorySize checks if a string is a valid Redis memory size.
func isValidMemorySize(s string) bool {
	pattern := regexp.MustCompile(`^(\d+)(b|kb|mb|gb|tb)?$`)
	return pattern.MatchString(strings.ToLower(s))
}

// Normalize normalizes the Redis configuration to canonical form.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Ensure consistent casing for loglevel
	if loglevel, ok := configMap["loglevel"].(string); ok {
		configMap["loglevel"] = strings.ToLower(loglevel)
	}

	// Normalize bind to array
	if bind, ok := configMap["bind"].(string); ok {
		configMap["bind"] = []string{bind}
	}

	return configMap, nil
}

// ToNative converts the configuration to native Redis config format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var buf bytes.Buffer

	// Get sorted keys for consistent output
	keys := make([]string, 0, len(configMap))
	for k := range configMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := configMap[key]
		if err := writeRedisDirective(&buf, key, value); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// writeRedisDirective writes a single Redis config directive.
func writeRedisDirective(buf *bytes.Buffer, key string, value any) error {
	switch v := value.(type) {
	case nil:
		// Skip nil values
		return nil
	case bool:
		if v {
			fmt.Fprintf(buf, "%s yes\n", key)
		} else {
			fmt.Fprintf(buf, "%s no\n", key)
		}
	case int, int64, float64:
		fmt.Fprintf(buf, "%s %v\n", key, v)
	case string:
		if v == "" {
			fmt.Fprintf(buf, "%s \"\"\n", key)
		} else if strings.ContainsAny(v, " \t\n\"") {
			fmt.Fprintf(buf, "%s %q\n", key, v)
		} else {
			fmt.Fprintf(buf, "%s %s\n", key, v)
		}
	case []any:
		for _, item := range v {
			if err := writeRedisDirective(buf, key, item); err != nil {
				return err
			}
		}
	case []string:
		for _, item := range v {
			if err := writeRedisDirective(buf, key, item); err != nil {
				return err
			}
		}
	case map[string]any:
		// Nested maps are flattened in Redis config
		return fmt.Errorf("nested maps not supported in Redis config")
	default:
		fmt.Fprintf(buf, "%s %v\n", key, v)
	}
	return nil
}

// FromNative parses native Redis configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	config := make(map[string]any)
	multiValueKeys := map[string]bool{
		"bind": true,
		"save": true,
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key value
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle quoted values
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}

		// Convert value
		converted := convertRedisValue(value)

		// Handle multi-value keys
		if multiValueKeys[key] {
			if existing, ok := config[key].([]any); ok {
				config[key] = append(existing, converted)
			} else if existing, ok := config[key]; ok {
				config[key] = []any{existing, converted}
			} else {
				config[key] = []any{converted}
			}
		} else {
			config[key] = converted
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	return config, nil
}

// convertRedisValue converts a Redis config value to appropriate Go type.
func convertRedisValue(s string) any {
	// Check for boolean
	lower := strings.ToLower(s)
	if lower == "yes" || lower == "true" {
		return true
	}
	if lower == "no" || lower == "false" {
		return false
	}

	// Check for integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}

	// Check for float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return s
}

// Merge merges two Redis configurations.
func (p *Plugin) Merge(base, overlay any) (any, error) {
	baseMap, ok := base.(map[string]any)
	if !ok {
		return overlay, nil
	}

	overlayMap, ok := overlay.(map[string]any)
	if !ok {
		return base, nil
	}

	result := make(map[string]any)

	// Copy base
	for k, v := range baseMap {
		result[k] = v
	}

	// Overlay values
	for k, v := range overlayMap {
		result[k] = v
	}

	return result, nil
}

// Diff computes the differences between two configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	var changes []plugin.Change

	oldMap, _ := old.(map[string]any)
	newMap, _ := new.(map[string]any)

	// Check for removed and modified keys
	for k, oldVal := range oldMap {
		newVal, exists := newMap[k]
		if !exists {
			changes = append(changes, plugin.Change{
				Path:     k,
				Type:     "remove",
				OldValue: oldVal,
			})
			continue
		}

		if !equalValues(oldVal, newVal) {
			changes = append(changes, plugin.Change{
				Path:     k,
				Type:     "modify",
				OldValue: oldVal,
				NewValue: newVal,
			})
		}
	}

	// Check for added keys
	for k, newVal := range newMap {
		if _, exists := oldMap[k]; !exists {
			changes = append(changes, plugin.Change{
				Path:     k,
				Type:     "add",
				NewValue: newVal,
			})
		}
	}

	return changes, nil
}

// equalValues compares two values for equality.
func equalValues(a, b any) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// Ensure Plugin implements AppPlugin interface.
var _ plugin.AppPlugin = (*Plugin)(nil)
