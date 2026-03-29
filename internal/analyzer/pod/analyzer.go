package pod

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/michaelkoro/kubewhy/internal/k8s"
)

// PodAnalyzer orchestrates the full pod diagnosis by running all
// diagnostic checks against a single pod and its events.
type PodAnalyzer struct {
	client *k8s.Client
}

// NewAnalyzer creates a PodAnalyzer backed by the given Kubernetes client.
func NewAnalyzer(client *k8s.Client) *PodAnalyzer {
	return &PodAnalyzer{client: client}
}

// Analyze fetches the named pod and its events, runs all diagnostic checks,
// and returns a PodDiagnosis summarising every issue found.
func (a *PodAnalyzer) Analyze(ctx context.Context, namespace, podName string) (*PodDiagnosis, error) {
	pod, err := a.client.GetPod(ctx, namespace, podName)
	if err != nil {
		return nil, fmt.Errorf("fetching pod %s/%s: %w", namespace, podName, err)
	}

	eventList, err := a.client.GetEvents(ctx, namespace, podName)
	if err != nil {
		return nil, fmt.Errorf("fetching events for %s/%s: %w", namespace, podName, err)
	}

	issues := a.runChecks(pod, eventList)

	return &PodDiagnosis{
		PodName:   podName,
		Namespace: namespace,
		Phase:     string(pod.Status.Phase),
		Issues:    issues,
		IsHealthy: len(issues) == 0,
	}, nil
}

// runChecks executes every diagnostic check in priority order and collects
// results. Each checker is a plain function defined in its own source file.
//
// Order matters: higher-severity / more fundamental problems (scheduling,
// image pull) are checked first so they appear at the top of the output.
func (a *PodAnalyzer) runChecks(pod *corev1.Pod, events *corev1.EventList) []DiagnosisResult {
	checkers := []func(*corev1.Pod, *corev1.EventList) []DiagnosisResult{
		checkPending,        // pending.go   — scheduling failures
		checkContainers,    // container.go — image pull, crash loop, OOMKilled, runtime errors
		checkInitContainers, // init.go      — init container failures
		checkProbes,         // probes.go    — liveness / readiness probe failures
		checkConfig,         // config.go    — missing ConfigMap / Secret
		checkVolumes,        // volume.go    — PVC and mount issues
		checkEviction,       // eviction.go  — node pressure eviction
		checkTerminating,    // terminating.go — stuck Terminating pods
	}

	var issues []DiagnosisResult
	for _, check := range checkers {
		issues = append(issues, check(pod, events)...)
	}
	return issues
}
