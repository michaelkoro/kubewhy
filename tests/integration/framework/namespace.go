package framework

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/michaelkoro/kubewhy/internal/k8s"
)

// Env holds the shared state for one integration test package run: a live
// Kubernetes clientset, the kubewhy client, and a dedicated namespace that is
// created before tests start and deleted when they finish.
type Env struct {
	Clientset *kubernetes.Clientset
	K8sClient *k8s.Client
	Namespace string
}

// NewEnv builds clients and creates a timestamped namespace. Call Teardown()
// in TestMain after m.Run() to delete the namespace and all resources in it.
func NewEnv() (*Env, error) {
	cs, err := NewClientset()
	if err != nil {
		return nil, fmt.Errorf("build clientset: %w", err)
	}

	k8sClient, err := NewK8sClient()
	if err != nil {
		return nil, fmt.Errorf("build k8s client: %w", err)
	}

	ns := fmt.Sprintf("kubewhy-integration-%d", time.Now().Unix())
	_, err = cs.CoreV1().Namespaces().Create(
		context.Background(),
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}},
		metav1.CreateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("create namespace %s: %w", ns, err)
	}

	return &Env{
		Clientset: cs,
		K8sClient: k8sClient,
		Namespace: ns,
	}, nil
}

// Teardown deletes the test namespace, cascading to all resources inside it.
func (e *Env) Teardown() {
	_ = e.Clientset.CoreV1().Namespaces().Delete(
		context.Background(), e.Namespace, metav1.DeleteOptions{},
	)
}
