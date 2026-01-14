// Package cli implements the command-line interface for kubewhy.
package cli

import (
	"fmt"

	"github.com/michaelkoro/kubewhy/internal/k8s"
	"github.com/spf13/cobra"
)

// checkCmd tests connectivity to the Kubernetes cluster.
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Test connection to the Kubernetes cluster",
	Long: `Test connectivity to the Kubernetes cluster by fetching the server version.

This is useful for verifying that your kubeconfig is set up correctly
and that the cluster is reachable.

Examples:
  # Test connection using default kubeconfig
  kubewhy check

  # Test connection using a specific kubeconfig
  kubewhy check --kubeconfig ~/my-config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := k8s.NewClient(GetKubeconfig())
		if err != nil {
			return fmt.Errorf("failed to create k8s client: %w", err)
		}

		version, err := client.CheckConnection(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to connect to cluster: %w", err)
		}

		fmt.Printf("Connected to Kubernetes cluster\n")
		fmt.Printf("  Server Version: %s\n", version.GitVersion)
		fmt.Printf("  Platform: %s\n", version.Platform)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

