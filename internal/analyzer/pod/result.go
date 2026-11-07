package pod

import (
	"fmt"

	"github.com/michaelkoro/kubewhy/internal/analyzer/framework"
)

// PodDiagnosis is the complete analysis result for a single pod.
type PodDiagnosis struct {
	PodName   string
	Namespace string
	Phase     string
	Issues    []framework.DiagnosisResult
	IsHealthy bool
}

// hasIssueBySeverity returns true if any diagnosed issue matches the given severity.
func (d *PodDiagnosis) hasIssueBySeverity(s framework.Severity) bool {
	for _, issue := range d.Issues {
		if issue.Severity == s {
			return true
		}
	}
	return false
}

// HasErrors returns true if any diagnosed issue has Error severity.
func (d *PodDiagnosis) HasErrors() bool { return d.hasIssueBySeverity(framework.SeverityError) }

// HasWarnings returns true if any diagnosed issue has Warning severity.
func (d *PodDiagnosis) HasWarnings() bool { return d.hasIssueBySeverity(framework.SeverityWarning) }

// Format renders the diagnosis as human-readable plain-English output,
// matching the output format described in the implementation plan.
func (d *PodDiagnosis) Format() string {
	var out string

	out += fmt.Sprintf("POD: %s (namespace: %s)\n", d.PodName, d.Namespace)
	out += fmt.Sprintf("Status: %s\n", d.Phase)

	if d.IsHealthy {
		out += "\nNo issues found. The pod appears to be healthy.\n"
		return out
	}

	for _, issue := range d.Issues {
		out += fmt.Sprintf("\n[%s] %s\n", issue.Severity, issue.Category)
		out += fmt.Sprintf("  %s\n", issue.Summary)
		if issue.Details != "" {
			out += fmt.Sprintf("\n  %s\n", issue.Details)
		}
		if issue.Suggestion != "" {
			out += fmt.Sprintf("\n  Suggestion: %s\n", issue.Suggestion)
		}
	}

	return out
}
