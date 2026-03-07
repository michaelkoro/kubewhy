package pod

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// containerMemoryLimit returns the memory limit string for the named container
// from the provided container spec slice, or empty if no limit is set.
// Pass pod.Spec.Containers for regular containers, pod.Spec.InitContainers for
// init containers.
func containerMemoryLimit(containerName string, containers []corev1.Container) string {
	for _, c := range containers {
		if c.Name != containerName {
			continue
		}
		if limit, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
			return limit.String()
		}
	}
	return ""
}

// checkImagePull detects image pull failures for a single container status.
func checkImagePull(cs *corev1.ContainerStatus, events *corev1.EventList) []DiagnosisResult {
	waiting := cs.State.Waiting
	if waiting == nil {
		return nil
	}

	switch waiting.Reason {
	case "ImagePullBackOff", "ErrImagePull", "ErrImageNeverPull":
	default:
		return nil
	}

	details := fmt.Sprintf("Container %q cannot pull its image.", cs.Name)
	if waiting.Message != "" {
		details += "\n  " + waiting.Message
	}

	// Enrich with event messages — they often carry the real error (auth
	// failure, image not found, rate limit, etc.)
	if hint := imagePullEventHint(cs.Name, events); hint != "" {
		details += "\n  " + hint
	}

	suggestion := "Check that the image name and tag are correct. " +
		"If the image is private, verify that an imagePullSecret is configured " +
		"and the credentials are valid."
	if waiting.Reason == "ErrImageNeverPull" {
		suggestion = "The image pull policy is set to Never but the image is not " +
			"present on the node. Either pre-pull the image or change the pull policy."
	}

	return []DiagnosisResult{{
		Severity:   SeverityError,
		Category:   "ImagePull",
		Summary:    fmt.Sprintf("Container %q cannot pull image (%s)", cs.Name, waiting.Reason),
		Details:    details,
		Suggestion: suggestion,
	}}
}

// imagePullEventHint scans events for pull-related messages and returns the
// most informative one as a one-line hint string.
func imagePullEventHint(containerName string, events *corev1.EventList) string {
	if events == nil {
		return ""
	}
	for _, ev := range events.Items {
		if ev.Reason != "Failed" && ev.Reason != "BackOff" {
			continue
		}
		msg := strings.ToLower(ev.Message)
		if strings.Contains(msg, "unauthorized") || strings.Contains(msg, "authentication") {
			return "Registry authentication failed — check your imagePullSecret."
		}
		if strings.Contains(msg, "not found") || strings.Contains(msg, "manifest unknown") {
			return "Image or tag not found in the registry — verify the image name and tag."
		}
		if strings.Contains(msg, "rate") {
			return "Registry rate limit exceeded — wait and retry, or authenticate to increase limits."
		}
		if ev.Message != "" {
			return ev.Message
		}
	}
	return ""
}

// checkCrashLoop detects containers stuck in CrashLoopBackOff and surfaces
// the exit code from the last termination to help narrow down the root cause.
func checkCrashLoop(cs *corev1.ContainerStatus) []DiagnosisResult {
	waiting := cs.State.Waiting
	if waiting == nil || waiting.Reason != "CrashLoopBackOff" {
		return nil
	}

	details := fmt.Sprintf(
		"Container %q has restarted %d time(s) and keeps crashing.",
		cs.Name, cs.RestartCount,
	)

	if last := cs.LastTerminationState.Terminated; last != nil {
		details += fmt.Sprintf(
			"\n  Last exit: code %d", last.ExitCode,
		)
		if last.Message != "" {
			details += " — " + last.Message
		}
		if hint := exitCodeHint(last.ExitCode); hint != "" {
			details += "\n  " + hint
		}
	}

	return []DiagnosisResult{{
		Severity: SeverityError,
		Category: "CrashLoop",
		Summary:  fmt.Sprintf("Container %q is in CrashLoopBackOff (%d restarts)", cs.Name, cs.RestartCount),
		Details:  details,
		Suggestion: "Check the container logs for the crash reason: " +
			"`kubectl logs <pod> -c " + cs.Name + " --previous`. " +
			"Common causes: application error, missing env var, misconfigured command/args.",
	}}
}

// exitCodeHint maps well-known exit codes to a human-readable explanation.
func exitCodeHint(code int32) string {
	switch code {
	case 1:
		return "Exit code 1 usually means the application encountered an unhandled error."
	case 2:
		return "Exit code 2 often indicates misuse of a shell built-in or a missing argument."
	case 126:
		return "Exit code 126: command found but not executable (check file permissions)."
	case 127:
		return "Exit code 127: command not found — verify the entrypoint / CMD in the image."
	case 128:
		return "Exit code 128: invalid argument to exit."
	case 130:
		return "Exit code 130: process terminated by Ctrl-C (SIGINT)."
	case 137:
		return "Exit code 137 (128+9): process killed by SIGKILL — likely OOMKilled or force-deleted."
	case 139:
		return "Exit code 139 (128+11): segmentation fault (SIGSEGV) — application crash."
	case 143:
		return "Exit code 143 (128+15): process terminated by SIGTERM — check prestop hooks and graceful shutdown."
	}
	return ""
}

// checkOOMKilled reports when a container was killed by the kernel OOM killer,
// either in its current state or its most recent termination. The caller is
// responsible for resolving memLimit from the appropriate container spec slice
// (pod.Spec.Containers for regular containers, pod.Spec.InitContainers for
// init containers) using containerMemoryLimit.
func checkOOMKilled(cs *corev1.ContainerStatus, memLimit string) []DiagnosisResult {
	isOOM := func(t *corev1.ContainerStateTerminated) bool {
		return t != nil && t.Reason == "OOMKilled"
	}

	if !isOOM(cs.State.Terminated) && !isOOM(cs.LastTerminationState.Terminated) {
		return nil
	}

	details := fmt.Sprintf("Container %q exceeded its memory limit and was killed by the kernel.", cs.Name)
	if memLimit != "" {
		details += fmt.Sprintf("\n  Configured memory limit: %s.", memLimit)
	} else {
		details += "\n  No memory limit is configured — the container consumed too much node memory."
	}

	return []DiagnosisResult{{
		Severity: SeverityError,
		Category: "OOMKilled",
		Summary:  fmt.Sprintf("Container %q was OOMKilled (out of memory)", cs.Name),
		Details:  details,
		Suggestion: "Increase the memory limit in the pod spec, or profile the application " +
			"to reduce its memory footprint. Run `kubectl top pod <pod>` to observe live usage.",
	}}
}

// checkRuntimeError detects containers that cannot start due to runtime-level
// problems such as a missing entrypoint or an admission-rejected security context.
// It covers two distinct Kubernetes representations of the same class of failure:
//   - Waiting{Reason:"ContainerCannotRun"|"CreateContainerError"} — set by kubelet
//     before the container process is ever launched.
//   - Terminated{Reason:"StartError"} — set by the container runtime (e.g.
//     containerd) when exec fails (e.g. binary not found, ENOENT).
func checkRuntimeError(cs *corev1.ContainerStatus) []DiagnosisResult {
	var reason, summary, details, suggestion string

	switch {
	case cs.State.Waiting != nil && cs.State.Waiting.Reason == "ContainerCannotRun":
		reason = cs.State.Waiting.Reason
		summary = fmt.Sprintf("Container %q cannot start: entrypoint not found or not executable", cs.Name)
		details = fmt.Sprintf("Container %q runtime error: %s", cs.Name, reason)
		if cs.State.Waiting.Message != "" {
			details += "\n  " + cs.State.Waiting.Message
		}
		suggestion = "Verify that the command / entrypoint in the pod spec exists inside the image " +
			"and has execute permissions."

	case cs.State.Waiting != nil && cs.State.Waiting.Reason == "CreateContainerError":
		reason = cs.State.Waiting.Reason
		summary = fmt.Sprintf("Container %q failed to create", cs.Name)
		details = fmt.Sprintf("Container %q runtime error: %s", cs.Name, reason)
		if cs.State.Waiting.Message != "" {
			details += "\n  " + cs.State.Waiting.Message
		}
		suggestion = "Check the security context settings (runAsUser, privileged, capabilities). " +
			"An admission webhook or PodSecurityPolicy may be rejecting the container."

	case cs.State.Terminated != nil && cs.State.Terminated.Reason == "StartError":
		t := cs.State.Terminated
		reason = t.Reason
		summary = fmt.Sprintf("Container %q cannot start: entrypoint not found or not executable", cs.Name)
		details = fmt.Sprintf("Container %q runtime error: %s (exit code %d)", cs.Name, reason, t.ExitCode)
		if t.Message != "" {
			details += "\n  " + t.Message
		}
		suggestion = "Verify that the command / entrypoint in the pod spec exists inside the image " +
			"and has execute permissions."

	default:
		return nil
	}

	return []DiagnosisResult{{
		Severity:   SeverityError,
		Category:   "RuntimeError",
		Summary:    summary,
		Details:    details,
		Suggestion: suggestion,
	}}
}
