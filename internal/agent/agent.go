package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aykay76/llmapi/pkg/ollama"
)

// Agent represents a coding agent that uses Ollama for completions
type Agent struct {
	client        *ollama.Client
	systemPrompts map[string]string
	modelName     string
}

// NewAgent creates a new coding agent
func NewAgent(ollamaClient *ollama.Client, modelName string) *Agent {
	if modelName == "" {
		modelName = "qwen3-coder:30b"
	}
	return &Agent{
		client:        ollamaClient,
		systemPrompts: make(map[string]string),
		modelName:     modelName,
	}
}

// LoadSystemPrompt loads a system prompt from a file
func (a *Agent) LoadSystemPrompt(name, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read system prompt file: %w", err)
	}

	a.systemPrompts[name] = string(data)
	return nil
}

// LoadSystemPromptDirectory loads all system prompts from a directory
func (a *Agent) LoadSystemPromptDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".txt" {
			name := filepath.Base(entry.Name())
			name = name[:len(name)-4] // Remove .txt extension
			err := a.LoadSystemPrompt(name, filepath.Join(dirPath, entry.Name()))
			if err != nil {
				return fmt.Errorf("failed to load prompt %s: %w", name, err)
			}
		}
	}

	return nil
}

// GetSystemPrompt returns a loaded system prompt by name
func (a *Agent) GetSystemPrompt(name string) (string, bool) {
	prompt, ok := a.systemPrompts[name]
	return prompt, ok
}
