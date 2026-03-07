//go:build integration

package podintegration

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestInitContainerDiagnosis runs all init-container failure scenarios as
// isolated subtests. Each subtest deploys a pod whose init container is
// deliberately broken, waits for it to enter the expected failure state, and
// asserts that the analyzer reports the correct "Init*" category.
func TestInitContainerDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("InitCrashLoop", testInitCrashLoop)
	t.Run("InitImagePull", testInitImagePull)
}

// testInitCrashLoop deploys a pod whose init container exits immediately with
// code 1. With RestartPolicy OnFailure the kubelet restarts it, which quickly
// produces CrashLoopBackOff in InitContainerStatuses.
func testInitCrashLoop(t *testing.T) {
	podName := fmt.Sprintf("test-init-crashloop-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — init container crash loop", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			InitContainers: []corev1.Container{{
				Name:    "init-crasher",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "exit 1"},
			}},
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
			}},
		},
	})

	waitForInitContainerWaitingReason(t, podName, "CrashLoopBackOff", defaultTimeout)
	assertCategory(t, podName, "InitCrashLoop")
}

// testInitImagePull deploys a pod whose init container references a nonexistent
// image. Kubernetes will repeatedly fail to pull it, landing the init container
// in ImagePullBackOff and blocking the regular containers from starting.
func testInitImagePull(t *testing.T) {
	podName := fmt.Sprintf("test-init-imagepull-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — init container bad image reference", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			InitContainers: []corev1.Container{{
				Name:            "init-bad-image",
				Image:           "does-not-exist-registry.io/fake-image:missing",
				ImagePullPolicy: corev1.PullAlways,
			}},
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
			}},
		},
	})

	waitForInitContainerWaitingReason(t, podName, "ImagePullBackOff", defaultTimeout)
	assertCategory(t, podName, "InitImagePull")
}
