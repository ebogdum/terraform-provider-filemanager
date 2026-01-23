// SPDX-License-Identifier: MIT

// Package tfvars_file implements the filemanager_tfvars_file resource.
package tfvars_file

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

// ResolveInterpolation resolves template references within vars.
// It builds a dependency graph among top-level var keys, topologically sorts them,
// and resolves self-references in order. Then resolves template_vars references.
func ResolveInterpolation(
	vars map[string]any,
	templateVars map[string]string,
	leftDelim, rightDelim string,
) (map[string]any, error) {
	if len(vars) == 0 {
		return vars, nil
	}

	// Extract all top-level keys
	knownKeys := make(map[string]bool, len(vars))
	allKeys := make([]string, 0, len(vars))
	for k := range vars {
		knownKeys[k] = true
		allKeys = append(allKeys, k)
	}

	// Build dependency graph: key → keys it references
	graph := make(map[string][]string, len(vars))
	for _, key := range allKeys {
		deps := collectDeps(key, vars, knownKeys, leftDelim, rightDelim)
		if len(deps) > 0 {
			graph[key] = deps
		}
	}

	// Topological sort
	sorted, err := topologicalSort(graph, allKeys)
	if err != nil {
		return nil, err
	}

	// Resolve in topological order
	resolved := make(map[string]any, len(vars))
	for _, key := range sorted {
		val := vars[key]
		resolvedVal, err := resolveDeep(val, resolved, leftDelim, rightDelim)
		if err != nil {
			return nil, fmt.Errorf("resolving var %q: %w", key, err)
		}
		resolved[key] = resolvedVal
	}

	// Second pass: resolve template_vars references
	if len(templateVars) > 0 {
		tvData := make(map[string]any, len(templateVars))
		for k, v := range templateVars {
			tvData[k] = v
		}
		// Merge resolved vars + template_vars (template_vars take precedence for unresolved refs)
		combined := make(map[string]any, len(resolved)+len(tvData))
		for k, v := range resolved {
			combined[k] = v
		}
		for k, v := range tvData {
			combined[k] = v
		}

		for key, val := range resolved {
			resolvedVal, err := resolveDeep(val, combined, leftDelim, rightDelim)
			if err != nil {
				return nil, fmt.Errorf("resolving template_vars in %q: %w", key, err)
			}
			resolved[key] = resolvedVal
		}
	}

	return resolved, nil
}

// resolveStringValue renders a single string value with Go templates.
func resolveStringValue(value string, data map[string]any, leftDelim, rightDelim string) (string, error) {
	if !strings.Contains(value, leftDelim) {
		return value, nil
	}

	tmpl, err := template.New("").
		Delims(leftDelim, rightDelim).
		Funcs(templateFuncs()).
		Option("missingkey=error").
		Parse(value)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}

	return buf.String(), nil
}

// resolveDeep recursively walks a value and resolves template strings at any depth.
func resolveDeep(value any, data map[string]any, leftDelim, rightDelim string) (any, error) {
	switch v := value.(type) {
	case string:
		return resolveStringValue(v, data, leftDelim, rightDelim)

	case map[string]any:
		result := make(map[string]any, len(v))
		for k, val := range v {
			resolved, err := resolveDeep(val, data, leftDelim, rightDelim)
			if err != nil {
				return nil, err
			}
			result[k] = resolved
		}
		return result, nil

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			resolved, err := resolveDeep(val, data, leftDelim, rightDelim)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil

	default:
		return value, nil
	}
}

// extractTopLevelRefs parses a string for {{ .X }} references that match top-level var keys.
func extractTopLevelRefs(value string, knownKeys map[string]bool, leftDelim, rightDelim string) []string {
	if !strings.Contains(value, leftDelim) {
		return nil
	}

	// Build regex pattern for the delimiters
	escapedLeft := regexp.QuoteMeta(leftDelim)
	escapedRight := regexp.QuoteMeta(rightDelim)
	pattern := escapedLeft + `\s*\.(\w+)\s*` + escapedRight
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(value, -1)
	seen := make(map[string]bool)
	var refs []string
	for _, match := range matches {
		if len(match) > 1 {
			ref := match[1]
			if knownKeys[ref] && !seen[ref] {
				seen[ref] = true
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// collectDeps recursively collects all dependencies for a top-level var.
func collectDeps(key string, vars map[string]any, knownKeys map[string]bool, leftDelim, rightDelim string) []string {
	seen := make(map[string]bool)
	collectDepsRecursive(vars[key], knownKeys, leftDelim, rightDelim, seen)
	// Remove self-reference from deps (handled by topo sort cycle detection)
	delete(seen, key)
	deps := make([]string, 0, len(seen))
	for dep := range seen {
		deps = append(deps, dep)
	}
	return deps
}

// collectDepsRecursive walks a value and collects template references.
func collectDepsRecursive(value any, knownKeys map[string]bool, leftDelim, rightDelim string, seen map[string]bool) {
	switch v := value.(type) {
	case string:
		refs := extractTopLevelRefs(v, knownKeys, leftDelim, rightDelim)
		for _, ref := range refs {
			seen[ref] = true
		}
	case map[string]any:
		for _, val := range v {
			collectDepsRecursive(val, knownKeys, leftDelim, rightDelim, seen)
		}
	case []any:
		for _, val := range v {
			collectDepsRecursive(val, knownKeys, leftDelim, rightDelim, seen)
		}
	}
}

// topologicalSort performs Kahn's algorithm. Returns error on cycles.
func topologicalSort(graph map[string][]string, allKeys []string) ([]string, error) {
	// Build in-degree map
	inDegree := make(map[string]int, len(allKeys))
	for _, k := range allKeys {
		inDegree[k] = 0
	}

	// Reverse adjacency: graph[A] = [B, C] means A depends on B and C
	// For Kahn's, we need: who depends on each node
	dependents := make(map[string][]string, len(allKeys))
	for node, deps := range graph {
		for _, dep := range deps {
			// dep must be processed before node
			dependents[dep] = append(dependents[dep], node)
			inDegree[node]++
		}
	}

	// Start with nodes that have no dependencies
	queue := make([]string, 0)
	for _, k := range allKeys {
		if inDegree[k] == 0 {
			queue = append(queue, k)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		for _, dependent := range dependents[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(allKeys) {
		// Find cycle for error message
		var cycleNodes []string
		for _, k := range allKeys {
			if inDegree[k] > 0 {
				cycleNodes = append(cycleNodes, k)
			}
		}
		return nil, fmt.Errorf("circular dependency detected among variables: %s", strings.Join(cycleNodes, " -> "))
	}

	return sorted, nil
}

// templateFuncs returns template function map.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"upper":   strings.ToUpper,
		"lower":   strings.ToLower,
		"trim":    strings.TrimSpace,
		"replace": strings.ReplaceAll,
		"split": func(sep, s string) []string {
			return strings.Split(s, sep)
		},
		"join": func(sep string, elems []string) string {
			return strings.Join(elems, sep)
		},
		"default": func(def, val string) string {
			if "" == val {
				return def
			}
			return val
		},
		"env": os.Getenv,
		"contains": func(substr, s string) bool {
			return strings.Contains(s, substr)
		},
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"trimPrefix": func(prefix, s string) string {
			return strings.TrimPrefix(s, prefix)
		},
		"trimSuffix": func(suffix, s string) string {
			return strings.TrimSuffix(s, suffix)
		},
	}
}
