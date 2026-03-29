//go:build integration

package podintegration

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestVolumeDiagnosis runs volume-mount failure scenarios as isolated subtests.
// Each subtest deploys a pod that references a non-existent PVC, waits for the
// expected failure state, and asserts the VolumeMount category.
func TestVolumeDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("MissingPVC", testMissingPVC)
}

// testMissingPVC deploys a pod mounting a PersistentVolumeClaim that does not
// exist. The kubelet emits FailedMount events while waiting to mount the
// volume, eventually timing out.
//
// We set nodeName to bypass the scheduler — a non-existent PVC causes the
// scheduler to leave the pod in Pending indefinitely, so no FailedMount event
// would ever be emitted. Assigning a node directly lets the kubelet attempt
// (and fail) the mount.
func testMissingPVC(t *testing.T) {
	podName := fmt.Sprintf("test-missing-pvc-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — mounting non-existent PVC", podName)

	nodeName := firstNodeName(t)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			NodeName:      nodeName,
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "data",
					MountPath: "/data",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "does-not-exist-pvc",
					},
				},
			}},
		},
	})

	waitForEvent(t, podName, "FailedMount", 2*time.Minute)
	assertCategory(t, podName, "VolumeMount")
}

// firstNodeName returns the name of the first Ready node in the cluster.
func firstNodeName(t *testing.T) string {
	t.Helper()
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes.Items) == 0 {
		t.Fatal("cluster has no nodes")
	}
	return nodes.Items[0].Name
}
