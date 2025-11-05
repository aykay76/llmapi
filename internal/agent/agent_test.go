package agent

import (
	"testing"

	"github.com/aykay76/llmapi/pkg/ollama"
)

func TestParseModelParameters(t *testing.T) {
	// This is a sample of what Ollama returns for model parameters
	// We'll need to update this with actual data from Ollama
	sampleParams := `template: "{{ .System }}\n\n{{ .Prompt }}"
context_length: 262144
embedding_length: 4096
gpu_layers: 115`

	client := ollama.NewClient("http://localhost:11434")
	agent := NewAgent(client, "qwen3:30b")

	info, err := agent.client.ShowModel(agent.modelName)
	if err != nil {
		t.Logf("Could not get real model info, using sample: %v", err)
		info = &ollama.ShowModelResponse{
			Parameters: sampleParams,
		}
	}

	t.Logf("Model Parameters:\n%s", info.Parameters)
}
