// Package renderer provides a way to render issues in different formats.
package renderer

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/ChainSafe/vm-compat/profile"
)

// TextRenderer formats the analysis report in a structured text format.
type TextRenderer struct {
	profile *profile.VMProfile
}

// NewTextRenderer creates a new instance of TextRenderer.
func NewTextRenderer(profile *profile.VMProfile) Renderer {
	return &TextRenderer{profile: profile}
}

// Render formats and writes the analysis report to the command line.
func (r *TextRenderer) Render(issues []*analyzer.Issue, output io.Writer) error {
	if len(issues) == 0 {
		return nil
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05 UTC")

	// Group issues by message
	groupedIssues := make(map[string][]*analyzer.Issue)
	for _, issue := range issues {
		groupedIssues[issue.Message] = append(groupedIssues[issue.Message], issue)
	}
	totalIssues := len(groupedIssues)

	// Sort issue messages for consistent output
	numOfCriticalIssues := 0
	var sortedMessages = make([]string, 0, len(groupedIssues))
	for msg, val := range groupedIssues {
		if val[0].Severity == analyzer.IssueSeverityCritical {
			numOfCriticalIssues++
		}
		sortedMessages = append(sortedMessages, msg)
	}
	sort.Strings(sortedMessages)

	// Build report template
	var report strings.Builder

	// Header Section
	report.WriteString("==============================\n")
	report.WriteString("🔍 Go Compatibility Analysis Report\n")
	report.WriteString("==============================\n\n")
	report.WriteString(fmt.Sprintf("🖥 VM Name: %s\n", r.profile.VMName))
	report.WriteString(fmt.Sprintf("⚙️ GOOS: %s\n", r.profile.GOOS))
	report.WriteString(fmt.Sprintf("🛠 GOARCH: %s\n", r.profile.GOARCH))
	report.WriteString(fmt.Sprintf("📅 Timestamp: %s\n", timestamp))
	report.WriteString("🔢 Analyzer Version: 1.0.0\n\n")
	report.WriteString("------------------------------\n")
	report.WriteString("🚨 Summary of Issues\n")
	report.WriteString("------------------------------\n")
	report.WriteString(fmt.Sprintf(" ❗ Critical Issues: %d\n", numOfCriticalIssues))
	report.WriteString(fmt.Sprintf("⚠️ Warnings: %d\n", totalIssues-numOfCriticalIssues))
	report.WriteString(fmt.Sprintf("ℹ️ Total Issues: %d\n\n", totalIssues))
	report.WriteString("------------------------------\n")
	report.WriteString("📌 Detailed Issues\n")
	report.WriteString("------------------------------\n\n")

	// Issues Section
	issueCounter := 1
	for _, msg := range sortedMessages {
		groupedIssue := groupedIssues[msg]
		report.WriteString(fmt.Sprintf("%d. [%s] %s\n", issueCounter, groupedIssue[0].Severity, msg))
		if len(groupedIssue[0].Impact) > 0 {
			report.WriteString(fmt.Sprintf("   - Impact: %s \n", groupedIssue[0].Impact))
		}
		if len(groupedIssue[0].Reference) > 0 {
			report.WriteString(fmt.Sprintf("   - Referance: %s \n", groupedIssue[0].Reference))
		}
		report.WriteString("   - CallStacks:")

		length := len(groupedIssue)
		for i, issue := range groupedIssue {
			report.WriteString(fmt.Sprintf("%s\n", buildCallStack(output, issue.CallStack, "")))
			// Don't spam the text output
			if i > 1 {
				report.WriteString(fmt.Sprintf("\n          ..... and %d more \n", length-i))
				break
			}
		}
		issueCounter++
	}

	// Recommendations Section
	report.WriteString("------------------------------\n")
	report.WriteString("✅ Recommendations\n")
	report.WriteString("------------------------------\n")
	report.WriteString("- Verify compatibility with the target runtime.\n")
	report.WriteString("🔚 End of Report\n")

	// Print the complete report at once
	_, err := output.Write([]byte(report.String()))
	return err
}

func buildCallStack(output io.Writer, source *analyzer.CallStack, str string) string {
	var fileInfo string
	if output == os.Stdout {
		fileInfo = fmt.Sprintf(
			" \033[94m\033]8;;file://%s:%d\033\\%s:%d\033]8;;\033\\\033[0m",
			source.AbsPath, source.Line, source.File, source.Line,
		)
	} else {
		fileInfo = fmt.Sprintf("%s:%d (%s)", source.File, source.Line, source.AbsPath)
	}

	str = strings.Join(
		[]string{
			str,
			fmt.Sprintf("-> %s : (%s)", fileInfo, source.Function)},
		"\n       ")
	if source.CallStack != nil {
		return buildCallStack(output, source.CallStack, str)
	}
	return str
}

// Format returns the format type.
func (r *TextRenderer) Format() string {
	return "text"
}
