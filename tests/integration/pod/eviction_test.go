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

	"github.com/michaelkoro/kubewhy/tests/integration/framework"
)

// TestEvictionDiagnosis verifies the eviction analyzer by patching a pod's
// status to simulate an eviction. Triggering real node pressure in a test
// cluster is impractical, so we patch the status subresource directly.
func TestEvictionDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("Evicted", testEvicted)
}

// testEvicted deploys a healthy pod, waits for it to be Running, then patches
// its status to Phase=Failed, Reason=Evicted with a MemoryPressure message.
// The analyzer should report the Eviction category.
func testEvicted(t *testing.T) {
	podName := fmt.Sprintf("test-eviction-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — will patch status to simulate eviction", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
			}},
		},
	})

	waitForPodRunning(t, podName, defaultTimeout)

	// Patch the status subresource to simulate an eviction.
	statusPatch, _ := json.Marshal(map[string]interface{}{
		"status": map[string]interface{}{
			"phase":   "Failed",
			"reason":  "Evicted",
			"message": "The node was low on resource: memory.",
		},
	})
	_, err := clientset.CoreV1().Pods(env.Namespace).Patch(
		context.Background(), podName,
		types.MergePatchType, statusPatch,
		metav1.PatchOptions{}, "status",
	)
	if err != nil {
		t.Fatalf("patch pod %q status: %v", podName, err)
	}
	tLog(t, "patched pod %q status to Phase=Failed, Reason=Evicted", podName)

	// Wait for the status to be observable.
	framework.WaitFor(t, defaultTimeout,
		func(ctx context.Context) (*corev1.Pod, error) {
			return clientset.CoreV1().Pods(env.Namespace).Get(ctx, podName, metav1.GetOptions{})
		},
		func(p *corev1.Pod) bool {
			return p.Status.Phase == corev1.PodFailed && p.Status.Reason == "Evicted"
		},
		func(p *corev1.Pod) string {
			return fmt.Sprintf("phase=%s reason=%s", p.Status.Phase, p.Status.Reason)
		},
	)

	assertCategory(t, podName, "Eviction")
}
