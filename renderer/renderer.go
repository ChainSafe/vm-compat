// Package renderer provides a way to render issues in different formats.
package renderer

import (
	"io"

	"github.com/ChainSafe/vm-compat/analyzer"
)

// Renderer defines the interface for rendering lint results in different formats.
type Renderer interface {
	// Render takes a list of issues and outputs them in the desired format to the provided writer.
	Render(issues []*analyzer.Issue, output io.Writer) error

	// Format returns the name of the output format (e.g., "json", "text", "html").
	Format() string
}
