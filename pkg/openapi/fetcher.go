package openapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/openapi"
	cachedopenapi "k8s.io/client-go/openapi/cached"
)

type SchemaFetcher struct {
	client openapi.Client
}

func NewSchemaFetcher(client openapi.Client) *SchemaFetcher {
	return &SchemaFetcher{client: cachedopenapi.NewClient(client)}
}

// FetchForResource fetches only the group-version schema that contains the
// requested resource type. This is dramatically faster than FetchAll because
// it downloads a single group-version's schema (~50-200KB) instead of every
// schema on the cluster (~5MB across 30-50 HTTP requests).
//
// It works in two phases:
//  1. Call Paths() to enumerate all group-version endpoints (one HTTP call)
//  2. Fetch schemas concurrently, stopping as soon as the target resource is
//     found. The returned Document contains only that group-version's schemas,
//     which is sufficient because $ref resolution is self-contained within each
//     group-version document.
func (f *SchemaFetcher) FetchForResource(resourceType string) (*Document, error) {
	paths, err := f.client.Paths()
	if err != nil {
		return nil, fmt.Errorf("fetch OpenAPI paths: %w", err)
	}

	lower := strings.ToLower(resourceType)

	// Phase 1: try well-known resource-to-path mappings first (zero HTTP calls).
	if path, ok := wellKnownResourcePath(lower); ok {
		if gv, exists := paths[path]; exists {
			doc, err := fetchAndParsePath(gv, path)
			if err == nil {
				return doc, nil
			}
		}
	}

	// Phase 2: fetch all paths concurrently, find the one containing
	// the target resource. The cachedopenapi wrapper ensures we don't
	// re-download paths already fetched in phase 1.
	type result struct {
		doc *Document
	}

	pathList := make([]struct {
		key string
		gv  openapi.GroupVersion
	}, 0, len(paths))
	for key, gv := range paths {
		pathList = append(pathList, struct {
			key string
			gv  openapi.GroupVersion
		}{key, gv})
	}

	results := make(chan result, len(pathList))
	var wg sync.WaitGroup

	for _, p := range pathList {
		wg.Add(1)
		go func(key string, gv openapi.GroupVersion) {
			defer wg.Done()
			doc, err := fetchAndParsePath(gv, key)
			if err != nil {
				return
			}
			if doc.hasResourceKind(lower) {
				results <- result{doc: doc}
			}
		}(p.key, p.gv)
	}

	// Close results channel when all goroutines complete.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Return the first match.
	for r := range results {
		if r.doc != nil {
			return r.doc, nil
		}
	}

	return nil, fmt.Errorf("no schema found for resource type %q", resourceType)
}

func (f *SchemaFetcher) FetchSchema(gvk schema.GroupVersionKind) (*Document, map[string]any, error) {
	paths, err := f.client.Paths()
	if err != nil {
		return nil, nil, fmt.Errorf("fetch OpenAPI paths: %w", err)
	}

	resourcePath := resourcePathFromGV(gvk.GroupVersion())
	gv, ok := paths[resourcePath]
	if !ok {
		return nil, nil, fmt.Errorf("no OpenAPI schema for path %s", resourcePath)
	}

	data, err := gv.Schema(runtime.ContentTypeJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch schema for %s: %w", resourcePath, err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		return nil, nil, fmt.Errorf("parse schema for %s: %w", resourcePath, err)
	}

	resourceSchema, err := doc.SchemaForGVK(gvk.Group, gvk.Version, gvk.Kind)
	if err != nil {
		return nil, nil, err
	}

	return doc, resourceSchema, nil
}

// ListGVKs fetches all group-version schemas concurrently and extracts the
// GVK metadata from each. This is used by --list and requires visiting all
// schemas.
func (f *SchemaFetcher) ListGVKs() ([]GVK, error) {
	paths, err := f.client.Paths()
	if err != nil {
		return nil, fmt.Errorf("fetch OpenAPI paths: %w", err)
	}

	type gvkResult struct {
		gvks []GVK
	}

	results := make([]gvkResult, len(paths))
	var wg sync.WaitGroup

	i := 0
	for _, gv := range paths {
		idx := i
		i++
		wg.Add(1)
		go func(gv openapi.GroupVersion, idx int) {
			defer wg.Done()
			data, err := gv.Schema(runtime.ContentTypeJSON)
			if err != nil {
				return
			}
			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				return
			}
			components, _ := raw["components"].(map[string]any)
			if components == nil {
				return
			}
			schemas, _ := components["schemas"].(map[string]any)
			var gvks []GVK
			for _, s := range schemas {
				schemaMap, ok := s.(map[string]any)
				if !ok {
					continue
				}
				gvks = append(gvks, extractGVKs(schemaMap)...)
			}
			results[idx] = gvkResult{gvks: gvks}
		}(gv, idx)
	}
	wg.Wait()

	var allGVKs []GVK
	for _, r := range results {
		allGVKs = append(allGVKs, r.gvks...)
	}
	return allGVKs, nil
}

// FetchAll fetches all group-version schemas concurrently and merges them
// into a single Document. This is needed for --list and scaffold (multiple
// resource types).
func (f *SchemaFetcher) FetchAll() (*Document, error) {
	paths, err := f.client.Paths()
	if err != nil {
		return nil, fmt.Errorf("fetch OpenAPI paths: %w", err)
	}

	type schemaResult struct {
		schemas map[string]any
	}

	results := make([]schemaResult, len(paths))
	var wg sync.WaitGroup

	i := 0
	for _, gv := range paths {
		idx := i
		i++
		wg.Add(1)
		go func(gv openapi.GroupVersion, idx int) {
			defer wg.Done()
			data, err := gv.Schema(runtime.ContentTypeJSON)
			if err != nil {
				return
			}
			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				return
			}
			components, _ := raw["components"].(map[string]any)
			if components == nil {
				return
			}
			schemas, _ := components["schemas"].(map[string]any)
			if schemas != nil {
				results[idx] = schemaResult{schemas: schemas}
			}
		}(gv, idx)
	}
	wg.Wait()

	merged := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{},
		},
	}
	mergedSchemas := merged["components"].(map[string]any)["schemas"].(map[string]any)

	for _, r := range results {
		for k, v := range r.schemas {
			mergedSchemas[k] = v
		}
	}

	data, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshal merged schemas: %w", err)
	}
	return ParseDocument(data)
}

func (f *SchemaFetcher) ServedGroupVersions() (map[string][]string, error) {
	paths, err := f.client.Paths()
	if err != nil {
		return nil, fmt.Errorf("fetch OpenAPI paths: %w", err)
	}

	available := make(map[string][]string)
	for pathKey := range paths {
		group, version := parseAPIPath(pathKey)
		if version == "" {
			continue
		}
		available[group] = append(available[group], version)
	}
	return available, nil
}

// fetchAndParsePath fetches a single group-version schema and parses it.
func fetchAndParsePath(gv openapi.GroupVersion, pathKey string) (*Document, error) {
	data, err := gv.Schema(runtime.ContentTypeJSON)
	if err != nil {
		return nil, fmt.Errorf("fetch schema for %s: %w", pathKey, err)
	}
	return ParseDocument(data)
}

func parseAPIPath(path string) (group, version string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case len(parts) == 2 && parts[0] == "api":
		return "", parts[1]
	case len(parts) == 3 && parts[0] == "apis":
		return parts[1], parts[2]
	default:
		return "", ""
	}
}

func resourcePathFromGV(gv schema.GroupVersion) string {
	if len(gv.Group) == 0 {
		return fmt.Sprintf("api/%s", gv.Version)
	}
	return fmt.Sprintf("apis/%s/%s", gv.Group, gv.Version)
}

// wellKnownResourcePath returns the API path for common resource types,
// allowing us to skip the full schema scan for the most frequently used
// resources. This maps resource type names (lowercase, singular and plural,
// short names) to their canonical API paths.
func wellKnownResourcePath(resourceType string) (string, bool) {
	m := map[string]string{
		// core/v1
		"pod": "api/v1", "pods": "api/v1", "po": "api/v1",
		"service": "api/v1", "services": "api/v1", "svc": "api/v1",
		"configmap": "api/v1", "configmaps": "api/v1", "cm": "api/v1",
		"secret": "api/v1", "secrets": "api/v1",
		"namespace": "api/v1", "namespaces": "api/v1", "ns": "api/v1",
		"serviceaccount": "api/v1", "serviceaccounts": "api/v1", "sa": "api/v1",
		"persistentvolumeclaim": "api/v1", "persistentvolumeclaims": "api/v1", "pvc": "api/v1",
		"persistentvolume": "api/v1", "persistentvolumes": "api/v1", "pv": "api/v1",
		"resourcequota": "api/v1", "resourcequotas": "api/v1", "quota": "api/v1",
		"limitrange": "api/v1", "limitranges": "api/v1", "limits": "api/v1",
		// apps/v1
		"deployment": "apis/apps/v1", "deployments": "apis/apps/v1", "deploy": "apis/apps/v1",
		"statefulset": "apis/apps/v1", "statefulsets": "apis/apps/v1", "sts": "apis/apps/v1",
		"daemonset": "apis/apps/v1", "daemonsets": "apis/apps/v1", "ds": "apis/apps/v1",
		"replicaset": "apis/apps/v1", "replicasets": "apis/apps/v1",
		// batch/v1
		"job": "apis/batch/v1", "jobs": "apis/batch/v1",
		"cronjob": "apis/batch/v1", "cronjobs": "apis/batch/v1", "cj": "apis/batch/v1",
		// networking.k8s.io/v1
		"ingress": "apis/networking.k8s.io/v1", "ingresses": "apis/networking.k8s.io/v1", "ing": "apis/networking.k8s.io/v1",
		"networkpolicy": "apis/networking.k8s.io/v1", "networkpolicies": "apis/networking.k8s.io/v1", "netpol": "apis/networking.k8s.io/v1",
		"ingressclass": "apis/networking.k8s.io/v1", "ingressclasses": "apis/networking.k8s.io/v1",
		// autoscaling/v2
		"horizontalpodautoscaler": "apis/autoscaling/v2", "horizontalpodautoscalers": "apis/autoscaling/v2", "hpa": "apis/autoscaling/v2",
		// policy/v1
		"poddisruptionbudget": "apis/policy/v1", "poddisruptionbudgets": "apis/policy/v1", "pdb": "apis/policy/v1",
		// rbac.authorization.k8s.io/v1
		"role": "apis/rbac.authorization.k8s.io/v1", "roles": "apis/rbac.authorization.k8s.io/v1",
		"clusterrole": "apis/rbac.authorization.k8s.io/v1", "clusterroles": "apis/rbac.authorization.k8s.io/v1",
		"rolebinding": "apis/rbac.authorization.k8s.io/v1", "rolebindings": "apis/rbac.authorization.k8s.io/v1",
		"clusterrolebinding": "apis/rbac.authorization.k8s.io/v1", "clusterrolebindings": "apis/rbac.authorization.k8s.io/v1",
		// storage.k8s.io/v1
		"storageclass": "apis/storage.k8s.io/v1", "storageclasses": "apis/storage.k8s.io/v1", "sc": "apis/storage.k8s.io/v1",
		// scheduling.k8s.io/v1
		"priorityclass": "apis/scheduling.k8s.io/v1", "priorityclasses": "apis/scheduling.k8s.io/v1", "pc": "apis/scheduling.k8s.io/v1",
		// node.k8s.io/v1
		"runtimeclass": "apis/node.k8s.io/v1", "runtimeclasses": "apis/node.k8s.io/v1",
		// admissionregistration.k8s.io/v1
		"validatingwebhookconfiguration": "apis/admissionregistration.k8s.io/v1", "vwc": "apis/admissionregistration.k8s.io/v1",
		"mutatingwebhookconfiguration": "apis/admissionregistration.k8s.io/v1", "mwc": "apis/admissionregistration.k8s.io/v1",
		// apiextensions.k8s.io/v1
		"customresourcedefinition": "apis/apiextensions.k8s.io/v1", "customresourcedefinitions": "apis/apiextensions.k8s.io/v1",
		"crd": "apis/apiextensions.k8s.io/v1", "crds": "apis/apiextensions.k8s.io/v1",
	}
	path, ok := m[resourceType]
	return path, ok
}
