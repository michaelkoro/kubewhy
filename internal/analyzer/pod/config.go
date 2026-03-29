package pod

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// checkConfig detects missing ConfigMap and Secret references. It covers two
// distinct failure modes:
//   - Container waiting with reason "CreateContainerConfigError" — an env
//     valueFrom references a ConfigMap or Secret that does not exist.
//   - Events with reason "FailedMount" whose message mentions "configmap" or
//     "secret" — a volume-projected ConfigMap/Secret is missing.
func checkConfig(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	var results []DiagnosisResult
	results = append(results, checkConfigEnv(pod)...)
	results = append(results, checkConfigMount(events)...)
	return results
}

// checkConfigEnv inspects container statuses for CreateContainerConfigError,
// which the kubelet sets when an env valueFrom references a missing ConfigMap
// or Secret key.
func checkConfigEnv(pod *corev1.Pod) []DiagnosisResult {
	var results []DiagnosisResult

	allStatuses := append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...)
	for i := range allStatuses {
		cs := &allStatuses[i]
		if cs.State.Waiting == nil || cs.State.Waiting.Reason != "CreateContainerConfigError" {
			continue
		}

		details := fmt.Sprintf("Container %q cannot start because a referenced ConfigMap or Secret does not exist.", cs.Name)
		if cs.State.Waiting.Message != "" {
			details += "\n  " + cs.State.Waiting.Message
		}

		results = append(results, DiagnosisResult{
			Severity: SeverityError,
			Category: "ConfigError",
			Summary:  fmt.Sprintf("Container %q has a missing ConfigMap or Secret reference", cs.Name),
			Details:  details,
			Suggestion: "Check that all ConfigMaps and Secrets referenced in the pod spec (env valueFrom, " +
				"envFrom) exist in the same namespace. Use `kubectl get configmaps` and `kubectl get secrets` to verify.",
		})
	}

	return results
}

// checkConfigMount scans events for FailedMount where the message indicates a
// missing ConfigMap or Secret (as opposed to a PVC, which is handled by
// checkVolumes).
func checkConfigMount(events *corev1.EventList) []DiagnosisResult {
	if events == nil {
		return nil
	}

	seen := map[string]bool{}
	var results []DiagnosisResult

	for _, ev := range events.Items {
		if ev.Reason != "FailedMount" {
			continue
		}
		msg := strings.ToLower(ev.Message)
		if !strings.Contains(msg, "configmap") && !strings.Contains(msg, "secret") {
			continue
		}
		if seen[ev.Message] {
			continue
		}
		seen[ev.Message] = true

		results = append(results, DiagnosisResult{
			Severity: SeverityError,
			Category: "ConfigError",
			Summary:  "Volume mount failed due to missing ConfigMap or Secret",
			Details:  "A volume-projected ConfigMap or Secret could not be mounted.\n  " + ev.Message,
			Suggestion: "Verify that the ConfigMap or Secret referenced in the pod's volumes section exists " +
				"in the same namespace. Use `kubectl get configmaps` and `kubectl get secrets` to check.",
		})
	}

	return results
}
