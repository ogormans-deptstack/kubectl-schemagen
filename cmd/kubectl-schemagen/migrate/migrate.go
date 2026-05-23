package migrate

import (
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
  kubectl get all -o yaml | kubectl schemagen migrate -`,
		Aliases: []string{"mig"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(args, kubeconfig)
		},
	}

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")

	return cmd
}

func runMigrate(files []string, kubeconfig string) error {
	available, err := cli.LoadAvailableAPIs(kubeconfig)
	if err != nil {
		return err
	}

	hasDeprecated := false
	hasRemoved := false

	for _, path := range files {
		var data []byte
		if path == "-" {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			path = "<stdin>"
		} else {
			data, err = os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
		}

		results, err := migrate.AnalyzeBytes(data, available)
		if err != nil {
			return fmt.Errorf("analyze %s: %w", path, err)
		}

		for _, r := range results {
			switch r.Status {
			case migrate.StatusOK:
				fmt.Printf("  OK  %s %s/%s\n", path, r.Manifest.APIVersion, r.Manifest.Kind)
			case migrate.StatusDeprecated:
				fmt.Printf(" DEP  %s %s/%s -> %s\n", path, r.Manifest.APIVersion, r.Manifest.Kind, r.Replacement)
				hasDeprecated = true
			case migrate.StatusRemoved:
				if r.Replacement != "" {
					fmt.Printf(" REM  %s %s/%s -> %s\n", path, r.Manifest.APIVersion, r.Manifest.Kind, r.Replacement)
				} else {
					fmt.Printf(" REM  %s %s/%s (no replacement found)\n", path, r.Manifest.APIVersion, r.Manifest.Kind)
				}
				hasRemoved = true
			}
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
