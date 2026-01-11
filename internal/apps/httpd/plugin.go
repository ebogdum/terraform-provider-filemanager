// SPDX-License-Identifier: MIT

// Package httpd provides an Apache HTTP Server configuration management plugin.
package httpd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Apache HTTP Server.
type Plugin struct{}

// New creates a new Apache HTTP Server plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "httpd"
}

// Version returns the supported Apache HTTP Server version range.
func (p *Plugin) Version() string {
	return ">=2.4.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Apache HTTP Server configuration management"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "apache"
}

// Schema returns the configuration schema for Apache HTTP Server.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "server",
				Required:    false,
				Multiple:    false,
				Description: "Core server configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "ServerRoot", Type: "string", Description: "The top of the directory tree under which the server's configuration, error, and log files are kept"},
					{Name: "ServerName", Type: "string", Description: "Hostname and port that the server uses to identify itself"},
					{Name: "ServerAdmin", Type: "string", Description: "Email address that the server includes in error messages sent to the client"},
					{Name: "ServerTokens", Type: "string", ValidValues: []string{"Full", "OS", "Minimal", "Minor", "Major", "Prod"}, Description: "Configures the Server HTTP response header"},
					{Name: "ServerSignature", Type: "string", ValidValues: []string{"On", "Off", "EMail"}, Description: "Configures the footer on server-generated documents"},
					{Name: "Timeout", Type: "int", Description: "Amount of time the server will wait for certain events before failing a request"},
					{Name: "KeepAlive", Type: "string", ValidValues: []string{"On", "Off"}, Description: "Whether to allow persistent connections"},
					{Name: "MaxKeepAliveRequests", Type: "int", Description: "Number of requests allowed on a persistent connection"},
					{Name: "KeepAliveTimeout", Type: "int", Description: "Amount of time the server will wait for subsequent requests on a persistent connection"},
				},
			},
			{
				Name:        "listen",
				Required:    false,
				Multiple:    false,
				Description: "Listen configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "Listen", Type: "string", Multiple: true, Description: "IP addresses and ports that the server listens to"},
				},
			},
			{
				Name:        "modules",
				Required:    false,
				Multiple:    false,
				Description: "Module loading configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "LoadModule", Type: "string", Multiple: true, Description: "Links in the object file or library and adds the module to the list of active modules"},
				},
			},
			{
				Name:        "user",
				Required:    false,
				Multiple:    false,
				Description: "User and group configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "User", Type: "string", Description: "The userid under which the server will answer requests"},
					{Name: "Group", Type: "string", Description: "Group under which the server will answer requests"},
				},
			},
			{
				Name:        "logging",
				Required:    false,
				Multiple:    false,
				Description: "Logging configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "ErrorLog", Type: "string", Description: "Location where the server will log errors"},
					{Name: "LogLevel", Type: "string", ValidValues: []string{"emerg", "alert", "crit", "error", "warn", "notice", "info", "debug", "trace1", "trace2", "trace3", "trace4", "trace5", "trace6", "trace7", "trace8"}, Description: "Controls the verbosity of the ErrorLog"},
					{Name: "CustomLog", Type: "string", Multiple: true, Description: "Sets filename and format of log file"},
					{Name: "LogFormat", Type: "string", Multiple: true, Description: "Describes a format for use in a log file"},
				},
			},
			{
				Name:        "document",
				Required:    false,
				Multiple:    false,
				Description: "Document root configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "DocumentRoot", Type: "string", Description: "Directory that forms the main document tree visible from the web"},
					{Name: "DirectoryIndex", Type: "string", Multiple: true, Description: "List of resources to look for when the client requests a directory"},
				},
			},
			{
				Name:        "includes",
				Required:    false,
				Multiple:    false,
				Description: "Configuration file includes",
				Directives: []plugin.DirectiveSchema{
					{Name: "Include", Type: "string", Multiple: true, Description: "Includes other configuration files from within the server configuration files"},
					{Name: "IncludeOptional", Type: "string", Multiple: true, Description: "Includes other configuration files from within the server configuration files (does not fail if missing)"},
				},
			},
			{
				Name:        "mpm",
				Required:    false,
				Multiple:    false,
				Description: "Multi-Processing Module configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "StartServers", Type: "int", Description: "Number of child server processes created at startup"},
					{Name: "MinSpareServers", Type: "int", Description: "Minimum number of idle child server processes"},
					{Name: "MaxSpareServers", Type: "int", Description: "Maximum number of idle child server processes"},
					{Name: "MaxRequestWorkers", Type: "int", Description: "Maximum number of connections that will be processed simultaneously"},
					{Name: "MaxConnectionsPerChild", Type: "int", Description: "Limit on the number of connections that an individual child server will handle during its life"},
					{Name: "ThreadsPerChild", Type: "int", Description: "Number of threads created by each child process"},
				},
			},
			{
				Name:        "directory",
				Required:    false,
				Multiple:    true,
				Description: "Directory block configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "Path", Type: "string", Required: true, Description: "Directory path this block applies to"},
					{Name: "Options", Type: "string", Multiple: true, Description: "Configures what features are available in a particular directory"},
					{Name: "AllowOverride", Type: "string", Description: "Types of directives that are allowed in .htaccess files"},
					{Name: "Require", Type: "string", Multiple: true, Description: "Tests whether an authenticated user is authorized by an authorization provider"},
					{Name: "Order", Type: "string", Description: "Controls the default access state and the order in which Allow and Deny are evaluated", Deprecated: true, DeprecatedBy: "Require"},
					{Name: "Allow", Type: "string", Description: "Controls which hosts can access an area of the server", Deprecated: true, DeprecatedBy: "Require"},
					{Name: "Deny", Type: "string", Description: "Controls which hosts are denied access to the server", Deprecated: true, DeprecatedBy: "Require"},
				},
			},
			{
				Name:        "virtualhost",
				Required:    false,
				Multiple:    true,
				Description: "Virtual host configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "Address", Type: "string", Required: true, Description: "IP address and port for this virtual host"},
					{Name: "ServerName", Type: "string", Description: "Hostname for this virtual host"},
					{Name: "ServerAlias", Type: "string", Multiple: true, Description: "Alternate names for a host"},
					{Name: "DocumentRoot", Type: "string", Description: "Directory that forms the main document tree for this virtual host"},
					{Name: "ErrorLog", Type: "string", Description: "Location where the server will log errors for this virtual host"},
					{Name: "CustomLog", Type: "string", Description: "Sets filename and format of log file for this virtual host"},
					{Name: "SSLEngine", Type: "string", ValidValues: []string{"on", "off"}, Description: "SSL Engine Operation Switch"},
					{Name: "SSLCertificateFile", Type: "string", Description: "Server PEM-encoded X.509 certificate data file"},
					{Name: "SSLCertificateKeyFile", Type: "string", Description: "Server PEM-encoded private key file"},
					{Name: "SSLCertificateChainFile", Type: "string", Description: "File of PEM-encoded Server CA Certificates"},
				},
				Subsections: []plugin.SectionSchema{
					{
						Name:        "directory",
						Required:    false,
						Multiple:    true,
						Description: "Directory block within virtual host",
						Directives: []plugin.DirectiveSchema{
							{Name: "Path", Type: "string", Required: true, Description: "Directory path"},
							{Name: "Options", Type: "string", Multiple: true, Description: "Directory options"},
							{Name: "AllowOverride", Type: "string", Description: "AllowOverride setting"},
							{Name: "Require", Type: "string", Multiple: true, Description: "Access control"},
						},
					},
				},
			},
			{
				Name:        "security",
				Required:    false,
				Multiple:    false,
				Description: "Security-related configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "SSLProtocol", Type: "string", Description: "Configure usable SSL/TLS protocol versions"},
					{Name: "SSLCipherSuite", Type: "string", Description: "Cipher Suite available for negotiation in SSL handshake"},
					{Name: "SSLHonorCipherOrder", Type: "string", ValidValues: []string{"on", "off"}, Description: "Option to prefer the server's cipher preference order"},
					{Name: "Header", Type: "string", Multiple: true, Description: "Configure HTTP response headers"},
					{Name: "TraceEnable", Type: "string", ValidValues: []string{"on", "off", "extended"}, Description: "Determines the behavior on TRACE requests"},
				},
			},
			{
				Name:        "proxy",
				Required:    false,
				Multiple:    false,
				Description: "Proxy configuration",
				Directives: []plugin.DirectiveSchema{
					{Name: "ProxyRequests", Type: "string", ValidValues: []string{"On", "Off"}, Description: "Enables forward (standard) proxy requests"},
					{Name: "ProxyPass", Type: "string", Multiple: true, Description: "Maps remote servers into the local server URL-space"},
					{Name: "ProxyPassReverse", Type: "string", Multiple: true, Description: "Adjusts the URL in HTTP response headers sent from a reverse proxied server"},
					{Name: "ProxyPreserveHost", Type: "string", ValidValues: []string{"On", "Off"}, Description: "Use incoming Host HTTP request header for proxy request"},
				},
			},
		},
	}
}

// DefaultConfig returns sensible defaults for Apache HTTP Server.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"ServerRoot":             "/etc/httpd",
		"Listen":                 []string{"80"},
		"User":                   "apache",
		"Group":                  "apache",
		"ServerAdmin":            "root@localhost",
		"DocumentRoot":           "/var/www/html",
		"DirectoryIndex":         []string{"index.html", "index.htm"},
		"ErrorLog":               "logs/error_log",
		"LogLevel":               "warn",
		"LogFormat":              []string{`"%h %l %u %t \"%r\" %>s %b" common`, `"%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\"" combined`},
		"CustomLog":              []string{"logs/access_log combined"},
		"Timeout":                60,
		"KeepAlive":              "On",
		"MaxKeepAliveRequests":   100,
		"KeepAliveTimeout":       5,
		"ServerTokens":           "Prod",
		"ServerSignature":        "Off",
		"TraceEnable":            "off",
		"StartServers":           8,
		"MinSpareServers":        5,
		"MaxSpareServers":        20,
		"MaxRequestWorkers":      256,
		"MaxConnectionsPerChild": 4000,
	}
}

// Validate validates the Apache HTTP Server configuration.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Validate Listen ports
	if listens, ok := configMap["Listen"]; ok {
		var listenList []string
		switch v := listens.(type) {
		case []any:
			for _, l := range v {
				if s, ok := l.(string); ok {
					listenList = append(listenList, s)
				}
			}
		case []string:
			listenList = v
		case string:
			listenList = []string{v}
		}

		for i, listen := range listenList {
			if err := validateListenDirective(listen); err != nil {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("Listen[%d]", i),
					Message: err.Error(),
					Value:   listen,
				})
			}
		}
	}

	// Validate LogLevel
	if loglevel, ok := configMap["LogLevel"].(string); ok {
		validLevels := []string{"emerg", "alert", "crit", "error", "warn", "notice", "info", "debug",
			"trace1", "trace2", "trace3", "trace4", "trace5", "trace6", "trace7", "trace8"}
		valid := false
		// LogLevel can have module-specific levels like "warn ssl:info"
		baseLoglevel := strings.Fields(loglevel)[0]
		for _, v := range validLevels {
			if strings.EqualFold(baseLoglevel, v) {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, plugin.ValidationError{
				Path:    "LogLevel",
				Message: fmt.Sprintf("invalid LogLevel: %s (must be one of: %s)", loglevel, strings.Join(validLevels, ", ")),
				Value:   loglevel,
			})
		}
	}

	// Validate ServerTokens
	if tokens, ok := configMap["ServerTokens"].(string); ok {
		validTokens := []string{"Full", "OS", "Minimal", "Minor", "Major", "Prod"}
		valid := false
		for _, v := range validTokens {
			if strings.EqualFold(tokens, v) {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, plugin.ValidationError{
				Path:    "ServerTokens",
				Message: fmt.Sprintf("invalid ServerTokens: %s (must be one of: %s)", tokens, strings.Join(validTokens, ", ")),
				Value:   tokens,
			})
		}
	}

	// Validate Timeout
	if timeout, ok := configMap["Timeout"]; ok {
		var timeoutNum int
		switch v := timeout.(type) {
		case int:
			timeoutNum = v
		case int64:
			timeoutNum = int(v)
		case float64:
			timeoutNum = int(v)
		}
		if timeoutNum < 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "Timeout",
				Message: fmt.Sprintf("invalid Timeout: %d (must be >= 0)", timeoutNum),
				Value:   timeout,
			})
		}
	}

	// Validate KeepAlive
	if keepalive, ok := configMap["KeepAlive"].(string); ok {
		validOptions := []string{"On", "Off"}
		valid := false
		for _, v := range validOptions {
			if strings.EqualFold(keepalive, v) {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, plugin.ValidationError{
				Path:    "KeepAlive",
				Message: fmt.Sprintf("invalid KeepAlive: %s (must be On or Off)", keepalive),
				Value:   keepalive,
			})
		}
	}

	// Validate MaxKeepAliveRequests
	if maxReq, ok := configMap["MaxKeepAliveRequests"]; ok {
		var maxReqNum int
		switch v := maxReq.(type) {
		case int:
			maxReqNum = v
		case int64:
			maxReqNum = int(v)
		case float64:
			maxReqNum = int(v)
		}
		if maxReqNum < 0 {
			errors = append(errors, plugin.ValidationError{
				Path:    "MaxKeepAliveRequests",
				Message: fmt.Sprintf("invalid MaxKeepAliveRequests: %d (must be >= 0)", maxReqNum),
				Value:   maxReq,
			})
		}
	}

	// Validate MaxRequestWorkers
	if workers, ok := configMap["MaxRequestWorkers"]; ok {
		var workersNum int
		switch v := workers.(type) {
		case int:
			workersNum = v
		case int64:
			workersNum = int(v)
		case float64:
			workersNum = int(v)
		}
		if workersNum < 1 {
			errors = append(errors, plugin.ValidationError{
				Path:    "MaxRequestWorkers",
				Message: fmt.Sprintf("invalid MaxRequestWorkers: %d (must be >= 1)", workersNum),
				Value:   workers,
			})
		}
	}

	// Validate VirtualHosts
	if vhosts, ok := configMap["VirtualHost"].([]any); ok {
		for i, vhost := range vhosts {
			vhostMap, ok := vhost.(map[string]any)
			if !ok {
				continue
			}
			if addr, ok := vhostMap["Address"].(string); ok {
				if err := validateListenDirective(addr); err != nil {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("VirtualHost[%d].Address", i),
						Message: err.Error(),
						Value:   addr,
					})
				}
			}
			// Validate SSL settings
			if sslEngine, ok := vhostMap["SSLEngine"].(string); ok && strings.EqualFold(sslEngine, "on") {
				if _, hasCert := vhostMap["SSLCertificateFile"]; !hasCert {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("VirtualHost[%d].SSLCertificateFile", i),
						Message: "SSLEngine is on but SSLCertificateFile is not set",
					})
				}
				if _, hasKey := vhostMap["SSLCertificateKeyFile"]; !hasKey {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("VirtualHost[%d].SSLCertificateKeyFile", i),
						Message: "SSLEngine is on but SSLCertificateKeyFile is not set",
					})
				}
			}
		}
	}

	// Validate Directory blocks
	if dirs, ok := configMap["Directory"].([]any); ok {
		for i, dir := range dirs {
			dirMap, ok := dir.(map[string]any)
			if !ok {
				continue
			}
			if _, hasPath := dirMap["Path"]; !hasPath {
				errors = append(errors, plugin.ValidationError{
					Path:    fmt.Sprintf("Directory[%d].Path", i),
					Message: "Directory block requires a Path",
				})
			}
		}
	}

	return errors, nil
}

// validateListenDirective validates a Listen directive value.
func validateListenDirective(listen string) error {
	// Valid formats:
	// - "80" (port only)
	// - "8080" (port only)
	// - "192.168.1.1:80" (IP:port)
	// - "[::]:80" (IPv6)
	// - "*:80" (all interfaces)
	// - "0.0.0.0:80" (all IPv4 interfaces)

	listen = strings.TrimSpace(listen)

	// Extract port part
	var portStr string
	if strings.HasPrefix(listen, "[") {
		// IPv6 format [::]:port
		closeBracket := strings.LastIndex(listen, "]")
		if closeBracket == -1 {
			return fmt.Errorf("invalid IPv6 format: %s", listen)
		}
		if closeBracket+1 < len(listen) && listen[closeBracket+1] == ':' {
			portStr = listen[closeBracket+2:]
		} else {
			portStr = listen[closeBracket+1:]
		}
	} else if strings.Contains(listen, ":") {
		// IP:port or *:port format
		parts := strings.Split(listen, ":")
		portStr = parts[len(parts)-1]
	} else {
		// Port only
		portStr = listen
	}

	// Validate port number
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", portStr)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number out of range: %d (must be 1-65535)", port)
	}

	return nil
}

// ValidateSemantic performs Apache-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Check if DocumentRoot exists
	if docRoot, ok := configMap["DocumentRoot"].(string); ok && docRoot != "" {
		if _, err := os.Stat(docRoot); os.IsNotExist(err) {
			errors = append(errors, plugin.ValidationError{
				Path:    "DocumentRoot",
				Message: fmt.Sprintf("DocumentRoot directory does not exist: %s", docRoot),
				Value:   docRoot,
			})
		}
	}

	// Check if ServerRoot exists
	if serverRoot, ok := configMap["ServerRoot"].(string); ok && serverRoot != "" {
		if _, err := os.Stat(serverRoot); os.IsNotExist(err) {
			errors = append(errors, plugin.ValidationError{
				Path:    "ServerRoot",
				Message: fmt.Sprintf("ServerRoot directory does not exist: %s", serverRoot),
				Value:   serverRoot,
			})
		}
	}

	// Check LoadModule references
	if modules, ok := configMap["LoadModule"]; ok {
		var moduleList []string
		switch v := modules.(type) {
		case []any:
			for _, m := range v {
				if s, ok := m.(string); ok {
					moduleList = append(moduleList, s)
				}
			}
		case []string:
			moduleList = v
		case string:
			moduleList = []string{v}
		}

		serverRoot := "/etc/httpd"
		if sr, ok := configMap["ServerRoot"].(string); ok && sr != "" {
			serverRoot = sr
		}

		for i, module := range moduleList {
			// LoadModule format: "module_name path/to/module.so"
			parts := strings.Fields(module)
			if len(parts) >= 2 {
				modulePath := parts[1]
				// If relative path, prepend ServerRoot
				if !strings.HasPrefix(modulePath, "/") {
					modulePath = serverRoot + "/" + modulePath
				}
				if _, err := os.Stat(modulePath); os.IsNotExist(err) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("LoadModule[%d]", i),
						Message: fmt.Sprintf("module file does not exist: %s", modulePath),
						Value:   module,
					})
				}
			}
		}
	}

	// Check SSL certificate files for VirtualHosts
	if vhosts, ok := configMap["VirtualHost"].([]any); ok {
		for i, vhost := range vhosts {
			vhostMap, ok := vhost.(map[string]any)
			if !ok {
				continue
			}

			if sslEngine, ok := vhostMap["SSLEngine"].(string); ok && strings.EqualFold(sslEngine, "on") {
				// Check certificate file
				if certFile, ok := vhostMap["SSLCertificateFile"].(string); ok && certFile != "" {
					if _, err := os.Stat(certFile); os.IsNotExist(err) {
						errors = append(errors, plugin.ValidationError{
							Path:    fmt.Sprintf("VirtualHost[%d].SSLCertificateFile", i),
							Message: fmt.Sprintf("SSL certificate file does not exist: %s", certFile),
							Value:   certFile,
						})
					}
				}
				// Check key file
				if keyFile, ok := vhostMap["SSLCertificateKeyFile"].(string); ok && keyFile != "" {
					if _, err := os.Stat(keyFile); os.IsNotExist(err) {
						errors = append(errors, plugin.ValidationError{
							Path:    fmt.Sprintf("VirtualHost[%d].SSLCertificateKeyFile", i),
							Message: fmt.Sprintf("SSL key file does not exist: %s", keyFile),
							Value:   keyFile,
						})
					}
				}
				// Check chain file if specified
				if chainFile, ok := vhostMap["SSLCertificateChainFile"].(string); ok && chainFile != "" {
					if _, err := os.Stat(chainFile); os.IsNotExist(err) {
						errors = append(errors, plugin.ValidationError{
							Path:    fmt.Sprintf("VirtualHost[%d].SSLCertificateChainFile", i),
							Message: fmt.Sprintf("SSL chain file does not exist: %s", chainFile),
							Value:   chainFile,
						})
					}
				}
			}
		}
	}

	// Check Include/IncludeOptional paths
	if includes, ok := configMap["Include"]; ok {
		var includeList []string
		switch v := includes.(type) {
		case []any:
			for _, inc := range v {
				if s, ok := inc.(string); ok {
					includeList = append(includeList, s)
				}
			}
		case []string:
			includeList = v
		case string:
			includeList = []string{v}
		}

		serverRoot := "/etc/httpd"
		if sr, ok := configMap["ServerRoot"].(string); ok && sr != "" {
			serverRoot = sr
		}

		for i, include := range includeList {
			includePath := include
			// If relative path, prepend ServerRoot
			if !strings.HasPrefix(includePath, "/") {
				includePath = serverRoot + "/" + includePath
			}
			// Include directive must exist (unlike IncludeOptional)
			// Check if it's a glob pattern
			if !strings.ContainsAny(includePath, "*?[") {
				if _, err := os.Stat(includePath); os.IsNotExist(err) {
					errors = append(errors, plugin.ValidationError{
						Path:    fmt.Sprintf("Include[%d]", i),
						Message: fmt.Sprintf("included file does not exist: %s", includePath),
						Value:   include,
					})
				}
			}
		}
	}

	// Security warnings
	if tokens, ok := configMap["ServerTokens"].(string); ok {
		if strings.EqualFold(tokens, "Full") || strings.EqualFold(tokens, "OS") {
			errors = append(errors, plugin.ValidationError{
				Path:    "ServerTokens",
				Message: "ServerTokens set to reveal detailed server information - consider using 'Prod' for production",
				Value:   tokens,
			})
		}
	}

	if signature, ok := configMap["ServerSignature"].(string); ok {
		if strings.EqualFold(signature, "On") || strings.EqualFold(signature, "EMail") {
			errors = append(errors, plugin.ValidationError{
				Path:    "ServerSignature",
				Message: "ServerSignature is enabled - consider disabling for production",
				Value:   signature,
			})
		}
	}

	if trace, ok := configMap["TraceEnable"].(string); ok {
		if strings.EqualFold(trace, "on") || strings.EqualFold(trace, "extended") {
			errors = append(errors, plugin.ValidationError{
				Path:    "TraceEnable",
				Message: "TRACE method is enabled - this can be a security risk",
				Value:   trace,
			})
		}
	}

	return errors, nil
}

// Normalize normalizes the Apache HTTP Server configuration to canonical form.
func (p *Plugin) Normalize(config any) (any, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	// Normalize Listen to array
	if listen, ok := configMap["Listen"].(string); ok {
		configMap["Listen"] = []string{listen}
	}

	// Normalize LoadModule to array
	if module, ok := configMap["LoadModule"].(string); ok {
		configMap["LoadModule"] = []string{module}
	}

	// Normalize DirectoryIndex to array
	if dirIndex, ok := configMap["DirectoryIndex"].(string); ok {
		configMap["DirectoryIndex"] = []string{dirIndex}
	}

	// Normalize Include to array
	if include, ok := configMap["Include"].(string); ok {
		configMap["Include"] = []string{include}
	}

	// Normalize IncludeOptional to array
	if include, ok := configMap["IncludeOptional"].(string); ok {
		configMap["IncludeOptional"] = []string{include}
	}

	// Normalize boolean-like values
	boolDirectives := []string{"KeepAlive", "SSLEngine", "ProxyRequests", "ProxyPreserveHost", "SSLHonorCipherOrder"}
	for _, directive := range boolDirectives {
		if val, ok := configMap[directive].(string); ok {
			lower := strings.ToLower(val)
			if lower == "on" {
				configMap[directive] = "On"
			} else if lower == "off" {
				configMap[directive] = "Off"
			}
		}
	}

	return configMap, nil
}

// ToNative converts the configuration to native Apache config format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var buf bytes.Buffer

	// Define directive order for consistent output
	directiveOrder := []string{
		"ServerRoot", "ServerName", "ServerAdmin", "ServerTokens", "ServerSignature",
		"Timeout", "KeepAlive", "MaxKeepAliveRequests", "KeepAliveTimeout",
		"Listen", "User", "Group",
		"LoadModule",
		"ErrorLog", "LogLevel", "LogFormat", "CustomLog",
		"DocumentRoot", "DirectoryIndex",
		"StartServers", "MinSpareServers", "MaxSpareServers", "MaxRequestWorkers",
		"MaxConnectionsPerChild", "ThreadsPerChild",
		"SSLProtocol", "SSLCipherSuite", "SSLHonorCipherOrder",
		"TraceEnable", "Header",
		"ProxyRequests", "ProxyPass", "ProxyPassReverse", "ProxyPreserveHost",
		"Include", "IncludeOptional",
	}

	// Write directives in order
	writtenKeys := make(map[string]bool)
	for _, key := range directiveOrder {
		if value, exists := configMap[key]; exists {
			if err := writeApacheDirective(&buf, key, value); err != nil {
				return nil, err
			}
			writtenKeys[key] = true
		}
	}

	// Write remaining keys that weren't in the order list
	keys := make([]string, 0)
	for k := range configMap {
		if !writtenKeys[k] && k != "Directory" && k != "VirtualHost" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := writeApacheDirective(&buf, key, configMap[key]); err != nil {
			return nil, err
		}
	}

	// Write Directory blocks
	if dirs, ok := configMap["Directory"].([]any); ok {
		for _, dir := range dirs {
			dirMap, ok := dir.(map[string]any)
			if !ok {
				continue
			}
			if err := writeDirectoryBlock(&buf, dirMap); err != nil {
				return nil, err
			}
		}
	}

	// Write VirtualHost blocks
	if vhosts, ok := configMap["VirtualHost"].([]any); ok {
		for _, vhost := range vhosts {
			vhostMap, ok := vhost.(map[string]any)
			if !ok {
				continue
			}
			if err := writeVirtualHostBlock(&buf, vhostMap); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

// writeApacheDirective writes a single Apache config directive.
func writeApacheDirective(buf *bytes.Buffer, key string, value any) error {
	switch v := value.(type) {
	case nil:
		return nil
	case bool:
		if v {
			fmt.Fprintf(buf, "%s On\n", key)
		} else {
			fmt.Fprintf(buf, "%s Off\n", key)
		}
	case int, int64, float64:
		fmt.Fprintf(buf, "%s %v\n", key, v)
	case string:
		if v != "" {
			// Check if value needs quoting
			if strings.ContainsAny(v, " \t") && !strings.HasPrefix(v, "\"") {
				fmt.Fprintf(buf, "%s %s\n", key, v)
			} else {
				fmt.Fprintf(buf, "%s %s\n", key, v)
			}
		}
	case []any:
		for _, item := range v {
			if err := writeApacheDirective(buf, key, item); err != nil {
				return err
			}
		}
	case []string:
		for _, item := range v {
			if err := writeApacheDirective(buf, key, item); err != nil {
				return err
			}
		}
	default:
		fmt.Fprintf(buf, "%s %v\n", key, v)
	}
	return nil
}

// writeDirectoryBlock writes a Directory block.
func writeDirectoryBlock(buf *bytes.Buffer, dirMap map[string]any) error {
	path, ok := dirMap["Path"].(string)
	if !ok {
		return fmt.Errorf("directory block missing Path")
	}

	fmt.Fprintf(buf, "\n<Directory %s>\n", path)

	// Write directives in a sensible order
	directives := []string{"Options", "AllowOverride", "Require", "Order", "Allow", "Deny"}
	for _, directive := range directives {
		if value, exists := dirMap[directive]; exists {
			switch v := value.(type) {
			case []any:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %v\n", directive, item)
				}
			case []string:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %s\n", directive, item)
				}
			case string:
				fmt.Fprintf(buf, "    %s %s\n", directive, v)
			}
		}
	}

	// Write any other directives
	for key, value := range dirMap {
		if key == "Path" {
			continue
		}
		found := false
		for _, d := range directives {
			if key == d {
				found = true
				break
			}
		}
		if !found {
			switch v := value.(type) {
			case string:
				fmt.Fprintf(buf, "    %s %s\n", key, v)
			case []any:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %v\n", key, item)
				}
			case []string:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %s\n", key, item)
				}
			}
		}
	}

	fmt.Fprintf(buf, "</Directory>\n")
	return nil
}

// writeVirtualHostBlock writes a VirtualHost block.
func writeVirtualHostBlock(buf *bytes.Buffer, vhostMap map[string]any) error {
	address, ok := vhostMap["Address"].(string)
	if !ok {
		return fmt.Errorf("VirtualHost block missing Address")
	}

	fmt.Fprintf(buf, "\n<VirtualHost %s>\n", address)

	// Write directives in a sensible order
	directives := []string{
		"ServerName", "ServerAlias", "DocumentRoot",
		"ErrorLog", "CustomLog",
		"SSLEngine", "SSLCertificateFile", "SSLCertificateKeyFile", "SSLCertificateChainFile",
	}

	for _, directive := range directives {
		if value, exists := vhostMap[directive]; exists {
			switch v := value.(type) {
			case []any:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %v\n", directive, item)
				}
			case []string:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %s\n", directive, item)
				}
			case string:
				if v != "" {
					fmt.Fprintf(buf, "    %s %s\n", directive, v)
				}
			}
		}
	}

	// Write any other directives (except Address and Directory)
	for key, value := range vhostMap {
		if key == "Address" || key == "Directory" {
			continue
		}
		found := false
		for _, d := range directives {
			if key == d {
				found = true
				break
			}
		}
		if !found {
			switch v := value.(type) {
			case string:
				if v != "" {
					fmt.Fprintf(buf, "    %s %s\n", key, v)
				}
			case []any:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %v\n", key, item)
				}
			case []string:
				for _, item := range v {
					fmt.Fprintf(buf, "    %s %s\n", key, item)
				}
			}
		}
	}

	// Write nested Directory blocks
	if dirs, ok := vhostMap["Directory"].([]any); ok {
		for _, dir := range dirs {
			dirMap, ok := dir.(map[string]any)
			if !ok {
				continue
			}
			path, ok := dirMap["Path"].(string)
			if !ok {
				continue
			}
			fmt.Fprintf(buf, "\n    <Directory %s>\n", path)
			for key, value := range dirMap {
				if key == "Path" {
					continue
				}
				switch v := value.(type) {
				case string:
					fmt.Fprintf(buf, "        %s %s\n", key, v)
				case []any:
					for _, item := range v {
						fmt.Fprintf(buf, "        %s %v\n", key, item)
					}
				case []string:
					for _, item := range v {
						fmt.Fprintf(buf, "        %s %s\n", key, item)
					}
				}
			}
			fmt.Fprintf(buf, "    </Directory>\n")
		}
	}

	fmt.Fprintf(buf, "</VirtualHost>\n")
	return nil
}

// FromNative parses native Apache configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	config := make(map[string]any)
	multiValueDirectives := map[string]bool{
		"Listen":           true,
		"LoadModule":       true,
		"Include":          true,
		"IncludeOptional":  true,
		"LogFormat":        true,
		"CustomLog":        true,
		"DirectoryIndex":   true,
		"ServerAlias":      true,
		"Header":           true,
		"ProxyPass":        true,
		"ProxyPassReverse": true,
	}

	var directories []map[string]any
	var virtualHosts []map[string]any

	scanner := bufio.NewScanner(bytes.NewReader(data))
	var currentBlock string
	var currentBlockContent map[string]any
	var blockStack []map[string]any
	var blockTypeStack []string
	indentLevel := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for block start
		if strings.HasPrefix(line, "<") && !strings.HasPrefix(line, "</") {
			blockMatch := regexp.MustCompile(`<(\w+)\s+(.*)>`).FindStringSubmatch(line)
			if len(blockMatch) >= 3 {
				blockType := blockMatch[1]
				blockArg := strings.TrimSuffix(blockMatch[2], ">")

				newBlock := make(map[string]any)

				if blockType == "Directory" {
					newBlock["Path"] = blockArg
				} else if blockType == "VirtualHost" {
					newBlock["Address"] = blockArg
				}

				if currentBlockContent != nil {
					blockStack = append(blockStack, currentBlockContent)
					blockTypeStack = append(blockTypeStack, currentBlock)
				}

				currentBlock = blockType
				currentBlockContent = newBlock
				indentLevel++
				continue
			}
		}

		// Check for block end
		if strings.HasPrefix(line, "</") {
			if currentBlockContent != nil {
				if currentBlock == "Directory" {
					if len(blockStack) > 0 {
						// Nested directory in VirtualHost
						parentBlock := blockStack[len(blockStack)-1]
						if existingDirs, ok := parentBlock["Directory"].([]any); ok {
							parentBlock["Directory"] = append(existingDirs, currentBlockContent)
						} else {
							parentBlock["Directory"] = []any{currentBlockContent}
						}
					} else {
						directories = append(directories, currentBlockContent)
					}
				} else if currentBlock == "VirtualHost" {
					virtualHosts = append(virtualHosts, currentBlockContent)
				}

				if len(blockStack) > 0 {
					currentBlockContent = blockStack[len(blockStack)-1]
					currentBlock = blockTypeStack[len(blockTypeStack)-1]
					blockStack = blockStack[:len(blockStack)-1]
					blockTypeStack = blockTypeStack[:len(blockTypeStack)-1]
				} else {
					currentBlockContent = nil
					currentBlock = ""
				}
				indentLevel--
			}
			continue
		}

		// Parse directive
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}

		key := parts[0]
		var value string
		if len(parts) >= 2 {
			value = strings.TrimSpace(parts[1])
		}

		// Handle quoted values
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}

		// Convert value
		converted := convertApacheValue(value)

		// Determine target map
		targetMap := config
		if currentBlockContent != nil {
			targetMap = currentBlockContent
		}

		// Handle multi-value directives
		if multiValueDirectives[key] {
			if existing, ok := targetMap[key].([]any); ok {
				targetMap[key] = append(existing, converted)
			} else if existing, ok := targetMap[key]; ok {
				targetMap[key] = []any{existing, converted}
			} else {
				targetMap[key] = []any{converted}
			}
		} else {
			targetMap[key] = converted
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	// Add directory and virtualhost blocks
	if len(directories) > 0 {
		dirInterfaces := make([]any, len(directories))
		for i, d := range directories {
			dirInterfaces[i] = d
		}
		config["Directory"] = dirInterfaces
	}
	if len(virtualHosts) > 0 {
		vhostInterfaces := make([]any, len(virtualHosts))
		for i, vh := range virtualHosts {
			vhostInterfaces[i] = vh
		}
		config["VirtualHost"] = vhostInterfaces
	}

	return config, nil
}

// convertApacheValue converts an Apache config value to appropriate Go type.
func convertApacheValue(s string) any {
	// Check for boolean
	lower := strings.ToLower(s)
	if lower == "on" || lower == "true" {
		return "On"
	}
	if lower == "off" || lower == "false" {
		return "Off"
	}

	// Check for integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}

	return s
}

// Merge merges two Apache configurations.
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
		// For array values, append rather than replace
		if baseVal, exists := result[k]; exists {
			if baseArr, ok := baseVal.([]any); ok {
				if overlayArr, ok := v.([]any); ok {
					result[k] = append(baseArr, overlayArr...)
					continue
				}
			}
		}
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
