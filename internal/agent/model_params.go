package agent

import (
	"strconv"
	"strings"
)

// ModelParameters represents the parsed parameters from a model's configuration
type ModelParameters struct {
	ContextLength   int    `json:"context_length,omitempty"`
	EmbeddingLength int    `json:"embedding_length,omitempty"`
	Template        string `json:"template,omitempty"`
	GPULayers       int    `json:"gpu_layers,omitempty"`
}

// parseModelParameters parses the raw parameter string from Ollama into a structured format
func parseModelParameters(params string) (*ModelParameters, error) {
	result := &ModelParameters{}

	// Split into lines and process each parameter
	lines := strings.Split(params, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove any surrounding quotes
		value = strings.Trim(value, `"'`)

		switch key {
		case "context_length":
			if n, err := strconv.Atoi(value); err == nil {
				result.ContextLength = n
			}
		case "embedding_length":
			if n, err := strconv.Atoi(value); err == nil {
				result.EmbeddingLength = n
			}
		case "gpu_layers":
			if n, err := strconv.Atoi(value); err == nil {
				result.GPULayers = n
			}
		case "template":
			// Remove any YAML-style block indicators
			value = strings.TrimPrefix(value, "|")
			value = strings.TrimSpace(value)
			result.Template = value
		}
	}

	return result, nil
}
