// SPDX-License-Identifier: MIT

// Package kubernetes provides a Kubernetes resource configuration management plugin.
package kubernetes

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// Plugin implements the AppPlugin interface for Kubernetes.
type Plugin struct{}

// New creates a new Kubernetes plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "kubernetes"
}

// Version returns the supported Kubernetes API version range.
func (p *Plugin) Version() string {
	return ">=1.16"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Kubernetes resource configuration management (Deployment, Service, ConfigMap, etc.)"
}

// NativeFormat returns the native format identifier.
func (p *Plugin) NativeFormat() string {
	return "yaml"
}

// Schema returns the configuration schema for Kubernetes resources.
func (p *Plugin) Schema() plugin.AppSchema {
	return plugin.AppSchema{
		Sections: []plugin.SectionSchema{
			{
				Name:        "apiVersion",
				Required:    true,
				Description: "Kubernetes API version",
				Directives: []plugin.DirectiveSchema{
					{Name: "apiVersion", Type: "string", Required: true, Description: "API version (e.g., apps/v1, v1)"},
				},
			},
			{
				Name:        "kind",
				Required:    true,
				Description: "Kubernetes resource kind",
				Directives: []plugin.DirectiveSchema{
					{Name: "kind", Type: "string", Required: true, Description: "Resource kind (e.g., Deployment, Service)"},
				},
			},
			{
				Name:        "metadata",
				Required:    true,
				Description: "Resource metadata",
				Directives: []plugin.DirectiveSchema{
					{Name: "name", Type: "string", Required: true, Description: "Resource name"},
					{Name: "namespace", Type: "string", Description: "Namespace"},
					{Name: "labels", Type: "map", Description: "Labels for the resource"},
					{Name: "annotations", Type: "map", Description: "Annotations for the resource"},
				},
			},
			{
				Name:        "spec",
				Required:    false,
				Description: "Resource specification (kind-specific)",
			},
			{
				Name:        "data",
				Required:    false,
				Description: "Data section (for ConfigMap/Secret)",
			},
		},
	}
}

// validKinds maps kind to required apiVersion prefix
var validKinds = map[string][]string{
	"Deployment":              {"apps/v1", "apps/v1beta1", "apps/v1beta2", "extensions/v1beta1"},
	"StatefulSet":             {"apps/v1", "apps/v1beta1", "apps/v1beta2"},
	"DaemonSet":               {"apps/v1", "apps/v1beta1", "apps/v1beta2", "extensions/v1beta1"},
	"ReplicaSet":              {"apps/v1", "apps/v1beta1", "apps/v1beta2", "extensions/v1beta1"},
	"Service":                 {"v1"},
	"ConfigMap":               {"v1"},
	"Secret":                  {"v1"},
	"ServiceAccount":          {"v1"},
	"PersistentVolumeClaim":   {"v1"},
	"PersistentVolume":        {"v1"},
	"Pod":                     {"v1"},
	"Namespace":               {"v1"},
	"Ingress":                 {"networking.k8s.io/v1", "networking.k8s.io/v1beta1", "extensions/v1beta1"},
	"IngressClass":            {"networking.k8s.io/v1"},
	"NetworkPolicy":           {"networking.k8s.io/v1"},
	"Job":                     {"batch/v1"},
	"CronJob":                 {"batch/v1", "batch/v1beta1"},
	"HorizontalPodAutoscaler": {"autoscaling/v2", "autoscaling/v2beta2", "autoscaling/v1"},
	"Role":                    {"rbac.authorization.k8s.io/v1"},
	"ClusterRole":             {"rbac.authorization.k8s.io/v1"},
	"RoleBinding":             {"rbac.authorization.k8s.io/v1"},
	"ClusterRoleBinding":      {"rbac.authorization.k8s.io/v1"},
}

// DefaultConfig returns a sample Kubernetes Deployment configuration.
func (p *Plugin) DefaultConfig() any {
	return map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name": "example",
			"labels": map[string]any{
				"app": "example",
			},
		},
		"spec": map[string]any{
			"replicas": 1,
			"selector": map[string]any{
				"matchLabels": map[string]any{
					"app": "example",
				},
			},
			"template": map[string]any{
				"metadata": map[string]any{
					"labels": map[string]any{
						"app": "example",
					},
				},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "example",
							"image": "nginx:latest",
							"ports": []any{
								map[string]any{
									"containerPort": 80,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Validate validates the Kubernetes resource structure.
func (p *Plugin) Validate(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "", Message: "configuration must be a map"}}, nil
	}

	// Validate apiVersion (required)
	apiVersion, ok := configMap["apiVersion"]
	if !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "apiVersion",
			Message: "apiVersion is required",
		})
	} else {
		apiVersionStr := fmt.Sprintf("%v", apiVersion)
		if !isValidAPIVersion(apiVersionStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    "apiVersion",
				Message: fmt.Sprintf("invalid apiVersion format: %s", apiVersionStr),
			})
		}
	}

	// Validate kind (required)
	kind, ok := configMap["kind"]
	if !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "kind",
			Message: "kind is required",
		})
	}

	// Validate apiVersion/kind combination
	if apiVersion != nil && kind != nil {
		kindStr := fmt.Sprintf("%v", kind)
		apiVersionStr := fmt.Sprintf("%v", apiVersion)

		if validVersions, exists := validKinds[kindStr]; exists {
			found := false
			for _, v := range validVersions {
				if v == apiVersionStr {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, plugin.ValidationError{
					Path:    "apiVersion",
					Message: fmt.Sprintf("apiVersion '%s' is not valid for kind '%s', expected one of: %v", apiVersionStr, kindStr, validVersions),
				})
			}
		}
	}

	// Validate metadata (required)
	metadata, ok := configMap["metadata"]
	if !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "metadata",
			Message: "metadata is required",
		})
	} else {
		metaErrors := p.validateMetadata(metadata)
		errors = append(errors, metaErrors...)
	}

	// Kind-specific validation
	if kind != nil {
		kindStr := fmt.Sprintf("%v", kind)
		kindErrors := p.validateKindSpecific(kindStr, configMap)
		errors = append(errors, kindErrors...)
	}

	return errors, nil
}

func (p *Plugin) validateMetadata(metadata any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	metaMap, ok := metadata.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "metadata", Message: "metadata must be a map"}}
	}

	// Name is required
	name, ok := metaMap["name"]
	if !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "metadata.name",
			Message: "name is required",
		})
	} else {
		nameStr := fmt.Sprintf("%v", name)
		if !isValidK8sName(nameStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    "metadata.name",
				Message: fmt.Sprintf("invalid name: %s (must be lowercase alphanumeric with - allowed, max 253 chars)", nameStr),
			})
		}
	}

	// Validate namespace if present
	if ns, ok := metaMap["namespace"]; ok {
		nsStr := fmt.Sprintf("%v", ns)
		if !isValidK8sName(nsStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    "metadata.namespace",
				Message: fmt.Sprintf("invalid namespace: %s", nsStr),
			})
		}
	}

	// Validate labels
	if labels, ok := metaMap["labels"]; ok {
		labelErrors := p.validateLabels(labels, "metadata.labels")
		errors = append(errors, labelErrors...)
	}

	// Validate annotations
	if annotations, ok := metaMap["annotations"]; ok {
		annoErrors := p.validateAnnotations(annotations, "metadata.annotations")
		errors = append(errors, annoErrors...)
	}

	return errors
}

func (p *Plugin) validateLabels(labels any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	labelMap, ok := labels.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "labels must be a map"}}
	}

	for k, v := range labelMap {
		// Validate label key
		if !isValidLabelKey(k) {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.%s", path, k),
				Message: fmt.Sprintf("invalid label key: %s", k),
			})
		}

		// Validate label value
		valStr := fmt.Sprintf("%v", v)
		if !isValidLabelValue(valStr) {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.%s", path, k),
				Message: fmt.Sprintf("invalid label value: %s", valStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateAnnotations(annotations any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	annoMap, ok := annotations.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "annotations must be a map"}}
	}

	for k := range annoMap {
		// Validate annotation key (same rules as label keys)
		if !isValidLabelKey(k) {
			errors = append(errors, plugin.ValidationError{
				Path:    fmt.Sprintf("%s.%s", path, k),
				Message: fmt.Sprintf("invalid annotation key: %s", k),
			})
		}
	}

	return errors
}

func (p *Plugin) validateKindSpecific(kind string, config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet":
		errors = append(errors, p.validateWorkload(kind, config)...)
	case "Service":
		errors = append(errors, p.validateService(config)...)
	case "ConfigMap":
		errors = append(errors, p.validateConfigMap(config)...)
	case "Secret":
		errors = append(errors, p.validateSecret(config)...)
	case "Ingress":
		errors = append(errors, p.validateIngress(config)...)
	}

	return errors
}

func (p *Plugin) validateWorkload(kind string, config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	spec, ok := config["spec"].(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "spec", Message: "spec is required for " + kind}}
	}

	// Validate selector
	selector, ok := spec["selector"].(map[string]any)
	if !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "spec.selector",
			Message: "selector is required",
		})
	}

	// Validate template
	template, ok := spec["template"].(map[string]any)
	if !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    "spec.template",
			Message: "template is required",
		})
		return errors
	}

	// Validate selector matches template labels
	if selector != nil && template != nil {
		if matchLabels, ok := selector["matchLabels"].(map[string]any); ok {
			if templateMeta, ok := template["metadata"].(map[string]any); ok {
				if templateLabels, ok := templateMeta["labels"].(map[string]any); ok {
					for k, v := range matchLabels {
						if tVal, exists := templateLabels[k]; !exists || fmt.Sprintf("%v", tVal) != fmt.Sprintf("%v", v) {
							errors = append(errors, plugin.ValidationError{
								Path:    "spec.selector.matchLabels",
								Message: fmt.Sprintf("selector label %s=%v does not match template labels", k, v),
							})
						}
					}
				}
			}
		}
	}

	// Validate containers
	if templateSpec, ok := template["spec"].(map[string]any); ok {
		if containers, ok := templateSpec["containers"].([]any); ok {
			if len(containers) == 0 {
				errors = append(errors, plugin.ValidationError{
					Path:    "spec.template.spec.containers",
					Message: "at least one container is required",
				})
			}
			for i, c := range containers {
				containerErrors := p.validateContainer(c, fmt.Sprintf("spec.template.spec.containers[%d]", i))
				errors = append(errors, containerErrors...)
			}
		} else {
			errors = append(errors, plugin.ValidationError{
				Path:    "spec.template.spec.containers",
				Message: "containers is required",
			})
		}
	}

	// Validate replicas
	if replicas, ok := spec["replicas"]; ok {
		switch r := replicas.(type) {
		case int:
			if r < 0 {
				errors = append(errors, plugin.ValidationError{
					Path:    "spec.replicas",
					Message: "replicas cannot be negative",
				})
			}
		case int64:
			if r < 0 {
				errors = append(errors, plugin.ValidationError{
					Path:    "spec.replicas",
					Message: "replicas cannot be negative",
				})
			}
		case float64:
			if r < 0 {
				errors = append(errors, plugin.ValidationError{
					Path:    "spec.replicas",
					Message: "replicas cannot be negative",
				})
			}
		}
	}

	return errors
}

func (p *Plugin) validateContainer(container any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	containerMap, ok := container.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "container must be a map"}}
	}

	// Name is required
	if _, ok := containerMap["name"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    path + ".name",
			Message: "container name is required",
		})
	}

	// Image is required
	if _, ok := containerMap["image"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    path + ".image",
			Message: "container image is required",
		})
	}

	// Validate resources if present
	if resources, ok := containerMap["resources"].(map[string]any); ok {
		errors = append(errors, p.validateResources(resources, path+".resources")...)
	}

	return errors
}

func (p *Plugin) validateResources(resources map[string]any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	requests, hasRequests := resources["requests"].(map[string]any)
	limits, hasLimits := resources["limits"].(map[string]any)

	if hasRequests && hasLimits {
		// Validate requests <= limits
		if reqCPU, ok := requests["cpu"]; ok {
			if limCPU, ok := limits["cpu"]; ok {
				reqVal := parseResourceQuantity(fmt.Sprintf("%v", reqCPU))
				limVal := parseResourceQuantity(fmt.Sprintf("%v", limCPU))
				if reqVal > limVal {
					errors = append(errors, plugin.ValidationError{
						Path:    path + ".requests.cpu",
						Message: fmt.Sprintf("requests.cpu (%v) exceeds limits.cpu (%v)", reqCPU, limCPU),
					})
				}
			}
		}
		if reqMem, ok := requests["memory"]; ok {
			if limMem, ok := limits["memory"]; ok {
				reqVal := parseResourceQuantity(fmt.Sprintf("%v", reqMem))
				limVal := parseResourceQuantity(fmt.Sprintf("%v", limMem))
				if reqVal > limVal {
					errors = append(errors, plugin.ValidationError{
						Path:    path + ".requests.memory",
						Message: fmt.Sprintf("requests.memory (%v) exceeds limits.memory (%v)", reqMem, limMem),
					})
				}
			}
		}
	}

	return errors
}

func (p *Plugin) validateService(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	spec, ok := config["spec"].(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "spec", Message: "spec is required for Service"}}
	}

	// Validate type if present
	if svcType, ok := spec["type"]; ok {
		typeStr := fmt.Sprintf("%v", svcType)
		validTypes := []string{"ClusterIP", "NodePort", "LoadBalancer", "ExternalName"}
		found := false
		for _, t := range validTypes {
			if t == typeStr {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    "spec.type",
				Message: fmt.Sprintf("invalid service type: %s", typeStr),
			})
		}
	}

	// Validate ports
	if ports, ok := spec["ports"].([]any); ok {
		for i, port := range ports {
			portErrors := p.validateServicePort(port, fmt.Sprintf("spec.ports[%d]", i))
			errors = append(errors, portErrors...)
		}
	}

	return errors
}

func (p *Plugin) validateServicePort(port any, path string) []plugin.ValidationError {
	var errors []plugin.ValidationError

	portMap, ok := port.(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: path, Message: "port must be a map"}}
	}

	// Port is required
	if _, ok := portMap["port"]; !ok {
		errors = append(errors, plugin.ValidationError{
			Path:    path + ".port",
			Message: "port is required",
		})
	}

	// Validate protocol if present
	if protocol, ok := portMap["protocol"]; ok {
		protocolStr := fmt.Sprintf("%v", protocol)
		validProtocols := []string{"TCP", "UDP", "SCTP"}
		found := false
		for _, p := range validProtocols {
			if p == protocolStr {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, plugin.ValidationError{
				Path:    path + ".protocol",
				Message: fmt.Sprintf("invalid protocol: %s", protocolStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateConfigMap(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError
	// ConfigMap can have data or binaryData, both are optional
	return errors
}

func (p *Plugin) validateSecret(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	// Validate type if present
	if secretType, ok := config["type"]; ok {
		typeStr := fmt.Sprintf("%v", secretType)
		validTypes := []string{
			"Opaque",
			"kubernetes.io/service-account-token",
			"kubernetes.io/dockercfg",
			"kubernetes.io/dockerconfigjson",
			"kubernetes.io/basic-auth",
			"kubernetes.io/ssh-auth",
			"kubernetes.io/tls",
			"bootstrap.kubernetes.io/token",
		}
		found := false
		for _, t := range validTypes {
			if t == typeStr {
				found = true
				break
			}
		}
		if !found && !strings.Contains(typeStr, "/") {
			errors = append(errors, plugin.ValidationError{
				Path:    "type",
				Message: fmt.Sprintf("unknown secret type: %s", typeStr),
			})
		}
	}

	return errors
}

func (p *Plugin) validateIngress(config map[string]any) []plugin.ValidationError {
	var errors []plugin.ValidationError

	spec, ok := config["spec"].(map[string]any)
	if !ok {
		return []plugin.ValidationError{{Path: "spec", Message: "spec is required for Ingress"}}
	}

	// Validate rules if present
	if rules, ok := spec["rules"].([]any); ok {
		for i, rule := range rules {
			if ruleMap, ok := rule.(map[string]any); ok {
				if http, ok := ruleMap["http"].(map[string]any); ok {
					if paths, ok := http["paths"].([]any); ok {
						for j, pathItem := range paths {
							if pathMap, ok := pathItem.(map[string]any); ok {
								// Validate pathType
								if pathType, ok := pathMap["pathType"]; ok {
									pathTypeStr := fmt.Sprintf("%v", pathType)
									validTypes := []string{"Exact", "Prefix", "ImplementationSpecific"}
									found := false
									for _, t := range validTypes {
										if t == pathTypeStr {
											found = true
											break
										}
									}
									if !found {
										errors = append(errors, plugin.ValidationError{
											Path:    fmt.Sprintf("spec.rules[%d].http.paths[%d].pathType", i, j),
											Message: fmt.Sprintf("invalid pathType: %s", pathTypeStr),
										})
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return errors
}

// ValidateSemantic performs Kubernetes-specific semantic validation.
func (p *Plugin) ValidateSemantic(config any) ([]plugin.ValidationError, error) {
	var errors []plugin.ValidationError

	configMap, ok := config.(map[string]any)
	if !ok {
		return errors, nil
	}

	kind := fmt.Sprintf("%v", configMap["kind"])

	// Check for common issues
	switch kind {
	case "Deployment", "StatefulSet":
		if spec, ok := configMap["spec"].(map[string]any); ok {
			// Warn about using 'latest' tag
			if template, ok := spec["template"].(map[string]any); ok {
				if templateSpec, ok := template["spec"].(map[string]any); ok {
					if containers, ok := templateSpec["containers"].([]any); ok {
						for i, c := range containers {
							if cMap, ok := c.(map[string]any); ok {
								if image, ok := cMap["image"].(string); ok {
									if strings.HasSuffix(image, ":latest") || !strings.Contains(image, ":") {
										errors = append(errors, plugin.ValidationError{
											Path:    fmt.Sprintf("spec.template.spec.containers[%d].image", i),
											Message: fmt.Sprintf("using 'latest' or no tag for image %s is not recommended for production", image),
										})
									}
								}
							}
						}
					}
				}
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

	// Copy with canonical key order
	keyOrder := []string{"apiVersion", "kind", "metadata", "spec", "data", "binaryData", "type", "status"}
	for _, k := range keyOrder {
		if v, ok := configMap[k]; ok {
			result[k] = v
		}
	}

	// Copy any remaining keys
	for k, v := range configMap {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result, nil
}

// ToNative converts the configuration to Kubernetes native YAML format.
func (p *Plugin) ToNative(config any) ([]byte, error) {
	return yaml.Marshal(config)
}

// FromNative parses Kubernetes native YAML format into configuration.
func (p *Plugin) FromNative(data []byte) (any, error) {
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Kubernetes config: %w", err)
	}
	return result, nil
}

// Merge merges two Kubernetes configurations.
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
		switch k {
		case "metadata":
			// Deep merge metadata
			if existing, ok := result[k]; ok {
				merged, _ := mergeDeep(existing, v)
				result[k] = merged
			} else {
				result[k] = deepCopy(v)
			}

		case "spec":
			// Deep merge spec
			if existing, ok := result[k]; ok {
				merged, _ := mergeDeep(existing, v)
				result[k] = merged
			} else {
				result[k] = deepCopy(v)
			}

		default:
			result[k] = deepCopy(v)
		}
	}

	return result, nil
}

// Diff detects changes between two Kubernetes configurations.
func (p *Plugin) Diff(old, new any) ([]plugin.Change, error) {
	return diffMaps(old, new, "")
}

// Helper functions

var apiVersionRegex = regexp.MustCompile(`^[a-z]+(\.[a-z0-9-]+)*/v\d+(alpha\d+|beta\d+)?$|^v\d+$`)

func isValidAPIVersion(s string) bool {
	return apiVersionRegex.MatchString(s)
}

var k8sNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func isValidK8sName(s string) bool {
	if len(s) > 253 {
		return false
	}
	return k8sNameRegex.MatchString(s)
}

var labelKeyRegex = regexp.MustCompile(`^([a-z0-9]([-a-z0-9]*[a-z0-9])?\.)*[a-z0-9]([-a-z0-9]*[a-z0-9])?/[a-zA-Z0-9]([-_.a-zA-Z0-9]*[a-zA-Z0-9])?$|^[a-zA-Z0-9]([-_.a-zA-Z0-9]*[a-zA-Z0-9])?$`)

func isValidLabelKey(s string) bool {
	if len(s) > 316 { // 253 (prefix) + 1 (/) + 63 (name)
		return false
	}
	return labelKeyRegex.MatchString(s)
}

var labelValueRegex = regexp.MustCompile(`^([a-zA-Z0-9]([-_.a-zA-Z0-9]*[a-zA-Z0-9])?)?$`)

func isValidLabelValue(s string) bool {
	if len(s) > 63 {
		return false
	}
	return labelValueRegex.MatchString(s)
}

func parseResourceQuantity(s string) float64 {
	s = strings.TrimSpace(s)

	multipliers := map[string]float64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
		"K":  1000,
		"M":  1000 * 1000,
		"G":  1000 * 1000 * 1000,
		"T":  1000 * 1000 * 1000 * 1000,
		"m":  0.001, // millicores
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			var num float64
			fmt.Sscanf(numStr, "%f", &num)
			return num * mult
		}
	}

	var num float64
	fmt.Sscanf(s, "%f", &num)
	return num
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

func mergeDeep(base, overlay any) (any, error) {
	baseMap, baseOk := base.(map[string]any)
	overlayMap, overlayOk := overlay.(map[string]any)

	if !baseOk || !overlayOk {
		return overlay, nil
	}

	result := make(map[string]any)
	for k, v := range baseMap {
		result[k] = v
	}
	for k, v := range overlayMap {
		if existing, ok := result[k]; ok {
			if _, isMap := existing.(map[string]any); isMap {
				if _, vIsMap := v.(map[string]any); vIsMap {
					merged, _ := mergeDeep(existing, v)
					result[k] = merged
					continue
				}
			}
		}
		result[k] = v
	}
	return result, nil
}

func diffMaps(old, new any, path string) ([]plugin.Change, error) {
	var changes []plugin.Change

	oldMap, oldOk := old.(map[string]any)
	newMap, newOk := new.(map[string]any)

	if !oldOk && !newOk {
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

	if !oldOk || !newOk {
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

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
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
			subChanges, _ := diffMaps(oldVal, newVal, keyPath)
			changes = append(changes, subChanges...)
		}
	}

	return changes, nil
}
