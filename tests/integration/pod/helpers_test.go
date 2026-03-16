//go:build integration

package podintegration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/michaelkoro/kubewhy/tests/integration/framework"
)

// tLog writes directly to stderr so the message appears immediately in the
// terminal, bypassing Go's test-output buffer which only flushes at subtest
// completion. Use this instead of t.Logf for progress messages that should
// be visible while the subtest is still running.
func tLog(t *testing.T, format string, args ...any) {
	t.Helper()
	fmt.Fprintf(os.Stderr, "\t    --- %s: %s\n", t.Name(), fmt.Sprintf(format, args...))
}

// deployPod creates the given pod in the shared test namespace and registers a
// t.Cleanup that force-deletes it when the test (or subtest) finishes.
func deployPod(t *testing.T, spec *corev1.Pod) *corev1.Pod {
	t.Helper()

	spec.Namespace = env.Namespace
	created, err := clientset.CoreV1().Pods(env.Namespace).Create(
		context.Background(), spec, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("create pod %q: %v", spec.Name, err)
	}
	tLog(t, "created pod %q (namespace: %s)", created.Name, created.Namespace)

	t.Cleanup(func() {
		grace := int64(0)
		_ = clientset.CoreV1().Pods(env.Namespace).Delete(
			context.Background(), created.Name,
			metav1.DeleteOptions{GracePeriodSeconds: &grace},
		)
	})

	return created
}

// waitForWaitingReason blocks until any container in the pod has the given
// waiting reason (e.g. "ImagePullBackOff", "CrashLoopBackOff").
func waitForWaitingReason(t *testing.T, podName, reason string, timeout time.Duration) {
	t.Helper()
	pod := framework.WaitFor(t, timeout,
		func(ctx context.Context) (*corev1.Pod, error) {
			return clientset.CoreV1().Pods(env.Namespace).Get(ctx, podName, metav1.GetOptions{})
		},
		func(p *corev1.Pod) bool {
			for _, cs := range p.Status.ContainerStatuses {
				if cs.State.Waiting != nil && cs.State.Waiting.Reason == reason {
					return true
				}
			}
			return false
		},
		describePod,
	)
	if pod != nil {
		tLog(t, "pod status: %s", describePod(pod))
	}
}

// waitForTerminatedReason blocks until any container in the pod has the given
// terminated reason in its current or last-terminated state (e.g. "OOMKilled").
func waitForTerminatedReason(t *testing.T, podName, reason string, timeout time.Duration) {
	t.Helper()
	pod := framework.WaitFor(t, timeout,
		func(ctx context.Context) (*corev1.Pod, error) {
			return clientset.CoreV1().Pods(env.Namespace).Get(ctx, podName, metav1.GetOptions{})
		},
		func(p *corev1.Pod) bool {
			for _, cs := range p.Status.ContainerStatuses {
				if cs.State.Terminated != nil && cs.State.Terminated.Reason == reason {
					return true
				}
				if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == reason {
					return true
				}
			}
			return false
		},
		describePod,
	)
	if pod != nil {
		tLog(t, "pod status: %s", describePod(pod))
	}
}

// assertCategory runs the pod analyzer and fails the test if no DiagnosisResult
// with the expected category is present in the output.
func assertCategory(t *testing.T, podName, wantCategory string) {
	t.Helper()

	diagnosis, err := analyzer.Analyze(context.Background(), env.Namespace, podName)
	if err != nil {
		t.Fatalf("analyzer.Analyze(%q): %v", podName, err)
	}

	for _, issue := range diagnosis.Issues {
		if issue.Category == wantCategory {
			tLog(t, "PASS: found expected category %q — %s", wantCategory, issue.Summary)
			return
		}
	}

	tLog(t, "Pod diagnosis output:\n%s", diagnosis.Format())
	t.Errorf("expected a DiagnosisResult with Category=%q, but none was found", wantCategory)

}

// waitForUnhealthyEvent blocks until the Kubernetes API reports at least one
// Unhealthy event for the named pod (fired when a liveness, readiness, or
// startup probe fails). It uses the EventList API so it works even when the
// pod object itself has not yet changed state.
func waitForUnhealthyEvent(t *testing.T, podName string, timeout time.Duration) {
	t.Helper()
	framework.WaitFor(t, timeout,
		func(ctx context.Context) (*corev1.EventList, error) {
			return clientset.CoreV1().Events(env.Namespace).List(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf(
					"involvedObject.name=%s,involvedObject.namespace=%s,reason=Unhealthy",
					podName, env.Namespace,
				),
			})
		},
		func(evs *corev1.EventList) bool {
			return len(evs.Items) > 0
		},
		func(evs *corev1.EventList) string {
			return fmt.Sprintf("waiting for Unhealthy event for pod %q (%d events so far)", podName, len(evs.Items))
		},
	)
	tLog(t, "pod %q has received an Unhealthy probe event", podName)
}

// waitForContainersNotReady blocks until the pod is Running and its
// ContainersReady condition is False, which is the observable state of a
// failing readiness probe (the container keeps running but receives no traffic).
func waitForContainersNotReady(t *testing.T, podName string, timeout time.Duration) {
	t.Helper()
	pod := framework.WaitFor(t, timeout,
		func(ctx context.Context) (*corev1.Pod, error) {
			return clientset.CoreV1().Pods(env.Namespace).Get(ctx, podName, metav1.GetOptions{})
		},
		func(p *corev1.Pod) bool {
			if p.Status.Phase != corev1.PodRunning {
				return false
			}
			for _, cond := range p.Status.Conditions {
				if cond.Type == corev1.ContainersReady && cond.Status == corev1.ConditionFalse {
					return true
				}
			}
			return false
		},
		func(p *corev1.Pod) string {
			return fmt.Sprintf("phase=%s %s", p.Status.Phase, describePod(p))
		},
	)
	if pod != nil {
		tLog(t, "pod %q is Running with ContainersReady=False", podName)
	}
}

// waitForUnschedulable blocks until the pod's PodScheduled condition is False,
// indicating the scheduler was unable to place it on any node.
func waitForUnschedulable(t *testing.T, podName string, timeout time.Duration) {
	t.Helper()
	pod := framework.WaitFor(t, timeout,
		func(ctx context.Context) (*corev1.Pod, error) {
			return clientset.CoreV1().Pods(env.Namespace).Get(ctx, podName, metav1.GetOptions{})
		},
		func(p *corev1.Pod) bool {
			for _, cond := range p.Status.Conditions {
				if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
					return true
				}
			}
			return false
		},
		func(p *corev1.Pod) string {
			for _, cond := range p.Status.Conditions {
				if cond.Type == corev1.PodScheduled {
					return fmt.Sprintf("phase=%s PodScheduled=%s: %s", p.Status.Phase, cond.Status, cond.Message)
				}
			}
			return fmt.Sprintf("phase=%s, no PodScheduled condition yet", p.Status.Phase)
		},
	)
	if pod != nil {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled {
				tLog(t, "scheduling condition: %s — %s", cond.Status, cond.Message)
				return
			}
		}
	}
}

// waitForInitContainerWaitingReason blocks until any init container in the pod
// has the given waiting reason (e.g. "ImagePullBackOff", "CrashLoopBackOff").
func waitForInitContainerWaitingReason(t *testing.T, podName, reason string, timeout time.Duration) {
	t.Helper()
	pod := framework.WaitFor(t, timeout,
		func(ctx context.Context) (*corev1.Pod, error) {
			return clientset.CoreV1().Pods(env.Namespace).Get(ctx, podName, metav1.GetOptions{})
		},
		func(p *corev1.Pod) bool {
			for _, cs := range p.Status.InitContainerStatuses {
				if cs.State.Waiting != nil && cs.State.Waiting.Reason == reason {
					return true
				}
			}
			return false
		},
		describeInitPod,
	)
	if pod != nil {
		tLog(t, "pod init container status: %s", describeInitPod(pod))
	}
}

// describePod formats all container states into a single log-friendly string,
// e.g.: bad-entrypoint→Terminated(StartError,exit=128) | app→Waiting(ImagePullBackOff)
func describePod(p *corev1.Pod) string {
	if len(p.Status.ContainerStatuses) == 0 {
		return fmt.Sprintf("phase=%s, no container statuses yet", p.Status.Phase)
	}
	parts := make([]string, 0, len(p.Status.ContainerStatuses))
	for _, cs := range p.Status.ContainerStatuses {
		switch {
		case cs.State.Waiting != nil:
			parts = append(parts, fmt.Sprintf("%s→Waiting(%s)", cs.Name, cs.State.Waiting.Reason))
		case cs.State.Running != nil:
			parts = append(parts, fmt.Sprintf("%s→Running", cs.Name))
		case cs.State.Terminated != nil:
			t := cs.State.Terminated
			parts = append(parts, fmt.Sprintf("%s→Terminated(%s,exit=%d)", cs.Name, t.Reason, t.ExitCode))
		default:
			parts = append(parts, fmt.Sprintf("%s→Unknown", cs.Name))
		}
	}
	return strings.Join(parts, " | ")
}

// describeInitPod formats all init container states into a single log-friendly
// string, e.g.: init→Waiting(CrashLoopBackOff) | setup→Terminated(Error,exit=1)
func describeInitPod(p *corev1.Pod) string {
	if len(p.Status.InitContainerStatuses) == 0 {
		return fmt.Sprintf("phase=%s, no init container statuses yet", p.Status.Phase)
	}
	parts := make([]string, 0, len(p.Status.InitContainerStatuses))
	for _, cs := range p.Status.InitContainerStatuses {
		switch {
		case cs.State.Waiting != nil:
			parts = append(parts, fmt.Sprintf("%s→Waiting(%s)", cs.Name, cs.State.Waiting.Reason))
		case cs.State.Running != nil:
			parts = append(parts, fmt.Sprintf("%s→Running", cs.Name))
		case cs.State.Terminated != nil:
			t := cs.State.Terminated
			parts = append(parts, fmt.Sprintf("%s→Terminated(%s,exit=%d)", cs.Name, t.Reason, t.ExitCode))
		default:
			parts = append(parts, fmt.Sprintf("%s→Unknown", cs.Name))
		}
	}
	return strings.Join(parts, " | ")
}
