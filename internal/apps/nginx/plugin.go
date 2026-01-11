// SPDX-License-Identifier: MIT

// Package nginx provides an Nginx configuration management plugin.
package nginx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Nginx.
type Plugin struct{}

// New creates a new Nginx plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "nginx"
}

// Version returns the supported Nginx version range.
func (p *Plugin) Version() string {
	return ">=1.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Nginx web server configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "nginx"
}

// Schema returns the configuration schema for Nginx.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "events",
				Required:    false,
				Multiple:    false,
				Description: "Event processing configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "worker_connections", Type: "int", Description: "Maximum connections per worker"},
					{Name: "use", Type: "string", ValidValues: []string{"epoll", "kqueue", "select", "poll"}, Description: "Event method"},
					{Name: "multi_accept", Type: "bool", Description: "Accept multiple connections"},
				},
			},
			{
				Name:        "http",
				Required:    false,
				Multiple:    false,
				Description: "HTTP server configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "include", Type: "string", Multiple: true, Description: "Include other config files"},
					{Name: "default_type", Type: "string", Description: "Default MIME type"},
					{Name: "sendfile", Type: "bool", Description: "Use sendfile"},
					{Name: "keepalive_timeout", Type: "duration", Description: "Keep-alive timeout"},
					{Name: "gzip", Type: "bool", Description: "Enable gzip compression"},
					{Name: "gzip_types", Type: "string", Multiple: true, Description: "MIME types to compress"},
				},
				Subsections: []plugin.SectionSchema{
					{
						Name:        "server",
						Required:    false,
						Multiple:    true,
						Description: "Virtual server configuration",
						Directives: []plugin.DirectiveSchema{
							{Name: "listen", Type: "string", Multiple: true, Required: true, Description: "Listen address/port"},
							{Name: "server_name", Type: "string", Multiple: true, Description: "Server names"},
							{Name: "root", Type: "string", Description: "Document root"},
							{Name: "index", Type: "string", Multiple: true, Description: "Index files"},
							{Name: "error_page", Type: "string", Multiple: true, Description: "Error page definitions"},
							{Name: "access_log", Type: "string", Description: "Access log path"},
							{Name: "error_log", Type: "string", Description: "Error log path"},
						},
						Subsections: []plugin.SectionSchema{
							{
								Name:        "location",
								Required:    false,
								Multiple:    true,
								Description: "Location block configuration",
								Directives: []plugin.DirectiveSchema{
									{Name: "root", Type: "string", Description: "Document root for location"},
									{Name: "alias", Type: "string", Description: "Alias path"},
									{Name: "index", Type: "string", Multiple: true, Description: "Index files"},
									{Name: "try_files", Type: "string", Description: "Try files directive"},
									{Name: "proxy_pass", Type: "string", Description: "Proxy upstream"},
									{Name: "proxy_set_header", Type: "string", Multiple: true, Description: "Proxy headers"},
									{Name: "fastcgi_pass", Type: "string", Description: "FastCGI upstream"},
									{Name: "return", Type: "string", Description: "Return directive"},
									{Name: "rewrite", Type: "string", Multiple: true, Description: "Rewrite rules"},
								},
							},
						},
					},
					{
						Name:        "upstream",
						Required:    false,
						Multiple:    true,
						Description: "Upstream server group",
						Directives: []plugin.DirectiveSchema{
							{Name: "server", Type: "string", Multiple: true, Required: true, Description: "Upstream servers"},
							{Name: "keepalive", Type: "int", Description: "Keepalive connections"},
							{Name: "least_conn", Type: "bool", Description: "Least connections algorithm"},
							{Name: "ip_hash", Type: "bool", Description: "IP hash algorithm"},
						},
					},
				},
			},
			{
				Name:        "stream",
				Required:    false,
				Multiple:    false,
				Description: "TCP/UDP stream configuration",
			},
			{
				Name:        "mail",
				Required:    false,
				Multiple:    false,
				Description: "Mail proxy configuration",
			},
		},
		Directives: []plugin.DirectiveSchema{
			{Name: "user", Type: "string", Description: "Worker process user"},
			{Name: "worker_processes", Type: "string", Description: "Number of worker processes (or 'auto')"},
			{Name: "error_log", Type: "string", Description: "Error log path and level"},
			{Name: "pid", Type: "string", Description: "PID file path"},
			{Name: "worker_rlimit_nofile", Type: "int", Description: "Worker file descriptor limit"},
			{Name: "include", Type: "string", Multiple: true, Description: "Include configuration files"},
		},
	}
}

// DefaultConfig returns sensible default Nginx configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"worker_processes": "auto",
		"error_log":        "/var/log/nginx/error.log",
		"pid":              "/run/nginx.pid",
		"events": map[string]any{
			"worker_connections": 1024,
		},
		"http": map[string]any{
			"include":           []string{"/etc/nginx/mime.types"},
			"default_type":      "application/octet-stream",
			"sendfile":          true,
			"keepalive_timeout": 65,
		},
	}
}

// Validate validates the Nginx configuration structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "", Message: "configuration must be a map"}}, nil
	}

	// Validate events block if present
	if events, ok := configMap["events"]; ok {
		if eventsMap, ok := events.(map[string]any); ok {
			if wc, ok := eventsMap["worker_connections"]; ok {
				if !isPositiveNumber(wc) {
					errors = append(errors, plugin.ValidationError{
						Path:    "events.worker_connections",
						Message: "worker_connections must be a positive number",
					})
				}
			}
		}
	}

	// Validate http block if present
	if http, ok := configMap["http"]; ok {
		httpErrors := p.validateHTTPBlock(http, "http")
		errors = append(errors, httpErrors...)
	}

	return errors, nil
}

// validateHTTPBlock validates the http block configuration.
func (p *Plugin) validateHTTPBlock(http any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	httpMap, ok := http.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "http must be a map"}}
	}

	// Validate servers
	if servers, ok := httpMap["server"]; ok {
		switch s := servers.(type) {
		case []any:
			for i, server := range s {
				serverPath := fmt.Sprintf("%s.server[%d]", path, i)
				serverErrors := p.validateServerBlock(server, serverPath)
				errors = append(errors, serverErrors...)
			}
		case map[string]any:
			serverErrors := p.validateServerBlock(s, path+".server")
			errors = append(errors, serverErrors...)
		}
	}

	return errors
}

// validateServerBlock validates a server block.
func (p *Plugin) validateServerBlock(server any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	serverMap, ok := server.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "server must be a map"}}
	}

	// Server should have a listen directive
	if _, ok := serverMap["listen"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: "server block should have a listen directive",
		})
	}

	// Validate locations
	if locations, ok := serverMap["location"]; ok {
		switch l := locations.(type) {
		case []any:
			for i, loc := range l {
				locPath := fmt.Sprintf("%s.location[%d]", path, i)
				locErrors := p.validateLocationBlock(loc, locPath)
				errors = append(errors, locErrors...)
			}
		case map[string]any:
			locErrors := p.validateLocationBlock(l, path+".location")
			errors = append(errors, locErrors...)
		}
	}

	return errors
}

// validateLocationBlock validates a location block.
func (p *Plugin) validateLocationBlock(location any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	locMap, ok := location.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "location must be a map"}}
	}

	// Location should have a path
	if _, ok := locMap["path"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: "location block should have a path",
		})
	}

	// Check for conflicting directives
	hasRoot := locMap["root"] != nil
	hasAlias := locMap["alias"] != nil
	if hasRoot && hasAlias {
		errors = append(errors, plugin.ValidationError{
			Path:    path,
			Message: "location cannot have both root and alias directives",
		})
	}

	return errors
}

// ValidateSemantic performs Nginx-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return errors, nil
	}

	// Check for common misconfigurations
	if http, ok := configMap["http"].(map[string]any); ok {
		// Check if gzip is enabled but gzip_types is not set
		if gzip, ok := http["gzip"].(bool); ok && gzip {
			if _, ok := http["gzip_types"]; !ok {
				errors = append(errors, plugin.ValidationError{
					Path:    "http.gzip_types",
					Message: "gzip is enabled but gzip_types is not set (recommended to set MIME types)",
				})
			}
		}
	}

	return errors, nil
}

// Normalize normalizes the configuration to a canonical form.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return config, nil
	}

	result := make(map[string]any)

	// Copy and normalize values
	for k, v := range configMap {
		switch k {
		case "worker_processes":
			// Normalize to string (can be "auto" or a number)
			result[k] = fmt.Sprintf("%v", v)
		case "keepalive_timeout":
			// Normalize duration
			result[k] = normalizeDuration(v)
		default:
			// Recursively normalize nested maps
			if nested, ok := v.(map[string]any); ok {
				normalized, _ := p.Normalize(nested)
				result[k] = normalized
			} else {
				result[k] = v
			}
		}
	}

	return result, nil
}

// ToNative converts the configuration to Nginx native format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	var buf bytes.Buffer
	if err := p.writeConfig(&buf, config, 0); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeConfig recursively writes configuration to a buffer.
func (p *Plugin) writeConfig(w io.Writer, config any, indent int) error {
	configMap, ok := config.(map[string]any)
	if !ok {
		return fmt.Errorf("config must be a map, got %T", config)
	}

	indentStr := strings.Repeat("    ", indent)

	// Sort keys for consistent output
	keys := make([]string, 0, len(configMap))
	for k := range configMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Separate directives from blocks
	var directives []string
	var blocks []string

	for _, k := range keys {
		v := configMap[k]
		if isBlock(k, v) {
			blocks = append(blocks, k)
		} else {
			directives = append(directives, k)
		}
	}

	// Write directives first
	for _, k := range directives {
		v := configMap[k]
		if err := p.writeDirective(w, k, v, indentStr); err != nil {
			return err
		}
	}

	// Add blank line between directives and blocks
	if len(directives) > 0 && len(blocks) > 0 {
		fmt.Fprintln(w)
	}

	// Write blocks
	for i, k := range blocks {
		v := configMap[k]
		if err := p.writeBlock(w, k, v, indent); err != nil {
			return err
		}
		if i < len(blocks)-1 {
			fmt.Fprintln(w)
		}
	}

	return nil
}

// writeDirective writes a single directive.
func (p *Plugin) writeDirective(w io.Writer, name string, value any, indent string) error {
	switch v := value.(type) {
	case []any:
		// Multiple values - write each on its own line
		for _, item := range v {
			fmt.Fprintf(w, "%s%s %v;\n", indent, name, formatValue(item))
		}
	case []string:
		for _, item := range v {
			fmt.Fprintf(w, "%s%s %s;\n", indent, name, item)
		}
	case bool:
		if v {
			fmt.Fprintf(w, "%s%s on;\n", indent, name)
		} else {
			fmt.Fprintf(w, "%s%s off;\n", indent, name)
		}
	default:
		fmt.Fprintf(w, "%s%s %v;\n", indent, name, formatValue(v))
	}
	return nil
}

// writeBlock writes a configuration block.
func (p *Plugin) writeBlock(w io.Writer, name string, value any, indent int) error {
	indentStr := strings.Repeat("    ", indent)

	switch v := value.(type) {
	case []any:
		// Multiple blocks with the same name
		for i, item := range v {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}

			// Handle location blocks specially (they have a path)
			if name == "location" {
				path := itemMap["path"]
				delete(itemMap, "path")
				modifier := itemMap["modifier"]
				delete(itemMap, "modifier")

				if modifier != nil {
					fmt.Fprintf(w, "%s%s %v %v {\n", indentStr, name, modifier, path)
				} else {
					fmt.Fprintf(w, "%s%s %v {\n", indentStr, name, path)
				}
			} else if name == "server" || name == "upstream" {
				// Named blocks
				blockName := itemMap["name"]
				delete(itemMap, "name")
				if blockName != nil {
					fmt.Fprintf(w, "%s%s %v {\n", indentStr, name, blockName)
				} else {
					fmt.Fprintf(w, "%s%s {\n", indentStr, name)
				}
			} else {
				fmt.Fprintf(w, "%s%s {\n", indentStr, name)
			}

			if err := p.writeConfig(w, itemMap, indent+1); err != nil {
				return err
			}
			fmt.Fprintf(w, "%s}\n", indentStr)

			if i < len(v)-1 {
				fmt.Fprintln(w)
			}
		}

	case map[string]any:
		// Single block
		blockName := v["name"]
		delete(v, "name")

		if name == "location" {
			path := v["path"]
			delete(v, "path")
			modifier := v["modifier"]
			delete(v, "modifier")

			if modifier != nil {
				fmt.Fprintf(w, "%s%s %v %v {\n", indentStr, name, modifier, path)
			} else {
				fmt.Fprintf(w, "%s%s %v {\n", indentStr, name, path)
			}
		} else if blockName != nil {
			fmt.Fprintf(w, "%s%s %v {\n", indentStr, name, blockName)
		} else {
			fmt.Fprintf(w, "%s%s {\n", indentStr, name)
		}

		if err := p.writeConfig(w, v, indent+1); err != nil {
			return err
		}
		fmt.Fprintf(w, "%s}\n", indentStr)
	}

	return nil
}

// FromNative parses Nginx native format into configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	parser := &nginxParser{
		scanner: bufio.NewScanner(bytes.NewReader(data)),
		line:    0,
	}
	return parser.parse()
}

// nginxParser parses Nginx configuration files.
type nginxParser struct {
	scanner *bufio.Scanner
	line    int
}

func (p *nginxParser) parse() (map[string]any, error) {
	result := make(map[string]any)

	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Tokenize the line
		tokens := tokenizeLine(line)
		if len(tokens) == 0 {
			continue
		}

		// Parse directive or block
		if err := p.parseDirectiveOrBlock(tokens, result); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.line, err)
		}
	}

	return result, p.scanner.Err()
}

func (p *nginxParser) parseDirectiveOrBlock(tokens []string, result map[string]any) error {
	if len(tokens) == 0 {
		return nil
	}

	name := tokens[0]

	// Check if it's a block (ends with {)
	lastToken := tokens[len(tokens)-1]
	if lastToken == "{" {
		// Parse block
		blockContent, err := p.parseBlock()
		if err != nil {
			return err
		}

		// Handle block arguments (like server_name, upstream name, location path)
		if len(tokens) > 2 {
			// Has arguments
			args := tokens[1 : len(tokens)-1]
			if name == "location" {
				// Location has special handling
				if len(args) == 1 {
					blockContent["path"] = args[0]
				} else if len(args) >= 2 {
					blockContent["modifier"] = args[0]
					blockContent["path"] = args[1]
				}
			} else if name == "upstream" {
				blockContent["name"] = args[0]
			}
		}

		// Add to result (handle multiple blocks with same name)
		if existing, ok := result[name]; ok {
			switch e := existing.(type) {
			case []any:
				result[name] = append(e, blockContent)
			default:
				result[name] = []any{e, blockContent}
			}
		} else {
			result[name] = blockContent
		}
	} else if lastToken == ";" {
		// Parse directive
		value := tokens[1 : len(tokens)-1]
		p.addDirective(result, name, value)
	} else {
		// Line doesn't end with ; or { - might span multiple lines
		// For simplicity, treat as directive
		value := tokens[1:]
		if len(value) > 0 && value[len(value)-1] == ";" {
			value = value[:len(value)-1]
		}
		p.addDirective(result, name, value)
	}

	return nil
}

func (p *nginxParser) parseBlock() (map[string]any, error) {
	result := make(map[string]any)
	braceCount := 1

	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for closing brace
		if line == "}" {
			braceCount--
			if braceCount == 0 {
				return result, nil
			}
			continue
		}

		// Tokenize and parse
		tokens := tokenizeLine(line)
		if len(tokens) == 0 {
			continue
		}

		// Track nested braces
		for _, t := range tokens {
			if t == "{" {
				braceCount++
			}
		}

		if err := p.parseDirectiveOrBlock(tokens, result); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("unexpected end of file in block")
}

func (p *nginxParser) addDirective(result map[string]any, name string, values []string) {
	var value any
	if len(values) == 0 {
		value = true
	} else if len(values) == 1 {
		value = parseDirectiveValue(values[0])
	} else {
		// Multiple values - join with space or keep as array
		value = strings.Join(values, " ")
	}

	// Handle multiple directives with same name
	if existing, ok := result[name]; ok {
		switch e := existing.(type) {
		case []any:
			result[name] = append(e, value)
		default:
			result[name] = []any{e, value}
		}
	} else {
		result[name] = value
	}
}

// Merge merges two Nginx configurations.
func (p *Plugin) Merge(base, overlay any) (any, error) {
	baseMap, baseOk := base.(map[string]any)
	overlayMap, overlayOk := overlay.(map[string]any)

	if !baseOk || !overlayOk {
		return overlay, nil
	}

	result := make(map[string]any)

	// Copy base
	for k, v := range baseMap {
		result[k] = deepCopy(v)
	}

	// Merge overlay
	for k, v := range overlayMap {
		if existing, ok := result[k]; ok {
			// Deep merge for blocks
			if isBlock(k, v) {
				merged, err := p.Merge(existing, v)
				if err != nil {
					return nil, err
				}
				result[k] = merged
			} else {
				result[k] = v
			}
		} else {
			result[k] = deepCopy(v)
		}
	}

	return result, nil
}

// Diff detects changes between two configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	return diffMaps(old, new, "")
}

// Helper functions

func isBlock(name string, value any) bool {
	blockNames := map[string]bool{
		"events": true, "http": true, "stream": true, "mail": true,
		"server": true, "location": true, "upstream": true,
		"if": true, "map": true, "geo": true, "limit_except": true,
		"types": true,
	}

	if blockNames[name] {
		return true
	}

	_, isMap := value.(map[string]any)
	_, isSlice := value.([]any)
	return isMap || isSlice
}

func isPositiveNumber(v any) bool {
	switch n := v.(type) {
	case int:
		return n > 0
	case int64:
		return n > 0
	case float64:
		return n > 0
	default:
		return false
	}
}

func normalizeDuration(v any) string {
	switch d := v.(type) {
	case int:
		return fmt.Sprintf("%d", d)
	case int64:
		return fmt.Sprintf("%d", d)
	case float64:
		return fmt.Sprintf("%.0f", d)
	case string:
		return d
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		// Quote if contains spaces or special characters
		if strings.ContainsAny(val, " \t\"'") {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(val, `"`, `\"`))
		}
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

var tokenRegex = regexp.MustCompile(`"[^"]*"|'[^']*'|[^\s]+`)

func tokenizeLine(line string) []string {
	matches := tokenRegex.FindAllString(line, -1)
	tokens := make([]string, 0, len(matches))
	for _, m := range matches {
		// Remove quotes from quoted strings
		if (strings.HasPrefix(m, `"`) && strings.HasSuffix(m, `"`)) ||
			(strings.HasPrefix(m, `'`) && strings.HasSuffix(m, `'`)) {
			m = m[1 : len(m)-1]
		}
		tokens = append(tokens, m)
	}
	return tokens
}

func parseDirectiveValue(s string) any {
	// Try to parse as number
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// Parse on/off as boolean
	if s == "on" {
		return true
	}
	if s == "off" {
		return false
	}
	return s
}

func deepCopy(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = deepCopy(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = deepCopy(v)
		}
		return result
	default:
		return v
	}
}

func diffMaps(old, new any, path string) ([]plugin.Change, error) {
	var changes []plugin.Change

	oldMap, oldOk := old.(map[string]any)
	newMap, newOk := new.(map[string]any)

	if !oldOk && !newOk {
		// Neither is a map - compare directly
		if old != new {
			changes = append(changes, plugin.Change{
				Type:     plugin.ChangeModified,
				Path:     path,
				OldValue: old,
				NewValue: new,
			})
		}
		return changes, nil
	}

	if !oldOk {
		// Old is not a map, new is - replacement
		changes = append(changes, plugin.Change{
			Type:     plugin.ChangeModified,
			Path:     path,
			OldValue: old,
			NewValue: new,
		})
		return changes, nil
	}

	if !newOk {
		// Old is a map, new is not - replacement
		changes = append(changes, plugin.Change{
			Type:     plugin.ChangeModified,
			Path:     path,
			OldValue: old,
			NewValue: new,
		})
		return changes, nil
	}

	// Both are maps - compare keys
	allKeys := make(map[string]bool)
	for k := range oldMap {
		allKeys[k] = true
	}
	for k := range newMap {
		allKeys[k] = true
	}

	for k := range allKeys {
		keyPath := k
		if path != "" {
			keyPath = path + "." + k
		}

		oldVal, oldHas := oldMap[k]
		newVal, newHas := newMap[k]

		if !oldHas {
			changes = append(changes, plugin.Change{
				Type:     plugin.ChangeAdded,
				Path:     keyPath,
				NewValue: newVal,
			})
		} else if !newHas {
			changes = append(changes, plugin.Change{
				Type:     plugin.ChangeRemoved,
				Path:     keyPath,
				OldValue: oldVal,
			})
		} else {
			// Both have the key - recurse
			subChanges, _ := diffMaps(oldVal, newVal, keyPath)
			changes = append(changes, subChanges...)
		}
	}

	return changes, nil
}
