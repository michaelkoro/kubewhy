package cli

import (
	"fmt"

	"github.com/michaelkoro/kubewhy/internal/k8s"
	"github.com/spf13/cobra"
)

// podCmd represents the 'kubewhy pod' subcommand.
// It diagnoses issues with Kubernetes pods and explains them in plain English.
var podCmd = &cobra.Command{
	Use:   "pod [name]",
	Short: "Diagnose why a pod is failing or pending",
	Long: `Analyzes a pod (or all pods) and explains any issues in plain English.

The command checks for common problems including:
  - Scheduling failures (insufficient resources, taints, affinity)
  - Image pull errors (wrong image name, registry auth issues)
  - Crash loops (application crashes, misconfiguration)
  - Config errors (missing ConfigMaps or Secrets)
  - Volume issues (unbound PVCs, mount failures)

Examples:
  # Diagnose a specific pod
  kubewhy pod my-pod

  # Diagnose all problematic pods in current namespace
  kubewhy pod

  # Diagnose pods in a specific namespace
  kubewhy pod -n kube-system

  # Diagnose pods across all namespaces
  kubewhy pod --all-namespaces`,
	RunE: runPodDiagnosis,
}

// init registers the pod subcommand with the root command.
func init() {
	rootCmd.AddCommand(podCmd)
}

// runPodDiagnosis is the main handler for the 'kubewhy pod' command.
// It creates a K8s client, fetches pod information, and runs diagnostics.
func runPodDiagnosis(cmd *cobra.Command, args []string) error {
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

	// Get pod name if provided
	var podName string
	if len(args) > 0 {
		podName = args[0]
	}

	// TODO: Implement pod analysis logic
	// This will be added when the pod-analyzer todo is implemented
	fmt.Printf("Diagnosing pods in namespace: %q, pod: %q\n", ns, podName)
	fmt.Println("Pod analysis not yet implemented - coming soon!")

	_ = client // Silence unused variable warning for now

	return nil
}

