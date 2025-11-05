# LLM API Client for Ollama

This is a Go package that provides a simple API client for interacting with Ollama, specifically designed for building coding agents. It includes support for loading and managing different system prompts, with a REPL-style interactive interface that maintains conversation history while streaming responses.

## Features

- Simple Ollama API client with streaming support
- Support for chat completions and text generation
- System prompt management and loading
- Interactive REPL coding agent with conversation history
- Streamed responses for real-time interaction
- Context-aware conversation management
- **Action-based code execution system** (file creation, command execution)
- **Unambiguous LLM intent parsing** with XML-style action tags
- Example usage and templates

## 🆕 Action-Based Coding Agent

The agent now supports **explicit action tags** for unambiguous code generation:

```xml
<create_file>
<path>main.go</path>
<content>package main...</content>
</create_file>
```

This solves the ambiguity problem: is `// filename.go` a comment or a directive? 

**See [docs/ACTIONS_SUMMARY.md](docs/ACTIONS_SUMMARY.md) for quick start** or [docs/ACTIONS.md](docs/ACTIONS.md) for full documentation.

## Installation

```bash
go get github.com/aykay76/llmapi
```

## Usage

### Basic Ollama Client

```go
import "github.com/aykay76/llmapi/pkg/ollama"

// Create a new client
client := ollama.NewClient("http://localhost:11434")

// Create a chat completion request
req := &ollama.ChatCompletionRequest{
    Model: "llama2",
    Messages: []ollama.ChatMessage{
        {
            Role:    "system",
            Content: "You are a helpful assistant.",
        },
        {
            Role:    "user",
            Content: "Hello!",
        },
    },
}

// Send the request
resp, err := client.CreateChatCompletion(req)
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Response)
```

### Using the Coding Agent REPL

The agent provides an interactive REPL interface for continuous coding assistance:

```bash
# Run the interactive agent
go run cmd/agent/main.go

# With custom model
go run cmd/agent/main.go -model "llama3:8b"

# With prompts directory
go run cmd/agent/main.go -prompts "./prompts"

# With system prompt
go run cmd/agent/main.go -system "You are a Go expert"
```

#### REPL Commands

- `/help` - Show available commands
- `/clear` - Clear conversation history
- `/model <name>` - Switch to a different model
- `/system <msg>` - Set system prompt
- `/prompt <name>` - Load a saved system prompt
- `/workdir <dir>` - Set working directory for action execution
- `/auto <on|off>` - Enable/disable auto-execution of actions
- `/exit` or `/quit` - Exit the REPL

### Programmatic Agent Usage

You can also use the agent programmatically with streaming:

```go
import (
    "context"
    "github.com/aykay76/llmapi/internal/agent"
    "github.com/aykay76/llmapi/pkg/ollama"
)

// Create a new Ollama client
client := ollama.NewClient("http://localhost:11434")
client.SetTimeout(0) // Disable timeout for streaming

// Create a new agent
agentInstance := agent.NewAgent(client, "qwen3-coder:30b")

// Load system prompts from directory
err := agentInstance.LoadSystemPromptDirectory("prompts")
if err != nil {
    log.Printf("Warning: %v", err)
}

// Set a system prompt
prompt, ok := agentInstance.GetSystemPrompt("coding-assistant")
if ok {
    agentInstance.SetSystemPrompt(prompt)
}

// Send a message with streaming response
ctx := context.Background()
err = agentInstance.SendMessage(ctx, "Explain SOLID principles", func(chunk string) error {
    fmt.Print(chunk) // Stream each chunk as it arrives
    return nil
})
if err != nil {
    log.Fatal(err)
}

// Follow-up messages maintain conversation context
err = agentInstance.SendMessage(ctx, "Give me a Go example", func(chunk string) error {
    fmt.Print(chunk)
    return nil
})
```

## Examples

### Run the Complete Example

```bash
go run examples/complete_example.go
```

You should see an output similar to `examples/example_output.txt`.
Note: this example assumes qwen-coder:30b as a model. You can change for whatever you have available.

### Run the Agent Example

```bash
go run examples/agent_example.go
```

This demonstrates programmatic usage with conversation history and streaming.

### Run the Interactive REPL

```bash
go run cmd/agent/main.go -prompts "./prompts"
```

This starts an interactive coding assistant where you can have continuous conversations.

## Project Structure

```
llmapi/
├── cmd/
│   └── agent/          # REPL agent executable
├── internal/
│   └── agent/          # Agent implementation
├── pkg/
│   └── ollama/         # Ollama API client
├── prompts/            # System prompt templates
├── examples/           # Usage examples
└── README.md
```

## How It Works

The agent maintains conversation history and uses Ollama's streaming API to provide real-time responses. Each interaction:

1. Adds the user message to conversation history
2. Sends the full history (with system prompt) to Ollama
3. Streams the response chunk-by-chunk for real-time display
4. Adds the assistant's response to history for context in future messages

This creates a natural, interactive coding assistant experience while leveraging the full power of local LLMs through Ollama.

## License

MIT License