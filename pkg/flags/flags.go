package flags

import (
	"sort"
	"strings"

	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/openapi"
)

type FlagDef struct {
	Name        string
	Description string
	Type        string
}

func ForResource(docRaw map[string]any, schema map[string]any) []FlagDef {
	props, err := openapi.SchemaProperties(docRaw, schema)
	if err != nil || props == nil {
		return nil
	}

	specSchema, ok := props["spec"].(map[string]any)
	if !ok {
		return nil
	}

	resolved, err := openapi.ResolveSchema(docRaw, specSchema)
	if err != nil {
		return nil
	}

	specProps, err := openapi.SchemaProperties(docRaw, resolved)
	if err != nil || specProps == nil {
		return nil
	}

	var defs []FlagDef
	for name, fieldSchema := range specProps {
		fieldMap, ok := fieldSchema.(map[string]any)
		if !ok {
			continue
		}
		resolvedField, err := openapi.ResolveSchema(docRaw, fieldMap)
		if err != nil {
			continue
		}
		fieldType := openapi.SchemaType(resolvedField)
		if !isFlaggable(fieldType) {
			continue
		}
		if isSkippedField(name) {
			continue
		}
		defs = append(defs, FlagDef{
			Name:        name,
			Description: trimDescription(resolvedField),
			Type:        fieldType,
		})
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

func isFlaggable(schemaType string) bool {
	return schemaType == "string" || schemaType == "integer" ||
		schemaType == "boolean" || schemaType == "number"
}

func isSkippedField(name string) bool {
	skipped := map[string]bool{
		"status":            true,
		"managedfields":     true,
		"resourceversion":   true,
		"uid":               true,
		"creationtimestamp": true,
		"deletiontimestamp": true,
		"generation":        true,
		"selflink":          true,
	}
	return skipped[strings.ToLower(name)]
}

func trimDescription(schema map[string]any) string {
	desc, _ := schema["description"].(string)
	if len(desc) > 80 {
		return desc[:77] + "..."
	}
	return desc
}
