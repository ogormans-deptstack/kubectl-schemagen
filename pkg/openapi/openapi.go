package openapi

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Document struct {
	raw map[string]any
}

func ParseDocument(data []byte) (*Document, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse OpenAPI document: %w", err)
	}
	return &Document{raw: raw}, nil
}

func (d *Document) Raw() map[string]any {
	return d.raw
}

func (d *Document) ComponentSchemas() map[string]any {
	components, ok := d.raw["components"].(map[string]any)
	if !ok {
		return nil
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return nil
	}
	return schemas
}

// hasResourceKind checks whether this document contains a schema with the
// given kind name (case-insensitive). This is used for targeted schema
// fetching to quickly identify which group-version document contains a
// requested resource type without building a full GVK index.
func (d *Document) hasResourceKind(lowerKind string) bool {
	schemas := d.ComponentSchemas()
	if schemas == nil {
		return false
	}
	for _, schema := range schemas {
		schemaMap, ok := schema.(map[string]any)
		if !ok {
			continue
		}
		for _, gvk := range extractGVKs(schemaMap) {
			if strings.ToLower(gvk.Kind) == lowerKind {
				return true
			}
		}
	}
	return false
}

func (d *Document) SchemaForGVK(group, version, kind string) (map[string]any, error) {
	schemas := d.ComponentSchemas()
	if schemas == nil {
		return nil, fmt.Errorf("no component schemas in document")
	}

	for _, schema := range schemas {
		schemaMap, ok := schema.(map[string]any)
		if !ok {
			continue
		}
		gvks := extractGVKs(schemaMap)
		for _, gvk := range gvks {
			if gvk.Group == group && gvk.Version == version && gvk.Kind == kind {
				return schemaMap, nil
			}
		}
	}
	return nil, fmt.Errorf("schema not found for %s/%s %s", group, version, kind)
}

type GVK struct {
	Group   string
	Version string
	Kind    string
}

func extractGVKs(schema map[string]any) []GVK {
	ext, ok := schema["x-kubernetes-group-version-kind"]
	if !ok {
		return nil
	}
	arr, ok := ext.([]any)
	if !ok {
		return nil
	}
	var gvks []GVK
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		gvks = append(gvks, GVK{
			Group:   stringVal(m, "group"),
			Version: stringVal(m, "version"),
			Kind:    stringVal(m, "kind"),
		})
	}
	return gvks
}

func stringVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func ResolveRef(doc map[string]any, ref string) (map[string]any, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, fmt.Errorf("unsupported ref format: %s", ref)
	}
	path := strings.Split(ref[2:], "/")
	var current any = doc
	for _, segment := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ref path %q: expected object at %s", ref, segment)
		}
		current, ok = m[segment]
		if !ok {
			return nil, fmt.Errorf("ref path %q: key %s not found", ref, segment)
		}
	}
	result, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ref %q resolved to non-object", ref)
	}
	return result, nil
}

func ResolveSchema(doc map[string]any, schema map[string]any) (map[string]any, error) {
	ref, ok := schema["$ref"].(string)
	if ok {
		return ResolveRef(doc, ref)
	}

	if allOf, ok := schema["allOf"].([]any); ok && len(allOf) == 1 {
		if itemMap, ok := allOf[0].(map[string]any); ok {
			if ref, ok := itemMap["$ref"].(string); ok {
				return ResolveRef(doc, ref)
			}
		}
	}

	return schema, nil
}

func SchemaProperties(doc map[string]any, schema map[string]any) (map[string]any, error) {
	resolved, err := ResolveSchema(doc, schema)
	if err != nil {
		return nil, err
	}

	if props, ok := resolved["properties"].(map[string]any); ok {
		return props, nil
	}

	if allOf, ok := resolved["allOf"].([]any); ok {
		merged := make(map[string]any)
		for _, item := range allOf {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			sub, err := ResolveSchema(doc, itemMap)
			if err != nil {
				continue
			}
			if props, ok := sub["properties"].(map[string]any); ok {
				for k, v := range props {
					merged[k] = v
				}
			}
		}
		if len(merged) > 0 {
			return merged, nil
		}
	}

	return nil, nil
}

func RequiredFields(schema map[string]any) []string {
	req, ok := schema["required"].([]any)
	if !ok {
		return nil
	}
	var fields []string
	for _, r := range req {
		if s, ok := r.(string); ok {
			fields = append(fields, s)
		}
	}
	return fields
}

func SchemaType(schema map[string]any) string {
	if t, ok := schema["type"].(string); ok {
		return t
	}
	if _, ok := schema["properties"]; ok {
		return "object"
	}
	if _, ok := schema["$ref"]; ok {
		return "object"
	}
	return "unknown"
}
