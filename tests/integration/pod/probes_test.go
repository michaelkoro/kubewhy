//go:build integration

// Package podintegration — probe failure integration tests.
package podintegration

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestProbeDiagnosis runs all probe failure scenarios as isolated subtests.
// Each subtest deploys a Running pod with a deliberately broken probe, waits
// for the expected failure state to appear, and asserts the analyzer reports
// the correct category.
func TestProbeDiagnosis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in -short mode")
	}

	t.Run("LivenessProbe", testLivenessProbeFailure)
	t.Run("ReadinessProbe", testReadinessProbeFailure)
}

// testLivenessProbeFailure deploys a pod that stays running (sleep 3600) but
// has a liveness probe that always exits non-zero. failureThreshold=10 gives a
// ~55 second window between the first Unhealthy event and the container being
// killed, which is well beyond the polling and analysis time.
func testLivenessProbeFailure(t *testing.T) {
	podName := fmt.Sprintf("test-liveness-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — always-failing liveness probe", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			// Never restart after the liveness-induced kill so the pod stays in
			// a stable Running state long enough for the test to observe it.
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"false"},
						},
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
					// High threshold gives ample time to detect the first
					// Unhealthy event and run the analyzer before the kubelet
					// kills the container.
					FailureThreshold: 10,
				},
			}},
		},
	})

	waitForUnhealthyEvent(t, podName, defaultTimeout)
	assertCategory(t, podName, "ProbeFailure")
}

// testReadinessProbeFailure deploys a pod whose readiness probe always fails.
// Unlike liveness, a failing readiness probe never restarts the container —
// the pod stays Running with ContainersReady=False indefinitely, which makes
// the timing safe regardless of how long analysis takes.
func testReadinessProbeFailure(t *testing.T) {
	podName := fmt.Sprintf("test-readiness-%d", time.Now().UnixNano())
	tLog(t, "deploying pod %q — always-failing readiness probe", podName)

	deployPod(t, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c", "sleep 3600"},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"false"},
						},
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
					FailureThreshold:    1,
				},
			}},
		},
	})

	waitForContainersNotReady(t, podName, defaultTimeout)
	assertCategory(t, podName, "ProbeFailure")
}
