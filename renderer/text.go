package renderer

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/ChainSafe/vm-compat/analyser"
)

// TextRenderer formats the analysis report in a structured text format.
type TextRenderer struct{}

// NewTextRenderer creates a new instance of TextRenderer.
func NewTextRenderer() Renderer {
	return &TextRenderer{}
}

// Render formats and writes the analysis report to the command line.
func (r *TextRenderer) Render(issues []*analyser.Issue, output io.Writer) error {
	if len(issues) == 0 {
		return nil
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05 UTC")

	// Group issues by message
	groupedIssues := make(map[string][]*analyser.Issue)
	for _, issue := range issues {
		groupedIssues[issue.Message] = append(groupedIssues[issue.Message], issue)
	}
	totalIssues := len(groupedIssues)

	// Sort issue messages for consistent output
	var sortedMessages = make([]string, 0, len(groupedIssues))
	for msg := range groupedIssues {
		sortedMessages = append(sortedMessages, msg)
	}
	sort.Strings(sortedMessages)

	// Build report template
	var report strings.Builder

	// Header Section
	report.WriteString("==============================\n")
	report.WriteString("🔍 Go Compatibility Analysis Report\n")
	report.WriteString("==============================\n\n")
	report.WriteString(fmt.Sprintf("📄 Analyzed File: %s\n", issues[0].File))
	report.WriteString(fmt.Sprintf("📅 Timestamp: %s\n", timestamp))
	report.WriteString("🔢 Analyzer Version: 1.0.0\n\n")
	report.WriteString("------------------------------\n")
	report.WriteString("🚨 Summary of Issues\n")
	report.WriteString("------------------------------\n")
	report.WriteString(fmt.Sprintf("❗ Critical Issues: %d\n", totalIssues))
	report.WriteString("⚠️ Warnings: 0\n")
	report.WriteString(fmt.Sprintf("ℹ️ Total Issues: %d\n\n", totalIssues))
	report.WriteString("------------------------------\n")
	report.WriteString("📌 Detailed Issues\n")
	report.WriteString("------------------------------\n\n")

	// Issues Section
	issueCounter := 1
	for _, msg := range sortedMessages {
		report.WriteString(fmt.Sprintf("%d. [CRITICAL] %s\n", issueCounter, msg))
		report.WriteString("   - Affected Segments:\n")

		for _, issue := range groupedIssues[msg] {
			report.WriteString(fmt.Sprintf("     - Source: %s \n", issue.Source))
		}

		report.WriteString("   - Recommendation: Ensure affected opcodes are supported in the target environment.\n\n")
		issueCounter++
	}

	// Recommendations Section
	report.WriteString("------------------------------\n")
	report.WriteString("✅ Recommendations\n")
	report.WriteString("------------------------------\n")
	report.WriteString("- Review critical issues and replace deprecated or unsupported opcodes.\n")
	report.WriteString("- Verify compatibility with the target runtime.\n")
	report.WriteString("- Ensure proper opcode translation for execution environments.\n\n")
	report.WriteString("🔚 End of Report\n")

	// Print the complete report at once
	_, err := output.Write([]byte(report.String()))
	return err
}

// Format returns the format type.
func (r *TextRenderer) Format() string {
	return "text"
}
