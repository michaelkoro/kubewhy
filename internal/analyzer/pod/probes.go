package pod

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// checkProbes detects liveness, readiness, and startup probe failures for a
// Running pod. It scans Unhealthy events first because liveness probe failures
// do not set ContainersReady=False — that condition is readiness-only. The
// condition check is used only as a fallback when events have expired.
func checkProbes(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	if pod.Status.Phase != corev1.PodRunning {
		return nil
	}

	if results := probeResultsFromEvents(events); len(results) > 0 {
		return results
	}

	if hasFailingReadyCondition(pod) {
		return []DiagnosisResult{{
			Severity: SeverityWarning,
			Category: "ProbeFailure",
			Summary:  "Pod is Running but not ready",
			Details: "The pod phase is Running but the Ready or ContainersReady condition is False.\n" +
				"  No Unhealthy probe events were found; they may have expired or the container is still initializing.",
			Suggestion: "Run `kubectl describe pod <pod>` to inspect probe configuration and recent events. " +
				"Check container logs for startup errors.",
		}}
	}

	return nil
}

// hasFailingReadyCondition returns true if the pod has a ContainersReady or
// PodReady condition with Status=False.
func hasFailingReadyCondition(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type != corev1.ContainersReady && cond.Type != corev1.PodReady {
			continue
		}
		if cond.Status == corev1.ConditionFalse {
			return true
		}
	}
	return false
}

// probeResultsFromEvents scans the event list for Unhealthy events, parses
// each into a probe type and container name, deduplicates by (container,
// probeType), and returns one DiagnosisResult per unique failure.
func probeResultsFromEvents(events *corev1.EventList) []DiagnosisResult {
	if events == nil {
		return nil
	}

	type key struct{ container, probeType string }
	seen := map[key]bool{}
	var results []DiagnosisResult

	for _, ev := range events.Items {
		if ev.Reason != "Unhealthy" {
			continue
		}

		probeType, containerName := parseUnhealthyEvent(ev)
		k := key{containerName, probeType}
		if seen[k] {
			continue
		}
		seen[k] = true

		nameLabel := containerName
		if nameLabel == "" {
			nameLabel = "<unknown>"
		}

		switch probeType {
		case "Liveness":
			results = append(results, DiagnosisResult{
				Severity: SeverityError,
				Category: "ProbeFailure",
				Summary:  fmt.Sprintf("Container %q liveness probe is failing", nameLabel),
				Details: fmt.Sprintf(
					"The liveness probe for container %q is failing.\n  %s\n"+
						"  Kubernetes will restart the container once the failure threshold is exceeded.",
					nameLabel, ev.Message,
				),
				Suggestion: "Check the liveness probe configuration (path, port, command, thresholds) with " +
					"`kubectl describe pod <pod>`. Review application logs: " +
					"`kubectl logs <pod> -c " + nameLabel + " --previous`.",
			})

		case "Readiness":
			results = append(results, DiagnosisResult{
				Severity: SeverityWarning,
				Category: "ProbeFailure",
				Summary:  fmt.Sprintf("Container %q readiness probe is failing", nameLabel),
				Details: fmt.Sprintf(
					"The readiness probe for container %q is failing.\n  %s\n"+
						"  The pod is excluded from Service endpoints and will receive no traffic. "+
						"The container is not restarted.",
					nameLabel, ev.Message,
				),
				Suggestion: "Check the readiness probe configuration with `kubectl describe pod <pod>`. " +
					"The application may still be initializing. Review logs: " +
					"`kubectl logs <pod> -c " + nameLabel + "`.",
			})

		default:
			// Startup probe or unrecognised probe type.
			results = append(results, DiagnosisResult{
				Severity: SeverityWarning,
				Category: "ProbeFailure",
				Summary:  fmt.Sprintf("Container %q probe is failing (%s)", nameLabel, probeType),
				Details: fmt.Sprintf(
					"A %s probe for container %q is failing.\n  %s",
					strings.ToLower(probeType), nameLabel, ev.Message,
				),
				Suggestion: "Check probe configuration with `kubectl describe pod <pod>` and review application logs.",
			})
		}
	}

	return results
}

// parseUnhealthyEvent extracts the probe type and container name from a
// Kubernetes Unhealthy event.
//
// The Message field has the form:
//
//	"Liveness probe failed: ..."
//	"Readiness probe failed: ..."
//	"Startup probe failed: ..."
//
// The container name is encoded in InvolvedObject.FieldPath as:
//
//	"spec.containers{my-container}"
//	"spec.initContainers{my-init}"
func parseUnhealthyEvent(ev corev1.Event) (probeType, containerName string) {
	msg := strings.ToLower(ev.Message)
	switch {
	case strings.HasPrefix(msg, "liveness"):
		probeType = "Liveness"
	case strings.HasPrefix(msg, "readiness"):
		probeType = "Readiness"
	case strings.HasPrefix(msg, "startup"):
		probeType = "Startup"
	default:
		probeType = "Unknown"
	}

	fp := ev.InvolvedObject.FieldPath
	if i := strings.Index(fp, "{"); i != -1 && strings.HasSuffix(fp, "}") {
		containerName = fp[i+1 : len(fp)-1]
	}
	return
}
