package generator

import "io"

type ResourceGenerator interface {
	Generate(resourceType string, overrides map[string]string, w io.Writer) error
	GenerateJSON(resourceType string, overrides map[string]string, w io.Writer) error
	GenerateAnnotated(resourceType string, overrides map[string]string, w io.Writer) error
	SupportedTypes() []string
}
