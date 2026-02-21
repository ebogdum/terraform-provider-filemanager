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
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var errors []plugin.ValidationError
	errors = append(errors, validateHTTPDListen(configMap)...)
	errors = append(errors, validateHTTPDLogLevel(configMap)...)
	errors = append(errors, validateHTTPDServerTokens(configMap)...)
	errors = append(errors, validateHTTPDTimeout(configMap)...)
	errors = append(errors, validateHTTPDKeepAlive(configMap)...)
	errors = append(errors, validateHTTPDMaxKeepAliveRequests(configMap)...)
	errors = append(errors, validateHTTPDMaxRequestWorkers(configMap)...)
	errors = append(errors, validateHTTPDVirtualHosts(configMap)...)
	errors = append(errors, validateHTTPDDirectoryBlocks(configMap)...)
	return errors, nil
}

func validateHTTPDListen(configMap map[string]any) []plugin.ValidationError {
	listens, exists := configMap["Listen"]
	if !exists {
		return nil
	}

	listenList := httpdStringList(listens)
	var errors []plugin.ValidationError
	for i, listen := range listenList {
		if err := validateListenDirective(listen); err != nil {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("Listen[%d]", i),
				Message: err.Error(),
				Value:   listen,
			})
		}
	}
	return errors
}

func validateHTTPDLogLevel(configMap map[string]any) []plugin.ValidationError {
	loglevel, ok := configMap["LogLevel"].(string)
	if !ok {
		return nil
	}

	validLevels := []string{
		"emerg", "alert", "crit", "error", "warn", "notice", "info", "debug",
		"trace1", "trace2", "trace3", "trace4", "trace5", "trace6", "trace7", "trace8",
	}
	baseLoglevel := strings.Fields(loglevel)
	if len(baseLoglevel) == 0 {
		return nil
	}
	if httpdContainsFold(validLevels, baseLoglevel[0]) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "LogLevel",
		Message: fmt.Sprintf("invalid LogLevel: %s (must be one of: %s)", loglevel, strings.Join(validLevels, ", ")),
		Value:   loglevel,
	}}
}

func validateHTTPDServerTokens(configMap map[string]any) []plugin.ValidationError {
	tokens, ok := configMap["ServerTokens"].(string)
	if !ok {
		return nil
	}

	validTokens := []string{"Full", "OS", "Minimal", "Minor", "Major", "Prod"}
	if httpdContainsFold(validTokens, tokens) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "ServerTokens",
		Message: fmt.Sprintf("invalid ServerTokens: %s (must be one of: %s)", tokens, strings.Join(validTokens, ", ")),
		Value:   tokens,
	}}
}

func validateHTTPDTimeout(configMap map[string]any) []plugin.ValidationError {
	timeoutNum, ok := httpdAnyToInt(configMap["Timeout"])
	if !ok || timeoutNum >= 0 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "Timeout",
		Message: fmt.Sprintf("invalid Timeout: %d (must be >= 0)", timeoutNum),
		Value:   configMap["Timeout"],
	}}
}

func validateHTTPDKeepAlive(configMap map[string]any) []plugin.ValidationError {
	keepalive, ok := configMap["KeepAlive"].(string)
	if !ok {
		return nil
	}

	validOptions := []string{"On", "Off"}
	if httpdContainsFold(validOptions, keepalive) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "KeepAlive",
		Message: fmt.Sprintf("invalid KeepAlive: %s (must be On or Off)", keepalive),
		Value:   keepalive,
	}}
}

func validateHTTPDMaxKeepAliveRequests(configMap map[string]any) []plugin.ValidationError {
	maxReqNum, ok := httpdAnyToInt(configMap["MaxKeepAliveRequests"])
	if !ok || maxReqNum >= 0 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "MaxKeepAliveRequests",
		Message: fmt.Sprintf("invalid MaxKeepAliveRequests: %d (must be >= 0)", maxReqNum),
		Value:   configMap["MaxKeepAliveRequests"],
	}}
}

func validateHTTPDMaxRequestWorkers(configMap map[string]any) []plugin.ValidationError {
	workersNum, ok := httpdAnyToInt(configMap["MaxRequestWorkers"])
	if !ok || workersNum >= 1 {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "MaxRequestWorkers",
		Message: fmt.Sprintf("invalid MaxRequestWorkers: %d (must be >= 1)", workersNum),
		Value:   configMap["MaxRequestWorkers"],
	}}
}

func validateHTTPDVirtualHosts(configMap map[string]any) []plugin.ValidationError {
	vhosts, ok := configMap["VirtualHost"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for i, vhost := range vhosts {
		vhostMap, ok := vhost.(map[string]any)
		if !ok {
			continue
		}
		errors = append(errors, validateHTTPDVirtualHostAddress(vhostMap, i)...)
		errors = append(errors, validateHTTPDVirtualHostSSL(vhostMap, i)...)
	}
	return errors
}

func validateHTTPDVirtualHostAddress(vhostMap map[string]any, index int) []plugin.ValidationError {
	addr, ok := vhostMap["Address"].(string)
	if !ok {
		return nil
	}

	if err := validateListenDirective(addr); err != nil {
		return []plugin.ValidationError{{
			Path:    fmt.Sprintf("VirtualHost[%d].Address", index),
			Message: err.Error(),
			Value:   addr,
		}}
	}
	return nil
}

func validateHTTPDVirtualHostSSL(vhostMap map[string]any, index int) []plugin.ValidationError {
	sslEngine, ok := vhostMap["SSLEngine"].(string)
	if !ok || !strings.EqualFold(sslEngine, "on") {
		return nil
	}

	var errors []plugin.ValidationError
	if _, hasCert := vhostMap["SSLCertificateFile"]; !hasCert {
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("VirtualHost[%d].SSLCertificateFile", index),
			Message: "SSLEngine is on but SSLCertificateFile is not set",
		})
	}
	if _, hasKey := vhostMap["SSLCertificateKeyFile"]; !hasKey {
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("VirtualHost[%d].SSLCertificateKeyFile", index),
			Message: "SSLEngine is on but SSLCertificateKeyFile is not set",
		})
	}
	return errors
}

func validateHTTPDDirectoryBlocks(configMap map[string]any) []plugin.ValidationError {
	dirs, ok := configMap["Directory"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for i, dir := range dirs {
		dirMap, ok := dir.(map[string]any)
		if !ok {
			continue
		}
		if _, hasPath := dirMap["Path"]; hasPath {
			continue
		}
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("Directory[%d].Path", i),
			Message: "Directory block requires a Path",
		})
	}
	return errors
}

func httpdStringList(v any) []string {
	switch values := v.(type) {
	case []any:
		result := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return values
	case string:
		return []string{values}
	default:
		return nil
	}
}

func httpdAnyToInt(v any) (int, bool) {
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

func httpdContainsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
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
	configMap, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config must be a map")
	}

	var errors []plugin.ValidationError
	errors = append(errors, validateHTTPDPathExists(configMap, "DocumentRoot", "DocumentRoot directory does not exist: %s")...)
	errors = append(errors, validateHTTPDPathExists(configMap, "ServerRoot", "ServerRoot directory does not exist: %s")...)
	errors = append(errors, validateHTTPDLoadModules(configMap)...)
	errors = append(errors, validateHTTPDVirtualHostCertificates(configMap)...)
	errors = append(errors, validateHTTPDIncludePaths(configMap)...)
	errors = append(errors, validateHTTPDSecurityWarnings(configMap)...)
	return errors, nil
}

func validateHTTPDPathExists(configMap map[string]any, field string, messageFmt string) []plugin.ValidationError {
	path, ok := configMap[field].(string)
	if !ok || path == "" {
		return nil
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    field,
		Message: fmt.Sprintf(messageFmt, path),
		Value:   path,
	}}
}

func validateHTTPDLoadModules(configMap map[string]any) []plugin.ValidationError {
	moduleList := httpdStringList(configMap["LoadModule"])
	if len(moduleList) == 0 {
		return nil
	}

	serverRoot := httpdServerRoot(configMap)
	var errors []plugin.ValidationError
	for i, module := range moduleList {
		modulePath, ok := resolveLoadModulePath(module, serverRoot)
		if !ok {
			continue
		}

		if _, err := os.Stat(modulePath); !os.IsNotExist(err) {
			continue
		}

		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("LoadModule[%d]", i),
			Message: fmt.Sprintf("module file does not exist: %s", modulePath),
			Value:   module,
		})
	}
	return errors
}

func validateHTTPDVirtualHostCertificates(configMap map[string]any) []plugin.ValidationError {
	vhosts, ok := configMap["VirtualHost"].([]any)
	if !ok {
		return nil
	}

	var errors []plugin.ValidationError
	for i, vhost := range vhosts {
		vhostMap, ok := vhost.(map[string]any)
		if !ok {
			continue
		}
		errors = append(errors, validateHTTPDVirtualHostCertFiles(vhostMap, i)...)
	}
	return errors
}

func validateHTTPDVirtualHostCertFiles(vhostMap map[string]any, index int) []plugin.ValidationError {
	sslEngine, ok := vhostMap["SSLEngine"].(string)
	if !ok || !strings.EqualFold(sslEngine, "on") {
		return nil
	}

	var errors []plugin.ValidationError
	errors = append(errors, validateHTTPDFileExists(vhostMap, "SSLCertificateFile", fmt.Sprintf("VirtualHost[%d].SSLCertificateFile", index), "SSL certificate file does not exist: %s")...)
	errors = append(errors, validateHTTPDFileExists(vhostMap, "SSLCertificateKeyFile", fmt.Sprintf("VirtualHost[%d].SSLCertificateKeyFile", index), "SSL key file does not exist: %s")...)
	errors = append(errors, validateHTTPDFileExists(vhostMap, "SSLCertificateChainFile", fmt.Sprintf("VirtualHost[%d].SSLCertificateChainFile", index), "SSL chain file does not exist: %s")...)
	return errors
}

func validateHTTPDIncludePaths(configMap map[string]any) []plugin.ValidationError {
	includeList := httpdStringList(configMap["Include"])
	if len(includeList) == 0 {
		return nil
	}

	serverRoot := httpdServerRoot(configMap)
	var errors []plugin.ValidationError
	for i, include := range includeList {
		includePath := include
		if !strings.HasPrefix(includePath, "/") {
			includePath = serverRoot + "/" + includePath
		}
		if strings.ContainsAny(includePath, "*?[") {
			continue
		}
		if _, err := os.Stat(includePath); !os.IsNotExist(err) {
			continue
		}
		errors = append(errors, plugin.ValidationError{
			Path:    fmt.Sprintf("Include[%d]", i),
			Message: fmt.Sprintf("included file does not exist: %s", includePath),
			Value:   include,
		})
	}
	return errors
}

func validateHTTPDSecurityWarnings(configMap map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	errors = append(errors, validateHTTPDServerTokensWarning(configMap)...)
	errors = append(errors, validateHTTPDServerSignatureWarning(configMap)...)
	errors = append(errors, validateHTTPDTraceEnableWarning(configMap)...)
	return errors
}

func validateHTTPDServerTokensWarning(configMap map[string]any) []plugin.ValidationError {
	tokens, ok := configMap["ServerTokens"].(string)
	if !ok {
		return nil
	}
	if !strings.EqualFold(tokens, "Full") && !strings.EqualFold(tokens, "OS") {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "ServerTokens",
		Message: "ServerTokens set to reveal detailed server information - consider using 'Prod' for production",
		Value:   tokens,
	}}
}

func validateHTTPDServerSignatureWarning(configMap map[string]any) []plugin.ValidationError {
	signature, ok := configMap["ServerSignature"].(string)
	if !ok {
		return nil
	}
	if !strings.EqualFold(signature, "On") && !strings.EqualFold(signature, "EMail") {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "ServerSignature",
		Message: "ServerSignature is enabled - consider disabling for production",
		Value:   signature,
	}}
}

func validateHTTPDTraceEnableWarning(configMap map[string]any) []plugin.ValidationError {
	trace, ok := configMap["TraceEnable"].(string)
	if !ok {
		return nil
	}
	if !strings.EqualFold(trace, "on") && !strings.EqualFold(trace, "extended") {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    "TraceEnable",
		Message: "TRACE method is enabled - this can be a security risk",
		Value:   trace,
	}}
}

func httpdServerRoot(configMap map[string]any) string {
	serverRoot, ok := configMap["ServerRoot"].(string)
	if ok && serverRoot != "" {
		return serverRoot
	}
	return "/etc/httpd"
}

func resolveLoadModulePath(module string, serverRoot string) (string, bool) {
	parts := strings.Fields(module)
	if len(parts) < 2 {
		return "", false
	}

	modulePath := parts[1]
	if strings.HasPrefix(modulePath, "/") {
		return modulePath, true
	}

	return serverRoot + "/" + modulePath, true
}

func validateHTTPDFileExists(values map[string]any, field, path, messageFmt string) []plugin.ValidationError {
	filePath, ok := values[field].(string)
	if !ok || filePath == "" {
		return nil
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil
	}

	return []plugin.ValidationError{{
		Path:    path,
		Message: fmt.Sprintf(messageFmt, filePath),
		Value:   filePath,
	}}
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
	writtenKeys := make(map[string]bool)
	if err := writeOrderedApacheDirectives(&buf, configMap, writtenKeys); err != nil {
		return nil, err
	}
	if err := writeRemainingApacheDirectives(&buf, configMap, writtenKeys); err != nil {
		return nil, err
	}
	if err := writeApacheDirectoryBlocks(&buf, configMap); err != nil {
		return nil, err
	}
	if err := writeApacheVirtualHostBlocks(&buf, configMap); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeOrderedApacheDirectives(buf *bytes.Buffer, configMap map[string]any, writtenKeys map[string]bool) error {
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

	for _, key := range directiveOrder {
		value, exists := configMap[key]
		if !exists {
			continue
		}
		if err := writeApacheDirective(buf, key, value); err != nil {
			return err
		}
		writtenKeys[key] = true
	}
	return nil
}

func writeRemainingApacheDirectives(buf *bytes.Buffer, configMap map[string]any, writtenKeys map[string]bool) error {
	keys := make([]string, 0)
	for key := range configMap {
		if writtenKeys[key] || key == "Directory" || key == "VirtualHost" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if err := writeApacheDirective(buf, key, configMap[key]); err != nil {
			return err
		}
	}
	return nil
}

func writeApacheDirectoryBlocks(buf *bytes.Buffer, configMap map[string]any) error {
	dirs, ok := configMap["Directory"].([]any)
	if !ok {
		return nil
	}

	for _, dir := range dirs {
		dirMap, ok := dir.(map[string]any)
		if !ok {
			continue
		}
		if err := writeDirectoryBlock(buf, dirMap); err != nil {
			return err
		}
	}
	return nil
}

func writeApacheVirtualHostBlocks(buf *bytes.Buffer, configMap map[string]any) error {
	vhosts, ok := configMap["VirtualHost"].([]any)
	if !ok {
		return nil
	}

	for _, vhost := range vhosts {
		vhostMap, ok := vhost.(map[string]any)
		if !ok {
			continue
		}
		if err := writeVirtualHostBlock(buf, vhostMap); err != nil {
			return err
		}
	}
	return nil
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

	directives := []string{"Options", "AllowOverride", "Require", "Order", "Allow", "Deny"}
	writeOrderedBlockDirectives(buf, dirMap, directives, "    ")
	writeRemainingBlockDirectives(buf, dirMap, directives, "Path", "    ")

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

	directives := []string{
		"ServerName", "ServerAlias", "DocumentRoot",
		"ErrorLog", "CustomLog",
		"SSLEngine", "SSLCertificateFile", "SSLCertificateKeyFile", "SSLCertificateChainFile",
	}
	writeOrderedBlockDirectives(buf, vhostMap, directives, "    ")
	writeRemainingBlockDirectives(buf, vhostMap, directives, "Address", "    ")
	writeNestedVirtualHostDirectories(buf, vhostMap)

	fmt.Fprintf(buf, "</VirtualHost>\n")
	return nil
}

func writeOrderedBlockDirectives(buf *bytes.Buffer, block map[string]any, directives []string, indent string) {
	for _, directive := range directives {
		value, exists := block[directive]
		if !exists {
			continue
		}
		writeApacheBlockDirective(buf, indent, directive, value, true)
	}
}

func writeRemainingBlockDirectives(buf *bytes.Buffer, block map[string]any, directives []string, skipKey string, indent string) {
	for key, value := range block {
		if key == skipKey || key == "Directory" || containsDirective(directives, key) {
			continue
		}
		writeApacheBlockDirective(buf, indent, key, value, false)
	}
}

func writeApacheBlockDirective(buf *bytes.Buffer, indent, key string, value any, skipEmptyString bool) {
	switch typed := value.(type) {
	case string:
		if skipEmptyString && typed == "" {
			return
		}
		if typed != "" {
			fmt.Fprintf(buf, "%s%s %s\n", indent, key, typed)
		}
	case []any:
		for _, item := range typed {
			fmt.Fprintf(buf, "%s%s %v\n", indent, key, item)
		}
	case []string:
		for _, item := range typed {
			fmt.Fprintf(buf, "%s%s %s\n", indent, key, item)
		}
	}
}

func containsDirective(directives []string, key string) bool {
	for _, directive := range directives {
		if directive == key {
			return true
		}
	}
	return false
}

func writeNestedVirtualHostDirectories(buf *bytes.Buffer, vhostMap map[string]any) {
	dirs, ok := vhostMap["Directory"].([]any)
	if !ok {
		return
	}

	for _, dir := range dirs {
		writeNestedVirtualHostDirectory(buf, dir)
	}
}

func writeNestedVirtualHostDirectory(buf *bytes.Buffer, dir any) {
	dirMap, ok := dir.(map[string]any)
	if !ok {
		return
	}

	path, ok := dirMap["Path"].(string)
	if !ok {
		return
	}

	fmt.Fprintf(buf, "\n    <Directory %s>\n", path)
	for key, value := range dirMap {
		if key == "Path" {
			continue
		}
		writeApacheBlockDirective(buf, "        ", key, value, false)
	}
	fmt.Fprintf(buf, "    </Directory>\n")
}

// FromNative parses native Apache configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	config := make(map[string]any)
	multiValueDirectives := apacheMultiValueDirectives()

	var directories []map[string]any
	var virtualHosts []map[string]any

	scanner := bufio.NewScanner(bytes.NewReader(data))
	var currentBlock string
	var currentBlockContent map[string]any
	var blockStack []map[string]any
	var blockTypeStack []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if shouldSkipApacheLine(line) {
			continue
		}

		if blockType, blockArg, ok := parseApacheBlockStart(line); ok {
			currentBlock, currentBlockContent, blockStack, blockTypeStack = pushApacheBlock(blockType, blockArg, currentBlock, currentBlockContent, blockStack, blockTypeStack)
			continue
		}

		if isApacheBlockEnd(line) {
			currentBlock, currentBlockContent, blockStack, blockTypeStack, directories, virtualHosts =
				popApacheBlock(currentBlock, currentBlockContent, blockStack, blockTypeStack, directories, virtualHosts)
			continue
		}

		key, value, ok := parseApacheDirectiveLine(line)
		if !ok {
			continue
		}

		targetMap := config
		if currentBlockContent != nil {
			targetMap = currentBlockContent
		}
		assignApacheDirective(targetMap, key, convertApacheValue(value), multiValueDirectives)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	setApacheParsedBlocks(config, "Directory", directories)
	setApacheParsedBlocks(config, "VirtualHost", virtualHosts)
	return config, nil
}

func apacheMultiValueDirectives() map[string]bool {
	return map[string]bool{
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
}

func shouldSkipApacheLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

func parseApacheBlockStart(line string) (string, string, bool) {
	if !strings.HasPrefix(line, "<") || strings.HasPrefix(line, "</") {
		return "", "", false
	}

	blockMatch := regexp.MustCompile(`<(\w+)\s+(.*)>`).FindStringSubmatch(line)
	if len(blockMatch) < 3 {
		return "", "", false
	}

	return blockMatch[1], strings.TrimSuffix(blockMatch[2], ">"), true
}

func pushApacheBlock(
	blockType string,
	blockArg string,
	currentBlock string,
	currentBlockContent map[string]any,
	blockStack []map[string]any,
	blockTypeStack []string,
) (string, map[string]any, []map[string]any, []string) {
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

	return blockType, newBlock, blockStack, blockTypeStack
}

func isApacheBlockEnd(line string) bool {
	return strings.HasPrefix(line, "</")
}

func popApacheBlock(
	currentBlock string,
	currentBlockContent map[string]any,
	blockStack []map[string]any,
	blockTypeStack []string,
	directories []map[string]any,
	virtualHosts []map[string]any,
) (string, map[string]any, []map[string]any, []string, []map[string]any, []map[string]any) {
	if currentBlockContent == nil {
		return currentBlock, currentBlockContent, blockStack, blockTypeStack, directories, virtualHosts
	}

	if currentBlock == "Directory" {
		if len(blockStack) > 0 {
			parentBlock := blockStack[len(blockStack)-1]
			appendNestedDirectory(parentBlock, currentBlockContent)
		} else {
			directories = append(directories, currentBlockContent)
		}
	} else if currentBlock == "VirtualHost" {
		virtualHosts = append(virtualHosts, currentBlockContent)
	}

	if len(blockStack) == 0 {
		return "", nil, blockStack, blockTypeStack, directories, virtualHosts
	}

	nextBlock := blockStack[len(blockStack)-1]
	nextType := blockTypeStack[len(blockTypeStack)-1]
	blockStack = blockStack[:len(blockStack)-1]
	blockTypeStack = blockTypeStack[:len(blockTypeStack)-1]
	return nextType, nextBlock, blockStack, blockTypeStack, directories, virtualHosts
}

func appendNestedDirectory(parentBlock map[string]any, child map[string]any) {
	if existingDirs, ok := parentBlock["Directory"].([]any); ok {
		parentBlock["Directory"] = append(existingDirs, child)
		return
	}
	parentBlock["Directory"] = []any{child}
}

func parseApacheDirectiveLine(line string) (string, string, bool) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 1 {
		return "", "", false
	}

	key := parts[0]
	value := ""
	if len(parts) >= 2 {
		value = strings.TrimSpace(parts[1])
	}
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		value = value[1 : len(value)-1]
	}
	return key, value, true
}

func assignApacheDirective(targetMap map[string]any, key string, converted any, multiValueDirectives map[string]bool) {
	if multiValueDirectives[key] {
		if existing, ok := targetMap[key].([]any); ok {
			targetMap[key] = append(existing, converted)
		} else if existing, ok := targetMap[key]; ok {
			targetMap[key] = []any{existing, converted}
		} else {
			targetMap[key] = []any{converted}
		}
		return
	}

	targetMap[key] = converted
}

func setApacheParsedBlocks(config map[string]any, field string, blocks []map[string]any) {
	if len(blocks) == 0 {
		return
	}

	values := make([]any, len(blocks))
	for i, block := range blocks {
		values[i] = block
	}
	config[field] = values
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
