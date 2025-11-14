package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rom8726/chaoskit"
)

func main() {
	var (
		filePath = flag.String("file", "", "Path to JUnit XML report file")
		verbose  = flag.Bool("verbose", false, "Show detailed information for each test case")
	)
	flag.Parse()

	if *filePath == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s -file <path-to-junit-xml>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Read XML file
	data, err := os.ReadFile(*filePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse XML
	var suite chaoskit.JUnitTestSuite
	if err := xml.Unmarshal(data, &suite); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
		os.Exit(1)
	}

	// Display report
	displayReport(&suite, *verbose)
}

// TestCaseVerdict represents the verdict for a single test case
type TestCaseVerdict int

const (
	VerdictPass TestCaseVerdict = iota
	VerdictUnstable
	VerdictFail
)

func getTestCaseVerdict(testCase *chaoskit.JUnitTestCase) TestCaseVerdict {
	if testCase.Failure != nil {
		return VerdictFail
	}
	if testCase.Error != nil {
		return VerdictUnstable
	}

	return VerdictPass
}

func calculateOverallVerdict(testCases []chaoskit.JUnitTestCase) (TestCaseVerdict, int, int, int) {
	passCount := 0
	unstableCount := 0
	failCount := 0

	for _, testCase := range testCases {
		verdict := getTestCaseVerdict(&testCase)
		switch verdict {
		case VerdictPass:
			passCount++
		case VerdictUnstable:
			unstableCount++
		case VerdictFail:
			failCount++
		}
	}

	// Determine overall verdict: FAIL > UNSTABLE > PASS
	overallVerdict := VerdictPass
	if failCount > 0 {
		overallVerdict = VerdictFail
	} else if unstableCount > 0 {
		overallVerdict = VerdictUnstable
	}

	return overallVerdict, passCount, unstableCount, failCount
}

func displayReport(suite *chaoskit.JUnitTestSuite, verbose bool) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ChaosKit JUnit XML Report Viewer              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Calculate verdicts from test cases
	overallVerdict, passCount, unstableCount, failCount := calculateOverallVerdict(suite.TestCases)
	totalTests := len(suite.TestCases)
	if totalTests == 0 {
		totalTests = suite.Tests // Fallback to suite.Tests if no test cases
	}

	// Summary
	fmt.Println("📊 SUMMARY")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Test Suite:     %s\n", suite.Name)
	fmt.Printf("Total Tests:    %d\n", totalTests)
	fmt.Printf("Passed:         %d\n", passCount)
	fmt.Printf("Failures:       %d\n", failCount)
	fmt.Printf("Warnings:       %d\n", unstableCount)
	fmt.Printf("Duration:       %.2f seconds\n", suite.Time)
	if suite.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, suite.Timestamp); err == nil {
			fmt.Printf("Timestamp:      %s\n", t.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("Timestamp:      %s\n", suite.Timestamp)
		}
	}

	// Calculate success rate
	successRate := 0.0
	if totalTests > 0 {
		successRate = float64(passCount) / float64(totalTests) * 100
	}
	fmt.Printf("Success Rate:   %.2f%% (%d/%d)\n", successRate, passCount, totalTests)
	fmt.Println()

	// Overall status based on test cases
	fmt.Println("📈 STATUS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	switch overallVerdict {
	case VerdictFail:
		fmt.Printf("❌ FAILED: %d critical failure(s)\n", failCount)
	case VerdictUnstable:
		fmt.Printf("⚠️  UNSTABLE: %d warning(s)\n", unstableCount)
	case VerdictPass:
		fmt.Println("✅ ALL TESTS PASSED")
	}
	fmt.Println()

	// Test cases
	if len(suite.TestCases) > 0 {
		fmt.Println("🧪 TEST CASES")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		for i, testCase := range suite.TestCases {
			status := "✅"
			statusText := "PASS"
			if testCase.Failure != nil {
				status = "❌"
				statusText = "FAIL"
			} else if testCase.Error != nil {
				status = "⚠️"
				statusText = "ERROR"
			}

			fmt.Printf("%d. %s %s\n", i+1, status, testCase.Name)
			if verbose {
				fmt.Printf("   Class:  %s\n", testCase.Classname)
				fmt.Printf("   Status: %s\n", statusText)
				if testCase.Time > 0 {
					fmt.Printf("   Time:   %.3f seconds\n", testCase.Time)
				}

				if testCase.Failure != nil {
					fmt.Printf("   Failure Type: %s\n", testCase.Failure.Type)
					fmt.Printf("   Message: %s\n", testCase.Failure.Message)
					if testCase.Failure.Content != "" {
						content := strings.TrimSpace(testCase.Failure.Content)
						lines := strings.Split(content, "\n")
						if len(lines) > 0 {
							fmt.Println("   Details:")
							for _, line := range lines {
								if strings.TrimSpace(line) != "" {
									fmt.Printf("     %s\n", line)
								}
							}
						}
					}
				}

				if testCase.Error != nil {
					fmt.Printf("   Error Type: %s\n", testCase.Error.Type)
					fmt.Printf("   Message: %s\n", testCase.Error.Message)
					if testCase.Error.Content != "" {
						content := strings.TrimSpace(testCase.Error.Content)
						lines := strings.Split(content, "\n")
						if len(lines) > 0 {
							fmt.Println("   Details:")
							for _, line := range lines {
								if strings.TrimSpace(line) != "" {
									fmt.Printf("     %s\n", line)
								}
							}
						}
					}
				}
				fmt.Println()
			}
		}

		if !verbose {
			fmt.Println()
			fmt.Println("💡 Tip: Use -verbose flag to see detailed information for each test case")
		}
	}

	// Failures and errors summary
	if failCount > 0 || unstableCount > 0 {
		fmt.Println()
		fmt.Println("🔍 FAILURES & ERRORS")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		failureCount := 0
		errorCount := 0

		for _, testCase := range suite.TestCases {
			if testCase.Failure != nil {
				failureCount++
				fmt.Printf("\n❌ Failure #%d: %s\n", failureCount, testCase.Name)
				fmt.Printf("   Type:    %s\n", testCase.Failure.Type)
				fmt.Printf("   Message: %s\n", testCase.Failure.Message)
				if testCase.Failure.Content != "" {
					content := strings.TrimSpace(testCase.Failure.Content)
					lines := strings.Split(content, "\n")
					if len(lines) > 0 {
						fmt.Println("   Details:")
						for _, line := range lines {
							if strings.TrimSpace(line) != "" {
								fmt.Printf("     %s\n", line)
							}
						}
					}
				}
			}

			if testCase.Error != nil {
				errorCount++
				fmt.Printf("\n⚠️  Error #%d: %s\n", errorCount, testCase.Name)
				fmt.Printf("   Type:    %s\n", testCase.Error.Type)
				fmt.Printf("   Message: %s\n", testCase.Error.Message)
				if testCase.Error.Content != "" {
					content := strings.TrimSpace(testCase.Error.Content)
					lines := strings.Split(content, "\n")
					if len(lines) > 0 {
						fmt.Println("   Details:")
						for _, line := range lines {
							if strings.TrimSpace(line) != "" {
								fmt.Printf("     %s\n", line)
							}
						}
					}
				}
			}
		}
	}

	// Footer
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Report generated by ChaosKit JUnit XML Report Viewer")
}
