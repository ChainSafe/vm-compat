package comparer

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

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
		return nil, fmt.Errorf("error reading data while decoding issues: %v", err)
	}
	err = json.Unmarshal(data, &baseLineIssues)
	if err != nil {
		return nil, fmt.Errorf("error decoding issues: %v", err)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity < issues[j].Severity
		}
		return issues[i].Hash < issues[j].Hash
	})

	sort.Slice(baseLineIssues, func(i, j int) bool {
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity < issues[j].Severity
		}
		return issues[i].Hash < issues[j].Hash
	})

	newIssues := make([]*analyzer.Issue, 0)
	for i, newIssue := range issues {
		if i >= len(baseLineIssues) || newIssue.Hash != baseLineIssues[i].Hash {
			newIssues = append(newIssues, newIssue)
		}
	}
	return newIssues, nil
}

func (r *jsonComparer) Format() string {
	return "json"
}
