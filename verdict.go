package chaoskit

// Verdict represents the overall test outcome
type Verdict int

const (
	// VerdictPass indicates all critical validators passed
	VerdictPass Verdict = iota

	// VerdictUnstable indicates warnings detected but no critical failures
	VerdictUnstable

	// VerdictFail indicates critical validators failed
	VerdictFail
)

// String returns human-readable verdict
func (v Verdict) String() string {
	switch v {
	case VerdictPass:
		return "PASS"
	case VerdictUnstable:
		return "UNSTABLE"
	case VerdictFail:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

// ExitCode returns appropriate exit code for CI/CD
// Pass=0, Unstable=0, Fail=1
func (v Verdict) ExitCode() int {
	switch v {
	case VerdictPass, VerdictUnstable:
		return 0
	case VerdictFail:
		return 1
	default:
		return 1
	}
}

// ValidationSeverity indicates how critical a validator failure is
type ValidationSeverity int

const (
	// SeverityCritical blocks release - must be fixed
	SeverityCritical ValidationSeverity = iota

	// SeverityWarning should be investigated but doesn't block release
	SeverityWarning

	// SeverityInfo is informational only
	SeverityInfo
)

// String returns human-readable severity
func (s ValidationSeverity) String() string {
	switch s {
	case SeverityCritical:
		return "CRITICAL"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}
