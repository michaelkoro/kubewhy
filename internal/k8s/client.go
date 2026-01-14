// Package k8s provides a wrapper around the Kubernetes client-go library.
// It simplifies common operations needed for diagnosing Kubernetes resources.
package k8s

import (
	"context"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset and provides helper methods
// for common operations used in diagnostics.
type Client struct {
	// clientset is the underlying Kubernetes client
	clientset *kubernetes.Clientset
	// currentNamespace is the namespace from the current kubeconfig context
	currentNamespace string
}

// NewClient creates a new Kubernetes client.
//
// It attempts to load kubeconfig in the following order:
//  1. kubeconfigPath argument (if provided)
//  2. KUBECONFIG environment variable
//  3. ~/.kube/config (default location)
//
// Returns an error if no valid kubeconfig can be found or loaded.
func NewClient(kubeconfigPath string) (*Client, error) {
	// Determine kubeconfig path
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Extract current namespace from kubeconfig context
	namespace := "default"
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	if ns, _, err := kubeConfig.Namespace(); err == nil && ns != "" {
		namespace = ns
	}

	return &Client{
		clientset:        clientset,
		currentNamespace: namespace,
	}, nil
}

// CurrentNamespace returns the namespace from the current kubeconfig context.
// This is used as the default namespace when no -n flag is provided.
func (c *Client) CurrentNamespace() string {
	return c.currentNamespace
}

// GetPod retrieves a specific pod by name from the given namespace.
// Returns an error if the pod is not found or cannot be accessed.
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListPods lists all pods in the given namespace.
// If namespace is empty, lists pods across all namespaces.
func (c *Client) ListPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
}

// GetEvents retrieves all events associated with a specific object.
// This is useful for understanding why a resource is in a particular state.
func (c *Client) GetEvents(ctx context.Context, namespace, objectName string) (*corev1.EventList, error) {
	fieldSelector := "involvedObject.name=" + objectName
	return c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}

// GetNode retrieves a node by name.
// This is useful for checking node conditions when pods fail to schedule.
func (c *Client) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	return c.clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

// ListNodes lists all nodes in the cluster.
// This is useful for checking overall cluster capacity.
func (c *Client) ListNodes(ctx context.Context) (*corev1.NodeList, error) {
	return c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
}

// Clientset returns the underlying Kubernetes clientset.
// Use this for advanced operations not covered by the helper methods.
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

