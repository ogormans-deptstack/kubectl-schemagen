package scaffold

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ogormans-deptstack/kubectl-schemagen/internal/cli"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/generator"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/openapi"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/scaffold"
)

func NewCommand() *cobra.Command {
	var opts cli.ManifestOptions
	var outputDir string

	cmd := &cobra.Command{
		Use:   "scaffold RESOURCE_TYPE [RESOURCE_TYPE...]",
		Short: "Generate a kustomize base from resource types",
		Long: `Generates example YAML manifests for the given resource types and writes
them as a kustomize base directory with a kustomization.yaml file.`,
		Example: `  kubectl schemagen scaffold deployment service
  kubectl schemagen scaffold deployment --name=web --image=nginx -o ./base`,
		Aliases: []string{"sc"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ReplicasSet = cmd.Flags().Changed("replicas")
			return runScaffold(args, &opts, outputDir)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "base", "Output directory for kustomize base")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Resource name")
	cmd.Flags().StringVar(&opts.Image, "image", "", "Container image")
	cmd.Flags().IntVar(&opts.Replicas, "replicas", 0, "Replica count")
	cmd.Flags().StringArrayVar(&opts.Set, "set", nil, "Field override (key=value)")
	cmd.Flags().StringVar(&opts.Kubeconfig, "kubeconfig", "", "Path to kubeconfig")

	return cmd
}

func runScaffold(resourceTypes []string, opts *cli.ManifestOptions, outputDir string) error {
	var doc *openapi.Document
	var err error

	// Use targeted fetching for a single resource type to avoid
	// downloading all group-version schemas unnecessarily.
	if len(resourceTypes) == 1 {
		doc, err = cli.LoadResourceSchema(opts.Kubeconfig, resourceTypes[0])
	} else {
		doc, err = cli.LoadClusterDoc(opts.Kubeconfig)
	}
	if err != nil {
		return err
	}

	gen := generator.NewOpenAPIGenerator(doc)
	overrides, err := cli.CollectOverrides(opts)
	if err != nil {
		return err
	}

	for _, resourceType := range resourceTypes {
		var buf bytes.Buffer
		if err := gen.Generate(resourceType, overrides, &buf); err != nil {
			return fmt.Errorf("generate %s: %w", resourceType, err)
		}

		if err := scaffold.WriteKustomizeBase(outputDir, resourceType, buf.Bytes()); err != nil {
			return fmt.Errorf("write scaffold for %s: %w", resourceType, err)
		}
		fmt.Printf("wrote %s/%s.yaml\n", outputDir, strings.ToLower(resourceType))
	}

	fmt.Printf("kustomize base written to %s/\n", outputDir)
	return nil
}
