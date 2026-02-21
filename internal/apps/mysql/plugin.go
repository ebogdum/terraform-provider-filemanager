// SPDX-License-Identifier: MIT

// Package mysql provides a MySQL/MariaDB configuration management plugin.
package mysql

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
	"gopkg.in/ini.v1"
)

// Plugin implements the AppPlugin interface for MySQL/MariaDB.
type Plugin struct{}

// New creates a new MySQL plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "mysql"
}

// Version returns the supported MySQL version range.
func (p *Plugin) Version() string {
	return ">=5.7.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "MySQL/MariaDB database server configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "ini"
}

// Schema returns the configuration schema for MySQL.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "mysqld",
				Required:    false,
				Multiple:    false,
				Description: "MySQL server configuration",
				Directives: []plugin.DirectiveSchema{
					// Basic settings
					{Name: "datadir", Type: "string", Description: "Data directory path"},
					{Name: "socket", Type: "string", Description: "Unix socket file path"},
					{Name: "port", Type: "int", Default: 3306, Description: "TCP/IP port number"},
					{Name: "bind-address", Type: "string", Default: "127.0.0.1", Description: "IP address to bind to"},

					// Connection settings
					{Name: "max_connections", Type: "int", Default: 151, Description: "Maximum concurrent connections"},
					{Name: "max_allowed_packet", Type: "string", Default: "16M", Description: "Maximum packet size"},

					// Character set settings
					{Name: "character-set-server", Type: "string", Default: "utf8mb4", Description: "Default server character set"},
					{Name: "collation-server", Type: "string", Default: "utf8mb4_unicode_ci", Description: "Default server collation"},

					// InnoDB settings
					{Name: "innodb_buffer_pool_size", Type: "string", Default: "128M", Description: "InnoDB buffer pool size"},
					{Name: "innodb_log_file_size", Type: "string", Default: "48M", Description: "InnoDB log file size"},
					{Name: "innodb_file_per_table", Type: "bool", Default: true, Description: "Store each table in its own file"},
					{Name: "innodb_flush_log_at_trx_commit", Type: "int", Default: 1, ValidValues: []string{"0", "1", "2"}, Description: "InnoDB flush behavior"},
					{Name: "innodb_lock_wait_timeout", Type: "int", Default: 50, Description: "InnoDB lock wait timeout in seconds"},

					// Logging
					{Name: "log_error", Type: "string", Description: "Error log file path"},
					{Name: "slow_query_log", Type: "bool", Default: false, Description: "Enable slow query log"},
					{Name: "slow_query_log_file", Type: "string", Description: "Slow query log file path"},
					{Name: "long_query_time", Type: "float", Default: 10.0, Description: "Slow query threshold in seconds"},
					{Name: "general_log", Type: "bool", Default: false, Description: "Enable general query log"},
					{Name: "general_log_file", Type: "string", Description: "General query log file path"},
					{Name: "log_bin", Type: "string", Description: "Binary log file base name"},

					// Performance settings
					{Name: "query_cache_type", Type: "int", ValidValues: []string{"0", "1", "2"}, Description: "Query cache type", Deprecated: true, DeprecatedBy: "MySQL 8.0 removes query cache"},
					{Name: "query_cache_size", Type: "string", Description: "Query cache size", Deprecated: true, DeprecatedBy: "MySQL 8.0 removes query cache"},
					{Name: "thread_cache_size", Type: "int", Default: 8, Description: "Thread cache size"},
					{Name: "table_open_cache", Type: "int", Default: 2000, Description: "Table open cache size"},

					// Security
					{Name: "skip-name-resolve", Type: "bool", Default: false, Description: "Skip DNS hostname resolution"},
					{Name: "local_infile", Type: "bool", Default: false, Description: "Allow LOAD DATA LOCAL"},
					{Name: "sql_mode", Type: "string", Description: "SQL mode settings"},

					// Replication
					{Name: "server-id", Type: "int", Description: "Unique server ID for replication"},
					{Name: "read_only", Type: "bool", Default: false, Description: "Read-only mode"},
					{Name: "log_slave_updates", Type: "bool", Default: false, Description: "Log replica updates to binary log"},
					{Name: "relay_log", Type: "string", Description: "Relay log file base name"},
					{Name: "gtid_mode", Type: "string", ValidValues: []string{"OFF", "OFF_PERMISSIVE", "ON_PERMISSIVE", "ON"}, Description: "GTID mode"},
				},
			},
			{
				Name:        "mysql",
				Required:    false,
				Multiple:    false,
				Description: "MySQL client configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "socket", Type: "string", Description: "Unix socket file path"},
					{Name: "port", Type: "int", Default: 3306, Description: "TCP/IP port number"},
					{Name: "default-character-set", Type: "string", Default: "utf8mb4", Description: "Default character set"},
					{Name: "no-auto-rehash", Type: "bool", Default: false, Description: "Disable automatic rehashing"},
					{Name: "safe-updates", Type: "bool", Default: false, Description: "Allow only safe UPDATE/DELETE"},
				},
			},
			{
				Name:        "client",
				Required:    false,
				Multiple:    false,
				Description: "General client configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "socket", Type: "string", Description: "Unix socket file path"},
					{Name: "port", Type: "int", Default: 3306, Description: "TCP/IP port number"},
					{Name: "default-character-set", Type: "string", Default: "utf8mb4", Description: "Default character set"},
					{Name: "user", Type: "string", Description: "Default user name"},
					{Name: "password", Type: "string", Description: "Default password"},
				},
			},
			{
				Name:        "mysqldump",
				Required:    false,
				Multiple:    false,
				Description: "mysqldump client configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "quick", Type: "bool", Default: true, Description: "Retrieve rows one at a time"},
					{Name: "max_allowed_packet", Type: "string", Default: "16M", Description: "Maximum packet size"},
					{Name: "single-transaction", Type: "bool", Default: false, Description: "Use single transaction for dump"},
					{Name: "routines", Type: "bool", Default: false, Description: "Dump stored routines"},
					{Name: "triggers", Type: "bool", Default: true, Description: "Dump triggers"},
					{Name: "events", Type: "bool", Default: false, Description: "Dump events"},
				},
			},
			{
				Name:        "mysqld_safe",
				Required:    false,
				Multiple:    false,
				Description: "mysqld_safe wrapper configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "socket", Type: "string", Description: "Unix socket file path"},
					{Name: "nice", Type: "int", Description: "Nice priority value"},
					{Name: "open-files-limit", Type: "int", Description: "Open files limit"},
					{Name: "syslog", Type: "bool", Default: false, Description: "Log to syslog"},
					{Name: "malloc-lib", Type: "string", Description: "Alternative malloc library"},
				},
			},
		},
	}
}

// DefaultConfig returns sensible defaults for MySQL.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"mysqld": map[string]any{
			"port":                           3306,
			"bind-address":                   "127.0.0.1",
			"max_connections":                151,
			"max_allowed_packet":             "16M",
			"character-set-server":           "utf8mb4",
			"collation-server":               "utf8mb4_unicode_ci",
			"innodb_buffer_pool_size":        "128M",
			"innodb_log_file_size":           "48M",
			"innodb_file_per_table":          true,
			"innodb_flush_log_at_trx_commit": 1,
			"innodb_lock_wait_timeout":       50,
			"slow_query_log":                 false,
			"long_query_time":                10.0,
			"general_log":                    false,
			"thread_cache_size":              8,
			"table_open_cache":               2000,
			"skip-name-resolve":              false,
			"local_infile":                   false,
			"read_only":                      false,
		},
		"mysql": map[string]any{
			"port":                  3306,
			"default-character-set": "utf8mb4",
			"no-auto-rehash":        false,
			"safe-updates":          false,
		},
		"client": map[string]any{
			"port":                  3306,
			"default-character-set": "utf8mb4",
		},
		"mysqldump": map[string]any{
			"quick":              true,
			"max_allowed_packet": "16M",
			"single-transaction": false,
			"routines":           false,
			"triggers":           true,
			"events":             false,
		},
	}
}

// Validate validates the MySQL configuration.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Validate mysqld section
	if mysqld, ok := configMap["mysqld"].(map[string]any); ok {
		errors = append(errors, p.validateMysqldSection(mysqld)...)
	}

	// Validate mysql section
	if mysql, ok := configMap["mysql"].(map[string]any); ok {
		errors = append(errors, p.validateClientSection("mysql", mysql)...)
	}

	// Validate client section
	if client, ok := configMap["client"].(map[string]any); ok {
		errors = append(errors, p.validateClientSection("client", client)...)
	}

	return errors, nil
}

// validateMysqldSection validates the mysqld section.
func (p *Plugin) validateMysqldSection(mysqld map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	errors = append(errors, validateMySQLPort(mysqld)...)
	errors = append(errors, validateMySQLBufferSizes(mysqld)...)
	errors = append(errors, validateMySQLFlushLog(mysqld)...)
	errors = append(errors, validateMySQLMaxConnections(mysqld)...)
	errors = append(errors, validateMySQLGTIDMode(mysqld)...)
	return errors
}

func validateMySQLPort(mysqld map[string]any) []plugin.ValidationError {
	port, ok := mysqld["port"]
	if !ok {
		return nil
	}
	portNum := toInt(port)
	if portNum >= 0 && portNum <= 65535 {
		return nil
	}
	return []plugin.ValidationError{{
		Path:    "mysqld.port",
		Message: fmt.Sprintf("invalid port number: %d (must be 0-65535)", portNum),
		Value:   port,
	}}
}

func validateMySQLBufferSizes(mysqld map[string]any) []plugin.ValidationError {
	fields := map[string]string{
		"innodb_buffer_pool_size": "128M, 1G",
		"innodb_log_file_size":    "48M, 1G",
		"max_allowed_packet":      "16M, 1G",
	}
	var errors []plugin.ValidationError
	for field, example := range fields {
		size, ok := mysqld[field].(string)
		if !ok || size == "" || isValidBufferSize(size) {
			continue
		}
		errors = append(errors, plugin.ValidationError{
			Path:    "mysqld." + field,
			Message: fmt.Sprintf("invalid buffer size format: %s (expected: %s, etc.)", size, example),
			Value:   size,
		})
	}
	return errors
}

func validateMySQLFlushLog(mysqld map[string]any) []plugin.ValidationError {
	val, ok := mysqld["innodb_flush_log_at_trx_commit"]
	if !ok {
		return nil
	}
	v := toInt(val)
	if v >= 0 && v <= 2 {
		return nil
	}
	return []plugin.ValidationError{{
		Path:    "mysqld.innodb_flush_log_at_trx_commit",
		Message: fmt.Sprintf("invalid value: %d (must be 0, 1, or 2)", v),
		Value:   val,
	}}
}

func validateMySQLMaxConnections(mysqld map[string]any) []plugin.ValidationError {
	val, ok := mysqld["max_connections"]
	if !ok {
		return nil
	}
	v := toInt(val)
	if v >= 1 && v <= 100000 {
		return nil
	}
	return []plugin.ValidationError{{
		Path:    "mysqld.max_connections",
		Message: fmt.Sprintf("invalid value: %d (must be 1-100000)", v),
		Value:   val,
	}}
}

func validateMySQLGTIDMode(mysqld map[string]any) []plugin.ValidationError {
	gtidMode, ok := mysqld["gtid_mode"].(string)
	if !ok || gtidMode == "" {
		return nil
	}
	validModes := []string{"OFF", "OFF_PERMISSIVE", "ON_PERMISSIVE", "ON"}
	for _, mode := range validModes {
		if strings.EqualFold(gtidMode, mode) {
			return nil
		}
	}
	return []plugin.ValidationError{{
		Path:    "mysqld.gtid_mode",
		Message: fmt.Sprintf("invalid gtid_mode: %s (must be one of: %s)", gtidMode, strings.Join(validModes, ", ")),
		Value:   gtidMode,
	}}
}

// validateClientSection validates a client section (mysql, client).
func (p *Plugin) validateClientSection(sectionName string, section map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate port
	if port, ok := section["port"]; ok {
		portNum := toInt(port)
		if portNum < 0 || portNum > 65535 {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.port", sectionName),
				Message: fmt.Sprintf("invalid port number: %d (must be 0-65535)", portNum),
				Value:   port,
			})
		}
	}

	return errors
}

// ValidateSemantic performs MySQL-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	mysqld, _ := configMap["mysqld"].(map[string]any)
	if mysqld == nil {
		return errors, nil
	}

	// Check datadir path validation
	if datadir, ok := mysqld["datadir"].(string); ok && datadir != "" {
		if !strings.HasPrefix(datadir, "/") {
			errors = append(errors, plugin.ValidationError{
				Path:    "mysqld.datadir",
				Message: fmt.Sprintf("datadir should be an absolute path: %s", datadir),
				Value:   datadir,
			})
		}
	}

	// Check bind-address security
	if bindAddr, ok := mysqld["bind-address"].(string); ok {
		if bindAddr == "0.0.0.0" || bindAddr == "*" || bindAddr == "::" {
			errors = append(errors, plugin.ValidationError{
				Path:    "mysqld.bind-address",
				Message: "binding to all interfaces exposes MySQL to the network - ensure firewall is configured",
				Value:   bindAddr,
			})
		}
	}

	// Check innodb_buffer_pool_size warning
	if size, ok := mysqld["innodb_buffer_pool_size"].(string); ok && size != "" {
		sizeBytes := parseBufferSize(size)
		// Warn if buffer pool is larger than 16GB (arbitrary but reasonable threshold for warning)
		if sizeBytes > 16*1024*1024*1024 {
			errors = append(errors, plugin.ValidationError{
				Path:    "mysqld.innodb_buffer_pool_size",
				Message: fmt.Sprintf("very large buffer pool size (%s) - ensure system has sufficient memory", size),
				Value:   size,
			})
		}
	}

	// Check replication configuration consistency
	if serverID, ok := mysqld["server-id"]; ok {
		if toInt(serverID) > 0 {
			// Server has a server-id, likely for replication
			if _, hasLogBin := mysqld["log_bin"]; !hasLogBin {
				errors = append(errors, plugin.ValidationError{
					Path:    "mysqld.server-id",
					Message: "server-id is set but log_bin is not configured - binary logging is typically required for replication",
					Value:   serverID,
				})
			}
		}
	}

	// Check if slow query log is enabled but no file specified
	if slowLog, ok := mysqld["slow_query_log"].(bool); ok && slowLog {
		if _, hasFile := mysqld["slow_query_log_file"]; !hasFile {
			errors = append(errors, plugin.ValidationError{
				Path:    "mysqld.slow_query_log",
				Message: "slow_query_log is enabled but slow_query_log_file is not specified",
				Value:   slowLog,
			})
		}
	}

	// Check if general log is enabled but no file specified
	if generalLog, ok := mysqld["general_log"].(bool); ok && generalLog {
		if _, hasFile := mysqld["general_log_file"]; !hasFile {
			errors = append(errors, plugin.ValidationError{
				Path:    "mysqld.general_log",
				Message: "general_log is enabled but general_log_file is not specified",
				Value:   generalLog,
			})
		}
	}

	// Warn about deprecated query cache settings
	if _, ok := mysqld["query_cache_type"]; ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "mysqld.query_cache_type",
			Message: "query_cache_type is deprecated and removed in MySQL 8.0",
			Value:   mysqld["query_cache_type"],
		})
	}
	if _, ok := mysqld["query_cache_size"]; ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "mysqld.query_cache_size",
			Message: "query_cache_size is deprecated and removed in MySQL 8.0",
			Value:   mysqld["query_cache_size"],
		})
	}

	return errors, nil
}

// isValidBufferSize checks if a string is a valid MySQL buffer size.
func isValidBufferSize(s string) bool {
	pattern := regexp.MustCompile(`^(\d+)(K|M|G|T)?$`)
	return pattern.MatchString(strings.ToUpper(s))
}

// parseBufferSize parses a buffer size string to bytes.
func parseBufferSize(s string) int64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	pattern := regexp.MustCompile(`^(\d+)(K|M|G|T)?$`)
	matches := pattern.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0
	}

	if len(matches) == 3 {
		switch matches[2] {
		case "K":
			value *= 1024
		case "M":
			value *= 1024 * 1024
		case "G":
			value *= 1024 * 1024 * 1024
		case "T":
			value *= 1024 * 1024 * 1024 * 1024
		}
	}

	return value
}

// toInt converts various numeric types to int.
func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

// Normalize normalizes the MySQL configuration to canonical form.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	result := make(map[string]any)

	// Normalize each section
	for section, content := range configMap {
		if sectionMap, ok := content.(map[string]any); ok {
			normalized := make(map[string]any)
			for k, v := range sectionMap {
				// Normalize buffer sizes to uppercase
				if isBufferSizeKey(k) {
					if str, ok := v.(string); ok {
						normalized[k] = strings.ToUpper(str)
					} else {
						normalized[k] = v
					}
				} else {
					normalized[k] = v
				}
			}
			result[section] = normalized
		} else {
			result[section] = content
		}
	}

	return result, nil
}

// isBufferSizeKey checks if a key typically contains buffer size values.
func isBufferSizeKey(key string) bool {
	sizeKeys := []string{
		"innodb_buffer_pool_size",
		"innodb_log_file_size",
		"max_allowed_packet",
		"query_cache_size",
		"key_buffer_size",
		"sort_buffer_size",
		"read_buffer_size",
		"read_rnd_buffer_size",
		"join_buffer_size",
		"tmp_table_size",
		"max_heap_table_size",
	}
	for _, sk := range sizeKeys {
		if key == sk {
			return true
		}
	}
	return false
}

// ToNative converts the configuration to native MySQL INI format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	cfg := ini.Empty()

	// Get sorted section names for consistent output
	sections := make([]string, 0, len(configMap))
	for section := range configMap {
		sections = append(sections, section)
	}
	sort.Strings(sections)

	for _, sectionName := range sections {
		sectionData := configMap[sectionName]
		sectionMap, ok := sectionData.(map[string]any)
		if !ok {
			continue
		}

		section, err := cfg.NewSection(sectionName)
		if err != nil {
			return nil, fmt.Errorf("failed to create section %s: %w", sectionName, err)
		}

		// Get sorted keys for consistent output
		keys := make([]string, 0, len(sectionMap))
		for k := range sectionMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := sectionMap[key]
			if err := writeINIValue(section, key, value); err != nil {
				return nil, err
			}
		}
	}

	var buf strings.Builder
	_, err := cfg.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write INI: %w", err)
	}

	return []byte(buf.String()), nil
}

// writeINIValue writes a single INI value.
func writeINIValue(section *ini.Section, key string, value any) error {
	switch v := value.(type) {
	case nil:
		return nil
	case bool:
		if v {
			_, err := section.NewKey(key, "1")
			return err
		}
		_, err := section.NewKey(key, "0")
		return err
	case int, int64, float64:
		_, err := section.NewKey(key, fmt.Sprintf("%v", v))
		return err
	case string:
		_, err := section.NewKey(key, v)
		return err
	default:
		_, err := section.NewKey(key, fmt.Sprintf("%v", v))
		return err
	}
}

// FromNative parses native MySQL INI configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse INI: %w", err)
	}

	config := make(map[string]any)

	for _, section := range cfg.Sections() {
		sectionName := section.Name()
		if sectionName == "DEFAULT" {
			continue
		}

		sectionMap := make(map[string]any)
		for _, key := range section.Keys() {
			sectionMap[key.Name()] = convertINIValue(key.Value())
		}

		if len(sectionMap) > 0 {
			config[sectionName] = sectionMap
		}
	}

	return config, nil
}

// convertINIValue converts an INI value to appropriate Go type.
func convertINIValue(s string) any {
	// Check for boolean
	lower := strings.ToLower(s)
	if lower == "1" || lower == "true" || lower == "yes" || lower == "on" {
		return true
	}
	if lower == "0" || lower == "false" || lower == "no" || lower == "off" {
		return false
	}

	// Check for integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}

	// Check for float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		// Only return float if it has decimal places
		if strings.Contains(s, ".") {
			return f
		}
	}

	return s
}

// Merge merges two MySQL configurations.
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

	// Copy base sections
	for section, content := range baseMap {
		if sectionMap, ok := content.(map[string]any); ok {
			newSection := make(map[string]any)
			for k, v := range sectionMap {
				newSection[k] = v
			}
			result[section] = newSection
		} else {
			result[section] = content
		}
	}

	// Merge overlay sections
	for section, content := range overlayMap {
		if overlaySectionMap, ok := content.(map[string]any); ok {
			if baseSectionMap, ok := result[section].(map[string]any); ok {
				// Merge into existing section
				for k, v := range overlaySectionMap {
					baseSectionMap[k] = v
				}
			} else {
				// New section
				newSection := make(map[string]any)
				for k, v := range overlaySectionMap {
					newSection[k] = v
				}
				result[section] = newSection
			}
		} else {
			result[section] = content
		}
	}

	return result, nil
}

// Diff computes the differences between two configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	var changes []plugin.Change

	oldMap, _ := old.(map[string]any)
	newMap, _ := new.(map[string]any)

	// Check for removed and modified sections
	for section, oldContent := range oldMap {
		newContent, exists := newMap[section]
		if !exists {
			changes = append(changes, plugin.Change{
				Path:     section,
				Type:     "remove",
				OldValue: oldContent,
			})
			continue
		}

		// Compare section contents
		sectionChanges := diffSections(section, oldContent, newContent)
		changes = append(changes, sectionChanges...)
	}

	// Check for added sections
	for section, newContent := range newMap {
		if _, exists := oldMap[section]; !exists {
			changes = append(changes, plugin.Change{
				Path:     section,
				Type:     "add",
				NewValue: newContent,
			})
		}
	}

	return changes, nil
}

// diffSections compares two section contents.
func diffSections(sectionName string, oldContent, newContent any) []plugin.Change {
	var changes []plugin.Change

	oldMap, oldOk := oldContent.(map[string]any)
	newMap, newOk := newContent.(map[string]any)

	if !oldOk || !newOk {
		if !equalValues(oldContent, newContent) {
			changes = append(changes, plugin.Change{
				Path:     sectionName,
				Type:     "modify",
				OldValue: oldContent,
				NewValue: newContent,
			})
		}
		return changes
	}

	// Check for removed and modified keys
	for key, oldVal := range oldMap {
		path := fmt.Sprintf("%s.%s", sectionName, key)
		newVal, exists := newMap[key]
		if !exists {
			changes = append(changes, plugin.Change{
				Path:     path,
				Type:     "remove",
				OldValue: oldVal,
			})
			continue
		}

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
	for key, newVal := range newMap {
		if _, exists := oldMap[key]; !exists {
			path := fmt.Sprintf("%s.%s", sectionName, key)
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

// Ensure Plugin implements AppPlugin interface.
var _ plugin.AppPlugin = (*Plugin)(nil)
