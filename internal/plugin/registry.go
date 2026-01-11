// SPDX-License-Identifier: MIT

package plugin

import (
	"fmt"
	"sync"
)

// -----------------------------------------------------------------------------
// Backend Registry
// -----------------------------------------------------------------------------

// BackendRegistry manages registered backend plugins.
type BackendRegistry struct {
	mu       sync.RWMutex
	backends map[string]BackendFactory
	aliases  map[string]Backend // Configured backend instances with aliases
}

// BackendFactory creates a new Backend instance.
type BackendFactory func() Backend

// NewBackendRegistry creates a new backend registry.
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{
		backends: make(map[string]BackendFactory),
		aliases:  make(map[string]Backend),
	}
}

// Register registers a backend factory by scheme.
func (r *BackendRegistry) Register(scheme string, factory BackendFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[scheme]; exists {
		return fmt.Errorf("%w: backend with scheme %q already registered", ErrPluginExists, scheme)
	}

	r.backends[scheme] = factory
	return nil
}

// Get returns a backend factory by scheme.
func (r *BackendRegistry) Get(scheme string) (BackendFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.backends[scheme]
	if !ok {
		return nil, fmt.Errorf("%w: backend scheme %q", ErrPluginNotFound, scheme)
	}
	return factory, nil
}

// SetAlias registers a configured backend instance with an alias.
func (r *BackendRegistry) SetAlias(alias string, backend Backend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.aliases[alias]; exists {
		return fmt.Errorf("%w: backend alias %q already exists", ErrPluginExists, alias)
	}

	r.aliases[alias] = backend
	return nil
}

// GetAlias returns a configured backend instance by alias.
func (r *BackendRegistry) GetAlias(alias string) (Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backend, ok := r.aliases[alias]
	if !ok {
		return nil, fmt.Errorf("%w: backend alias %q", ErrPluginNotFound, alias)
	}
	return backend, nil
}

// RemoveAlias removes a backend alias.
func (r *BackendRegistry) RemoveAlias(alias string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.aliases, alias)
}

// Schemes returns all registered backend schemes.
func (r *BackendRegistry) Schemes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemes := make([]string, 0, len(r.backends))
	for scheme := range r.backends {
		schemes = append(schemes, scheme)
	}
	return schemes
}

// Aliases returns all registered backend aliases.
func (r *BackendRegistry) Aliases() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	aliases := make([]string, 0, len(r.aliases))
	for alias := range r.aliases {
		aliases = append(aliases, alias)
	}
	return aliases
}

// CloseAll closes all aliased backend instances.
func (r *BackendRegistry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for alias, backend := range r.aliases {
		if err := backend.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close backend %q: %w", alias, err)
		}
	}
	r.aliases = make(map[string]Backend)
	return lastErr
}

// -----------------------------------------------------------------------------
// Format Registry
// -----------------------------------------------------------------------------

// FormatRegistry manages registered format plugins.
type FormatRegistry struct {
	mu          sync.RWMutex
	formats     map[string]FormatPlugin
	byExtension map[string]FormatPlugin
	byMimeType  map[string]FormatPlugin
}

// NewFormatRegistry creates a new format registry.
func NewFormatRegistry() *FormatRegistry {
	return &FormatRegistry{
		formats:     make(map[string]FormatPlugin),
		byExtension: make(map[string]FormatPlugin),
		byMimeType:  make(map[string]FormatPlugin),
	}
}

// Register registers a format plugin.
func (r *FormatRegistry) Register(plugin FormatPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.formats[name]; exists {
		return fmt.Errorf("%w: format %q already registered", ErrPluginExists, name)
	}

	r.formats[name] = plugin

	// Index by extensions
	for _, ext := range plugin.Extensions() {
		r.byExtension[ext] = plugin
	}

	// Index by MIME types
	for _, mimeType := range plugin.MimeTypes() {
		r.byMimeType[mimeType] = plugin
	}

	return nil
}

// Get returns a format plugin by name.
func (r *FormatRegistry) Get(name string) (FormatPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.formats[name]
	if !ok {
		return nil, fmt.Errorf("%w: format %q", ErrPluginNotFound, name)
	}
	return plugin, nil
}

// GetByExtension returns a format plugin by file extension.
func (r *FormatRegistry) GetByExtension(ext string) (FormatPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.byExtension[ext]
	if !ok {
		return nil, fmt.Errorf("%w: format for extension %q", ErrPluginNotFound, ext)
	}
	return plugin, nil
}

// GetByMimeType returns a format plugin by MIME type.
func (r *FormatRegistry) GetByMimeType(mimeType string) (FormatPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.byMimeType[mimeType]
	if !ok {
		return nil, fmt.Errorf("%w: format for MIME type %q", ErrPluginNotFound, mimeType)
	}
	return plugin, nil
}

// Names returns all registered format names.
func (r *FormatRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.formats))
	for name := range r.formats {
		names = append(names, name)
	}
	return names
}

// -----------------------------------------------------------------------------
// App Registry
// -----------------------------------------------------------------------------

// AppRegistry manages registered application plugins.
type AppRegistry struct {
	mu   sync.RWMutex
	apps map[string]AppPlugin
}

// NewAppRegistry creates a new app registry.
func NewAppRegistry() *AppRegistry {
	return &AppRegistry{
		apps: make(map[string]AppPlugin),
	}
}

// Register registers an app plugin.
func (r *AppRegistry) Register(plugin AppPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.apps[name]; exists {
		return fmt.Errorf("%w: app %q already registered", ErrPluginExists, name)
	}

	r.apps[name] = plugin
	return nil
}

// Get returns an app plugin by name.
func (r *AppRegistry) Get(name string) (AppPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.apps[name]
	if !ok {
		return nil, fmt.Errorf("%w: app %q", ErrPluginNotFound, name)
	}
	return plugin, nil
}

// Names returns all registered app names.
func (r *AppRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.apps))
	for name := range r.apps {
		names = append(names, name)
	}
	return names
}

// -----------------------------------------------------------------------------
// Master Registry
// -----------------------------------------------------------------------------

// Registry provides unified access to all plugin registries.
type Registry struct {
	Backends *BackendRegistry
	Formats  *FormatRegistry
	Apps     *AppRegistry
}

// NewRegistry creates a new master registry with all sub-registries.
func NewRegistry() *Registry {
	return &Registry{
		Backends: NewBackendRegistry(),
		Formats:  NewFormatRegistry(),
		Apps:     NewAppRegistry(),
	}
}

// Close closes all resources held by the registry.
func (r *Registry) Close() error {
	return r.Backends.CloseAll()
}
