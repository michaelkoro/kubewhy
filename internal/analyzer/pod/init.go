package pod

import (
	corev1 "k8s.io/api/core/v1"
)

// checkInitContainers inspects every init container in the pod and returns
// issues for image pull errors, crash loops, OOMKilled, and runtime errors.
// A failing init container prevents all regular containers from starting, which
// is made explicit in every result's Details field.
func checkInitContainers(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	var results []DiagnosisResult

	for i := range pod.Status.InitContainerStatuses {
		cs := &pod.Status.InitContainerStatuses[i]
		results = append(results, initResultsFor(cs, pod, events)...)
	}

	return results
}

// initResultsFor runs the same sub-checks as checkContainers but for a single
// init container status. It resolves the memory limit from pod.Spec.InitContainers
// and post-processes every result to:
//   - prefix the Category with "Init" (e.g. "CrashLoop" → "InitCrashLoop")
//   - prepend a blocking-note to Details so the output makes the impact clear
func initResultsFor(cs *corev1.ContainerStatus, pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	memLimit := containerMemoryLimit(cs.Name, pod.Spec.InitContainers)

	var raw []DiagnosisResult
	raw = append(raw, checkImagePull(cs, events)...)
	raw = append(raw, checkCrashLoop(cs)...)
	raw = append(raw, checkOOMKilled(cs, memLimit)...)
	raw = append(raw, checkRuntimeError(cs)...)

	for i := range raw {
		raw[i].Category = "Init" + raw[i].Category
		raw[i].Details = "Init container failure — regular containers will not start until all init containers succeed.\n  " + raw[i].Details
	}

	return raw
}
