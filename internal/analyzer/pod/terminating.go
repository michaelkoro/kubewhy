package pod

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// checkTerminating detects pods that are stuck in the Terminating state. A pod
// is stuck when it has a DeletionTimestamp but has not been removed, typically
// because a finalizer is blocking deletion.
func checkTerminating(pod *corev1.Pod, _ *corev1.EventList) []DiagnosisResult {
	if pod.DeletionTimestamp == nil {
		return nil
	}

	gracePeriod := int64(30)
	if pod.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriod = *pod.Spec.TerminationGracePeriodSeconds
	}

	stuckFor := time.Since(pod.DeletionTimestamp.Time)
	if stuckFor < time.Duration(gracePeriod)*time.Second {
		return nil
	}

	details := fmt.Sprintf(
		"The pod has been terminating for %s (grace period: %ds).",
		stuckFor.Round(time.Second), gracePeriod,
	)

	if len(pod.Finalizers) > 0 {
		details += fmt.Sprintf(
			"\n  Blocking finalizers: %s",
			strings.Join(pod.Finalizers, ", "),
		)
	}

	return []DiagnosisResult{{
		Severity:   SeverityWarning,
		Category:   "StuckTerminating",
		Summary:    fmt.Sprintf("Pod stuck terminating for %s", stuckFor.Round(time.Second)),
		Details:    details,
		Suggestion: terminatingSuggestion(pod.Finalizers),
	}}
}

// terminatingSuggestion returns advice based on whether finalizers are present.
func terminatingSuggestion(finalizers []string) string {
	if len(finalizers) > 0 {
		return "The pod has finalizers that must be resolved before deletion completes. " +
			"Investigate the controller managing the finalizer(s): " +
			strings.Join(finalizers, ", ") + ". " +
			"If the controller is gone, you can remove the finalizer with " +
			"`kubectl patch pod <pod> -p '{\"metadata\":{\"finalizers\":null}}'` " +
			"or force-delete with `kubectl delete pod <pod> --force --grace-period=0`."
	}
	return "The pod has no finalizers but is still stuck. A container may not be " +
		"responding to SIGTERM. Try force-deleting: " +
		"`kubectl delete pod <pod> --force --grace-period=0`."
}
