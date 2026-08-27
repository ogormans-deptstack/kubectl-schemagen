package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/defaults"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/fuzzy"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/openapi"
)

// FieldAnnotation holds schema metadata for a single field, used by the
// annotated YAML emitter to render inline comments.
type FieldAnnotation struct {
	Description string
	Type        string
	Enums       []string
	Required    bool
}

type OpenAPIGenerator struct {
	doc         *openapi.Document
	gvkIndex    map[string]openapi.GVK
	overrides   map[string]string
	annotations map[string]FieldAnnotation // dot-path -> annotation
	annotate    bool                       // whether to collect annotations
	pathStack   []string                   // current field path during walk
	inVCT       bool
	isCRD       bool // true when generating for a CRD (group contains '.')
}

func NewOpenAPIGenerator(doc *openapi.Document) *OpenAPIGenerator {
	return &OpenAPIGenerator{
		doc:      doc,
		gvkIndex: buildGVKIndex(doc),
	}
}

func (g *OpenAPIGenerator) Generate(resourceType string, overrides map[string]string, w io.Writer) error {
	gvk, ok := g.resolveGVK(resourceType)
	if !ok {
		suggestions := fuzzy.Suggest(resourceType, g.SupportedTypes(), 3)
		if len(suggestions) > 0 {
			return fmt.Errorf("no example available for %q. Did you mean: %s? Try --list", resourceType, strings.Join(suggestions, ", "))
		}
		return fmt.Errorf("no example available for %q. Try --list", resourceType)
	}

	schema, err := g.doc.SchemaForGVK(gvk.Group, gvk.Version, gvk.Kind)
	if err != nil {
		return fmt.Errorf("schema lookup failed for %s: %w", gvk.Kind, err)
	}

	g.overrides = overrides
	g.isCRD = isCRDGroup(gvk.Group)
	manifest := g.buildManifest(gvk, schema)

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	yamlOut, err := jsonToYAML(data)
	if err != nil {
		return fmt.Errorf("convert to YAML: %w", err)
	}

	_, err = w.Write(yamlOut)
	return err
}

// GenerateJSON writes the manifest as indented JSON instead of YAML.
func (g *OpenAPIGenerator) GenerateJSON(resourceType string, overrides map[string]string, w io.Writer) error {
	gvk, ok := g.resolveGVK(resourceType)
	if !ok {
		suggestions := fuzzy.Suggest(resourceType, g.SupportedTypes(), 3)
		if len(suggestions) > 0 {
			return fmt.Errorf("no example available for %q. Did you mean: %s? Try --list", resourceType, strings.Join(suggestions, ", "))
		}
		return fmt.Errorf("no example available for %q. Try --list", resourceType)
	}

	schema, err := g.doc.SchemaForGVK(gvk.Group, gvk.Version, gvk.Kind)
	if err != nil {
		return fmt.Errorf("schema lookup failed for %s: %w", gvk.Kind, err)
	}

	g.overrides = overrides
	g.isCRD = isCRDGroup(gvk.Group)
	manifest := g.buildManifest(gvk, schema)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')

	_, err = w.Write(data)
	return err
}

// GenerateAnnotated writes the manifest as YAML with inline comments
// showing field descriptions, types, and enum values from the schema.
func (g *OpenAPIGenerator) GenerateAnnotated(resourceType string, overrides map[string]string, w io.Writer) error {
	gvk, ok := g.resolveGVK(resourceType)
	if !ok {
		suggestions := fuzzy.Suggest(resourceType, g.SupportedTypes(), 3)
		if len(suggestions) > 0 {
			return fmt.Errorf("no example available for %q. Did you mean: %s? Try --list", resourceType, strings.Join(suggestions, ", "))
		}
		return fmt.Errorf("no example available for %q. Try --list", resourceType)
	}

	schema, err := g.doc.SchemaForGVK(gvk.Group, gvk.Version, gvk.Kind)
	if err != nil {
		return fmt.Errorf("schema lookup failed for %s: %w", gvk.Kind, err)
	}

	g.overrides = overrides
	g.isCRD = isCRDGroup(gvk.Group)
	g.annotate = true
	g.annotations = make(map[string]FieldAnnotation)
	g.pathStack = nil
	manifest := g.buildManifest(gvk, schema)
	g.annotate = false

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	yamlOut, err := jsonToAnnotatedYAML(data, g.annotations)
	if err != nil {
		return fmt.Errorf("convert to annotated YAML: %w", err)
	}

	_, err = w.Write(yamlOut)
	return err
}

// Annotations returns the collected field annotations from the last
// GenerateAnnotated call. This is primarily useful for testing.
func (g *OpenAPIGenerator) Annotations() map[string]FieldAnnotation {
	return g.annotations
}

func (g *OpenAPIGenerator) SupportedTypes() []string {
	var types []string
	seen := make(map[string]bool)
	for _, gvk := range g.gvkIndex {
		if !seen[gvk.Kind] {
			types = append(types, gvk.Kind)
			seen[gvk.Kind] = true
		}
	}
	sort.Strings(types)
	return types
}

func (g *OpenAPIGenerator) SupportedTypesWithAliases() []string {
	kinds := g.SupportedTypes()
	result := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		aliases := aliasesForKind(kind)
		if len(aliases) == 0 {
			result = append(result, kind)
			continue
		}
		var shortNames []string
		kindLen := len(kind)
		for _, a := range aliases {
			if len(a) < kindLen {
				shortNames = append(shortNames, a)
			}
		}
		if len(shortNames) == 0 {
			result = append(result, kind)
			continue
		}
		result = append(result, kind+" ("+strings.Join(shortNames, ", ")+")")
	}
	return result
}

func (g *OpenAPIGenerator) buildManifest(gvk openapi.GVK, schema map[string]any) map[string]any {
	manifest := newOrderedMap()

	apiVersion := gvk.Version
	if gvk.Group != "" {
		apiVersion = gvk.Group + "/" + gvk.Version
	}
	manifest.set("apiVersion", apiVersion)
	manifest.set("kind", gvk.Kind)

	name := fmt.Sprintf("example-%s", strings.ToLower(gvk.Kind))
	if override, ok := g.overrides["name"]; ok {
		name = override
	}

	labels := map[string]any{"app.kubernetes.io/name": name}
	metadata := map[string]any{
		"name":   name,
		"labels": labels,
	}
	g.applyMetadataOverrides(metadata)
	manifest.set("metadata", metadata)

	props, err := openapi.SchemaProperties(g.doc.Raw(), schema)
	if err != nil || props == nil {
		return manifest.toMap()
	}

	specSchema, ok := props["spec"].(map[string]any)
	if !ok {
		g.addTopLevelFields(manifest, gvk, props)
		return manifest.toMap()
	}

	g.pushPath("spec")
	spec := g.walkSchema(specSchema, gvk.Kind, 0)
	g.popPath()
	if spec != nil {
		g.injectTemplateLabels(spec, name)
		g.injectTemplateRestartPolicy(spec, gvk.Kind)
		g.fixStrategyDefaults(spec, gvk.Kind)
		g.injectServiceSelector(spec, gvk.Kind, name)
		g.fixCRDDefaults(spec, manifest, gvk.Kind)
		g.fixPDBDefaults(spec, gvk.Kind)
		g.fixPVDefaults(spec, gvk.Kind)
		g.fixIngressClassDefaults(spec, gvk.Kind)
		g.fixIssuerDefaults(spec, gvk.Kind)
		g.fixArgoDefaults(spec, gvk.Kind)
		g.fixLimitRangeDefaults(spec, gvk.Kind)
		g.stripNoisyFields(spec, gvk.Kind)
		g.applyOverrides(spec, gvk.Kind)
		manifest.set("spec", spec)
	}

	return manifest.toMap()
}

func (g *OpenAPIGenerator) addTopLevelFields(manifest *orderedMap, gvk openapi.GVK, props map[string]any) {
	kind := gvk.Kind

	for fieldName, fieldSchema := range props {
		lower := strings.ToLower(fieldName)
		if lower == "apiversion" || lower == "kind" || lower == "metadata" {
			continue
		}

		if !defaults.IsImportantField(fieldName) {
			continue
		}

		if v := defaults.ValueForField(fieldName, "object", "", kind); v != nil {
			manifest.set(fieldName, v)
			continue
		}

		fieldMap, ok := fieldSchema.(map[string]any)
		if !ok {
			continue
		}
		resolved, err := openapi.ResolveSchema(g.doc.Raw(), fieldMap)
		if err != nil {
			continue
		}
		val := g.generateValue(fieldName, resolved, kind, 0)
		if val != nil {
			manifest.set(fieldName, val)
		}
	}

	if kind == "Secret" {
		if _, ok := props["type"]; ok {
			manifest.set("type", "Opaque")
		}
	}
}

func (g *OpenAPIGenerator) walkSchema(schema map[string]any, kind string, depth int) map[string]any {
	if depth > 6 {
		return nil
	}

	resolved, err := openapi.ResolveSchema(g.doc.Raw(), schema)
	if err != nil {
		return nil
	}

	props, err := openapi.SchemaProperties(g.doc.Raw(), resolved)
	if err != nil || props == nil {
		return nil
	}

	required := make(map[string]bool)
	for _, r := range openapi.RequiredFields(resolved) {
		required[r] = true
	}

	_, hasSiblingContainers := props["containers"]

	result := make(map[string]any)
	for fieldName, fieldSchema := range props {
		fieldMap, ok := fieldSchema.(map[string]any)
		if !ok {
			continue
		}
		if isExcludedField(fieldName, depth) {
			continue
		}
		if fieldName == "selector" && (kind == "Job" || kind == "CronJob" || kind == "PersistentVolumeClaim" || g.inVCT) {
			continue
		}
		if fieldName == "resources" && hasSiblingContainers {
			continue
		}

		resolvedField, err := openapi.ResolveSchema(g.doc.Raw(), fieldMap)
		if err != nil {
			continue
		}

		isRequired := required[fieldName]
		isImportant := defaults.IsImportantField(fieldName)
		fieldType := openapi.SchemaType(resolvedField)
		isPrimitive := fieldType == "string" || fieldType == "integer" || fieldType == "number" || fieldType == "boolean"

		if !isRequired && !isImportant {
			if depth > 1 {
				continue
			}
			if isPrimitive {
				continue
			}
		}

		if !isRequired && hasOneOf(fieldMap) {
			continue
		}

		val := g.generateValue(fieldName, resolvedField, kind, depth)
		if val != nil {
			result[fieldName] = val
			g.recordAnnotation(fieldName, resolvedField, isRequired)
		}
	}

	g.resolveDiscriminatedUnions(result, props, resolved, kind, depth)

	if len(result) == 0 {
		return nil
	}
	return result
}

func (g *OpenAPIGenerator) generateValue(fieldName string, schema map[string]any, kind string, depth int) any {
	schemaType := openapi.SchemaType(schema)
	format, _ := schema["format"].(string)

	if format == "int-or-string" {
		if v, ok := g.overrides[fieldName]; ok {
			return parseIntOrString(v)
		}
		if v, ok := defaults.FieldDefault(fieldName, kind); ok {
			return v
		}
		return 1
	}

	switch schemaType {
	case "object":
		if v := defaults.ValueForField(fieldName, "object", format, kind); v != nil {
			if m, ok := v.(map[string]any); ok {
				return m
			}
		}
		if isResourceQuantityMap(fieldName, schema) {
			return g.resourceQuantityDefaults(fieldName, kind)
		}
		if hasAdditionalProperties(schema) {
			m := g.mapDefault(fieldName, kind)
			if m == nil {
				return nil
			}
			return m
		}
		g.pushPath(fieldName)
		sub := g.walkSchema(schema, kind, depth+1)
		g.popPath()
		if sub == nil {
			return nil
		}
		return sub

	case "array":
		if v := defaults.ValueForField(fieldName, "array", format, kind); v != nil {
			return v
		}
		items := arrayItemSchema(schema)
		if items == nil {
			return nil
		}
		resolved, err := openapi.ResolveSchema(g.doc.Raw(), items)
		if err != nil {
			return nil
		}
		itemType := openapi.SchemaType(resolved)
		if itemType == "object" {
			isVCT := strings.ToLower(fieldName) == "volumeclaimtemplates"
			if isVCT {
				g.inVCT = true
			}
			g.pushPath(fieldName)
			elem := g.walkSchema(resolved, kind, depth+1)
			g.popPath()
			if isVCT {
				g.inVCT = false
			}
			if elem != nil {
				return []any{elem}
			}
			return nil
		}
		return nil

	case "string", "integer", "number", "boolean":
		// User overrides always win.
		if v, ok := g.overrides[fieldName]; ok {
			if schemaType == "integer" {
				return parseIntOrString(v)
			}
			return v
		}

		// For CRDs, schema-provided example/default takes priority
		// over hardcoded field defaults since CRD authors set these
		// intentionally for their specific types.
		if g.isCRD {
			if v, ok := schemaExample(schema); ok {
				return v
			}
			if v, ok := schemaDefault(schema); ok {
				return v
			}
		}

		if v, ok := defaults.FieldDefault(fieldName, kind); ok {
			return v
		}

		// For built-in types, schema defaults come after field defaults
		// since our hardcoded defaults produce better output.
		if !g.isCRD {
			if v, ok := schemaDefault(schema); ok {
				return v
			}
		}

		if schemaType == "string" {
			if pattern, ok := schema["pattern"].(string); ok {
				if v := generatePatternExample(pattern); v != "" {
					return v
				}
			}
		}
		if enums := schemaEnums(schema); len(enums) > 0 {
			return enums[0]
		}

		// For integers and numbers, respect min/max constraints.
		if schemaType == "integer" || schemaType == "number" {
			return constrainedNumericDefault(schema, schemaType, format)
		}
		return defaults.TypeDefault(schemaType, format)
	}
	return nil
}

func (g *OpenAPIGenerator) pushPath(segment string) {
	if g.annotate {
		g.pathStack = append(g.pathStack, segment)
	}
}

func (g *OpenAPIGenerator) popPath() {
	if g.annotate && len(g.pathStack) > 0 {
		g.pathStack = g.pathStack[:len(g.pathStack)-1]
	}
}

func (g *OpenAPIGenerator) currentPath(fieldName string) string {
	parts := make([]string, 0, len(g.pathStack)+1)
	parts = append(parts, g.pathStack...)
	parts = append(parts, fieldName)
	return strings.Join(parts, ".")
}

func (g *OpenAPIGenerator) recordAnnotation(fieldName string, schema map[string]any, isRequired bool) {
	if !g.annotate {
		return
	}
	path := g.currentPath(fieldName)
	desc, _ := schema["description"].(string)
	fieldType := openapi.SchemaType(schema)
	format, _ := schema["format"].(string)
	if format != "" {
		fieldType = fieldType + " (" + format + ")"
	}
	enums := schemaEnums(schema)

	// Only record if there's useful metadata.
	if desc == "" && len(enums) == 0 {
		return
	}

	// Clean description: collapse newlines and trim for single-line comments.
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.Join(strings.Fields(desc), " ")
	// Truncate long descriptions to keep comments readable.
	if len(desc) > 120 {
		desc = desc[:117] + "..."
	}

	g.annotations[path] = FieldAnnotation{
		Description: desc,
		Type:        fieldType,
		Enums:       enums,
		Required:    isRequired,
	}
}

func (g *OpenAPIGenerator) resourceQuantityDefaults(fieldName, kind string) map[string]any {
	if kind == "PersistentVolumeClaim" || g.inVCT {
		return map[string]any{"storage": "1Gi"}
	}
	lower := strings.ToLower(fieldName)
	switch lower {
	case "limits":
		return map[string]any{"cpu": "500m", "memory": "256Mi"}
	case "requests":
		return map[string]any{"cpu": "250m", "memory": "128Mi"}
	}
	return map[string]any{"cpu": "250m", "memory": "128Mi"}
}

func (g *OpenAPIGenerator) mapDefault(fieldName, kind string) map[string]any {
	if v := defaults.ValueForField(fieldName, "object", "", kind); v != nil {
		if m, ok := v.(map[string]string); ok {
			result := make(map[string]any, len(m))
			for k, v := range m {
				result[k] = v
			}
			return result
		}
	}
	return nil
}

func (g *OpenAPIGenerator) applyOverrides(spec map[string]any, kind string) {
	for k, v := range g.overrides {
		switch k {
		case "name":
			continue
		case "replicas":
			if _, hasReplicas := spec["replicas"]; hasReplicas {
				spec["replicas"] = parseIntOrString(v)
			}
		case "image":
			g.setContainerImage(spec, v)
		default:
			if strings.HasPrefix(k, "metadata.") {
				continue
			}
			if strings.Contains(k, ".") {
				g.setNestedField(spec, k, v)
			}
		}
	}
}

func (g *OpenAPIGenerator) applyMetadataOverrides(metadata map[string]any) {
	for k, v := range g.overrides {
		if !strings.HasPrefix(k, "metadata.") {
			continue
		}
		path := strings.TrimPrefix(k, "metadata.")
		parts := strings.Split(path, ".")
		current := metadata
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = parseIntOrString(v)
				break
			}
			next, ok := current[part].(map[string]any)
			if !ok {
				next = make(map[string]any)
				current[part] = next
			}
			current = next
		}
	}
}

func (g *OpenAPIGenerator) setNestedField(root map[string]any, path string, value string) {
	parts := strings.Split(path, ".")
	if len(parts) > 1 && parts[0] == "spec" {
		parts = parts[1:]
	}

	current := root
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = parseIntOrString(value)
			return
		}

		idx, fieldName, hasIndex := parseArrayIndex(part)
		if hasIndex {
			arr, ok := current[fieldName].([]any)
			if !ok || idx >= len(arr) {
				return
			}
			next, ok := arr[idx].(map[string]any)
			if !ok {
				return
			}
			current = next
		} else {
			next, ok := current[part].(map[string]any)
			if !ok {
				return
			}
			current = next
		}
	}
}

func parseArrayIndex(part string) (idx int, fieldName string, ok bool) {
	openBracket := strings.Index(part, "[")
	if openBracket < 0 {
		return 0, "", false
	}
	closeBracket := strings.Index(part, "]")
	if closeBracket < 0 {
		return 0, "", false
	}
	fieldName = part[:openBracket]
	idxStr := part[openBracket+1 : closeBracket]
	idx = 0
	for _, ch := range idxStr {
		if ch < '0' || ch > '9' {
			return 0, "", false
		}
		idx = idx*10 + int(ch-'0')
	}
	return idx, fieldName, true
}

func (g *OpenAPIGenerator) setContainerImage(spec map[string]any, image string) {
	if tmpl, ok := spec["template"].(map[string]any); ok {
		if tmplSpec, ok := tmpl["spec"].(map[string]any); ok {
			g.setContainerImage(tmplSpec, image)
			return
		}
	}
	if containers, ok := spec["containers"].([]any); ok && len(containers) > 0 {
		if c, ok := containers[0].(map[string]any); ok {
			c["image"] = image
		}
	}
	if jt, ok := spec["jobTemplate"].(map[string]any); ok {
		if jtSpec, ok := jt["spec"].(map[string]any); ok {
			if tmpl, ok := jtSpec["template"].(map[string]any); ok {
				if tmplSpec, ok := tmpl["spec"].(map[string]any); ok {
					g.setContainerImage(tmplSpec, image)
				}
			}
		}
	}
}

func (g *OpenAPIGenerator) injectTemplateLabels(spec map[string]any, name string) {
	label := map[string]any{"app.kubernetes.io/name": name}

	injectLabels := func(tmpl map[string]any) {
		md, ok := tmpl["metadata"].(map[string]any)
		if !ok {
			md = make(map[string]any)
			tmpl["metadata"] = md
		}
		md["labels"] = map[string]any{"app.kubernetes.io/name": name}
	}

	injectMatchLabels := func(parent map[string]any) {
		if sel, ok := parent["selector"].(map[string]any); ok {
			if _, ok := sel["matchLabels"]; ok {
				sel["matchLabels"] = label
			}
		}
	}

	if tmpl, ok := spec["template"].(map[string]any); ok {
		injectLabels(tmpl)
		injectMatchLabels(spec)
	}
	if jt, ok := spec["jobTemplate"].(map[string]any); ok {
		if jtSpec, ok := jt["spec"].(map[string]any); ok {
			if tmpl, ok := jtSpec["template"].(map[string]any); ok {
				injectLabels(tmpl)
				injectMatchLabels(jtSpec)
			}
		}
	}
}

func (g *OpenAPIGenerator) injectTemplateRestartPolicy(spec map[string]any, kind string) {
	if kind != "Job" && kind != "CronJob" {
		return
	}
	inject := func(podSpec map[string]any) {
		podSpec["restartPolicy"] = "Never"
	}

	if tmpl, ok := spec["template"].(map[string]any); ok {
		if tmplSpec, ok := tmpl["spec"].(map[string]any); ok {
			inject(tmplSpec)
		}
	}
	if jt, ok := spec["jobTemplate"].(map[string]any); ok {
		if jtSpec, ok := jt["spec"].(map[string]any); ok {
			if tmpl, ok := jtSpec["template"].(map[string]any); ok {
				if tmplSpec, ok := tmpl["spec"].(map[string]any); ok {
					inject(tmplSpec)
				}
			}
		}
	}
}

func (g *OpenAPIGenerator) fixStrategyDefaults(spec map[string]any, kind string) {
	if strategy, ok := spec["strategy"].(map[string]any); ok {
		strategy["type"] = "RollingUpdate"
	}
	if strategy, ok := spec["updateStrategy"].(map[string]any); ok {
		strategy["type"] = "RollingUpdate"
	}
}

func (g *OpenAPIGenerator) fixCRDDefaults(spec map[string]any, manifest *orderedMap, kind string) {
	if kind != "CustomResourceDefinition" {
		return
	}
	delete(spec, "conversion")

	names, ok := spec["names"].(map[string]any)
	if !ok {
		return
	}
	plural, _ := names["plural"].(string)
	group, _ := spec["group"].(string)
	if plural != "" && group != "" {
		meta, _ := manifest.values["metadata"].(map[string]any)
		if meta != nil {
			crdName := plural + "." + group
			meta["name"] = crdName
			meta["labels"] = map[string]string{"app.kubernetes.io/name": crdName}
		}
	}
}

func (g *OpenAPIGenerator) fixPDBDefaults(spec map[string]any, kind string) {
	if kind != "PodDisruptionBudget" {
		return
	}
	if _, hasMin := spec["minAvailable"]; hasMin {
		delete(spec, "maxUnavailable")
	}
}

func (g *OpenAPIGenerator) fixPVDefaults(spec map[string]any, kind string) {
	if kind != "PersistentVolume" {
		return
	}
	volumeTypes := []string{
		"awsElasticBlockStore", "azureDisk", "azureFile", "cephfs", "cinder",
		"csi", "fc", "flexVolume", "flocker", "gcePersistentDisk", "glusterfs",
		"iscsi", "local", "nfs", "photonPersistentDisk", "portworxVolume",
		"quobyte", "rbd", "scaleIO", "storageos", "vsphereVolume",
	}
	for _, vt := range volumeTypes {
		delete(spec, vt)
	}
}

func (g *OpenAPIGenerator) fixIngressClassDefaults(spec map[string]any, kind string) {
	if kind != "IngressClass" {
		return
	}
	delete(spec, "parameters")
}

func (g *OpenAPIGenerator) fixIssuerDefaults(spec map[string]any, kind string) {
	if kind != "Issuer" && kind != "ClusterIssuer" {
		return
	}
	// acme, ca, vault, venafi are mutually exclusive issuer types.
	// Keep only acme (most common) to avoid required-field validation
	// errors from the other types (e.g. vault.auth is required).
	delete(spec, "ca")
	delete(spec, "vault")
	delete(spec, "venafi")
}

func (g *OpenAPIGenerator) fixArgoDefaults(spec map[string]any, kind string) {
	argoKinds := map[string]bool{
		"Workflow": true, "CronWorkflow": true,
		"WorkflowTemplate": true, "ClusterWorkflowTemplate": true,
	}
	if !argoKinds[kind] {
		return
	}

	// CronWorkflow wraps the workflow spec inside spec.workflowSpec
	target := spec
	if kind == "CronWorkflow" {
		if _, ok := spec["schedules"]; !ok {
			spec["schedules"] = []any{"*/5 * * * *"}
		}
		if ws, ok := spec["workflowSpec"].(map[string]any); ok {
			target = ws
		}
	}

	fixArgoPromNames := func(m map[string]any) {
		metrics, ok := m["metrics"].(map[string]any)
		if !ok {
			return
		}
		prom, ok := metrics["prometheus"].([]any)
		if !ok {
			return
		}
		for _, entry := range prom {
			if p, ok := entry.(map[string]any); ok {
				if name, ok := p["name"].(string); ok {
					p["name"] = strings.ReplaceAll(name, "-", "_")
				}
			}
		}
	}

	stripArgoTemplateFields := func(tmpl map[string]any) {
		// data requires data.source, memoize requires memoize.cache
		delete(tmpl, "data")
		delete(tmpl, "memoize")
		// Argo templates allow at most one template type; keep container
		for _, exclusiveType := range []string{"containerSet", "dag", "http", "resource", "script", "steps", "suspend", "plugin"} {
			delete(tmpl, exclusiveType)
		}
		fixArgoPromNames(tmpl)
	}

	fixArgoPromNames(target)

	if templates, ok := target["templates"].([]any); ok {
		for _, t := range templates {
			if tmpl, ok := t.(map[string]any); ok {
				stripArgoTemplateFields(tmpl)
			}
		}
	}

	if td, ok := target["templateDefaults"].(map[string]any); ok {
		stripArgoTemplateFields(td)
	}
}

func (g *OpenAPIGenerator) fixLimitRangeDefaults(spec map[string]any, kind string) {
	if kind != "LimitRange" {
		return
	}
	limits, ok := spec["limits"].([]any)
	if !ok || len(limits) == 0 {
		return
	}
	entry, ok := limits[0].(map[string]any)
	if !ok {
		return
	}
	delete(entry, "maxLimitRequestRatio")
	entry["default"] = map[string]any{"cpu": "500m", "memory": "256Mi"}
	entry["defaultRequest"] = map[string]any{"cpu": "100m", "memory": "64Mi"}
	entry["min"] = map[string]any{"cpu": "50m", "memory": "32Mi"}
	entry["max"] = map[string]any{"cpu": "2", "memory": "1Gi"}
}

func (g *OpenAPIGenerator) stripNoisyFields(spec map[string]any, kind string) {
	noisy := []string{"tolerations", "topologySpreadConstraints", "overhead", "readinessGates"}
	stripFromPodSpec := func(podSpec map[string]any) {
		for _, field := range noisy {
			delete(podSpec, field)
		}
	}

	switch kind {
	case "Pod":
		stripFromPodSpec(spec)
	case "Deployment", "StatefulSet", "DaemonSet", "Job", "ReplicaSet":
		if tmpl, ok := spec["template"].(map[string]any); ok {
			if tmplSpec, ok := tmpl["spec"].(map[string]any); ok {
				stripFromPodSpec(tmplSpec)
			}
		}
	case "CronJob":
		if jt, ok := spec["jobTemplate"].(map[string]any); ok {
			if jtSpec, ok := jt["spec"].(map[string]any); ok {
				if tmpl, ok := jtSpec["template"].(map[string]any); ok {
					if tmplSpec, ok := tmpl["spec"].(map[string]any); ok {
						stripFromPodSpec(tmplSpec)
					}
				}
			}
		}
	}
}

func (g *OpenAPIGenerator) injectServiceSelector(spec map[string]any, kind, name string) {
	if kind != "Service" {
		return
	}
	if _, ok := spec["selector"]; ok {
		return
	}
	spec["selector"] = map[string]any{"app.kubernetes.io/name": name}
}

func (g *OpenAPIGenerator) resolveGVK(resourceType string) (openapi.GVK, bool) {
	lower := strings.ToLower(resourceType)
	if gvk, ok := g.gvkIndex[lower]; ok {
		return gvk, true
	}
	if gvk, ok := g.gvkIndex[singularize(lower)]; ok {
		return gvk, true
	}
	return openapi.GVK{}, false
}

func buildGVKIndex(doc *openapi.Document) map[string]openapi.GVK {
	idx := make(map[string]openapi.GVK)
	schemas := doc.ComponentSchemas()
	if schemas == nil {
		return idx
	}

	preferredVersions := map[string]string{
		"Deployment":              "apps/v1",
		"StatefulSet":             "apps/v1",
		"DaemonSet":               "apps/v1",
		"ReplicaSet":              "apps/v1",
		"Job":                     "batch/v1",
		"CronJob":                 "batch/v1",
		"Ingress":                 "networking.k8s.io/v1",
		"NetworkPolicy":           "networking.k8s.io/v1",
		"HorizontalPodAutoscaler": "autoscaling/v2",
	}

	for _, schema := range schemas {
		schemaMap, ok := schema.(map[string]any)
		if !ok {
			continue
		}
		ext, ok := schemaMap["x-kubernetes-group-version-kind"]
		if !ok {
			continue
		}
		arr, ok := ext.([]any)
		if !ok {
			continue
		}
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			gvk := openapi.GVK{
				Group:   stringVal(m, "group"),
				Version: stringVal(m, "version"),
				Kind:    stringVal(m, "kind"),
			}

			lower := strings.ToLower(gvk.Kind)
			existing, exists := idx[lower]
			if exists {
				pref, hasPref := preferredVersions[gvk.Kind]
				gv := gvk.Group + "/" + gvk.Version
				if gvk.Group == "" {
					gv = gvk.Version
				}
				existingGV := existing.Group + "/" + existing.Version
				if existing.Group == "" {
					existingGV = existing.Version
				}
				if hasPref && gv == pref {
					idx[lower] = gvk
				} else if !hasPref && !exists {
					idx[lower] = gvk
				}
				_ = existingGV
			} else {
				idx[lower] = gvk
			}

			for _, alias := range aliasesForKind(gvk.Kind) {
				if _, exists := idx[alias]; !exists {
					idx[alias] = gvk
				}
			}
		}
	}
	return idx
}

func aliasesForKind(kind string) []string {
	aliases := map[string][]string{
		"Pod":                            {"po", "pods"},
		"Deployment":                     {"deploy", "deployments"},
		"Service":                        {"svc", "services"},
		"ConfigMap":                      {"cm", "configmaps"},
		"Secret":                         {"secrets"},
		"Job":                            {"jobs"},
		"CronJob":                        {"cj", "cronjobs"},
		"Ingress":                        {"ing", "ingresses"},
		"NetworkPolicy":                  {"netpol", "networkpolicies"},
		"StatefulSet":                    {"sts", "statefulsets"},
		"DaemonSet":                      {"ds", "daemonsets"},
		"PersistentVolumeClaim":          {"pvc", "persistentvolumeclaims"},
		"HorizontalPodAutoscaler":        {"hpa", "horizontalpodautoscalers"},
		"ServiceAccount":                 {"sa", "serviceaccounts"},
		"Namespace":                      {"ns", "namespaces"},
		"PodDisruptionBudget":            {"pdb", "poddisruptionbudgets"},
		"ResourceQuota":                  {"quota", "resourcequotas"},
		"LimitRange":                     {"limits", "limitranges"},
		"PersistentVolume":               {"pv", "persistentvolumes"},
		"IngressClass":                   {"ingressclasses"},
		"StorageClass":                   {"sc", "storageclasses"},
		"PriorityClass":                  {"pc", "priorityclasses"},
		"Role":                           {"roles"},
		"ClusterRole":                    {"clusterroles"},
		"RoleBinding":                    {"rolebindings"},
		"ClusterRoleBinding":             {"clusterrolebindings"},
		"ValidatingWebhookConfiguration": {"vwc"},
		"MutatingWebhookConfiguration":   {"mwc"},
		"CustomResourceDefinition":       {"crd", "crds", "customresourcedefinitions"},
		"RuntimeClass":                   {"runtimeclasses"},
	}
	return aliases[kind]
}

func singularize(s string) string {
	irregulars := map[string]string{
		"ingresses":              "ingress",
		"networkpolicies":        "networkpolicy",
		"persistentvolumeclaims": "persistentvolumeclaim",
		"storageclasses":         "storageclass",
		"ingressclasses":         "ingressclass",
		"priorityclasses":        "priorityclass",
		"runtimeclasses":         "runtimeclass",
		"limitranges":            "limitrange",
	}
	if v, ok := irregulars[s]; ok {
		return v
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

func isExcludedField(name string, depth int) bool {
	lower := strings.ToLower(name)

	if depth == 0 {
		topLevel := map[string]bool{
			"apiversion": true,
			"kind":       true,
			"metadata":   true,
		}
		if topLevel[lower] {
			return true
		}
	}

	if depth >= 1 {
		deepOnly := map[string]bool{
			"restartpolicy": true,
		}
		if deepOnly[lower] {
			return true
		}
	}

	excluded := map[string]bool{
		"status":            true,
		"managedfields":     true,
		"resourceversion":   true,
		"uid":               true,
		"creationtimestamp": true,
		"deletiontimestamp": true,
		"generation":        true,
		"selflink":          true,
		"finalizers":        true,
		"ownerreferences":   true,
		"clustername":       true,

		"initcontainers":             true,
		"ephemeralcontainers":        true,
		"volumes":                    true,
		"volumemounts":               true,
		"volumedevices":              true,
		"matchexpressions":           true,
		"podfailurepolicy":           true,
		"datasource":                 true,
		"datasourceref":              true,
		"hostaliases":                true,
		"topologyspreadconstraints":  true,
		"overhead":                   true,
		"readinessgates":             true,
		"schedulinggates":            true,
		"resourceclaims":             true,
		"defaultbackend":             true,
		"lifecycle":                  true,
		"livenessprobe":              true,
		"readinessprobe":             true,
		"startupprobe":               true,
		"resizepolicy":               true,
		"env":                        true,
		"securitycontext":            true,
		"os":                         true,
		"affinity":                   true,
		"dnsconfig":                  true,
		"podreplacementpolicy":       true,
		"successfuljobshistorylimit": true,
		"failedjobshistorylimit":     true,
		"concurrencypolicy":          true,
		"suspend":                    true,
		"startingdeadlineseconds":    true,
		"command":                    true,
		"args":                       true,
		"workingdir":                 true,
		"restartpolicyrules":         true,
		"workloadref":                true,
		"podgroup":                   true,
	}
	return excluded[lower]
}

func isResourceQuantityMap(fieldName string, schema map[string]any) bool {
	lower := strings.ToLower(fieldName)
	if lower == "limits" || lower == "requests" {
		return true
	}
	ap, ok := schema["additionalProperties"].(map[string]any)
	if !ok {
		return false
	}
	ref, _ := ap["$ref"].(string)
	return strings.Contains(ref, "Quantity")
}

func hasAdditionalProperties(schema map[string]any) bool {
	_, ok := schema["additionalProperties"]
	return ok
}

func arrayItemSchema(schema map[string]any) map[string]any {
	items, ok := schema["items"].(map[string]any)
	if ok {
		return items
	}
	return nil
}

func stringVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func parseIntOrString(s string) any {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
		return i
	}
	return s
}

func schemaEnums(schema map[string]any) []string {
	enums, ok := schema["enum"].([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, e := range enums {
		if s, ok := e.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func generatePatternExample(pattern string) string {
	if strings.Contains(pattern, "\\/") || strings.Contains(pattern, "/") {
		return "example.com/example"
	}
	return ""
}

func schemaDefault(schema map[string]any) (any, bool) {
	val, ok := schema["default"]
	return val, ok
}

func schemaExample(schema map[string]any) (any, bool) {
	val, ok := schema["example"]
	return val, ok
}

// constrainedNumericDefault returns an integer or number value that respects
// the schema's minimum, maximum, exclusiveMinimum, and multipleOf constraints.
func constrainedNumericDefault(schema map[string]any, schemaType, format string) any {
	val := 1
	if min, ok := toFloat64(schema["minimum"]); ok {
		if val < int(min) {
			val = int(min)
		}
	}
	if exMin, ok := toFloat64(schema["exclusiveMinimum"]); ok {
		if val <= int(exMin) {
			val = int(exMin) + 1
		}
	}
	if max, ok := toFloat64(schema["maximum"]); ok {
		if val > int(max) {
			val = int(max)
		}
	}
	if mult, ok := toFloat64(schema["multipleOf"]); ok && mult > 0 {
		m := int(mult)
		if val%m != 0 {
			val = ((val / m) + 1) * m
		}
	}
	if schemaType == "number" {
		return float64(val)
	}
	return val
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func hasOneOf(schema map[string]any) bool {
	if _, ok := schema["oneOf"]; ok {
		return true
	}
	if items, ok := schema["items"].(map[string]any); ok {
		if _, ok := items["oneOf"]; ok {
			return true
		}
	}
	return false
}

func (g *OpenAPIGenerator) resolveDiscriminatedUnions(result map[string]any, props map[string]any, parentSchema map[string]any, kind string, depth int) {
	typeVal, ok := result["type"].(string)
	if !ok {
		return
	}

	typeRequired := false
	for _, r := range openapi.RequiredFields(parentSchema) {
		if r == "type" {
			typeRequired = true
			break
		}
	}
	if !typeRequired {
		return
	}

	siblingName := lowerFirst(typeVal)
	if _, alreadyPresent := result[siblingName]; alreadyPresent {
		return
	}

	siblingSchema, ok := props[siblingName].(map[string]any)
	if !ok {
		return
	}

	resolved, err := openapi.ResolveSchema(g.doc.Raw(), siblingSchema)
	if err != nil {
		return
	}

	siblingType := openapi.SchemaType(resolved)
	switch siblingType {
	case "object":
		val := g.generateValue(siblingName, resolved, kind, depth)
		if val != nil {
			result[siblingName] = val
			return
		}
		result[siblingName] = map[string]any{}
	case "string":
		val := g.generateValue(siblingName, resolved, kind, depth)
		if val != nil {
			result[siblingName] = val
		}
	}
}

// isCRDGroup returns true if the API group belongs to a custom resource
// definition (not a built-in Kubernetes API group). Built-in groups either
// have no dots (core ""), end with ".k8s.io", or are well-known system groups.
func isCRDGroup(group string) bool {
	if group == "" {
		return false
	}
	if !strings.Contains(group, ".") {
		return false
	}
	// Built-in K8s API groups end with .k8s.io
	if strings.HasSuffix(group, ".k8s.io") {
		return false
	}
	return true
}

func lowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
