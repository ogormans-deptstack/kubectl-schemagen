package flags

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/openapi"
)

func TestForResource(t *testing.T) {
	doc := loadMergedFixture(t)
	raw := doc.Raw()

	t.Run("Deployment has replicas flag", func(t *testing.T) {
		schema := schemaForKind(t, doc, "apps", "v1", "Deployment")
		defs := ForResource(raw, schema)
		flag := findFlag(defs, "replicas")
		if flag == nil {
			t.Fatal("expected replicas flag for Deployment")
		}
		if flag.Type != "integer" {
			t.Errorf("expected replicas type=integer, got %s", flag.Type)
		}
	})

	t.Run("Deployment has paused flag", func(t *testing.T) {
		schema := schemaForKind(t, doc, "apps", "v1", "Deployment")
		defs := ForResource(raw, schema)
		flag := findFlag(defs, "paused")
		if flag == nil {
			t.Fatal("expected paused flag for Deployment")
		}
		if flag.Type != "boolean" {
			t.Errorf("expected paused type=boolean, got %s", flag.Type)
		}
	})

	t.Run("CronJob has schedule flag", func(t *testing.T) {
		schema := schemaForKind(t, doc, "batch", "v1", "CronJob")
		defs := ForResource(raw, schema)
		flag := findFlag(defs, "schedule")
		if flag == nil {
			t.Fatal("expected schedule flag for CronJob")
		}
		if flag.Type != "string" {
			t.Errorf("expected schedule type=string, got %s", flag.Type)
		}
	})

	t.Run("ConfigMap has no spec flags", func(t *testing.T) {
		schema := schemaForKind(t, doc, "", "v1", "ConfigMap")
		defs := ForResource(raw, schema)
		if len(defs) != 0 {
			names := make([]string, len(defs))
			for i, d := range defs {
				names[i] = d.Name
			}
			t.Errorf("expected no flags for ConfigMap (no spec), got %d: %v", len(defs), names)
		}
	})

	t.Run("flags are sorted by name", func(t *testing.T) {
		schema := schemaForKind(t, doc, "apps", "v1", "Deployment")
		defs := ForResource(raw, schema)
		for i := 1; i < len(defs); i++ {
			if defs[i].Name < defs[i-1].Name {
				t.Errorf("flags not sorted: %s < %s", defs[i].Name, defs[i-1].Name)
			}
		}
	})
}

func findFlag(defs []FlagDef, name string) *FlagDef {
	for i := range defs {
		if defs[i].Name == name {
			return &defs[i]
		}
	}
	return nil
}

func schemaForKind(t *testing.T, doc *openapi.Document, group, version, kind string) map[string]any {
	t.Helper()
	schema, err := doc.SchemaForGVK(group, version, kind)
	if err != nil {
		t.Fatalf("schema for %s/%s %s: %v", group, version, kind, err)
	}
	return schema
}

func loadMergedFixture(t *testing.T) *openapi.Document {
	t.Helper()
	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "openapi")
	files, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Skipf("fixtures not found at %s: %v", fixtureDir, err)
		return nil
	}

	merged := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{},
		},
	}
	mergedSchemas := merged["components"].(map[string]any)["schemas"].(map[string]any)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixtureDir, f.Name()))
		if err != nil {
			t.Fatalf("read fixture %s: %v", f.Name(), err)
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("parse fixture %s: %v", f.Name(), err)
		}
		components, _ := raw["components"].(map[string]any)
		if components == nil {
			continue
		}
		schemas, _ := components["schemas"].(map[string]any)
		for k, v := range schemas {
			mergedSchemas[k] = v
		}
	}

	data, err := json.Marshal(merged)
	if err != nil {
		t.Fatalf("marshal merged: %v", err)
	}
	doc, err := openapi.ParseDocument(data)
	if err != nil {
		t.Fatalf("parse merged: %v", err)
	}
	return doc
}
