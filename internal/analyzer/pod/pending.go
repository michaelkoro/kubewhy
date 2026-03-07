package pod

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// checkPending is the top-level check for scheduling failures. It is a no-op
// unless the pod is in the Pending phase.
func checkPending(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	if pod.Status.Phase != corev1.PodPending {
		return nil
	}
	return checkScheduling(pod, events)
}

// checkScheduling looks for a PodScheduled=False condition and returns a
// DiagnosisResult describing why the pod cannot be placed on any node.
func checkScheduling(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	var schedCond *corev1.PodCondition
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == corev1.PodScheduled {
			schedCond = &pod.Status.Conditions[i]
			break
		}
	}

	if schedCond == nil || schedCond.Status != corev1.ConditionFalse {
		return nil
	}

	details := "The pod cannot be scheduled to any node."
	if schedCond.Message != "" {
		details += "\n  " + schedCond.Message
	}
	if hint := schedulingEventHint(events); hint != "" {
		details += "\n  " + hint
	}

	return []DiagnosisResult{{
		Severity:   SeverityError,
		Category:   "Scheduling",
		Summary:    "Pod cannot be scheduled",
		Details:    details,
		Suggestion: schedulingSuggestion(schedCond.Message, events),
	}}
}

// schedulingEventHint returns the first non-empty message from a
// FailedScheduling event, which typically contains the full scheduler reason.
func schedulingEventHint(events *corev1.EventList) string {
	if events == nil {
		return ""
	}
	for _, ev := range events.Items {
		if ev.Reason == "FailedScheduling" && ev.Message != "" {
			return ev.Message
		}
	}
	return ""
}

// schedulingSuggestion returns a targeted fix recommendation by matching the
// condition message and FailedScheduling event messages against known patterns.
func schedulingSuggestion(conditionMsg string, events *corev1.EventList) string {
	combined := strings.ToLower(conditionMsg)
	if events != nil {
		for _, ev := range events.Items {
			if ev.Reason == "FailedScheduling" {
				combined += " " + strings.ToLower(ev.Message)
			}
		}
	}

	switch {
	case strings.Contains(combined, "insufficient cpu"):
		return "Reduce the CPU request in the pod spec, or add nodes with more available CPU capacity."
	case strings.Contains(combined, "insufficient memory"):
		return "Reduce the memory request in the pod spec, or add nodes with more available memory."
	case strings.Contains(combined, "node affinity") || strings.Contains(combined, "didn't match pod affinity"):
		return "Check the nodeAffinity / podAffinity rules in the pod spec. No node matches the required labels."
	case strings.Contains(combined, "taint") || strings.Contains(combined, "tolerat"):
		return "The pod does not tolerate the taints on available nodes. Add a matching toleration or remove the taint."
	case strings.Contains(combined, "unschedulable"):
		return "All nodes are marked as unschedulable (cordoned). Check node conditions with `kubectl describe node`."
	default:
		return "Run `kubectl describe pod <pod>` for full scheduling details. " +
			"Check node capacity, affinity rules, and taints/tolerations."
	}
}
