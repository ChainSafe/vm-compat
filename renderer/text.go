package renderer

import (
	"fmt"
	"io"

	"github.com/ChainSafe/vm-compat/analyser"
)

// TextRenderer renders issues in plain text format.
type TextRenderer struct{}

func NewTextRenderer() Renderer {
	return &TextRenderer{}
}

func (r *TextRenderer) Render(issues []analyser.Issue, output io.Writer) error {
	for _, issue := range issues {
		_, err := fmt.Fprintf(output, "File: %s, Line: %d, Message: %s\n",
			issue.File, issue.Line, issue.Message)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *TextRenderer) Format() string {
	return "text"
}
