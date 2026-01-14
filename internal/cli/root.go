// Package cli implements the command-line interface for kubewhy.
// It uses Cobra to define commands and flags for diagnosing Kubernetes resources.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Global flags available to all subcommands
var (
	// kubeconfig is the path to the kubeconfig file
	kubeconfig string
	// namespace is the Kubernetes namespace to operate in
	namespace string
	// allNamespaces indicates whether to search across all namespaces
	allNamespaces bool
)

// rootCmd is the base command for kubewhy.
// All other commands are added as subcommands to this.
var rootCmd = &cobra.Command{
	Use:   "kubewhy",
	Short: "Explains Kubernetes failures in plain English",
	Long: `kubewhy is a diagnostic CLI that analyzes failing Kubernetes resources
and explains what's wrong in plain English.

It helps you quickly understand why pods are pending, crash-looping,
or failing to start — without digging through kubectl describe output.

Examples:
  # Diagnose why a specific pod won't start
  kubewhy pod my-failing-pod

  # Diagnose all failing pods in current namespace
  kubewhy pod

  # Diagnose pods across all namespaces
  kubewhy pod --all-namespaces`,
}

// init registers global flags that apply to all commands.
func init() {
	// --kubeconfig flag: path to kubeconfig file
	rootCmd.PersistentFlags().StringVar(
		&kubeconfig,
		"kubeconfig",
		"",
		"path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)",
	)

	// -n, --namespace flag: target namespace
	rootCmd.PersistentFlags().StringVarP(
		&namespace,
		"namespace",
		"n",
		"",
		"kubernetes namespace (default: current context namespace)",
	)

	// -A, --all-namespaces flag: search all namespaces
	rootCmd.PersistentFlags().BoolVarP(
		&allNamespaces,
		"all-namespaces",
		"A",
		false,
		"diagnose resources across all namespaces",
	)
}

// Execute runs the root command and handles errors.
// This is called from main() to start the CLI.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// GetKubeconfig returns the kubeconfig path from the --kubeconfig flag.
// Returns empty string if not set (will use default locations).
func GetKubeconfig() string {
	return kubeconfig
}

// GetNamespace returns the namespace from the -n/--namespace flag.
// Returns empty string if not set (will use current context namespace).
func GetNamespace() string {
	return namespace
}

// GetAllNamespaces returns true if -A/--all-namespaces flag is set.
func GetAllNamespaces() bool {
	return allNamespaces
}

