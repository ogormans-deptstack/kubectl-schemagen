package migrate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/ogormans-deptstack/kubectl-schemagen/internal/cli"
	"github.com/ogormans-deptstack/kubectl-schemagen/pkg/migrate"
)

// exitCode distinguishes between deprecated and removed API results.
// Exit 1 = removed APIs found, Exit 2 = deprecated APIs found (none removed).
type exitCode struct {
	code int
}

func (e *exitCode) Error() string {
	switch e.code {
	case 1:
		return "removed APIs found"
	case 2:
		return "deprecated APIs found"
	default:
		return "issues found"
	}
}

func (e *exitCode) ExitCode() int {
	return e.code
}

func NewCommand() *cobra.Command {
	var kubeconfig string
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "migrate [FILE...]",
		Short: "Detect deprecated or removed Kubernetes APIs in manifests",
		Long: `Reads YAML manifests and detects deprecated or removed API versions by
comparing against the connected cluster's OpenAPI schema. Reports which
resources use outdated API versions and suggests replacements.

Use "-" as a filename to read from stdin, enabling piped workflows:
  kubectl get deploy -o yaml | kubectl schemagen migrate -

Exit codes:
  0  All APIs are current
  1  Removed APIs found (or runtime error)
  2  Deprecated APIs found (none removed)`,
		Example: `  kubectl schemagen migrate deployment.yaml
  kubectl schemagen migrate manifests/*.yaml
  kubectl schemagen migrate -o json deployment.yaml
  kubectl get all -o yaml | kubectl schemagen migrate -`,
		Aliases: []string{"mig"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(args, kubeconfig, outputFormat)
		},
	}

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json)")

	return cmd
}

// migrateOutput is the JSON-serializable output for a single result.
type migrateOutput struct {
	File        string `json:"file"`
	APIVersion  string `json:"apiVersion"`
	Kind        string `json:"kind"`
	Name        string `json:"name,omitempty"`
	Status      string `json:"status"`
	Replacement string `json:"replacement,omitempty"`
}

func runMigrate(files []string, kubeconfig, outputFormat string) error {
	available, err := cli.LoadAvailableAPIs(kubeconfig)
	if err != nil {
		return err
	}

	hasDeprecated := false
	hasRemoved := false
	var jsonResults []migrateOutput

	for _, path := range files {
		var data []byte
		displayPath := path
		if path == "-" {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			displayPath = "<stdin>"
		} else {
			data, err = os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
		}

		results, err := migrate.AnalyzeBytes(data, available)
		if err != nil {
			return fmt.Errorf("analyze %s: %w", displayPath, err)
		}

		for _, r := range results {
			switch r.Status {
			case migrate.StatusDeprecated:
				hasDeprecated = true
			case migrate.StatusRemoved:
				hasRemoved = true
			}

			switch outputFormat {
			case "json":
				jsonResults = append(jsonResults, migrateOutput{
					File:        displayPath,
					APIVersion:  r.Manifest.APIVersion,
					Kind:        r.Manifest.Kind,
					Name:        r.Manifest.Name,
					Status:      r.Status.String(),
					Replacement: r.Replacement,
				})
			default:
				printTextResult(displayPath, r)
			}
		}
	}

	if outputFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(jsonResults); err != nil {
			return fmt.Errorf("encode JSON: %w", err)
		}
	}

	if hasRemoved {
		return &exitCode{code: 1}
	}
	if hasDeprecated {
		return &exitCode{code: 2}
	}
	return nil
}

func printTextResult(path string, r migrate.Result) {
	switch r.Status {
	case migrate.StatusOK:
		fmt.Printf("  OK  %s %s/%s\n", path, r.Manifest.APIVersion, r.Manifest.Kind)
	case migrate.StatusDeprecated:
		fmt.Printf(" DEP  %s %s/%s -> %s\n", path, r.Manifest.APIVersion, r.Manifest.Kind, r.Replacement)
	case migrate.StatusRemoved:
		if r.Replacement != "" {
			fmt.Printf(" REM  %s %s/%s -> %s\n", path, r.Manifest.APIVersion, r.Manifest.Kind, r.Replacement)
		} else {
			fmt.Printf(" REM  %s %s/%s (no replacement found)\n", path, r.Manifest.APIVersion, r.Manifest.Kind)
		}
	}
}
