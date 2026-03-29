package cli

import (
	"context"
	"fmt"

	"github.com/michaelkoro/kubewhy/internal/analyzer/pod"
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
  - OOMKilled (container exceeded memory limit)
  - Runtime errors (missing entrypoint, security context violations)
  - Init container failures (any of the above in init containers)
  - Probe failures (liveness and readiness probe issues)
  - Config errors (missing ConfigMaps or Secrets)
  - Volume issues (unbound PVCs, mount failures)
  - Eviction (node pressure — disk, memory, PID)
  - Stuck Terminating (finalizers blocking pod deletion)

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

	ctx := context.Background()
	analyzer := pod.NewAnalyzer(client)

	// Mode 1: specific pod name provided
	if len(args) > 0 {
		return diagnoseSinglePod(ctx, analyzer, ns, args[0])
	}

	// Mode 2: no name — scan all pods in the namespace
	return diagnoseAllPods(ctx, client, analyzer, ns)
}

// diagnoseSinglePod analyzes one pod by name and prints its full diagnosis,
// including a healthy confirmation when no issues are found.
func diagnoseSinglePod(ctx context.Context, analyzer *pod.PodAnalyzer, ns, name string) error {
	diagnosis, err := analyzer.Analyze(ctx, ns, name)
	if err != nil {
		return err
	}
	fmt.Print(diagnosis.Format())
	return nil
}

// diagnoseAllPods fetches every pod in the namespace, analyzes each one, and
// prints only those with issues. Finishes with an analyzed/issues summary line.
func diagnoseAllPods(ctx context.Context, client *k8s.Client, analyzer *pod.PodAnalyzer, ns string) error {
	podList, err := client.ListPods(ctx, ns)
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}

	var issueCount int
	for i := range podList.Items {
		p := &podList.Items[i]
		diagnosis, err := analyzer.Analyze(ctx, p.Namespace, p.Name)
		if err != nil {
			fmt.Printf("Warning: could not analyze pod %s/%s: %v\n", p.Namespace, p.Name, err)
			continue
		}
		if !diagnosis.IsHealthy {
			fmt.Print(diagnosis.Format())
			fmt.Println()
			issueCount++
		}
	}

	fmt.Printf("Analyzed %d pod(s), %d with issues.\n", len(podList.Items), issueCount)
	return nil
}

