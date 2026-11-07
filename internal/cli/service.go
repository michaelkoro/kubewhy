package cli

import (
	"context"
	"fmt"

	"github.com/michaelkoro/kubewhy/internal/analyzer/svc"
	"github.com/michaelkoro/kubewhy/internal/k8s"
	"github.com/spf13/cobra"
)

// svcCmd represents the 'kubewhy svc' subcommand.
// It diagnoses issues with Kubernetes services and explains them in plain English.
var svcCmd = &cobra.Command{
	Use:   "svc [name]",
	Short: "Diagnose why a service is not working",
	Long: `Analyzes a Kubernetes service and explains any issues in plain English.

The command checks for common problems including:
  - Service not found
  - No endpoints available
  - Misconfigured selectors
  - Port mismatches
  - Network policy restrictions

Examples:
  # Diagnose a specific service
  kubewhy svc my-service

  # Diagnose all services in current namespace
  kubewhy svc

  # Diagnose services in a specific namespace
  kubewhy svc -n kube-system

  # Diagnose services across all namespaces
  kubewhy svc --all-namespaces`,
	RunE: runServiceDiagnosis,
}

// init registers the svc subcommand with the root command.
func init() {
	rootCmd.AddCommand(svcCmd)
}

// runServiceDiagnosis is the main handler for the 'kubewhy svc' command.
// It creates a K8s client, fetches svc information, and runs diagnostics.
func runServiceDiagnosis(cmd *cobra.Command, args []string) error {
	// Create Kubernetes client
	client, err := k8s.NewClient(GetKubeconfig())
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Determine which namespace to query
	ns := GetNamespace()
	if ns == "" && !GetAllNamespaces() {
		// Use the current context's namespace
		ns = client.CurrentNamespace()
	}
	if GetAllNamespaces() {
		// Empty namespace means "all namespaces" in client-go
		ns = ""
	}

	ctx := context.Background()
	analyzer := svc.NewAnalyzer(client)

	// Mode 1: specific svc name provided
	if len(args) > 0 {
		return diagnoseSingleSvc(ctx, analyzer, ns, args[0])
	}

	// Mode 2: no name — scan all svcs in the namespace
	return diagnoseAllSvcs(ctx, client, analyzer, ns)
}

// diagnoseSingleSvc analyzes one service by name and prints its full diagnosis,
// including a healthy confirmation when no issues are found.
func diagnoseSingleSvc(ctx context.Context, analyzer *svc.SvcAnalyzer, ns, name string) error {
	/*diagnosis, err := analyzer.Analyze(ctx, ns, name)
	if err != nil {
		return err
	}

	fmt.Print(diagnosis.Format())*/
	return nil
}
func diagnoseAllSvcs(ctx context.Context, client *k8s.Client, analyzer *svc.SvcAnalyzer, ns string) error {
	println("Diagnosing all services in namespace:", ns)
	return nil
}
