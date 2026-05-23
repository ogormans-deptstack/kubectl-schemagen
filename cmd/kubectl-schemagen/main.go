package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ogormans-deptstack/kubectl-schemagen/cmd/kubectl-schemagen/manifest"
	"github.com/ogormans-deptstack/kubectl-schemagen/cmd/kubectl-schemagen/migrate"
	"github.com/ogormans-deptstack/kubectl-schemagen/cmd/kubectl-schemagen/scaffold"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// exitCoder is implemented by errors that carry a specific exit code.
type exitCoder interface {
	Error() string
	ExitCode() int
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		var ec exitCoder
		if errors.As(err, &ec) {
			fmt.Fprintf(os.Stderr, "%s\n", ec.Error())
			os.Exit(ec.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl-schemagen",
		Short: "OpenAPI schema-powered Kubernetes tools",
		Long: `kubectl-schemagen provides a suite of tools that leverage the cluster's
OpenAPI v3 schema for manifest generation, API migration, and scaffolding.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       formatVersion(),
	}

	cmd.AddCommand(manifest.NewCommand())
	cmd.AddCommand(migrate.NewCommand())
	cmd.AddCommand(scaffold.NewCommand())

	return cmd
}

func formatVersion() string {
	if commit == "unknown" && date == "unknown" {
		return version
	}
	short := commit
	if len(short) > 7 {
		short = short[:7]
	}
	return fmt.Sprintf("%s (commit %s, built %s)", version, short, date)
}
