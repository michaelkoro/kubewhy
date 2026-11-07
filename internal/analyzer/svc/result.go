package svc

import "github.com/michaelkoro/kubewhy/internal/analyzer/framework"

type SvcDiagnosis struct {
	ServiceName string
	Namespace   string
	Issues      []framework.DiagnosisResult
	IsHealthy   bool
}
