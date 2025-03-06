// Package renderer provides a way to render issues in different formats.
package renderer

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/ChainSafe/vm-compat/analyzer"
)

// JSONRenderer renders issues in JSON format.
type JSONRenderer struct{}

func NewJSONRenderer() Renderer {
	return &JSONRenderer{}
}

func (r *JSONRenderer) Render(issues []*analyzer.Issue, output io.Writer) error {
	sort.Slice(issues, func(i, j int) bool {
		// Sort by severity first
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity < issues[j].Severity
		}
		// If severity is the same, sort by hash lexicographically
		return issues[i].Hash < issues[j].Hash
	})
	return json.NewEncoder(output).Encode(issues)
}

func (r *JSONRenderer) Format() string {
	return "json"
}
