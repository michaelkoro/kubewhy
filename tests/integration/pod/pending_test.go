//go:build integration

package podintegration

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestPendingDiagnosis runs all scheduling-failure scenarios as isolated
// subtests. Each subtest deploys a pod that cannot be scheduled, waits for
// it to enter the Unschedulable state, and asserts the Scheduling category.
func TestPendingDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("InsufficientCPU", testInsufficientCPU)
	t.Run("NodeAffinityMismatch", testNodeAffinityMismatch)
}

// testInsufficientCPU deploys a pod requesting 9999 CPU cores, far exceeding
// any real cluster's capacity. The scheduler immediately marks it unschedulable
// with reason "Insufficient cpu".
func testInsufficientCPU(t *testing.T) {
	podName := fmt.Sprintf("test-insufficient-cpu-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — excessive CPU request (9999 cores)", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "cpu-hog",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("9999"),
					},
				},
			}},
		},
	})

	waitForUnschedulable(t, podName, defaultTimeout)
	assertCategory(t, podName, "Scheduling")
}

// testNodeAffinityMismatch deploys a pod with a RequiredDuringScheduling node
// affinity rule that no node in the cluster satisfies. The scheduler marks the
// pod unschedulable because "kubewhy.io/nonexistent-label=required" does not
// exist on any node.
func testNodeAffinityMismatch(t *testing.T) {
	podName := fmt.Sprintf("test-affinity-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — unsatisfied RequiredDuringScheduling node affinity", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{{
							MatchExpressions: []corev1.NodeSelectorRequirement{{
								Key:      "kubewhy.io/nonexistent-label",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{"required"},
							}},
						}},
					},
				},
			},
			Containers: []corev1.Container{{
				Name:    "affinity-test",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
			}},
		},
	})

	waitForUnschedulable(t, podName, defaultTimeout)
	assertCategory(t, podName, "Scheduling")
}
