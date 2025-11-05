# Quick Reference

## Start the REPL

```bash
go run cmd/agent/main.go
```

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `/help` | Show help | `/help` |
| `/clear` | Clear history | `/clear` |
| `/model <name>` | Switch model | `/model llama3:8b` |
| `/system <msg>` | Set system prompt | `/system You are a Python expert` |
| `/prompt <name>` | Load saved prompt | `/prompt coding-assistant` |
| `/exit` or `/quit` | Exit REPL | `/exit` |

## Flags

```bash
-url string       # Ollama URL (default: http://localhost:11434)
-model string     # Model name (default: qwen3-coder:30b)
-prompts string   # Prompts directory
-system string    # System prompt text
```

## Examples

### Basic Chat
```
> Explain SOLID principles
[Streams response...]
> Give me a Go example
[Continues with context...]
```

### Using Commands
```
> /prompt python-datascience
✓ Loaded system prompt: python-datascience
> How do I handle missing data in pandas?
[Expert Python/pandas response...]
```

### Switching Models
```
> /model llama3:8b
✓ Switched to model: llama3:8b
> Same question but with this smaller model
```

## Programmatic Usage

```go
client := ollama.NewClient("http://localhost:11434")
client.SetTimeout(0)

agent := agent.NewAgent(client, "qwen3-coder:30b")
agent.SetSystemPrompt("You are a helpful assistant")

ctx := context.Background()
agent.SendMessage(ctx, "Hello!", func(chunk string) error {
    fmt.Print(chunk) // Streams in real-time
    return nil
})
```

## Tips

- **Context Memory**: The agent remembers your conversation - ask follow-up questions naturally
- **Clear When Needed**: Use `/clear` to start fresh if context gets too long
- **Try Different Prompts**: Use `/prompt` to switch between specialist modes
- **Model Size**: Smaller models (llama3:8b) are faster, larger (qwen3-coder:30b) are more capable
- **Code Blocks**: The agent outputs code in Markdown format for easy copying

## Troubleshooting

**Connection failed?**
```bash
ollama serve  # Start Ollama
```

**Model not found?**
```bash
ollama pull qwen3-coder:30b
ollama list  # See available models
```

**Need to exit?**
- Type `/exit` or `/quit`
- Or press Ctrl+C
