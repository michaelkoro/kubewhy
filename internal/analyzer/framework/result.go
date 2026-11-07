package framework

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
