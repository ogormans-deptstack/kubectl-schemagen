package openapi

import (
	"testing"
)

func TestResolveRef(t *testing.T) {
	doc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"io.k8s.api.core.v1.Pod": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"metadata": map[string]any{"type": "object"},
					},
				},
			},
		},
	}

	t.Run("resolves valid ref", func(t *testing.T) {
		result, err := ResolveRef(doc, "#/components/schemas/io.k8s.api.core.v1.Pod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["type"] != "object" {
			t.Errorf("expected type=object, got %v", result["type"])
		}
	})

	t.Run("rejects non-local ref", func(t *testing.T) {
		_, err := ResolveRef(doc, "http://example.com/schema")
		if err == nil {
			t.Fatal("expected error for non-local ref")
		}
	})

	t.Run("returns error for missing key", func(t *testing.T) {
		_, err := ResolveRef(doc, "#/components/schemas/nonexistent")
		if err == nil {
			t.Fatal("expected error for missing key")
		}
	})
}

func TestSchemaForGVK(t *testing.T) {
	docJSON := `{
		"components": {
			"schemas": {
				"io.k8s.api.core.v1.Pod": {
					"type": "object",
					"x-kubernetes-group-version-kind": [
						{"group": "", "version": "v1", "kind": "Pod"}
					],
					"properties": {
						"spec": {"type": "object"}
					}
				},
				"io.k8s.api.apps.v1.Deployment": {
					"type": "object",
					"x-kubernetes-group-version-kind": [
						{"group": "apps", "version": "v1", "kind": "Deployment"}
					],
					"properties": {
						"spec": {"type": "object"}
					}
				}
			}
		}
	}`

	doc, err := ParseDocument([]byte(docJSON))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	t.Run("finds Pod schema", func(t *testing.T) {
		schema, err := doc.SchemaForGVK("", "v1", "Pod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema["type"] != "object" {
			t.Error("expected type=object")
		}
	})

	t.Run("finds Deployment schema", func(t *testing.T) {
		schema, err := doc.SchemaForGVK("apps", "v1", "Deployment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema["type"] != "object" {
			t.Error("expected type=object")
		}
	})

	t.Run("returns error for unknown GVK", func(t *testing.T) {
		_, err := doc.SchemaForGVK("apps", "v1", "Nonexistent")
		if err == nil {
			t.Fatal("expected error for unknown GVK")
		}
	})
}

func TestSchemaProperties(t *testing.T) {
	t.Run("extracts direct properties", func(t *testing.T) {
		doc := map[string]any{}
		schema := map[string]any{
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "integer"},
			},
		}
		props, err := SchemaProperties(doc, schema)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(props) != 2 {
			t.Errorf("expected 2 properties, got %d", len(props))
		}
	})

	t.Run("merges allOf properties", func(t *testing.T) {
		doc := map[string]any{
			"components": map[string]any{
				"schemas": map[string]any{
					"Base": map[string]any{
						"properties": map[string]any{
							"metadata": map[string]any{"type": "object"},
						},
					},
				},
			},
		}
		schema := map[string]any{
			"allOf": []any{
				map[string]any{"$ref": "#/components/schemas/Base"},
				map[string]any{
					"properties": map[string]any{
						"spec": map[string]any{"type": "object"},
					},
				},
			},
		}
		props, err := SchemaProperties(doc, schema)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(props) != 2 {
			t.Errorf("expected 2 merged properties, got %d", len(props))
		}
	})
}

func TestRequiredFields(t *testing.T) {
	schema := map[string]any{
		"required": []any{"apiVersion", "kind", "metadata"},
	}
	req := RequiredFields(schema)
	if len(req) != 3 {
		t.Errorf("expected 3 required fields, got %d", len(req))
	}
}

func TestSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]any
		expected string
	}{
		{"explicit string", map[string]any{"type": "string"}, "string"},
		{"explicit integer", map[string]any{"type": "integer"}, "integer"},
		{"inferred object from properties", map[string]any{"properties": map[string]any{}}, "object"},
		{"inferred object from ref", map[string]any{"$ref": "#/foo"}, "object"},
		{"unknown", map[string]any{}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SchemaType(tt.schema)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestHasResourceKind(t *testing.T) {
	docJSON := `{
		"components": {
			"schemas": {
				"io.k8s.api.core.v1.Pod": {
					"type": "object",
					"x-kubernetes-group-version-kind": [
						{"group": "", "version": "v1", "kind": "Pod"}
					]
				},
				"io.k8s.api.apps.v1.Deployment": {
					"type": "object",
					"x-kubernetes-group-version-kind": [
						{"group": "apps", "version": "v1", "kind": "Deployment"}
					]
				}
			}
		}
	}`

	doc, err := ParseDocument([]byte(docJSON))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tests := []struct {
		kind     string
		expected bool
	}{
		{"pod", true},
		{"deployment", true},
		{"Pod", false}, // hasResourceKind expects lowercase input
		{"service", false},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got := doc.hasResourceKind(tt.kind)
			if got != tt.expected {
				t.Errorf("hasResourceKind(%q) = %v, want %v", tt.kind, got, tt.expected)
			}
		})
	}
}

func TestParseDocument(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		doc, err := ParseDocument([]byte(`{"openapi": "3.0.0"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if doc.Raw()["openapi"] != "3.0.0" {
			t.Error("expected openapi 3.0.0")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ParseDocument([]byte(`not json`))
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}
