// Package framework provides shared infrastructure for kubewhy integration
// tests. It is a regular (non-test) package so that every resource-specific
// test package (pod/, deployment/, service/, …) can import it.
package framework

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/michaelkoro/kubewhy/internal/k8s"
)

// NewClientset builds a *kubernetes.Clientset from the default kubeconfig,
// using the same resolution order as the kubewhy CLI.
func NewClientset() (*kubernetes.Clientset, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules, &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

// NewK8sClient builds the kubewhy k8s.Client from the default kubeconfig.
func NewK8sClient() (*k8s.Client, error) {
	return k8s.NewClient("")
}
