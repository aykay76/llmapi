# Usage Guide

## Getting Started with the Coding Agent REPL

### Quick Start

1. Make sure Ollama is running locally:
   ```bash
   ollama serve
   ```

2. Pull a coding model (if you haven't already):
   ```bash
   ollama pull qwen3-coder:30b
   # or use a smaller model
   ollama pull llama3:8b
   ```

3. Run the agent:
   ```bash
   go run cmd/agent/main.go
   ```

### Command Line Options

```bash
go run cmd/agent/main.go [options]

Options:
  -url string
        Ollama API URL (default "http://localhost:11434")
  -model string
        Model name to use (default "qwen3-coder:30b")
  -prompts string
        Directory containing system prompt files
  -system string
        System prompt to use
```

### Examples

#### Using a Different Model
```bash
go run cmd/agent/main.go -model "llama3:8b"
```

#### Loading System Prompts
```bash
go run cmd/agent/main.go -prompts "./prompts"
```

#### Setting a Custom System Prompt
```bash
go run cmd/agent/main.go -system "You are a Python expert specializing in data science"
```

#### Connecting to Remote Ollama Instance
```bash
go run cmd/agent/main.go -url "http://your-server:11434"
```

## REPL Commands

### `/help`
Display available commands and their usage.

### `/clear`
Clear the conversation history. Use this to start fresh without the context of previous messages.

```
> /clear
✓ Conversation history cleared
```

### `/model <name>`
Switch to a different model during the session.

```
> /model llama3:8b
✓ Switched to model: llama3:8b
```

View current model:
```
> /model
Current model: qwen3-coder:30b
```

### `/system <message>`
Set or update the system prompt during the session.

```
> /system You are a Go expert specializing in microservices
✓ System prompt updated
```

View current system prompt:
```
> /system
Current system prompt:
You are a Go expert specializing in microservices
```

### `/prompt <name>`
Load a pre-configured system prompt from the prompts directory.

```
> /prompt coding-assistant
✓ Loaded system prompt: coding-assistant
```

List available prompts:
```
> /prompt
Available prompts:
  - coding-assistant
  - python-expert
  - devops-helper
```

### `/exit` or `/quit`
Exit the REPL session.

## Creating System Prompts

System prompts are text files stored in the `prompts/` directory. Each file should have a `.txt` extension.

Example: `prompts/go-expert.txt`
```
You are an expert Go developer with deep knowledge of:
- Go idioms and best practices
- Concurrent programming with goroutines and channels
- Standard library usage
- Performance optimization
- Testing strategies

When helping with Go code:
- Follow Go conventions (gofmt, golint)
- Use clear variable names
- Handle errors explicitly
- Write testable code
- Consider performance implications
```

## Usage Tips

### Maintaining Context
The agent maintains conversation history, so you can ask follow-up questions:

```
> Explain the Strategy pattern

[Agent explains Strategy pattern]

> Can you show me a Go implementation?

[Agent provides Go code example using context from previous message]
```

### Code Reviews
Paste your code and ask for review:

```
> Can you review this code?
func processData(data []string) {
    for _, item := range data {
        // process item
    }
}

[Agent provides review with suggestions]
```

### Debugging Help
Share error messages for assistance:

```
> I'm getting this error: "panic: runtime error: invalid memory address"
Here's my code: [paste code]

[Agent helps identify the issue]
```

### Learning New Concepts
Ask for explanations and examples:

```
> Explain how channels work in Go and give me practical examples

[Agent provides detailed explanation with code]
```

## Programmatic Usage

For integration into your own tools:

```go
package main

import (
    "context"
    "fmt"
    "github.com/aykay76/llmapi/internal/agent"
    "github.com/aykay76/llmapi/pkg/ollama"
)

func main() {
    // Setup
    client := ollama.NewClient("http://localhost:11434")
    client.SetTimeout(0)
    
    agentInstance := agent.NewAgent(client, "qwen3-coder:30b")
    agentInstance.SetSystemPrompt("You are a helpful coding assistant")
    
    // Send message with streaming
    ctx := context.Background()
    err := agentInstance.SendMessage(ctx, "Hello!", func(chunk string) error {
        fmt.Print(chunk)
        return nil
    })
    
    if err != nil {
        panic(err)
    }
    
    // Clear history if needed
    agentInstance.ClearHistory()
}
```

## Troubleshooting

### Connection Issues
If you can't connect to Ollama:
1. Verify Ollama is running: `ollama list`
2. Check the URL: default is `http://localhost:11434`
3. Try specifying the URL: `-url "http://localhost:11434"`

### Model Not Found
If the model isn't available:
```bash
ollama pull qwen3-coder:30b
# or
ollama list  # to see available models
```

### Slow Responses
For faster responses:
- Use smaller models: `llama3:8b` instead of `qwen3-coder:30b`
- Ensure sufficient GPU/CPU resources
- Check Ollama logs: `ollama logs`

### Context Too Long
If context becomes too long:
- Use `/clear` to reset conversation history
- Start a new session
- Use shorter prompts and responses
