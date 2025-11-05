package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/aykay76/llmapi/pkg/ollama"
)

// Agent represents a coding agent that can interact with an LLM
type Agent struct {
	client              *ollama.Client
	systemPrompts       map[string]string
	modelName           string
	modelParams         *ModelParameters
	conversationHistory []ollama.ChatMessage
	systemPrompt        string
	workDir             string
	autoExecuteActions  bool
	actionParser        *ActionParser
	pendingActions      []Action
	lastResponseStats   *ollama.GenerateResponse
}

// NewAgent creates a new coding agent
func NewAgent(ollamaClient *ollama.Client, modelName string) *Agent {
	if modelName == "" {
		modelName = "qwen3-coder:30b"
	}

	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	agent := &Agent{
		client:              ollamaClient,
		systemPrompts:       make(map[string]string),
		modelName:           modelName,
		conversationHistory: make([]ollama.ChatMessage, 0),
		actionParser:        NewActionParser(),
		workDir:             workDir,
		autoExecuteActions:  false, // Default to false for safety
	}

	// Initialize model parameters
	if info, err := ollamaClient.ShowModel(modelName); err == nil {
		if params, err := parseModelParameters(info.Parameters); err == nil {
			agent.modelParams = params
		}
	}

	return agent
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

// SetSystemPrompt sets the active system prompt for the agent
func (a *Agent) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
} // SetWorkDir sets the working directory for action execution
func (a *Agent) SetWorkDir(dir string) {
	a.workDir = dir
}

// SetAutoExecuteActions enables/disables automatic action execution
func (a *Agent) SetAutoExecuteActions(enabled bool) {
	a.autoExecuteActions = enabled
}

// ClearHistory clears the conversation history
func (a *Agent) ClearHistory() {
	a.conversationHistory = make([]ollama.ChatMessage, 0)
}

// SendMessage sends a message to the agent and streams the response
func (a *Agent) SendMessage(ctx context.Context, message string, onChunk func(string) error) error {
	// Add user message to history
	a.conversationHistory = append(a.conversationHistory, ollama.ChatMessage{
		Role:    "user",
		Content: message,
	})

	// Build messages array with system prompt if set
	messages := make([]ollama.ChatMessage, 0)
	if a.systemPrompt != "" {
		messages = append(messages, ollama.ChatMessage{
			Role:    "system",
			Content: a.systemPrompt,
		})
	}
	messages = append(messages, a.conversationHistory...)

	// Create generate request by flattening the conversation history into
	// a single prompt. Some Ollama setups return streaming text under the
	// generate endpoint; use that to avoid empty-chat-format responses.
	// The system prompt is supplied separately in the GenerateRequest.System
	// field when available.
	var promptBuilder strings.Builder
	for i, m := range messages {
		if i > 0 {
			promptBuilder.WriteString("\n\n")
		}
		role := strings.Title(m.Role)
		promptBuilder.WriteString(role)
		promptBuilder.WriteString(": ")
		promptBuilder.WriteString(m.Content)
	}

	req := &ollama.GenerateRequest{
		Model:  a.modelName,
		System: a.systemPrompt,
		Prompt: promptBuilder.String(),
		Stream: true,
	}

	// Accumulate assistant response
	var fullResponse strings.Builder
	wrappedOnChunk := func(chunk string) error {
		fullResponse.WriteString(chunk)
		return onChunk(chunk)
	}

	// Stream the response (use Generate stream to match server streaming format)
	var lastChunk ollama.GenerateResponse
	wrappedOnChunkWithStats := func(chunk string) error {
		if err := json.Unmarshal([]byte(chunk), &lastChunk); err == nil {
			return wrappedOnChunk(lastChunk.Response)
		}
		return wrappedOnChunk(chunk)
	}

	err := a.client.StreamGenerateWithContext(ctx, req, wrappedOnChunkWithStats)
	if err != nil {
		return fmt.Errorf("failed to stream chat: %w", err)
	}

	// Print model statistics
	fmt.Printf("\n📊 Model Stats:\n")

	// Model context capacity
	if a.modelParams != nil && a.modelParams.ContextLength > 0 {
		fmt.Printf("  • Model Context: %d tokens\n", a.modelParams.ContextLength)
	}

	// Usage statistics
	fmt.Printf("  • Context Messages: %d\n", len(messages))
	fmt.Printf("  • Response Length: %d chars\n", len(fullResponse.String()))
	fmt.Printf("  • Total Duration: %dms\n", lastChunk.TotalDuration/1e6)
	fmt.Printf("  • Load Duration: %dms\n", lastChunk.LoadDuration/1e6)

	// Context window usage
	if len(lastChunk.Context) > 0 {
		usedTokens := len(lastChunk.Context)
		if a.modelParams != nil && a.modelParams.ContextLength > 0 {
			usagePercent := float64(usedTokens) / float64(a.modelParams.ContextLength) * 100
			fmt.Printf("  • Context Usage: %d/%d tokens (%.1f%%)\n",
				usedTokens, a.modelParams.ContextLength, usagePercent)
		} else {
			fmt.Printf("  • Context Tokens Used: %d\n", usedTokens)
		}
	} else {
		fmt.Printf("  • Context Usage: No context used yet\n")
	} // Add assistant response to history
	a.conversationHistory = append(a.conversationHistory, ollama.ChatMessage{
		Role:    "assistant",
		Content: fullResponse.String(),
	})

	// Parse actions
	actions := a.actionParser.Parse(fullResponse.String())
	if len(actions) > 0 {
		fmt.Printf("\n\n📋 Detected %d action(s):\n", len(actions))
		for i, action := range actions {
			fmt.Printf("  %d. %s\n", i+1, action.String())
		}

		// Store as pending actions so the user can run /execute later
		a.pendingActions = actions

		if a.autoExecuteActions {
			fmt.Println("\n⚙️  Auto-executing actions...")
			if err := ExecuteActions(ctx, actions, a.workDir); err != nil {
				return fmt.Errorf("failed to execute actions: %w", err)
			}
			a.pendingActions = nil
			fmt.Println("✅ All actions completed successfully")
		} else {
			fmt.Println("\n💡 Tip: Use /execute to run these actions, or enable auto-execution with /auto on")
		}
	}

	return nil
}

// RunREPL starts an interactive REPL session with the agent
func (a *Agent) RunREPL(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	var currentCancel context.CancelFunc
	var interrupted bool

	// Handle Ctrl+C in a separate goroutine
	go func() {
		for range sigChan {
			interrupted = true
			if currentCancel != nil {
				currentCancel()
				fmt.Println("\n🛑 Interrupted! Stream stopped.")
			} else {
				fmt.Print("\n> ")
			}
		}
	}()

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          Coding Agent REPL - Powered by Ollama            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("Model: %s\n", a.modelName)
	fmt.Println("\nCommands:")
	fmt.Println("  /help         - Show this help message")
	fmt.Println("  /clear        - Clear conversation history")
	fmt.Println("  /model <name> - Switch to a different model")
	fmt.Println("  /system <msg> - Set system prompt")
	fmt.Println("  /prompt <name>- Load a saved system prompt")
	fmt.Println("  /workdir <dir>- Set working directory for actions")
	fmt.Println("  /auto <on|off>- Enable/disable auto-execution of actions")
	fmt.Println("  /exit or /quit- Exit the REPL")
	fmt.Println("\nType your message and press Enter to chat.")
	fmt.Println()

	for {
		if !interrupted {
			fmt.Print("\n> ")
		}
		interrupted = false

		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "interrupt" {
				continue
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Create a new cancellable context for this interaction
		streamCtx, cancel := context.WithCancel(ctx)
		currentCancel = cancel
		defer func() {
			currentCancel = nil
			cancel()
		}()

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if err := a.handleCommand(input); err != nil {
				if err.Error() == "exit" {
					fmt.Println("\nGoodbye!")
					return nil
				}
				fmt.Printf("Error: %v\n", err)
			}
			continue
		}

		// Send message and stream response
		fmt.Println()
		err = a.SendMessage(streamCtx, input, func(chunk string) error {
			fmt.Print(chunk)
			return nil
		})
		if err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				fmt.Print("\n💡 Tip: The response was interrupted. Continue with your next question!\n\n> ")
				continue
			}
			fmt.Printf("\nError: %v\n\n> ", err)
			continue
		}
		fmt.Println()
	}
}

// handleCommand processes REPL commands
func (a *Agent) handleCommand(cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/help":
		fmt.Println("\nAvailable Commands:")
		fmt.Println("  /help         - Show this help message")
		fmt.Println("  /clear        - Clear conversation history")
		fmt.Println("  /model <name> - Switch to a different model")
		fmt.Println("  /system <msg> - Set system prompt")
		fmt.Println("  /prompt <name>- Load a saved system prompt")
		fmt.Println("  /workdir <dir>- Set working directory for actions")
		fmt.Println("  /auto <on|off>- Enable/disable auto-execution of actions")
		fmt.Println("  /exit, /quit  - Exit the REPL")

	case "/clear":
		a.ClearHistory()
		fmt.Println("✓ Conversation history cleared")

	case "/model":
		if len(parts) < 2 {
			fmt.Printf("Current model: %s\n", a.modelName)
			fmt.Println("Usage: /model <model-name>")
		} else {
			a.modelName = parts[1]
			// Get model parameters and details
			if info, err := a.client.ShowModel(a.modelName); err == nil {
				if params, err := parseModelParameters(info.Parameters); err == nil {
					a.modelParams = params
				}
				fmt.Printf("\n🤖 Model Information:\n")
				fmt.Printf("  • Name: %s\n", a.modelName)
				if info.License != "" {
					fmt.Printf("  • License: %s\n", info.License)
				}
				if info.Details.Format != "" {
					fmt.Printf("  • Format: %s\n", info.Details.Format)
				}
				if info.Details.Family != "" {
					fmt.Printf("  • Family: %s\n", info.Details.Family)
				}
				if info.Details.ParameterSize != "" {
					fmt.Printf("  • Size: %s\n", info.Details.ParameterSize)
				}
				if info.Details.QuantizationLevel != "" {
					fmt.Printf("  • Quantization: %s\n", info.Details.QuantizationLevel)
				}

				if a.modelParams != nil {
					fmt.Printf("\n⚙️ Model Parameters:\n")
					if a.modelParams.ContextLength > 0 {
						fmt.Printf("  • Context Window: %d tokens\n", a.modelParams.ContextLength)
					}
					if a.modelParams.EmbeddingLength > 0 {
						fmt.Printf("  • Embedding Size: %d\n", a.modelParams.EmbeddingLength)
					}
					if a.modelParams.GPULayers > 0 {
						fmt.Printf("  • GPU Layers: %d\n", a.modelParams.GPULayers)
					}
					if a.modelParams.Template != "" {
						fmt.Printf("  • Template: %s\n", a.modelParams.Template)
					}
				}
				fmt.Printf("\n✓ Successfully switched to model\n")
			} else {
				fmt.Printf("✓ Switched to model: %s (could not fetch details: %v)\n", a.modelName, err)
			}
		}

	case "/system":
		if len(parts) < 2 {
			if a.systemPrompt == "" {
				fmt.Println("No system prompt set")
			} else {
				fmt.Printf("Current system prompt:\n%s\n", a.systemPrompt)
			}
			fmt.Println("Usage: /system <name|message>  (if <name> matches a loaded prompt it will be used)")
		} else {
			// If the argument matches a loaded prompt name, use that prompt.
			nameOrMsg := strings.Join(parts[1:], " ")
			if prompt, ok := a.systemPrompts[nameOrMsg]; ok {
				a.systemPrompt = prompt
				fmt.Printf("✓ Loaded system prompt: %s\n", nameOrMsg)
			} else {
				// No matching prompt name — treat the argument as the inline system message.
				a.systemPrompt = nameOrMsg
				fmt.Println("✓ System prompt updated")
			}
		}

	case "/prompt":
		if len(parts) < 2 {
			fmt.Println("Available prompts:")
			for name := range a.systemPrompts {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println("Usage: /prompt <name>")
		} else {
			prompt, ok := a.GetSystemPrompt(parts[1])
			if !ok {
				return fmt.Errorf("prompt '%s' not found", parts[1])
			}
			a.systemPrompt = prompt
			fmt.Printf("✓ Loaded system prompt: %s\n", parts[1])
		}

	case "/workdir":
		if len(parts) < 2 {
			fmt.Printf("Current working directory: %s\n", a.workDir)
			fmt.Println("Usage: /workdir <directory>")
		} else {
			newDir := strings.Join(parts[1:], " ")
			// Expand ~ to home directory
			if strings.HasPrefix(newDir, "~") {
				home, err := os.UserHomeDir()
				if err == nil {
					newDir = filepath.Join(home, newDir[1:])
				}
			}

			// Check if directory exists
			if info, err := os.Stat(newDir); err != nil || !info.IsDir() {
				return fmt.Errorf("directory does not exist: %s", newDir)
			}

			a.workDir = newDir
			fmt.Printf("✓ Working directory set to: %s\n", a.workDir)
		}

	case "/auto":
		if len(parts) < 2 {
			status := "disabled"
			if a.autoExecuteActions {
				status = "enabled"
			}
			fmt.Printf("Auto-execution is currently: %s\n", status)
			fmt.Println("Usage: /auto <on|off>")
		} else {
			switch strings.ToLower(parts[1]) {
			case "on", "true", "1", "yes":
				a.autoExecuteActions = true
				fmt.Println("✓ Auto-execution enabled")
			case "off", "false", "0", "no":
				a.autoExecuteActions = false
				fmt.Println("✓ Auto-execution disabled")
			default:
				return fmt.Errorf("invalid value: %s (use 'on' or 'off')", parts[1])
			}
		}

	case "/exit", "/quit":
		return fmt.Errorf("exit")

	case "/execute":
		if len(a.pendingActions) == 0 {
			fmt.Println("No pending actions to execute")
			return nil
		}
		fmt.Println("\n⚙️  Executing pending actions...")
		// Execute with a background context; REPL has its own cancellation elsewhere
		if err := ExecuteActions(context.Background(), a.pendingActions, a.workDir); err != nil {
			return fmt.Errorf("execution failed: %w", err)
		}
		a.pendingActions = nil
		fmt.Println("✅ All actions completed successfully")

	default:
		return fmt.Errorf("unknown command: %s (type /help for available commands)", parts[0])
	}

	return nil
}
