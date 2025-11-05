# Summary: Action-Based Coding Agent

## The Problem You Identified ✓

When an LLM generates output like:
```go
// main.go
package main
```

**Is that a comment or a directive to create `main.go`?** → Ambiguous!

## The Solution: Explicit Action Tags

Tell the LLM (via system prompt) to use XML-style tags for actions:

```xml
<create_file>
<path>main.go</path>
<content>package main</content>
</create_file>
```

Regular markdown blocks are **only for explanations**, not actions.

## How It Works

1. **System Prompt** (`prompts/coding-agent-with-actions.txt`) tells the LLM:
   - Use `<create_file>` for files to create
   - Use `<execute_command>` for commands to run
   - Use ` ```language ` only for examples/explanations

2. **Action Parser** (`internal/agent/actions.go`) extracts tags using regex

3. **Agent** (`internal/agent/agent.go`) displays actions and executes them

## Available Actions

| Action | Purpose |
|--------|---------|
| `<create_file>` | Create a file with content |
| `<create_directory>` | Create a directory |
| `<execute_command>` | Run a shell command |
| `<modify_file>` | Search/replace in existing file |
| `<read_file>` | Read file for LLM context |

## Quick Start

```bash
# Start agent
go run cmd/agent/main.go -prompts ./prompts

# Load action-aware prompt
> /prompt coding-agent-with-actions

# Set working directory
> /workdir /path/to/project

# Enable auto-execution (optional)
> /auto on

# Ask for code
> Create a web server in Go

# Agent shows detected actions:
📋 Detected 2 action(s):
  1. CREATE_FILE: main.go (234 bytes)
  2. CREATE_FILE: go.mod (25 bytes)

# If auto-execution is on, files are created automatically
✅ All actions completed successfully
```

## Why This Approach?

✅ **Unambiguous** - No guessing what's an action vs example  
✅ **Safe** - Validates paths, prevents `..` traversal  
✅ **Flexible** - Auto-execute or manual approval  
✅ **Extensible** - Easy to add new action types  
✅ **MCP-Like** - Similar to how Claude/Copilot work  

## Comparison to Your Question

**You asked:** "Do we need to tell the LLM what capabilities we have - a bit like MCP tools?"

**Answer:** **YES!** That's exactly what we do:

1. System prompt lists available actions (like MCP tool definitions)
2. LLM uses structured tags (like MCP tool calls)
3. Agent parses and executes (like MCP tool execution)

This is **much better** than:
- ❌ Heuristic pattern matching (`// filename.go` ambiguity)
- ❌ Hoping the LLM formats consistently
- ❌ Complex regex to guess intent

## Files Created

- `prompts/coding-agent-with-actions.txt` - System prompt with action instructions
- `internal/agent/actions.go` - Action types and parser
- `internal/agent/actions_test.go` - Tests for action system
- `internal/agent/agent.go` - Extended with action support
- `docs/ACTIONS.md` - Full documentation
- `examples/action_example_request.txt` - Example LLM response

## Testing

```bash
go test ./internal/agent/... -v
```

All tests pass ✓

## Next Steps

You can now:
1. Run the agent with action-aware prompts
2. Generate multi-file projects automatically
3. Execute commands safely
4. Add new action types as needed

The system is production-ready and follows the same principles as modern coding agents like GitHub Copilot Workspace!
