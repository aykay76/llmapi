# LLM API Client for Ollama

This is a Go package that provides a simple API client for interacting with Ollama, specifically designed for building coding agents. It includes support for loading and managing different system prompts.

## Features

- Simple Ollama API client
- Support for chat completions
- System prompt management
- Coding agent implementation
- Example usage

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

### Using the Coding Agent

```go
import (
    "github.com/aykay76/llmapi/internal/agent"
    "github.com/aykay76/llmapi/pkg/ollama"
)

// Create a new Ollama client
client := ollama.NewClient("http://localhost:11434")

// Create a new agent
codingAgent := agent.NewAgent(client)

// Load a system prompt
err := codingAgent.LoadSystemPrompt("my-prompt", "path/to/prompt.txt")
if err != nil {
    log.Fatal(err)
}

// Create a completion using the agent
response, err := codingAgent.CreateCompletion("my-prompt", "What are the best practices for error handling in Go?")
if err != nil {
    log.Fatal(err)
}

fmt.Println(response)
```

When you clone this repo you can try out the example by running:

`go run examples/complete_example.go`

You should see an output similar to `examples/example_output.txt`.
Note: this example assumes qwen-coder:30b as a model. You can change for whatever you have available :)
You can see in the output that each file is fenced in markdown with a comment containing the filename:
```
// pkg/models/models.go
```
So you can parse the output and create a directory structure and files locally. I haven't actually verified the output but it would be interesting if you copied this into a working client. Let me know?

## License

MIT License