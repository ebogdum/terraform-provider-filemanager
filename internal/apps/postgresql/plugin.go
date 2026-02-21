// SPDX-License-Identifier: MIT

// Package postgresql provides a PostgreSQL configuration management plugin.
package postgresql

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

// Plugin implements the AppPlugin interface for PostgreSQL.
type Plugin struct{}

// New creates a new PostgreSQL plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "postgresql"
}

// Version returns the supported PostgreSQL version range.
func (p *Plugin) Version() string {
	return ">=12.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "PostgreSQL database configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "postgresql"
}

// Schema returns the configuration schema for PostgreSQL.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "connection",
				Required:    false,
				Multiple:    false,
				Description: "Connection settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "listen_addresses", Type: "string", Description: "IP addresses to listen on (comma-separated or '*')"},
					{Name: "port", Type: "int", Default: 5432, Description: "TCP port to listen on"},
					{Name: "max_connections", Type: "int", Default: 100, Description: "Maximum number of concurrent connections"},
					{Name: "superuser_reserved_connections", Type: "int", Default: 3, Description: "Connections reserved for superusers"},
					{Name: "unix_socket_directories", Type: "string", Description: "Directories for Unix socket"},
					{Name: "unix_socket_permissions", Type: "string", Description: "Unix socket permissions"},
					{Name: "bonjour", Type: "bool", Description: "Enable Bonjour service discovery"},
					{Name: "tcp_keepalives_idle", Type: "int", Description: "TCP keepalive idle time"},
					{Name: "tcp_keepalives_interval", Type: "int", Description: "TCP keepalive interval"},
					{Name: "tcp_keepalives_count", Type: "int", Description: "TCP keepalive count"},
				},
			},
			{
				Name:        "memory",
				Required:    false,
				Multiple:    false,
				Description: "Memory settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "shared_buffers", Type: "string", Default: "128MB", Description: "Shared memory buffers"},
					{Name: "effective_cache_size", Type: "string", Default: "4GB", Description: "Planner's assumption about disk cache"},
					{Name: "work_mem", Type: "string", Default: "4MB", Description: "Memory for internal sort operations"},
					{Name: "maintenance_work_mem", Type: "string", Default: "64MB", Description: "Memory for maintenance operations"},
					{Name: "temp_buffers", Type: "string", Default: "8MB", Description: "Maximum memory for temporary buffers"},
					{Name: "huge_pages", Type: "string", ValidValues: []string{"off", "on", "try"}, Description: "Use of huge pages"},
					{Name: "effective_io_concurrency", Type: "int", Default: 1, Description: "Number of concurrent disk I/O operations"},
					{Name: "max_worker_processes", Type: "int", Default: 8, Description: "Maximum number of background processes"},
					{Name: "max_parallel_workers_per_gather", Type: "int", Default: 2, Description: "Maximum parallel workers per gather"},
					{Name: "max_parallel_workers", Type: "int", Default: 8, Description: "Maximum number of parallel workers"},
					{Name: "max_parallel_maintenance_workers", Type: "int", Default: 2, Description: "Maximum parallel maintenance workers"},
				},
			},
			{
				Name:        "wal",
				Required:    false,
				Multiple:    false,
				Description: "Write-ahead log settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "wal_level", Type: "string", Default: "replica", ValidValues: []string{"minimal", "replica", "logical"}, Description: "WAL level"},
					{Name: "max_wal_size", Type: "string", Default: "1GB", Description: "Maximum WAL size before checkpoint"},
					{Name: "min_wal_size", Type: "string", Default: "80MB", Description: "Minimum WAL size to retain"},
					{Name: "checkpoint_completion_target", Type: "float", Default: 0.9, Description: "Checkpoint completion target"},
					{Name: "checkpoint_timeout", Type: "string", Default: "5min", Description: "Time between automatic checkpoints"},
					{Name: "wal_buffers", Type: "string", Default: "-1", Description: "WAL buffer size (-1 for auto)"},
					{Name: "wal_writer_delay", Type: "string", Default: "200ms", Description: "WAL writer delay"},
					{Name: "commit_delay", Type: "int", Default: 0, Description: "Commit delay in microseconds"},
					{Name: "commit_siblings", Type: "int", Default: 5, Description: "Minimum concurrent transactions for commit_delay"},
					{Name: "fsync", Type: "bool", Default: true, Description: "Force synchronization of updates to disk"},
					{Name: "synchronous_commit", Type: "string", ValidValues: []string{"off", "local", "remote_write", "remote_apply", "on"}, Description: "Synchronous commit level"},
					{Name: "wal_sync_method", Type: "string", Description: "WAL synchronization method"},
					{Name: "full_page_writes", Type: "bool", Default: true, Description: "Write full pages to WAL after checkpoint"},
					{Name: "wal_compression", Type: "string", ValidValues: []string{"off", "pglz", "lz4", "zstd", "on"}, Description: "WAL compression method"},
				},
			},
			{
				Name:        "logging",
				Required:    false,
				Multiple:    false,
				Description: "Logging settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "log_destination", Type: "string", Default: "stderr", Description: "Log destination (stderr, csvlog, syslog, eventlog)"},
					{Name: "logging_collector", Type: "bool", Default: false, Description: "Enable log collector"},
					{Name: "log_directory", Type: "string", Default: "log", Description: "Log directory"},
					{Name: "log_filename", Type: "string", Default: "postgresql-%Y-%m-%d_%H%M%S.log", Description: "Log filename pattern"},
					{Name: "log_rotation_age", Type: "string", Default: "1d", Description: "Log rotation age"},
					{Name: "log_rotation_size", Type: "string", Default: "10MB", Description: "Log rotation size"},
					{Name: "log_min_messages", Type: "string", ValidValues: []string{"DEBUG5", "DEBUG4", "DEBUG3", "DEBUG2", "DEBUG1", "INFO", "NOTICE", "WARNING", "ERROR", "LOG", "FATAL", "PANIC"}, Description: "Minimum message level to log"},
					{Name: "log_min_error_statement", Type: "string", ValidValues: []string{"DEBUG5", "DEBUG4", "DEBUG3", "DEBUG2", "DEBUG1", "INFO", "NOTICE", "WARNING", "ERROR", "LOG", "FATAL", "PANIC"}, Description: "Minimum error level to log statements"},
					{Name: "log_min_duration_statement", Type: "int", Default: -1, Description: "Log statements running longer than this (ms, -1 to disable)"},
					{Name: "log_checkpoints", Type: "bool", Default: true, Description: "Log checkpoints"},
					{Name: "log_connections", Type: "bool", Default: false, Description: "Log connections"},
					{Name: "log_disconnections", Type: "bool", Default: false, Description: "Log disconnections"},
					{Name: "log_duration", Type: "bool", Default: false, Description: "Log statement duration"},
					{Name: "log_error_verbosity", Type: "string", ValidValues: []string{"TERSE", "DEFAULT", "VERBOSE"}, Description: "Error verbosity level"},
					{Name: "log_line_prefix", Type: "string", Default: "%m [%p] ", Description: "Log line prefix format"},
					{Name: "log_lock_waits", Type: "bool", Default: false, Description: "Log lock waits"},
					{Name: "log_statement", Type: "string", ValidValues: []string{"none", "ddl", "mod", "all"}, Description: "Log statement types"},
					{Name: "log_temp_files", Type: "int", Default: -1, Description: "Log temp files of this size or larger (-1 to disable)"},
					{Name: "log_timezone", Type: "string", Description: "Timezone for log timestamps"},
				},
			},
			{
				Name:        "replication",
				Required:    false,
				Multiple:    false,
				Description: "Replication settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "max_wal_senders", Type: "int", Default: 10, Description: "Maximum number of WAL sender processes"},
					{Name: "max_replication_slots", Type: "int", Default: 10, Description: "Maximum number of replication slots"},
					{Name: "wal_keep_size", Type: "string", Default: "0", Description: "Minimum WAL to keep for standbys"},
					{Name: "max_slot_wal_keep_size", Type: "string", Default: "-1", Description: "Maximum WAL kept for replication slots"},
					{Name: "hot_standby", Type: "bool", Default: true, Description: "Allow queries during recovery"},
					{Name: "hot_standby_feedback", Type: "bool", Default: false, Description: "Send feedback to prevent query conflicts"},
					{Name: "synchronous_standby_names", Type: "string", Description: "Standby servers for synchronous replication"},
					{Name: "primary_conninfo", Type: "string", Description: "Connection string to primary server"},
					{Name: "primary_slot_name", Type: "string", Description: "Replication slot on primary server"},
					{Name: "recovery_target_timeline", Type: "string", Description: "Recovery timeline"},
					{Name: "archive_mode", Type: "string", ValidValues: []string{"off", "on", "always"}, Description: "Archive mode"},
					{Name: "archive_command", Type: "string", Description: "Archive command"},
					{Name: "archive_timeout", Type: "string", Default: "0", Description: "Archive timeout"},
					{Name: "restore_command", Type: "string", Description: "Restore command for recovery"},
				},
			},
			{
				Name:        "performance",
				Required:    false,
				Multiple:    false,
				Description: "Performance tuning settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "random_page_cost", Type: "float", Default: 4.0, Description: "Planner's estimate of random page cost"},
					{Name: "seq_page_cost", Type: "float", Default: 1.0, Description: "Planner's estimate of sequential page cost"},
					{Name: "cpu_tuple_cost", Type: "float", Default: 0.01, Description: "Planner's estimate of CPU tuple processing cost"},
					{Name: "cpu_index_tuple_cost", Type: "float", Default: 0.005, Description: "Planner's estimate of CPU index tuple processing cost"},
					{Name: "cpu_operator_cost", Type: "float", Default: 0.0025, Description: "Planner's estimate of CPU operator processing cost"},
					{Name: "parallel_tuple_cost", Type: "float", Default: 0.1, Description: "Planner's estimate of parallel tuple cost"},
					{Name: "parallel_setup_cost", Type: "float", Default: 1000.0, Description: "Planner's estimate of parallel setup cost"},
					{Name: "min_parallel_table_scan_size", Type: "string", Default: "8MB", Description: "Minimum table size for parallel scan"},
					{Name: "min_parallel_index_scan_size", Type: "string", Default: "512kB", Description: "Minimum index size for parallel scan"},
					{Name: "default_statistics_target", Type: "int", Default: 100, Description: "Default statistics target"},
					{Name: "constraint_exclusion", Type: "string", ValidValues: []string{"off", "on", "partition"}, Description: "Constraint exclusion mode"},
					{Name: "cursor_tuple_fraction", Type: "float", Default: 0.1, Description: "Planner's estimate of cursor row fetch fraction"},
					{Name: "from_collapse_limit", Type: "int", Default: 8, Description: "FROM list collapse limit"},
					{Name: "join_collapse_limit", Type: "int", Default: 8, Description: "JOIN collapse limit"},
					{Name: "jit", Type: "bool", Default: true, Description: "Enable JIT compilation"},
					{Name: "jit_above_cost", Type: "float", Default: 100000, Description: "Query cost above which JIT is used"},
					{Name: "jit_inline_above_cost", Type: "float", Default: 500000, Description: "Query cost above which JIT inlines functions"},
					{Name: "jit_optimize_above_cost", Type: "float", Default: 500000, Description: "Query cost above which JIT optimizes"},
				},
			},
			{
				Name:        "autovacuum",
				Required:    false,
				Multiple:    false,
				Description: "Autovacuum settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "autovacuum", Type: "bool", Default: true, Description: "Enable autovacuum"},
					{Name: "autovacuum_max_workers", Type: "int", Default: 3, Description: "Maximum autovacuum workers"},
					{Name: "autovacuum_naptime", Type: "string", Default: "1min", Description: "Time between autovacuum runs"},
					{Name: "autovacuum_vacuum_threshold", Type: "int", Default: 50, Description: "Minimum tuple updates before vacuum"},
					{Name: "autovacuum_analyze_threshold", Type: "int", Default: 50, Description: "Minimum tuple updates before analyze"},
					{Name: "autovacuum_vacuum_scale_factor", Type: "float", Default: 0.2, Description: "Fraction of table to trigger vacuum"},
					{Name: "autovacuum_analyze_scale_factor", Type: "float", Default: 0.1, Description: "Fraction of table to trigger analyze"},
					{Name: "autovacuum_freeze_max_age", Type: "int", Default: 200000000, Description: "Age at which to autovacuum to prevent wraparound"},
					{Name: "autovacuum_multixact_freeze_max_age", Type: "int", Default: 400000000, Description: "Multixact age at which to autovacuum"},
					{Name: "autovacuum_vacuum_cost_delay", Type: "string", Default: "2ms", Description: "Vacuum cost delay"},
					{Name: "autovacuum_vacuum_cost_limit", Type: "int", Default: -1, Description: "Vacuum cost limit (-1 for default)"},
				},
			},
			{
				Name:        "security",
				Required:    false,
				Multiple:    false,
				Description: "Security settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "authentication_timeout", Type: "string", Default: "1min", Description: "Maximum time for authentication"},
					{Name: "password_encryption", Type: "string", Default: "scram-sha-256", ValidValues: []string{"md5", "scram-sha-256"}, Description: "Password encryption method"},
					{Name: "ssl", Type: "bool", Default: false, Description: "Enable SSL connections"},
					{Name: "ssl_ca_file", Type: "string", Description: "SSL CA file"},
					{Name: "ssl_cert_file", Type: "string", Description: "SSL certificate file"},
					{Name: "ssl_key_file", Type: "string", Description: "SSL key file"},
					{Name: "ssl_ciphers", Type: "string", Description: "SSL cipher suites"},
					{Name: "ssl_prefer_server_ciphers", Type: "bool", Default: true, Description: "Prefer server cipher order"},
					{Name: "ssl_min_protocol_version", Type: "string", ValidValues: []string{"TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"}, Description: "Minimum SSL protocol version"},
					{Name: "ssl_max_protocol_version", Type: "string", ValidValues: []string{"TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"}, Description: "Maximum SSL protocol version"},
					{Name: "row_security", Type: "bool", Default: true, Description: "Enable row security"},
				},
			},
			{
				Name:        "client",
				Required:    false,
				Multiple:    false,
				Description: "Client connection defaults",
				Directives: []plugin.DirectiveSchema{
					{Name: "search_path", Type: "string", Description: "Schema search path"},
					{Name: "default_tablespace", Type: "string", Description: "Default tablespace"},
					{Name: "temp_tablespaces", Type: "string", Description: "Tablespaces for temporary objects"},
					{Name: "default_transaction_isolation", Type: "string", ValidValues: []string{"serializable", "repeatable read", "read committed", "read uncommitted"}, Description: "Default transaction isolation level"},
					{Name: "default_transaction_read_only", Type: "bool", Default: false, Description: "Default transaction read-only mode"},
					{Name: "default_transaction_deferrable", Type: "bool", Default: false, Description: "Default transaction deferrable mode"},
					{Name: "session_replication_role", Type: "string", ValidValues: []string{"origin", "replica", "local"}, Description: "Session replication role"},
					{Name: "statement_timeout", Type: "int", Default: 0, Description: "Statement timeout in milliseconds"},
					{Name: "lock_timeout", Type: "int", Default: 0, Description: "Lock timeout in milliseconds"},
					{Name: "idle_in_transaction_session_timeout", Type: "int", Default: 0, Description: "Idle transaction timeout in milliseconds"},
					{Name: "idle_session_timeout", Type: "int", Default: 0, Description: "Idle session timeout in milliseconds"},
					{Name: "client_min_messages", Type: "string", ValidValues: []string{"DEBUG5", "DEBUG4", "DEBUG3", "DEBUG2", "DEBUG1", "LOG", "NOTICE", "WARNING", "ERROR"}, Description: "Minimum client message level"},
					{Name: "timezone", Type: "string", Description: "Default timezone"},
					{Name: "datestyle", Type: "string", Description: "Date display format"},
					{Name: "lc_messages", Type: "string", Description: "Locale for messages"},
					{Name: "lc_monetary", Type: "string", Description: "Locale for monetary formatting"},
					{Name: "lc_numeric", Type: "string", Description: "Locale for numeric formatting"},
					{Name: "lc_time", Type: "string", Description: "Locale for time formatting"},
				},
			},
			{
				Name:        "resource",
				Required:    false,
				Multiple:    false,
				Description: "Resource usage settings",
				Directives: []plugin.DirectiveSchema{
					{Name: "max_files_per_process", Type: "int", Default: 1000, Description: "Maximum open files per process"},
					{Name: "shared_preload_libraries", Type: "string", Description: "Libraries to preload at server start"},
					{Name: "dynamic_shared_memory_type", Type: "string", ValidValues: []string{"posix", "sysv", "windows", "mmap"}, Description: "Dynamic shared memory implementation"},
					{Name: "vacuum_cost_delay", Type: "string", Default: "0", Description: "Vacuum cost delay"},
					{Name: "vacuum_cost_page_hit", Type: "int", Default: 1, Description: "Vacuum cost for buffer hit"},
					{Name: "vacuum_cost_page_miss", Type: "int", Default: 2, Description: "Vacuum cost for buffer miss"},
					{Name: "vacuum_cost_page_dirty", Type: "int", Default: 20, Description: "Vacuum cost for dirtying a page"},
					{Name: "vacuum_cost_limit", Type: "int", Default: 200, Description: "Vacuum cost limit before sleep"},
					{Name: "bgwriter_delay", Type: "string", Default: "200ms", Description: "Background writer delay"},
					{Name: "bgwriter_lru_maxpages", Type: "int", Default: 100, Description: "Background writer maximum pages"},
					{Name: "bgwriter_lru_multiplier", Type: "float", Default: 2.0, Description: "Background writer LRU multiplier"},
					{Name: "bgwriter_flush_after", Type: "string", Default: "512kB", Description: "Background writer flush after"},
				},
			},
		},
	}
}

// DefaultConfig returns sensible defaults for PostgreSQL.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		// Connection
		"listen_addresses":               "localhost",
		"port":                           5432,
		"max_connections":                100,
		"superuser_reserved_connections": 3,
		// Memory
		"shared_buffers":                  "128MB",
		"effective_cache_size":            "4GB",
		"work_mem":                        "4MB",
		"maintenance_work_mem":            "64MB",
		"effective_io_concurrency":        1,
		"max_worker_processes":            8,
		"max_parallel_workers_per_gather": 2,
		"max_parallel_workers":            8,
		// WAL
		"wal_level":                    "replica",
		"max_wal_size":                 "1GB",
		"min_wal_size":                 "80MB",
		"checkpoint_completion_target": 0.9,
		"fsync":                        true,
		"synchronous_commit":           "on",
		"full_page_writes":             true,
		// Logging
		"log_destination":         "stderr",
		"logging_collector":       false,
		"log_directory":           "log",
		"log_filename":            "postgresql-%Y-%m-%d_%H%M%S.log",
		"log_checkpoints":         true,
		"log_line_prefix":         "%m [%p] ",
		"log_min_messages":        "WARNING",
		"log_min_error_statement": "ERROR",
		// Replication
		"max_wal_senders":       10,
		"max_replication_slots": 10,
		"hot_standby":           true,
		"archive_mode":          "off",
		// Performance
		"random_page_cost":          4.0,
		"seq_page_cost":             1.0,
		"default_statistics_target": 100,
		"jit":                       true,
		// Autovacuum
		"autovacuum":                      true,
		"autovacuum_max_workers":          3,
		"autovacuum_naptime":              "1min",
		"autovacuum_vacuum_threshold":     50,
		"autovacuum_analyze_threshold":    50,
		"autovacuum_vacuum_scale_factor":  0.2,
		"autovacuum_analyze_scale_factor": 0.1,
		// Security
		"authentication_timeout": "1min",
		"password_encryption":    "scram-sha-256",
		"ssl":                    false,
		"row_security":           true,
		// Client
		"default_transaction_read_only":       false,
		"statement_timeout":                   0,
		"lock_timeout":                        0,
		"idle_in_transaction_session_timeout": 0,
	}
}

// Validate validates the PostgreSQL configuration.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var errors []plugin.ValidationError
	errors = append(errors, validatePostgresPort(configMap)...)
	errors = append(errors, validatePostgresMaxConnections(configMap)...)
	errors = append(errors, validatePostgresWalLevel(configMap)...)
	errors = append(errors, validatePostgresMemorySizes(configMap)...)
	errors = append(errors, validatePostgresSynchronousCommit(configMap)...)
	errors = append(errors, validatePostgresArchiveMode(configMap)...)
	errors = append(errors, validatePostgresPasswordEncryption(configMap)...)
	errors = append(errors, validatePostgresLogStatement(configMap)...)
	errors = append(errors, validatePostgresCheckpointCompletionTarget(configMap)...)
	return errors, nil
}

// ValidateSemantic performs PostgreSQL-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var errors []plugin.ValidationError
	errors = append(errors, validatePostgresSharedBuffers(configMap)...)
	errors = append(errors, validatePostgresReplicationWal(configMap)...)
	errors = append(errors, validatePostgresArchiveSettings(configMap)...)
	errors = append(errors, validatePostgresListenSSL(configMap)...)
	errors = append(errors, validatePostgresSynchronousReplication(configMap)...)
	errors = append(errors, validatePostgresWorkMemBudget(configMap)...)
	return errors, nil
}

func validatePostgresPort(configMap map[string]any) []plugin.ValidationError {
	port, ok := pgAnyToInt(configMap["port"])
	if !ok {
		return nil
	}
	if port >= 1 && port <= 65535 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "port",
		Message: fmt.Sprintf("invalid port number: %d (must be 1-65535)", port),
		Value:   configMap["port"],
	}}
}

func validatePostgresMaxConnections(configMap map[string]any) []plugin.ValidationError {
	maxConn, ok := pgAnyToInt(configMap["max_connections"])
	if !ok {
		return nil
	}
	if maxConn >= 1 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "max_connections",
		Message: fmt.Sprintf("invalid max_connections: %d (must be at least 1)", maxConn),
		Value:   configMap["max_connections"],
	}}
}

func validatePostgresWalLevel(configMap map[string]any) []plugin.ValidationError {
	walLevel, ok := configMap["wal_level"].(string)
	if !ok {
		return nil
	}

	validLevels := []string{"minimal", "replica", "logical"}
	if pgContainsFold(validLevels, walLevel) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "wal_level",
		Message: fmt.Sprintf("invalid wal_level: %s (must be one of: %s)", walLevel, strings.Join(validLevels, ", ")),
		Value:   walLevel,
	}}
}

func validatePostgresMemorySizes(configMap map[string]any) []plugin.ValidationError {
	memorySizeFields := []string{
		"shared_buffers", "effective_cache_size", "work_mem", "maintenance_work_mem",
		"temp_buffers", "max_wal_size", "min_wal_size", "wal_buffers",
	}

	var errors []plugin.ValidationError
	for _, field := range memorySizeFields {
		val, ok := configMap[field].(string)
		if !ok || val == "" || val == "-1" {
			continue
		}
		if isValidMemorySize(val) {
			continue
		}

		errors = append(errors, plugin.ValidationError{
			Path:    field,
			Message: fmt.Sprintf("invalid memory size format: %s (expected: 128MB, 1GB, 4kB, etc.)", val),
			Value:   val,
		})
	}
	return errors
}

func validatePostgresSynchronousCommit(configMap map[string]any) []plugin.ValidationError {
	syncCommit, ok := configMap["synchronous_commit"].(string)
	if !ok {
		return nil
	}

	validOptions := []string{"off", "local", "remote_write", "remote_apply", "on"}
	if pgContainsFold(validOptions, syncCommit) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "synchronous_commit",
		Message: fmt.Sprintf("invalid synchronous_commit: %s (must be one of: %s)", syncCommit, strings.Join(validOptions, ", ")),
		Value:   syncCommit,
	}}
}

func validatePostgresArchiveMode(configMap map[string]any) []plugin.ValidationError {
	archiveMode, ok := configMap["archive_mode"].(string)
	if !ok {
		return nil
	}

	validOptions := []string{"off", "on", "always"}
	if pgContainsFold(validOptions, archiveMode) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "archive_mode",
		Message: fmt.Sprintf("invalid archive_mode: %s (must be one of: %s)", archiveMode, strings.Join(validOptions, ", ")),
		Value:   archiveMode,
	}}
}

func validatePostgresPasswordEncryption(configMap map[string]any) []plugin.ValidationError {
	passEnc, ok := configMap["password_encryption"].(string)
	if !ok {
		return nil
	}

	validOptions := []string{"md5", "scram-sha-256"}
	if pgContainsFold(validOptions, passEnc) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "password_encryption",
		Message: fmt.Sprintf("invalid password_encryption: %s (must be one of: %s)", passEnc, strings.Join(validOptions, ", ")),
		Value:   passEnc,
	}}
}

func validatePostgresLogStatement(configMap map[string]any) []plugin.ValidationError {
	logStmt, ok := configMap["log_statement"].(string)
	if !ok {
		return nil
	}

	validOptions := []string{"none", "ddl", "mod", "all"}
	if pgContainsFold(validOptions, logStmt) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "log_statement",
		Message: fmt.Sprintf("invalid log_statement: %s (must be one of: %s)", logStmt, strings.Join(validOptions, ", ")),
		Value:   logStmt,
	}}
}

func validatePostgresCheckpointCompletionTarget(configMap map[string]any) []plugin.ValidationError {
	cct, ok := pgAnyToFloat(configMap["checkpoint_completion_target"])
	if !ok {
		return nil
	}
	if cct >= 0 && cct <= 1 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "checkpoint_completion_target",
		Message: fmt.Sprintf("invalid checkpoint_completion_target: %v (must be between 0.0 and 1.0)", configMap["checkpoint_completion_target"]),
		Value:   configMap["checkpoint_completion_target"],
	}}
}

func validatePostgresSharedBuffers(configMap map[string]any) []plugin.ValidationError {
	sharedBuffers, ok := configMap["shared_buffers"].(string)
	if !ok {
		return nil
	}

	sharedBytes := parseMemorySize(sharedBuffers)
	if sharedBytes <= 0 {
		return nil
	}

	var errors []plugin.ValidationError
	if sharedBytes < 128*1024*1024 {
		errors = append(errors, plugin.ValidationError{
			Path:    "shared_buffers",
			Message: fmt.Sprintf("shared_buffers (%s) is below recommended minimum of 128MB for production workloads", sharedBuffers),
			Value:   sharedBuffers,
		})
	}

	effectiveCache, ok := configMap["effective_cache_size"].(string)
	if !ok {
		return errors
	}

	cacheBytes := parseMemorySize(effectiveCache)
	if cacheBytes > 0 && cacheBytes < sharedBytes {
		errors = append(errors, plugin.ValidationError{
			Path:    "effective_cache_size",
			Message: fmt.Sprintf("effective_cache_size (%s) should typically be larger than shared_buffers (%s)", effectiveCache, sharedBuffers),
			Value:   effectiveCache,
		})
	}
	return errors
}

func validatePostgresReplicationWal(configMap map[string]any) []plugin.ValidationError {
	walLevel, _ := configMap["wal_level"].(string)
	maxWalSenders, _ := pgAnyToInt(configMap["max_wal_senders"])
	if maxWalSenders <= 0 || !strings.EqualFold(walLevel, "minimal") {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "wal_level",
		Message: "wal_level must be 'replica' or 'logical' when max_wal_senders > 0 for replication",
		Value:   walLevel,
	}}
}

func validatePostgresArchiveSettings(configMap map[string]any) []plugin.ValidationError {
	archiveMode, _ := configMap["archive_mode"].(string)
	archiveCommand, _ := configMap["archive_command"].(string)
	if (archiveMode != "on" && archiveMode != "always") || archiveCommand != "" {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "archive_mode",
		Message: "archive_mode is enabled but archive_command is not set",
		Value:   archiveMode,
	}}
}

func validatePostgresListenSSL(configMap map[string]any) []plugin.ValidationError {
	listenAddresses, _ := configMap["listen_addresses"].(string)
	sslEnabled, _ := configMap["ssl"].(bool)
	if (listenAddresses != "*" && listenAddresses != "0.0.0.0") || sslEnabled {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "listen_addresses",
		Message: "listening on all interfaces without SSL is a security risk",
		Value:   listenAddresses,
	}}
}

func validatePostgresSynchronousReplication(configMap map[string]any) []plugin.ValidationError {
	syncStandbyNames, _ := configMap["synchronous_standby_names"].(string)
	syncCommit, _ := configMap["synchronous_commit"].(string)
	if syncStandbyNames == "" || (syncCommit != "off" && syncCommit != "local") {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "synchronous_commit",
		Message: "synchronous_standby_names is set but synchronous_commit doesn't enable remote synchronization",
		Value:   syncCommit,
	}}
}

func validatePostgresWorkMemBudget(configMap map[string]any) []plugin.ValidationError {
	workMem, ok := configMap["work_mem"].(string)
	if !ok {
		return nil
	}

	workBytes := parseMemorySize(workMem)
	if workBytes <= 0 {
		return nil
	}

	maxConn := 100
	if parsed, ok := pgAnyToInt(configMap["max_connections"]); ok {
		maxConn = parsed
	}

	totalWorkMem := workBytes * int64(maxConn) * 2
	if totalWorkMem <= 64*1024*1024*1024 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "work_mem",
		Message: fmt.Sprintf("work_mem (%s) x max_connections (%d) could use significant memory (%s total)", workMem, maxConn, formatBytes(totalWorkMem)),
		Value:   workMem,
	}}
}

func pgAnyToInt(v any) (int, bool) {
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

func pgAnyToFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func pgContainsFold(values []string, candidate string) bool {
	for _, value := range values {
		if strings.EqualFold(value, candidate) {
			return true
		}
	}
	return false
}

// isValidMemorySize checks if a string is a valid PostgreSQL memory size.
func isValidMemorySize(s string) bool {
	pattern := regexp.MustCompile(`^(-?\d+)\s*(B|kB|KB|MB|GB|TB|PB)?$`)
	return pattern.MatchString(s)
}

// parseMemorySize parses a PostgreSQL memory size string to bytes.
func parseMemorySize(s string) int64 {
	pattern := regexp.MustCompile(`^(-?\d+)\s*(B|kB|KB|MB|GB|TB|PB)?$`)
	matches := pattern.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0
	}

	unit := "B"
	if len(matches) >= 3 && matches[2] != "" {
		unit = strings.ToUpper(matches[2])
	}

	switch unit {
	case "B":
		return value
	case "KB":
		return value * 1024
	case "MB":
		return value * 1024 * 1024
	case "GB":
		return value * 1024 * 1024 * 1024
	case "TB":
		return value * 1024 * 1024 * 1024 * 1024
	case "PB":
		return value * 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return value
	}
}

// formatBytes formats bytes to human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Normalize normalizes the PostgreSQL configuration to canonical form.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Normalize wal_level to lowercase
	if walLevel, ok := configMap["wal_level"].(string); ok {
		configMap["wal_level"] = strings.ToLower(walLevel)
	}

	// Normalize synchronous_commit to lowercase
	if syncCommit, ok := configMap["synchronous_commit"].(string); ok {
		configMap["synchronous_commit"] = strings.ToLower(syncCommit)
	}

	// Normalize archive_mode to lowercase
	if archiveMode, ok := configMap["archive_mode"].(string); ok {
		configMap["archive_mode"] = strings.ToLower(archiveMode)
	}

	// Normalize boolean string values
	boolFields := []string{"fsync", "ssl", "hot_standby", "autovacuum", "logging_collector", "full_page_writes", "jit", "row_security"}
	for _, field := range boolFields {
		if val, ok := configMap[field].(string); ok {
			lower := strings.ToLower(val)
			if lower == "on" || lower == "true" || lower == "yes" || lower == "1" {
				configMap[field] = true
			} else if lower == "off" || lower == "false" || lower == "no" || lower == "0" {
				configMap[field] = false
			}
		}
	}

	return configMap, nil
}

// ToNative converts the configuration to native PostgreSQL config format.
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
		if err := writePostgresDirective(&buf, key, value); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// writePostgresDirective writes a single PostgreSQL config directive.
func writePostgresDirective(buf *bytes.Buffer, key string, value any) error {
	switch v := value.(type) {
	case nil:
		// Skip nil values
		return nil
	case bool:
		if v {
			fmt.Fprintf(buf, "%s = on\n", key)
		} else {
			fmt.Fprintf(buf, "%s = off\n", key)
		}
	case int, int64:
		fmt.Fprintf(buf, "%s = %v\n", key, v)
	case float64:
		// Format floats without unnecessary decimals
		if v == float64(int(v)) {
			fmt.Fprintf(buf, "%s = %d\n", key, int(v))
		} else {
			fmt.Fprintf(buf, "%s = %v\n", key, v)
		}
	case string:
		if v == "" {
			fmt.Fprintf(buf, "%s = ''\n", key)
		} else if needsQuoting(v) {
			// Escape single quotes by doubling them
			escaped := strings.ReplaceAll(v, "'", "''")
			fmt.Fprintf(buf, "%s = '%s'\n", key, escaped)
		} else {
			fmt.Fprintf(buf, "%s = %s\n", key, v)
		}
	case []any:
		// PostgreSQL doesn't have native array syntax in postgresql.conf
		// These would typically be comma-separated values
		strVals := make([]string, len(v))
		for i, item := range v {
			strVals[i] = fmt.Sprintf("%v", item)
		}
		joined := strings.Join(strVals, ", ")
		fmt.Fprintf(buf, "%s = '%s'\n", key, joined)
	case []string:
		joined := strings.Join(v, ", ")
		fmt.Fprintf(buf, "%s = '%s'\n", key, joined)
	default:
		fmt.Fprintf(buf, "%s = %v\n", key, v)
	}
	return nil
}

// needsQuoting checks if a value needs to be quoted in PostgreSQL config.
func needsQuoting(s string) bool {
	// Quote if contains spaces, special chars, or looks like a path/identifier
	if strings.ContainsAny(s, " \t\n'\"#=,/\\") {
		return true
	}
	// Quote if it starts with a letter and contains non-alphanumeric chars
	if len(s) > 0 {
		// Memory sizes like 128MB, 1GB don't need quotes
		memPattern := regexp.MustCompile(`^\d+[kKmMgGtTpP]?[bB]?$`)
		if memPattern.MatchString(s) {
			return false
		}
		// Duration values like 5min, 200ms don't need quotes
		durationPattern := regexp.MustCompile(`^\d+(ms|s|min|h|d)$`)
		if durationPattern.MatchString(s) {
			return false
		}
	}
	// Quote paths, URIs, and complex strings
	if strings.Contains(s, "/") || strings.Contains(s, "%") {
		return true
	}
	return false
}

// FromNative parses native PostgreSQL configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	config := make(map[string]any)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key = value
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])

		// Remove inline comments
		if commentIdx := strings.Index(value, "#"); commentIdx > 0 {
			// Make sure we're not inside quotes
			if !isInsideQuotes(value, commentIdx) {
				value = strings.TrimSpace(value[:commentIdx])
			}
		}

		// Handle quoted values
		if (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) ||
			(strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) {
			value = value[1 : len(value)-1]
			// Unescape doubled quotes
			value = strings.ReplaceAll(value, "''", "'")
			value = strings.ReplaceAll(value, "\"\"", "\"")
		}

		// Convert value
		config[key] = convertPostgresValue(value)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	return config, nil
}

// isInsideQuotes checks if position is inside a quoted string.
func isInsideQuotes(s string, pos int) bool {
	inSingleQuote := false
	inDoubleQuote := false
	for i := 0; i < pos && i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}
	}
	return inSingleQuote || inDoubleQuote
}

// convertPostgresValue converts a PostgreSQL config value to appropriate Go type.
func convertPostgresValue(s string) any {
	// Check for boolean
	lower := strings.ToLower(s)
	if lower == "on" || lower == "true" || lower == "yes" {
		return true
	}
	if lower == "off" || lower == "false" || lower == "no" {
		return false
	}

	// Check for integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}

	// Check for float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		// Keep as float only if it has decimal part
		if f != float64(int64(f)) {
			return f
		}
		return int(f)
	}

	return s
}

// Merge merges two PostgreSQL configurations.
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
