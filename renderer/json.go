package renderer

import (
	"encoding/json"
	"io"

	"github.com/ChainSafe/vm-compat/analyser"
)

// JSONRenderer renders issues in JSON format.
type JSONRenderer struct{}

func NewJSONRenderer() Renderer {
	return &JSONRenderer{}
}

func (r *JSONRenderer) Render(issues []*analyser.Issue, output io.Writer) error {
	return json.NewEncoder(output).Encode(issues)
}

func (r *JSONRenderer) Format() string {
	return "json"
}
