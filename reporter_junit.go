package chaoskit

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

// JUnitTestSuite represents JUnit XML test suite format
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// JUnitError represents a test error
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// GenerateJUnitXML converts report to JUnit XML format
func (r *Reporter) GenerateJUnitXML(report *Report) (string, error) {
	suite := JUnitTestSuite{
		Name:      report.ScenarioName,
		Tests:     report.TotalIterations,
		Failures:  len(report.CriticalFailures),
		Errors:    len(report.Warnings),
		Time:      report.Duration.Seconds(),
		Timestamp: report.ExecutionTime.Format(time.RFC3339),
		TestCases: make([]JUnitTestCase, 0),
	}

	// Add overall verdict as a test case
	verdictCase := JUnitTestCase{
		Name:      "chaos-test-verdict",
		Classname: "chaoskit." + report.ScenarioName,
		Time:      report.Duration.Seconds(),
	}

	switch report.Verdict {
	case VerdictFail:
		verdictCase.Failure = &JUnitFailure{
			Message: report.Summary,
			Type:    "VerdictFail",
			Content: formatFailuresForJUnit(report),
		}
	case VerdictUnstable:
		verdictCase.Error = &JUnitError{
			Message: report.Summary,
			Type:    "VerdictUnstable",
			Content: formatWarningsForJUnit(report),
		}
	}

	suite.TestCases = append(suite.TestCases, verdictCase)

	// Add individual validator results as test cases
	for _, failure := range report.CriticalFailures {
		testCase := JUnitTestCase{
			Name:      failure.ValidatorName,
			Classname: "chaoskit.validator",
			Time:      0, // Validators don't have individual duration
			Failure: &JUnitFailure{
				Message: failure.Message,
				Type:    "CriticalValidatorFailure",
				Content: fmt.Sprintf("Validator %s failed %d times\nFirst seen: %s\nLast seen: %s",
					failure.ValidatorName, failure.Occurrences,
					failure.FirstSeen.Format(time.RFC3339),
					failure.LastSeen.Format(time.RFC3339)),
			},
		}
		suite.TestCases = append(suite.TestCases, testCase)
	}

	for _, warning := range report.Warnings {
		testCase := JUnitTestCase{
			Name:      warning.ValidatorName,
			Classname: "chaoskit.validator",
			Time:      0,
			Error: &JUnitError{
				Message: warning.Message,
				Type:    "ValidatorWarning",
				Content: fmt.Sprintf("Validator %s produced %d warnings",
					warning.ValidatorName, warning.Occurrences),
			},
		}
		suite.TestCases = append(suite.TestCases, testCase)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JUnit XML: %w", err)
	}

	return xml.Header + string(output), nil
}

// SaveJUnitXML writes JUnit XML report to file
func (r *Reporter) SaveJUnitXML(report *Report, path string) error {
	xmlStr, err := r.GenerateJUnitXML(report)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(xmlStr), 0644)
}

func formatFailuresForJUnit(report *Report) string {
	var content string
	content += fmt.Sprintf("Verdict: %s\n", report.Verdict)
	content += fmt.Sprintf("Success Rate: %.2f%%\n", report.SuccessRate*100)
	content += fmt.Sprintf("Critical Failures: %d\n\n", len(report.CriticalFailures))

	for _, failure := range report.CriticalFailures {
		content += fmt.Sprintf("- %s: %s (%d times)\n",
			failure.ValidatorName, failure.Message, failure.Occurrences)
	}

	return content
}

func formatWarningsForJUnit(report *Report) string {
	var content string
	content += fmt.Sprintf("Verdict: %s\n", report.Verdict)
	content += fmt.Sprintf("Success Rate: %.2f%%\n", report.SuccessRate*100)
	content += fmt.Sprintf("Warnings: %d\n\n", len(report.Warnings))

	for _, warning := range report.Warnings {
		content += fmt.Sprintf("- %s: %s (%d times)\n",
			warning.ValidatorName, warning.Message, warning.Occurrences)
	}

	return content
}
