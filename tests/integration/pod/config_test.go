//go:build integration

package podintegration

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestConfigDiagnosis runs all config-error scenarios as isolated subtests.
// Each subtest deploys a pod that references a non-existent ConfigMap or
// Secret, waits for the expected failure state, and asserts the ConfigError
// category.
func TestConfigDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("MissingConfigMap", testMissingConfigMap)
	t.Run("MissingSecret", testMissingSecret)
}

// testMissingConfigMap deploys a pod with envFrom referencing a ConfigMap that
// does not exist. The kubelet sets the container waiting reason to
// CreateContainerConfigError.
func testMissingConfigMap(t *testing.T) {
	podName := fmt.Sprintf("test-missing-cm-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — envFrom referencing non-existent ConfigMap", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
				EnvFrom: []corev1.EnvFromSource{{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "does-not-exist-configmap",
						},
					},
				}},
			}},
		},
	})

	waitForWaitingReason(t, podName, "CreateContainerConfigError", defaultTimeout)
	assertCategory(t, podName, "ConfigError")
}

// testMissingSecret deploys a pod with envFrom referencing a Secret that does
// not exist. The kubelet sets the container waiting reason to
// CreateContainerConfigError.
func testMissingSecret(t *testing.T) {
	podName := fmt.Sprintf("test-missing-secret-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — envFrom referencing non-existent Secret", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
				EnvFrom: []corev1.EnvFromSource{{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "does-not-exist-secret",
						},
					},
				}},
			}},
		},
	})

	waitForWaitingReason(t, podName, "CreateContainerConfigError", defaultTimeout)
	assertCategory(t, podName, "ConfigError")
}
