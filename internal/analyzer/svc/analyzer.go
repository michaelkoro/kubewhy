package svc

import (
	"context"

	"github.com/michaelkoro/kubewhy/internal/k8s"
)

// SvcAnalyzer orchestrates the full svc diagnosis by running all
// diagnostic checks against a single svc and its events.
type SvcAnalyzer struct {
	client *k8s.Client
}

// NewAnalyzer creates a SvcAnalyzer backed by the given Kubernetes client.
func NewAnalyzer(client *k8s.Client) *SvcAnalyzer {
	return &SvcAnalyzer{client: client}
}

func (a *SvcAnalyzer) Analyze(ctx context.Context, ns, name string) (*SvcDiagnosis, error) {
	return nil, nil
}
