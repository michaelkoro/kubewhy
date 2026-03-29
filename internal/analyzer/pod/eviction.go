package pod

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// checkEviction detects pods that were evicted due to node resource pressure
// (DiskPressure, MemoryPressure, PIDPressure, etc.). Evicted pods have
// Phase=Failed and Reason="Evicted" with a message describing the pressure.
func checkEviction(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	if pod.Status.Phase != corev1.PodFailed || pod.Status.Reason != "Evicted" {
		return nil
	}

	details := "The pod was evicted from its node due to resource pressure."
	if pod.Status.Message != "" {
		details += "\n  " + pod.Status.Message
	}

	if hint := evictionEventHint(events); hint != "" {
		details += "\n  " + hint
	}

	return []DiagnosisResult{{
		Severity:   SeverityError,
		Category:   "Eviction",
		Summary:    fmt.Sprintf("Pod was evicted (%s)", evictionPressure(pod.Status.Message)),
		Details:    details,
		Suggestion: evictionSuggestion(pod.Status.Message),
	}}
}

// evictionEventHint returns the first Evicted event message if present.
func evictionEventHint(events *corev1.EventList) string {
	if events == nil {
		return ""
	}
	for _, ev := range events.Items {
		if ev.Reason == "Evicted" && ev.Message != "" {
			return ev.Message
		}
	}
	return ""
}

// evictionPressure extracts a short pressure label from the eviction message.
func evictionPressure(message string) string {
	msg := strings.ToLower(message)
	switch {
	case strings.Contains(msg, "diskpressure") || strings.Contains(msg, "ephemeral-storage"):
		return "DiskPressure"
	case strings.Contains(msg, "memorypressure") || strings.Contains(msg, "memory"):
		return "MemoryPressure"
	case strings.Contains(msg, "pidpressure") || strings.Contains(msg, "pid"):
		return "PIDPressure"
	default:
		return "node pressure"
	}
}

// evictionSuggestion returns a targeted fix recommendation based on the
// eviction message.
func evictionSuggestion(message string) string {
	msg := strings.ToLower(message)
	switch {
	case strings.Contains(msg, "diskpressure") || strings.Contains(msg, "ephemeral-storage"):
		return "The node ran out of disk space. Clean up unused images and containers, " +
			"or add nodes with more disk capacity. Check node conditions with " +
			"`kubectl describe node <node>`."
	case strings.Contains(msg, "memorypressure") || strings.Contains(msg, "memory"):
		return "The node ran out of memory. Set appropriate memory requests and limits " +
			"on your pods to prevent overcommit, or add nodes with more memory. " +
			"Check node conditions with `kubectl describe node <node>`."
	case strings.Contains(msg, "pidpressure") || strings.Contains(msg, "pid"):
		return "The node ran out of process IDs. Investigate containers that may be " +
			"spawning too many processes. Check node conditions with " +
			"`kubectl describe node <node>`."
	default:
		return "Check the node conditions to understand the resource pressure: " +
			"`kubectl describe node <node>`. Consider setting resource requests and limits " +
			"to prevent overcommit."
	}
}
