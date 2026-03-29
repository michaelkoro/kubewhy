//go:build integration

package podintegration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestTerminatingDiagnosis runs stuck-terminating scenarios as isolated
// subtests. Each subtest deploys a pod with a finalizer, deletes it, and
// asserts that the analyzer reports StuckTerminating.
func TestTerminatingDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("StuckFinalizer", testStuckFinalizer)
}

// testStuckFinalizer deploys a pod with a custom finalizer, then deletes it
// with grace period 0. Because the finalizer is never resolved, the pod stays
// in a Terminating state with DeletionTimestamp set. The analyzer should
// report StuckTerminating once the grace period has elapsed.
func testStuckFinalizer(t *testing.T) {
	podName := fmt.Sprintf("test-stuck-terminating-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — with blocking finalizer", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:       podName,
			Finalizers: []string{"kubewhy.io/test-blocker"},
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: int64Ptr(1),
			RestartPolicy:                corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
			}},
		},
	})

	// Wait for the pod to be Running before issuing the delete.
	waitForPodRunning(t, podName, defaultTimeout)

	// Delete the pod — it will stay in Terminating because of the finalizer.
	grace := int64(0)
	err := clientset.CoreV1().Pods(env.Namespace).Delete(
		context.Background(), podName,
		metav1.DeleteOptions{GracePeriodSeconds: &grace},
	)
	if err != nil {
		t.Fatalf("delete pod %q: %v", podName, err)
	}
	tLog(t, "issued delete for pod %q", podName)

	waitForDeletionTimestamp(t, podName, defaultTimeout)

	// Wait past the 1-second grace period so the analyzer flags it as stuck.
	time.Sleep(3 * time.Second)

	assertCategory(t, podName, "StuckTerminating")

	// Clean up: remove the finalizer so the pod can be garbage collected.
	removeFinalizer(t, podName)
}

// removeFinalizer patches the pod to clear its finalizers list so Kubernetes
// can complete deletion.
func removeFinalizer(t *testing.T, podName string) {
	t.Helper()
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": nil,
		},
	})
	_, err := clientset.CoreV1().Pods(env.Namespace).Patch(
		context.Background(), podName,
		types.MergePatchType, patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		t.Logf("warning: failed to remove finalizer from %q: %v", podName, err)
	}
}

func int64Ptr(v int64) *int64 { return &v }
