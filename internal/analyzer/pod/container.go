package pod

import (
	corev1 "k8s.io/api/core/v1"
)

// checkContainers inspects every regular container in the pod and returns
// issues for image pull errors, crash loops, OOMKilled, and runtime errors.
// Init containers are handled separately in init.go.
// Sub-check logic lives in checks.go and is shared with init container detection.
func checkContainers(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	var results []DiagnosisResult

	for i := range pod.Status.ContainerStatuses {
		cs := &pod.Status.ContainerStatuses[i]
		memLimit := containerMemoryLimit(cs.Name, pod.Spec.Containers)
		results = append(results, checkImagePull(cs, events)...)
		results = append(results, checkCrashLoop(cs)...)
		results = append(results, checkOOMKilled(cs, memLimit)...)
		results = append(results, checkRuntimeError(cs)...)
	}

	return results
}
