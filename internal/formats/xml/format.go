// SPDX-License-Identifier: MIT

// Package xml provides XML format plugin implementation with XPath support.
package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Format implements the FormatPlugin interface for XML.
type Format struct{}

// New creates a new XML format plugin.
func New() *Format {
	return &Format{}
}

// Name returns the format name.
func (f *Format) Name() string {
	return "xml"
}

// Extensions returns the supported file extensions.
func (f *Format) Extensions() []string {
	return []string{".xml"}
}

// MimeTypes returns the supported MIME types.
func (f *Format) MimeTypes() []string {
	return []string{"application/xml", "text/xml"}
}

// XMLNode represents a node in the XML tree.
type XMLNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Content  string     `xml:",chardata"`
	Children []*XMLNode `xml:",any"`
	Comment  string     `xml:",comment"`
	InnerXML string     `xml:",innerxml"`
}

// Parse parses XML data into a Go value.
// Returns a map[string]any representation of the XML.
func (f *Format) Parse(data []byte) (any, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("XML parse error: %w", err)
	}

	return xmlNodeToMap(doc), nil
}

// xmlNodeToMap converts an xmlquery.Node to a map representation.
func xmlNodeToMap(node *xmlquery.Node) any {
	if node == nil {
		return nil
	}

	switch node.Type {
	case xmlquery.DocumentNode:
		// Document node - return the root element
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == xmlquery.ElementNode {
				result := make(map[string]any)
				val, err := elementToMap(child, 0)
				if err != nil {
					return nil
				}
				result[child.Data] = val
				return result
			}
		}
		return nil

	case xmlquery.ElementNode:
		result := make(map[string]any)
		val, err := elementToMap(node, 0)
		if err != nil {
			return nil
		}
		result[node.Data] = val
		return result

	case xmlquery.TextNode:
		return strings.TrimSpace(node.Data)

	default:
		return nil
	}
}

const maxXMLDepth = 500

// elementToMap converts an XML element to a map.
func elementToMap(node *xmlquery.Node, depth ...int) (any, error) {
	currentDepth := 0
	if len(depth) > 0 {
		currentDepth = depth[0]
	}
	if currentDepth > maxXMLDepth {
		return nil, fmt.Errorf("XML nesting depth exceeds maximum of %d", maxXMLDepth)
	}

	if node == nil {
		return nil, nil
	}

	result := make(map[string]any)

	// Handle attributes
	for _, attr := range node.Attr {
		attrKey := "@" + attr.Name.Local
		if attr.Name.Space != "" {
			attrKey = "@" + attr.Name.Space + ":" + attr.Name.Local
		}
		result[attrKey] = attr.Value
	}

	// Collect children by tag name
	childMap := make(map[string][]any)
	var textContent strings.Builder
	hasElementChildren := false

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case xmlquery.ElementNode:
			hasElementChildren = true
			childValue, err := elementToMap(child, currentDepth+1)
			if err != nil {
				return nil, err
			}
			childMap[child.Data] = append(childMap[child.Data], childValue)

		case xmlquery.TextNode:
			text := strings.TrimSpace(child.Data)
			if text != "" {
				textContent.WriteString(text)
			}
		}
	}

	// Add children to result
	for name, children := range childMap {
		if len(children) == 1 {
			result[name] = children[0]
		} else {
			result[name] = children
		}
	}

	// Handle text content
	if textContent.Len() > 0 {
		if hasElementChildren || len(result) > 0 {
			result["#text"] = textContent.String()
		} else {
			// Pure text element
			return textContent.String(), nil
		}
	}

	// If result is empty and no text, return empty map
	if len(result) == 0 {
		return make(map[string]any), nil
	}

	return result, nil
}

// Serialize serializes a Go value to XML.
func (f *Format) Serialize(value any, opts plugin.SerializeOptions) ([]byte, error) {
	var buf bytes.Buffer

	// Write XML declaration
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	if !opts.Compact {
		buf.WriteString("\n")
	}

	indent := ""
	if !opts.Compact {
		if opts.IndentChar != "" {
			indent = strings.Repeat(opts.IndentChar, opts.Indent)
		} else if opts.Indent > 0 {
			indent = strings.Repeat(" ", opts.Indent)
		} else {
			indent = "  "
		}
	}

	// Serialize the value
	if err := serializeValue(&buf, value, "", indent, opts); err != nil {
		return nil, fmt.Errorf("XML serialize error: %w", err)
	}

	result := buf.Bytes()
	if opts.TrailingNewline && len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}

	return result, nil
}

// serializeValue serializes a value to XML.
func serializeValue(w io.Writer, value any, currentIndent, indentStep string, opts plugin.SerializeOptions) error {
	switch v := value.(type) {
	case map[string]any:
		return serializeMap(w, v, currentIndent, indentStep, opts)

	case []any:
		// Arrays at root level aren't valid XML, wrap in container
		for _, item := range v {
			if err := serializeValue(w, item, currentIndent, indentStep, opts); err != nil {
				return err
			}
		}
		return nil

	case string:
		_, err := io.WriteString(w, escapeXML(v))
		return err

	case nil:
		return nil

	default:
		_, err := fmt.Fprintf(w, "%v", v)
		return err
	}
}

// serializeMap serializes a map to XML elements.
func serializeMap(w io.Writer, m map[string]any, currentIndent, indentStep string, opts plugin.SerializeOptions) error {
	// Get sorted keys if requested
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if opts.SortKeys {
		sort.Strings(keys)
	}

	for _, key := range keys {
		val := m[key]

		// Skip special keys at this level (they're attributes or text)
		if strings.HasPrefix(key, "@") || key == "#text" {
			continue
		}

		if err := serializeElement(w, key, val, currentIndent, indentStep, opts); err != nil {
			return err
		}
	}

	return nil
}

// xmlNameRegex validates XML element names (simplified check).
var xmlNameRegex = regexp.MustCompile(`^[a-zA-Z_:][\w.\-:]*$`)

// serializeElement serializes a single XML element.
func serializeElement(w io.Writer, name string, value any, currentIndent, indentStep string, opts plugin.SerializeOptions) error {
	if !xmlNameRegex.MatchString(name) {
		return fmt.Errorf("invalid XML element name: %q", name)
	}
	switch v := value.(type) {
	case []any:
		// Multiple elements with same name
		for _, item := range v {
			if err := serializeElement(w, name, item, currentIndent, indentStep, opts); err != nil {
				return err
			}
		}
		return nil

	case map[string]any:
		return serializeMapElement(w, name, v, currentIndent, indentStep, opts)

	case string:
		if _, err := fmt.Fprintf(w, "%s<%s>%s</%s>", currentIndent, name, escapeXML(v), name); err != nil {
			return err
		}
		if !opts.Compact {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		return nil

	case nil:
		if _, err := fmt.Fprintf(w, "%s<%s/>", currentIndent, name); err != nil {
			return err
		}
		if !opts.Compact {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		return nil

	default:
		if _, err := fmt.Fprintf(w, "%s<%s>%v</%s>", currentIndent, name, v, name); err != nil {
			return err
		}
		if !opts.Compact {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		return nil
	}
}

// serializeMapElement serializes a map as an XML element with attributes and children.
func serializeMapElement(w io.Writer, name string, m map[string]any, currentIndent, indentStep string, opts plugin.SerializeOptions) error {
	if _, err := fmt.Fprintf(w, "%s<%s", currentIndent, name); err != nil {
		return err
	}

	keys, attrs, textContent, hasChildren := collectMapElementParts(m, opts.SortKeys)
	for _, attr := range attrs {
		if _, err := io.WriteString(w, attr); err != nil {
			return err
		}
	}

	if !hasChildren && textContent == "" {
		if _, err := io.WriteString(w, "/>"); err != nil {
			return err
		}
		return writeOptionalNewline(w, opts.Compact)
	}

	if _, err := io.WriteString(w, ">"); err != nil {
		return err
	}

	if hasChildren {
		if err := writeOptionalNewline(w, opts.Compact); err != nil {
			return err
		}

		childIndent := currentIndent + indentStep
		if err := writeMapChildren(w, m, keys, childIndent, indentStep, opts); err != nil {
			return err
		}

		if textContent != "" {
			if _, err := io.WriteString(w, escapeXML(textContent)); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintf(w, "%s</%s>", currentIndent, name); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(w, escapeXML(textContent)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "</%s>", name); err != nil {
			return err
		}
	}

	return writeOptionalNewline(w, opts.Compact)
}

// escapeXML escapes special characters in XML text content.
func escapeXML(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// escapeXMLAttr escapes special characters in XML attribute values.
func escapeXMLAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func collectMapElementParts(m map[string]any, sortKeys bool) (keys []string, attrs []string, textContent string, hasChildren bool) {
	keys = make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if sortKeys {
		sort.Strings(keys)
	}

	for _, k := range keys {
		v := m[k]
		switch {
		case strings.HasPrefix(k, "@"):
			attrName := strings.TrimPrefix(k, "@")
			attrs = append(attrs, fmt.Sprintf(` %s="%s"`, attrName, escapeXMLAttr(fmt.Sprintf("%v", v))))
		case k == "#text":
			textContent = fmt.Sprintf("%v", v)
		default:
			hasChildren = true
		}
	}
	return keys, attrs, textContent, hasChildren
}

func writeMapChildren(w io.Writer, m map[string]any, keys []string, childIndent, indentStep string, opts plugin.SerializeOptions) error {
	for _, k := range keys {
		if strings.HasPrefix(k, "@") || k == "#text" {
			continue
		}
		if err := serializeElement(w, k, m[k], childIndent, indentStep, opts); err != nil {
			return err
		}
	}
	return nil
}

func writeOptionalNewline(w io.Writer, compact bool) error {
	if compact {
		return nil
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// Merge merges two XML values according to the strategy.
func (f *Format) Merge(base, overlay any, strategy plugin.MergeStrategy) (any, error) {
	switch strategy {
	case plugin.MergeReplace:
		return overlay, nil

	case plugin.MergeDeep:
		return deepMerge(base, overlay), nil

	case plugin.MergeAppend:
		return appendMerge(base, overlay), nil

	case plugin.MergeConcat:
		return appendMerge(base, overlay), nil

	case plugin.MergeUnion:
		return unionMerge(base, overlay), nil

	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", strategy)
	}
}

// Query queries an XML value using XPath.
func (f *Format) Query(data any, path string) (any, error) {
	// Convert the data back to XML for XPath query
	xmlBytes, err := f.Serialize(data, plugin.SerializeOptions{Compact: true})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize for XPath query: %w", err)
	}

	doc, err := xmlquery.Parse(bytes.NewReader(xmlBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML for XPath: %w", err)
	}

	// Execute XPath query
	nodes, err := xmlquery.QueryAll(doc, path)
	if err != nil {
		return nil, fmt.Errorf("XPath query error: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("XPath query returned no results: %s", path)
	}

	// Convert results
	if len(nodes) == 1 {
		return nodeToValue(nodes[0]), nil
	}

	results := make([]any, len(nodes))
	for i, node := range nodes {
		results[i] = nodeToValue(node)
	}
	return results, nil
}

// nodeToValue converts an xmlquery.Node to a Go value.
func nodeToValue(node *xmlquery.Node) any {
	if node == nil {
		return nil
	}

	switch node.Type {
	case xmlquery.TextNode, xmlquery.CharDataNode:
		return strings.TrimSpace(node.Data)

	case xmlquery.AttributeNode:
		return node.InnerText()

	case xmlquery.ElementNode:
		val, err := elementToMap(node, 0)
		if err != nil {
			return nil
		}
		return val

	default:
		return node.InnerText()
	}
}

// Set sets a value at the specified XPath.
// Note: This is a simplified implementation that works with the map representation.
func (f *Format) Set(data any, path string, value any) (any, error) {
	// For simple paths like /root/element, convert to map path
	mapPath := xpathToMapPath(path)
	if mapPath == "" {
		return nil, fmt.Errorf("cannot set at XPath: %s (only simple element paths supported)", path)
	}

	return setMapPath(data, mapPath, value)
}

// Delete deletes a value at the specified XPath.
func (f *Format) Delete(data any, path string) (any, error) {
	// For simple paths, convert to map path
	mapPath := xpathToMapPath(path)
	if mapPath == "" {
		return nil, fmt.Errorf("cannot delete at XPath: %s (only simple element paths supported)", path)
	}

	return deleteMapPath(data, mapPath)
}

// xpathToMapPath converts simple XPath to a dot-separated map path.
func xpathToMapPath(xpath string) string {
	// Handle simple element paths: /root/child or //element
	xpath = strings.TrimPrefix(xpath, "//")
	xpath = strings.TrimPrefix(xpath, "/")

	// Check for complex XPath features
	if strings.ContainsAny(xpath, "[]@*()") {
		return ""
	}

	return strings.ReplaceAll(xpath, "/", ".")
}

// setMapPath sets a value at a dot-separated path.
func setMapPath(data any, path string, value any) (any, error) {
	if path == "" {
		return value, nil
	}

	result := deepCopy(data)
	if result == nil {
		result = make(map[string]any)
	}

	parts := strings.Split(path, ".")
	current := result

	for i, part := range parts[:len(parts)-1] {
		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				v[part] = make(map[string]any)
				next = v[part]
			}
			current = next
		default:
			return nil, fmt.Errorf("cannot traverse %T at %s (part %d)", current, part, i)
		}
	}

	// Set final value
	lastPart := parts[len(parts)-1]
	switch v := current.(type) {
	case map[string]any:
		v[lastPart] = value
	default:
		return nil, fmt.Errorf("cannot set in %T", current)
	}

	return result, nil
}

// deleteMapPath deletes a value at a dot-separated path.
func deleteMapPath(data any, path string) (any, error) {
	if path == "" {
		return nil, nil
	}

	result := deepCopy(data)
	if result == nil {
		return nil, nil
	}

	parts := strings.Split(path, ".")
	current := result

	for _, part := range parts[:len(parts)-1] {
		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return result, nil // Path doesn't exist
			}
			current = next
		default:
			return result, nil
		}
	}

	// Delete final key
	lastPart := parts[len(parts)-1]
	if m, ok := current.(map[string]any); ok {
		delete(m, lastPart)
	}

	return result, nil
}

// Validate validates XML data against a schema.
func (f *Format) Validate(data any, schema any) ([]plugin.ValidationError, error) {
	// XML Schema (XSD) validation would require a full XSD implementation
	// For now, we just validate that the data is well-formed XML
	xmlBytes, err := f.Serialize(data, plugin.SerializeOptions{Compact: true})
	if err != nil {
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("invalid XML structure: %v", err),
			},
		}, nil
	}

	// Try to parse to verify well-formedness
	_, err = xmlquery.Parse(bytes.NewReader(xmlBytes))
	if err != nil {
		return []plugin.ValidationError{
			{
				Path:    "",
				Message: fmt.Sprintf("XML parse error: %v", err),
			},
		}, nil
	}

	return nil, nil
}

// GetSchema returns the Terraform schema for XML-specific attributes.
func (f *Format) GetSchema() plugin.FormatSchema {
	return plugin.FormatSchema{
		Attributes: map[string]plugin.SchemaAttribute{
			"indent": {
				Type:        "number",
				Optional:    true,
				Default:     2,
				Description: "Indentation spaces",
			},
			"sort_keys": {
				Type:        "bool",
				Optional:    true,
				Description: "Sort element names alphabetically",
			},
			"compact": {
				Type:        "bool",
				Optional:    true,
				Description: "Output compact XML without whitespace",
			},
			"xpath": {
				Type:        "string",
				Optional:    true,
				Description: "XPath expression to query specific elements",
			},
		},
	}
}

// deepMerge performs a recursive deep merge of two values.
func deepMerge(base, overlay any) any {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	baseMap, baseIsMap := base.(map[string]any)
	overlayMap, overlayIsMap := overlay.(map[string]any)

	if baseIsMap && overlayIsMap {
		result := make(map[string]any)
		for k, v := range baseMap {
			result[k] = v
		}
		for k, v := range overlayMap {
			if existing, ok := result[k]; ok {
				result[k] = deepMerge(existing, v)
			} else {
				result[k] = v
			}
		}
		return result
	}

	return overlay
}

// appendMerge appends arrays and deep merges objects.
func appendMerge(base, overlay any) any {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	baseArr, baseIsArr := base.([]any)
	overlayArr, overlayIsArr := overlay.([]any)

	if baseIsArr && overlayIsArr {
		result := make([]any, len(baseArr)+len(overlayArr))
		copy(result, baseArr)
		copy(result[len(baseArr):], overlayArr)
		return result
	}

	return deepMerge(base, overlay)
}

// unionMerge creates a union of arrays and deep merges objects.
func unionMerge(base, overlay any) any {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	baseArr, baseIsArr := base.([]any)
	overlayArr, overlayIsArr := overlay.([]any)

	if baseIsArr && overlayIsArr {
		seen := make(map[string]bool)
		result := make([]any, 0)

		for _, v := range baseArr {
			key := fmt.Sprintf("%v", v)
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		}
		for _, v := range overlayArr {
			key := fmt.Sprintf("%v", v)
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		}
		return result
	}

	return deepMerge(base, overlay)
}

// deepCopy creates a deep copy of a value.
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

// ParseXPath is a helper function to execute an XPath query on XML data.
func ParseXPath(xmlData []byte, xpath string) ([]string, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	nodes, err := xmlquery.QueryAll(doc, xpath)
	if err != nil {
		return nil, fmt.Errorf("XPath query error: %w", err)
	}

	results := make([]string, len(nodes))
	for i, node := range nodes {
		results[i] = node.InnerText()
	}
	return results, nil
}

// XPathMatch checks if the given XML matches the XPath expression.
var xpathPatternRegex = regexp.MustCompile(`^[a-zA-Z0-9_/\[\]@='".\-*]+$`)

// IsValidXPath performs basic validation of an XPath expression.
func IsValidXPath(xpath string) bool {
	// Basic validation - real XPath can be quite complex
	if xpath == "" {
		return false
	}
	return xpathPatternRegex.MatchString(xpath)
}

// GetNodeText extracts text content from an XML node path.
func GetNodeText(xmlData []byte, xpath string) (string, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlData))
	if err != nil {
		return "", fmt.Errorf("failed to parse XML: %w", err)
	}

	node := xmlquery.FindOne(doc, xpath)
	if node == nil {
		return "", fmt.Errorf("node not found: %s", xpath)
	}

	return strings.TrimSpace(node.InnerText()), nil
}

// GetNodeAttr extracts an attribute value from an XML node.
func GetNodeAttr(xmlData []byte, xpath string, attrName string) (string, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlData))
	if err != nil {
		return "", fmt.Errorf("failed to parse XML: %w", err)
	}

	node := xmlquery.FindOne(doc, xpath)
	if node == nil {
		return "", fmt.Errorf("node not found: %s", xpath)
	}

	for _, attr := range node.Attr {
		if attr.Name.Local == attrName {
			return attr.Value, nil
		}
	}

	return "", fmt.Errorf("attribute not found: %s", attrName)
}

// CountNodes counts nodes matching an XPath expression.
func CountNodes(xmlData []byte, xpath string) (int, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlData))
	if err != nil {
		return 0, fmt.Errorf("failed to parse XML: %w", err)
	}

	nodes, err := xmlquery.QueryAll(doc, xpath)
	if err != nil {
		return 0, fmt.Errorf("XPath query error: %w", err)
	}

	return len(nodes), nil
}

// TransformXML applies a simple transformation based on XPath selectors and values.
type XMLTransform struct {
	XPath string
	Value string
	Op    string // "set", "delete", "append"
}

// ApplyTransforms applies a list of transformations to XML data.
func ApplyTransforms(xmlData []byte, transforms []XMLTransform) ([]byte, error) {
	f := New()

	// Parse the XML
	data, err := f.Parse(xmlData)
	if err != nil {
		return nil, err
	}

	// Apply each transform
	for _, t := range transforms {
		switch t.Op {
		case "set":
			data, err = f.Set(data, t.XPath, t.Value)
		case "delete":
			data, err = f.Delete(data, t.XPath)
		case "append":
			// Get existing value and append
			existing, queryErr := f.Query(data, t.XPath)
			if queryErr == nil {
				if existingStr, ok := existing.(string); ok {
					data, err = f.Set(data, t.XPath, existingStr+t.Value)
				}
			}
		default:
			err = fmt.Errorf("unknown transform operation: %s", t.Op)
		}

		if err != nil {
			return nil, fmt.Errorf("transform error at %s: %w", t.XPath, err)
		}
	}

	// Serialize back to XML
	return f.Serialize(data, plugin.SerializeOptions{Indent: 2})
}
