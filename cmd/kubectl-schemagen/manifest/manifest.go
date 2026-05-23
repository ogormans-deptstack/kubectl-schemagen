package manifest

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ogormans-deptstack/kubectl-schemagen/internal/cli"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/generator"
)

func NewCommand() *cobra.Command {
	var opts cli.ManifestOptions
	var list bool
	var outputFormat string
	var annotate bool

	cmd := &cobra.Command{
		Use:   "manifest RESOURCE_TYPE",
		Short: "Generate example YAML manifests from the OpenAPI spec",
		Long: `Generates example Kubernetes resource YAML from the cluster's OpenAPI v3 spec.
The generated manifest includes sensible defaults and can be piped directly to kubectl create.

Use --annotate to include schema descriptions and enum values as YAML
comments, making the output self-documenting.`,
		Example: `  kubectl schemagen manifest pod
  kubectl schemagen manifest deployment --name=web --replicas=3 | kubectl create -f -
  kubectl schemagen manifest deployment --annotate
  kubectl schemagen manifest deployment -o json | jq .
  kubectl schemagen manifest --list`,
		Aliases: []string{"m"},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ReplicasSet = cmd.Flags().Changed("replicas")
			return runManifest(args, &opts, list, outputFormat, annotate)
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List all supported resource types")
	cmd.Flags().BoolVar(&annotate, "annotate", false, "Add schema descriptions as YAML comments")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "yaml", "Output format (yaml, json)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Resource name")
	cmd.Flags().StringVar(&opts.Image, "image", "", "Container image")
	cmd.Flags().IntVar(&opts.Replicas, "replicas", 0, "Replica count")
	cmd.Flags().StringArrayVar(&opts.Set, "set", nil, "Field override (key=value)")
	cmd.Flags().StringVar(&opts.Kubeconfig, "kubeconfig", "", "Path to kubeconfig")

	return cmd
}

func runManifest(args []string, opts *cli.ManifestOptions, list bool, outputFormat string, annotate bool) error {
	// --list requires all schemas to show every available type.
	if list {
		doc, err := cli.LoadClusterDoc(opts.Kubeconfig)
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
	// downloading all 30-50+ group-version schemas.
	doc, err := cli.LoadResourceSchema(opts.Kubeconfig, args[0])
	if err != nil {
		return err
	}

	gen := generator.NewOpenAPIGenerator(doc)
	overrides, err := cli.CollectOverrides(opts)
	if err != nil {
		return err
	}

	if annotate {
		return gen.GenerateAnnotated(args[0], overrides, os.Stdout)
	}

	switch outputFormat {
	case "json":
		return gen.GenerateJSON(args[0], overrides, os.Stdout)
	case "yaml", "":
		return gen.Generate(args[0], overrides, os.Stdout)
	default:
		return fmt.Errorf("unsupported output format %q (use yaml or json)", outputFormat)
	}
}
