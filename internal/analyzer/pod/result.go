package pod

import "fmt"

// Severity indicates the impact level of a diagnosed issue.
type Severity int

const (
	SeverityError   Severity = iota // Pod is broken or cannot start
	SeverityWarning                 // Pod is degraded but partially functional
	SeverityInfo                    // Noteworthy but not necessarily a problem
)

// String returns the display label for a Severity level.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// DiagnosisResult represents a single diagnosed issue found in a pod.
type DiagnosisResult struct {
	Severity   Severity // Error, Warning, or Info
	Category   string   // e.g. "Scheduling", "ImagePull", "CrashLoop"
	Summary    string   // One-line explanation shown in the header
	Details    string   // Extended explanation of the root cause
	Suggestion string   // Actionable fix recommendation
}

// PodDiagnosis is the complete analysis result for a single pod.
type PodDiagnosis struct {
	PodName   string
	Namespace string
	Phase     string
	Issues    []DiagnosisResult
	IsHealthy bool
}

// hasIssueBySeverity returns true if any diagnosed issue matches the given severity.
func (d *PodDiagnosis) hasIssueBySeverity(s Severity) bool {
	for _, issue := range d.Issues {
		if issue.Severity == s {
			return true
		}
	}
	return false
}

// HasErrors returns true if any diagnosed issue has Error severity.
func (d *PodDiagnosis) HasErrors() bool { return d.hasIssueBySeverity(SeverityError) }

// HasWarnings returns true if any diagnosed issue has Warning severity.
func (d *PodDiagnosis) HasWarnings() bool { return d.hasIssueBySeverity(SeverityWarning) }

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
