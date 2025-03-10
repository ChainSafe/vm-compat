// Package comparer provides a way to compare current issues with the baseline report.
package comparer

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ChainSafe/vm-compat/analyzer"
)

// Comparer defines the interface for comparing the issues
type Comparer interface {
	CompareReport(issues []*analyzer.Issue, reader io.Reader) ([]*analyzer.Issue, error)

	Format() string
}

type jsonComparer struct{}

func NewJSONComparer() Comparer {
	return &jsonComparer{}
}

func (r *jsonComparer) CompareReport(issues []*analyzer.Issue, reader io.Reader) ([]*analyzer.Issue, error) {
	baseLineIssues := make([]*analyzer.Issue, 0)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading data while decoding issues: %w", err)
	}
	err = json.Unmarshal(data, &baseLineIssues)
	if err != nil {
		return nil, fmt.Errorf("error decoding issues: %w", err)
	}

	baselineIssuesMap := make(map[string]*analyzer.Issue)
	for _, issue := range baseLineIssues {
		baselineIssuesMap[issue.Hash] = issue
	}

	newIssues := make([]*analyzer.Issue, 0)
	for _, newIssue := range issues {
		if _, ok := baselineIssuesMap[newIssue.Hash]; !ok {
			newIssues = append(newIssues, newIssue)
		}
	}
	return newIssues, nil
}

func (r *jsonComparer) Format() string {
	return "json"
}
