package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type orderedMap struct {
	keys   []string
	values map[string]any
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]any)}
}

func (m *orderedMap) set(key string, value any) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

func (m *orderedMap) toMap() map[string]any {
	result := make(map[string]any, len(m.keys))
	for _, k := range m.keys {
		result[k] = m.values[k]
	}
	return result
}

func jsonToYAML(data []byte) ([]byte, error) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writeYAML(&buf, obj, 0, false)
	return buf.Bytes(), nil
}

func jsonToAnnotatedYAML(data []byte, annotations map[string]FieldAnnotation) ([]byte, error) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writeAnnotatedYAML(&buf, obj, 0, false, nil, annotations)
	return buf.Bytes(), nil
}

func writeYAML(buf *bytes.Buffer, val any, indent int, inArray bool) {
	prefix := strings.Repeat("  ", indent)

	switch v := val.(type) {
	case map[string]any:
		keys := sortedMapKeys(v)
		prioritizeKeys(keys)
		for i, k := range keys {
			if i == 0 && inArray {
				fmt.Fprintf(buf, "%s:", k)
				writeMapValue(buf, v[k], indent)
			} else {
				fmt.Fprintf(buf, "%s%s:", prefix, k)
				writeMapValue(buf, v[k], indent)
			}
		}

	case []any:
		for _, item := range v {
			fmt.Fprintf(buf, "%s- ", prefix)
			switch elem := item.(type) {
			case map[string]any:
				writeYAML(buf, elem, indent+1, true)
			default:
				writeScalar(buf, elem)
				buf.WriteByte('\n')
			}
		}

	default:
		writeScalar(buf, v)
		buf.WriteByte('\n')
	}
}

func writeMapValue(buf *bytes.Buffer, val any, indent int) {
	switch v := val.(type) {
	case map[string]any:
		if len(v) == 0 {
			buf.WriteString(" {}\n")
		} else {
			buf.WriteByte('\n')
			writeYAML(buf, v, indent+1, false)
		}
	case []any:
		if len(v) == 0 {
			buf.WriteString(" []\n")
		} else {
			buf.WriteByte('\n')
			writeYAML(buf, v, indent+1, false)
		}
	default:
		buf.WriteByte(' ')
		writeScalar(buf, v)
		buf.WriteByte('\n')
	}
}

func writeScalar(buf *bytes.Buffer, val any) {
	switch v := val.(type) {
	case string:
		if needsQuoting(v) {
			fmt.Fprintf(buf, "%q", v)
		} else {
			buf.WriteString(v)
		}
	case float64:
		if v == float64(int64(v)) {
			fmt.Fprintf(buf, "%d", int64(v))
		} else {
			fmt.Fprintf(buf, "%g", v)
		}
	case bool:
		fmt.Fprintf(buf, "%t", v)
	case nil:
		buf.WriteString("null")
	default:
		fmt.Fprintf(buf, "%v", v)
	}
}

func needsQuoting(s string) bool {
	if s == "" || s == "true" || s == "false" || s == "null" || s == "yes" || s == "no" {
		return true
	}
	if strings.ContainsAny(s, ":{}[]|>&*!%#`@,?") {
		return true
	}
	if looksNumeric(s) {
		return true
	}
	return false
}

func looksNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	return (first >= '0' && first <= '9') || first == '-' || first == '+' || first == '.'
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeAnnotatedYAML(buf *bytes.Buffer, val any, indent int, inArray bool, path []string, annotations map[string]FieldAnnotation) {
	prefix := strings.Repeat("  ", indent)

	switch v := val.(type) {
	case map[string]any:
		keys := sortedMapKeys(v)
		prioritizeKeys(keys)
		for i, k := range keys {
			fieldPath := append(append([]string{}, path...), k)
			pathStr := strings.Join(fieldPath, ".")

			// Write annotation comment before the field.
			if ann, ok := annotations[pathStr]; ok {
				commentPrefix := prefix
				if i == 0 && inArray {
					commentPrefix = strings.Repeat("  ", indent-1) + "  "
				}
				writeAnnotationComment(buf, ann, commentPrefix)
			}

			if i == 0 && inArray {
				fmt.Fprintf(buf, "%s:", k)
				writeAnnotatedMapValue(buf, v[k], indent, fieldPath, annotations)
			} else {
				fmt.Fprintf(buf, "%s%s:", prefix, k)
				writeAnnotatedMapValue(buf, v[k], indent, fieldPath, annotations)
			}
		}

	case []any:
		for _, item := range v {
			fmt.Fprintf(buf, "%s- ", prefix)
			switch elem := item.(type) {
			case map[string]any:
				writeAnnotatedYAML(buf, elem, indent+1, true, path, annotations)
			default:
				writeScalar(buf, elem)
				buf.WriteByte('\n')
			}
		}

	default:
		writeScalar(buf, v)
		buf.WriteByte('\n')
	}
}

func writeAnnotatedMapValue(buf *bytes.Buffer, val any, indent int, path []string, annotations map[string]FieldAnnotation) {
	switch v := val.(type) {
	case map[string]any:
		if len(v) == 0 {
			buf.WriteString(" {}\n")
		} else {
			buf.WriteByte('\n')
			writeAnnotatedYAML(buf, v, indent+1, false, path, annotations)
		}
	case []any:
		if len(v) == 0 {
			buf.WriteString(" []\n")
		} else {
			buf.WriteByte('\n')
			writeAnnotatedYAML(buf, v, indent+1, false, path, annotations)
		}
	default:
		buf.WriteByte(' ')
		writeScalar(buf, v)
		buf.WriteByte('\n')
	}
}

func writeAnnotationComment(buf *bytes.Buffer, ann FieldAnnotation, prefix string) {
	if ann.Description != "" {
		fmt.Fprintf(buf, "%s# %s\n", prefix, ann.Description)
	}
	if len(ann.Enums) > 0 {
		fmt.Fprintf(buf, "%s# enum: [%s]\n", prefix, strings.Join(ann.Enums, ", "))
	}
}

func prioritizeKeys(keys []string) {
	priority := map[string]int{
		"apiVersion": 0,
		"kind":       1,
		"metadata":   2,
		"spec":       3,
		"data":       4,
		"type":       5,
		"name":       10,
		"labels":     11,
		"image":      12,
	}
	sort.SliceStable(keys, func(i, j int) bool {
		pi, oki := priority[keys[i]]
		pj, okj := priority[keys[j]]
		if oki && okj {
			return pi < pj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return false
	})
}
