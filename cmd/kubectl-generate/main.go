package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/generator"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/openapi"
)

var version = "dev"

type options struct {
	list        bool
	name        string
	image       string
	replicas    int
	replicasSet bool
	set         []string
	kubeconfig  string
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "kubectl-generate RESOURCE_TYPE",
		Short: "Generate example YAML manifests from the OpenAPI spec",
		Long: `Generates example Kubernetes resource YAML from the cluster's OpenAPI v3 spec.
The generated manifest includes sensible defaults and can be piped directly to kubectl create.`,
		Example: `  kubectl-generate pod
  kubectl-generate deployment --name=web --replicas=3 | kubectl create -f -
  kubectl-generate --list`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.replicasSet = cmd.Flags().Changed("replicas")
			return runGenerate(args, &opts)
		},
	}

	cmd.Flags().BoolVar(&opts.list, "list", false, "List all supported resource types")
	cmd.Flags().StringVar(&opts.name, "name", "", "Resource name")
	cmd.Flags().StringVar(&opts.image, "image", "", "Container image")
	cmd.Flags().IntVar(&opts.replicas, "replicas", 0, "Replica count")
	cmd.Flags().StringArrayVar(&opts.set, "set", nil, "Field override (key=value)")
	cmd.Flags().StringVar(&opts.kubeconfig, "kubeconfig", "", "Path to kubeconfig")

	return cmd
}

func runGenerate(args []string, opts *options) error {
	// --list requires all schemas to show every available type.
	if opts.list {
		doc, err := loadClusterDoc(opts.kubeconfig)
		if err != nil {
			return err
		}
		gen := generator.NewOpenAPIGenerator(doc)
		for _, t := range gen.SupportedTypesWithAliases() {
			fmt.Println(t)
		}
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("resource type required. Use --list to see available types")
	}

	// For a single resource type, use targeted fetching to avoid
	// downloading all group-version schemas.
	doc, err := loadResourceSchema(opts.kubeconfig, args[0])
	if err != nil {
		return err
	}

	gen := generator.NewOpenAPIGenerator(doc)
	overrides := collectOverrides(opts)
	return gen.Generate(args[0], overrides, os.Stdout)
}

func collectOverrides(opts *options) map[string]string {
	overrides := make(map[string]string)
	if opts.name != "" {
		overrides["name"] = opts.name
	}
	if opts.image != "" {
		overrides["image"] = opts.image
	}
	if opts.replicasSet {
		overrides["replicas"] = fmt.Sprintf("%d", opts.replicas)
	}
	for _, s := range opts.set {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) == 2 {
			overrides[parts[0]] = parts[1]
		}
	}
	return overrides
}

func loadResourceSchema(kubeconfigPath, resourceType string) (*openapi.Document, error) {
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

func loadClusterDoc(kubeconfigPath string) (*openapi.Document, error) {
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
