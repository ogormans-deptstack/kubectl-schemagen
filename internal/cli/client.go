package cli

import (
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/openapi"
)

func LoadClusterDoc(kubeconfigPath string) (*openapi.Document, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}

	fetcher := openapi.NewSchemaFetcher(disc.OpenAPIV3())
	return fetcher.FetchAll()
}

// LoadResourceSchema fetches only the group-version schema containing the
// requested resource type. This is significantly faster than LoadClusterDoc
// because it avoids downloading schemas for all API groups.
func LoadResourceSchema(kubeconfigPath, resourceType string) (*openapi.Document, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}

	fetcher := openapi.NewSchemaFetcher(disc.OpenAPIV3())
	return fetcher.FetchForResource(resourceType)
}

func LoadAvailableAPIs(kubeconfigPath string) (map[string][]string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}

	fetcher := openapi.NewSchemaFetcher(disc.OpenAPIV3())
	return fetcher.ServedGroupVersions()
}
