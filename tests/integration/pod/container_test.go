//go:build integration

// Package podintegration contains end-to-end tests that deploy faulty pods to
// a live Kubernetes cluster and verify that the container analyzer diagnoses
// each failure correctly.
//
// Run with:
//
//	go test -v -tags integration ./tests/integration/pod/... -timeout 3m
package podintegration

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultTimeout is how long each test waits for a pod to reach its target
// failure state before giving up. k3d is fast, but image pulls can be slow.
const defaultTimeout = 90 * time.Second

// TestContainerDiagnosis runs all container-level failure scenarios as
// isolated subtests. Each subtest deploys a deliberately broken pod, waits
// for it to enter the expected bad state, runs the analyzer, and asserts that
// the correct category is reported.
func TestContainerDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("ImagePull", testImagePull)
	t.Run("CrashLoop", testCrashLoop)
	t.Run("OOMKilled", testOOMKilled)
	t.Run("RuntimeError", testRuntimeError)
}

// testImagePull deploys a pod that references a nonexistent image. Kubernetes
// will repeatedly attempt and fail to pull it, landing in ImagePullBackOff.
func testImagePull(t *testing.T) {
	podName := fmt.Sprintf("test-imagepull-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — bad image reference", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:            "app",
				Image:           "does-not-exist-registry.io/fake-image:missing",
				ImagePullPolicy: corev1.PullAlways,
			}},
		},
	})

	waitForWaitingReason(t, podName, "ImagePullBackOff", defaultTimeout)
	assertCategory(t, podName, "ImagePull")
}

// testCrashLoop deploys a pod whose container exits immediately with code 1.
// After enough restarts Kubernetes puts it into CrashLoopBackOff.
func testCrashLoop(t *testing.T) {
	podName := fmt.Sprintf("test-crashloop-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — crash loop", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			// OnFailure causes Kubernetes to restart the container after each
			// crash, which is what triggers CrashLoopBackOff.
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{{
				Name:    "crasher",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "exit 1"},
			}},
		},
	})

	waitForWaitingReason(t, podName, "CrashLoopBackOff", defaultTimeout)
	assertCategory(t, podName, "CrashLoop")
}

// testOOMKilled deploys a pod with a very low memory limit whose container
// immediately tries to allocate far more memory than it is allowed. The kernel
// OOM killer terminates it with reason OOMKilled. Because RestartPolicy is
// OnFailure the pod restarts, so the analyzer may see OOMKilled in either the
// current or last-terminated state — both are checked by checkOOMKilled.
func testOOMKilled(t *testing.T) {
	podName := fmt.Sprintf("test-oomkilled-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — OOM killed", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{{
				Name:  "memory-hog",
				Image: "polinux/stress:latest",
				// Allocate 32 MB — well above the 4 Mi limit below.
				Command: []string{"stress", "--vm", "1", "--vm-bytes", "32M", "--vm-hang", "0"},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("4Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("4Mi"),
					},
				},
			}},
		},
	})

	waitForTerminatedReason(t, podName, "OOMKilled", defaultTimeout)
	assertCategory(t, podName, "OOMKilled")
}

// testRuntimeError deploys a pod whose entrypoint does not exist inside the
// image. The container runtime (containerd) cannot exec the process and reports
// Terminated reason "StartError".
func testRuntimeError(t *testing.T) {
	podName := fmt.Sprintf("test-runtimeerr-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — runtime error", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "bad-entrypoint",
				Image:   "busybox:latest",
				Command: []string{"/this-binary-does-not-exist"},
			}},
		},
	})

	waitForTerminatedReason(t, podName, "StartError", defaultTimeout)
	assertCategory(t, podName, "RuntimeError")
}
