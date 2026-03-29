package pod

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// checkVolumes detects PVC and general volume mount failures by scanning pod
// events. ConfigMap/Secret mount failures are handled by checkConfig and are
// explicitly excluded here to avoid duplicate reports.
func checkVolumes(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	if events == nil {
		return nil
	}

	seen := map[string]bool{}
	var results []DiagnosisResult

	for _, ev := range events.Items {
		switch ev.Reason {
		case "FailedMount":
			msg := strings.ToLower(ev.Message)
			if strings.Contains(msg, "configmap") || strings.Contains(msg, "secret") {
				continue
			}
		case "FailedAttachVolume":
		default:
			continue
		}

		if seen[ev.Message] {
			continue
		}
		seen[ev.Message] = true

		results = append(results, DiagnosisResult{
			Severity:   SeverityError,
			Category:   "VolumeMount",
			Summary:    "Volume mount or attach failed",
			Details:    "A volume could not be mounted or attached to the pod.\n  " + ev.Message,
			Suggestion: volumeSuggestion(ev.Reason, ev.Message),
		})
	}

	return results
}

// volumeSuggestion returns a targeted fix recommendation based on the event
// reason and message content.
func volumeSuggestion(reason, message string) string {
	msg := strings.ToLower(message)

	if reason == "FailedAttachVolume" {
		return "Check that the persistent volume exists and is not attached to another node. " +
			"For cloud volumes, verify the volume is in the same availability zone as the node."
	}

	switch {
	case strings.Contains(msg, "persistentvolumeclaim") || strings.Contains(msg, "pvc"):
		return "Verify that the PersistentVolumeClaim exists and is bound. " +
			"Use `kubectl get pvc` to check status. If the PVC is Pending, the StorageClass " +
			"may not have a provisioner or there may be no matching PersistentVolume."
	case strings.Contains(msg, "hostpath") || strings.Contains(msg, "host path"):
		return "The hostPath volume could not be mounted. Verify the path exists on the node " +
			"and the kubelet has permission to access it."
	default:
		return "Check that the volume referenced in the pod spec exists and is accessible. " +
			"Run `kubectl describe pod <pod>` for detailed mount error information."
	}
}
