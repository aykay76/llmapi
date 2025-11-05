# Architecture Overview

## Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        User / Developer                      │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                       REPL Interface                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Commands: /help /clear /model /system /prompt /exit │   │
│  │  Input: User messages and questions                  │   │
│  │  Output: Streamed responses                          │   │
│  └──────────────────────────────────────────────────────┘   │
│                    (cmd/agent/main.go)                       │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      Agent Core                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  • Conversation History Management                   │   │
│  │  • System Prompt Management                          │   │
│  │  • Message Building & Context                        │   │
│  │  • Stream Coordination                               │   │
│  │  • Command Processing                                │   │
│  └──────────────────────────────────────────────────────┘   │
│                 (internal/agent/agent.go)                    │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   Ollama API Client                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  • StreamChatWithContext()                           │   │
│  │  • StreamGenerateWithContext()                       │   │
│  │  • HTTP Request/Response Handling                    │   │
│  │  • JSON Parsing & Streaming                          │   │
│  │  • Context Cancellation Support                      │   │
│  └──────────────────────────────────────────────────────┘   │
│                   (pkg/ollama/client.go)                     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Ollama Server                             │
│                 (http://localhost:11434)                     │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  • Model Loading & Management                        │   │
│  │  • Token Generation                                  │   │
│  │  • Context Management                                │   │
│  │  • GPU/CPU Inference                                 │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### Message Sending Flow

```
User Input
    │
    ├─> Add to conversation history
    │
    ├─> Prepend system prompt (if set)
    │
    ├─> Build ChatRequest with full history
    │
    ├─> StreamChatWithContext()
    │       │
    │       ├─> HTTP POST to /api/chat
    │       │
    │       ├─> Read response line-by-line
    │       │
    │       └─> Parse JSON chunks
    │
    ├─> For each chunk:
    │       │
    │       ├─> Call onChunk callback
    │       │
    │       ├─> Display to user (fmt.Print)
    │       │
    │       └─> Accumulate full response
    │
    └─> Add complete response to history
```

### REPL Command Flow

```
User Input
    │
    ├─> Check if starts with "/"
    │       │
    │       ├─> YES: Process as command
    │       │       │
    │       │       ├─> /help     -> Show help text
    │       │       ├─> /clear    -> Clear conversation history
    │       │       ├─> /model    -> Switch model
    │       │       ├─> /system   -> Set system prompt
    │       │       ├─> /prompt   -> Load saved prompt
    │       │       └─> /exit     -> Exit REPL
    │       │
    │       └─> NO: Process as message
    │               │
    │               └─> (Follow message sending flow)
```

## Key Components

### 1. Agent (`internal/agent/agent.go`)
**Responsibilities:**
- Maintains conversation history
- Manages system prompts
- Coordinates with Ollama client
- Implements REPL logic
- Processes commands

**Key Data:**
```go
type Agent struct {
    client              *ollama.Client
    systemPrompts       map[string]string
    modelName           string
    conversationHistory []ollama.ChatMessage
    systemPrompt        string
}
```

### 2. Ollama Client (`pkg/ollama/client.go`)
**Responsibilities:**
- HTTP communication with Ollama
- Request serialization
- Response streaming
- Context cancellation
- Error handling

**Key Methods:**
```go
StreamChatWithContext(ctx, req, onChunk)
StreamGenerateWithContext(ctx, req, onChunk)
```

### 3. REPL Main (`cmd/agent/main.go`)
**Responsibilities:**
- CLI flag parsing
- Agent initialization
- Signal handling
- Prompt directory loading
- Starting REPL loop

## Conversation History Structure

```
Request to Ollama:
{
  "model": "qwen3-coder:30b",
  "messages": [
    {
      "role": "system",
      "content": "You are a coding assistant..."
    },
    {
      "role": "user",
      "content": "Explain SOLID principles"
    },
    {
      "role": "assistant",
      "content": "[Previous response about SOLID...]"
    },
    {
      "role": "user",
      "content": "Show me a Go example"
    }
  ],
  "stream": true
}
```

## Streaming Protocol

```
Ollama Response (line-by-line JSON):

{"response": "The", "done": false}
{"response": " Strategy", "done": false}
{"response": " pattern", "done": false}
...
{"response": ".", "done": true}

Each chunk triggers:
onChunk(chunk.response) -> fmt.Print() -> User sees text
```

## System Prompts

```
prompts/
├── coding-assistant.txt      # General coding help
├── python-datascience.txt    # Python/ML specialist
└── devops-expert.txt         # DevOps & infrastructure

Loaded at startup with -prompts flag
Accessed via /prompt <name> command
Applied to every request when set
```

## Error Handling

```
Connection Error
    └─> Suggest: Check ollama serve

Model Not Found
    └─> Suggest: ollama pull <model>

Context Cancelled
    └─> Graceful shutdown

Stream Error
    └─> Display error, continue REPL
```

## Extensibility Points

1. **New Commands**: Add cases to `handleCommand()`
2. **Custom Prompts**: Add .txt files to prompts/
3. **New Models**: Use -model flag or /model command
4. **Custom Streaming**: Modify onChunk callback
5. **Context Middleware**: Wrap SendMessage()
6. **History Management**: Add export/import to Agent

## Performance Characteristics

- **Latency**: First token ~100-500ms (model dependent)
- **Throughput**: 20-50 tokens/sec (hardware dependent)
- **Memory**: ~O(n) where n = conversation length
- **Network**: Minimal - streaming reduces buffering
- **Cancellation**: Instant via context.Context
