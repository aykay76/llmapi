# Action-Based Coding Agent System

This document explains how the action-based system works in the coding agent and how to use it.

## Overview

The agent uses an **explicit action tag system** to distinguish between:
- **Actions** the LLM wants executed (file creation, commands, etc.)
- **Explanations** and code examples shown to the user

This solves the ambiguity problem where `// filename.go` could be either a comment or a directive.

## How It Works

### 1. System Prompt Instructions

The LLM is instructed via system prompt to use XML-style tags for any action it wants executed:

```
<create_file>
<path>main.go</path>
<content>
package main
func main() {}
</content>
</create_file>
```

Regular markdown code blocks (` ```go ... ``` `) are used only for **explanations and examples**, not actions.

### 2. Action Parser

The `ActionParser` uses regex to extract action tags from LLM responses:

```go
parser := NewActionParser()
actions := parser.Parse(llmResponse)
```

### 3. Action Execution

Actions can be executed automatically or manually:

```go
// Execute all actions in sequence
ExecuteActions(ctx, actions, workDir)
```

## Available Actions

### 1. CREATE_FILE

Creates a new file with the specified content.

```xml
<create_file>
<path>relative/path/to/file.go</path>
<content>
package main

func main() {
    println("Hello")
}
</content>
</create_file>
```

**Features:**
- Automatically creates parent directories
- Overwrites existing files
- Validates path (prevents `..` directory traversal)

### 2. EXECUTE_COMMAND

Runs a shell command in the working directory.

```xml
<execute_command>
<command>go mod init myproject</command>
<description>Initialize Go module</description>
</execute_command>
```

**Features:**
- Executes in the configured working directory
- Streams stdout/stderr to console
- Description is optional but recommended

### 3. CREATE_DIRECTORY

Creates a directory (and all parent directories).

```xml
<create_directory>
<path>pkg/models</path>
</create_directory>
```

**Features:**
- Creates full path (like `mkdir -p`)
- No-op if directory already exists

### 4. MODIFY_FILE

Modifies an existing file by searching and replacing content.

```xml
<modify_file>
<path>main.go</path>
<search>
func oldFunction() {
    return 1
}
</search>
<replace>
func newFunction() {
    return 2
}
</replace>
</modify_file>
```

**Features:**
- Replaces first occurrence only
- Fails if search string not found
- Preserves file permissions

### 5. READ_FILE

Reads a file and displays its content (useful for giving LLM context).

```xml
<read_file>
<path>config.yaml</path>
</read_file>
```

**Note:** In a production system, this would feed the content back to the LLM context.

## Usage in REPL

### Start the Agent

```bash
go run cmd/agent/main.go -prompts ./prompts
```

### Load Action-Aware Prompt

```
> /prompt coding-agent-with-actions
✓ Loaded system prompt: coding-agent-with-actions
```

### Set Working Directory

```
> /workdir /path/to/project
✓ Working directory set to: /path/to/project
```

### Enable Auto-Execution (Optional)

```
> /auto on
✓ Auto-execution enabled
```

**Safety Note:** With auto-execution OFF (default), you'll see actions but must approve them manually.

### Ask for Code Generation

```
> Create a simple HTTP server in Go with health check

I'll create a simple HTTP server for you.

<create_file>
<path>main.go</path>
<content>
package main
...
</content>
</create_file>

<create_file>
<path>go.mod</path>
<content>
module httpserver
go 1.21
</content>
</create_file>

📋 Detected 2 action(s):
  1. CREATE_FILE: main.go (523 bytes)
  2. CREATE_FILE: go.mod (31 bytes)

💡 Tip: Use /execute to run these actions, or enable auto-execution with /auto on
```

### Execute Actions Manually

When auto-execution is disabled, actions are displayed but not executed. In a future enhancement, you could add:

```
> /execute

[1/2] CREATE_FILE: main.go (523 bytes)
✓ Completed

[2/2] CREATE_FILE: go.mod (31 bytes)
✓ Completed

✅ All actions completed successfully
```

## REPL Commands

| Command | Description |
|---------|-------------|
| `/help` | Show all commands |
| `/workdir <path>` | Set working directory for actions |
| `/auto on\|off` | Enable/disable auto-execution |
| `/prompt <name>` | Load a system prompt |
| `/model <name>` | Switch LLM model |
| `/clear` | Clear conversation history |
| `/exit` | Exit REPL |

## Action Validation

All actions implement validation:

```go
type Action interface {
    Execute(ctx context.Context, workDir string) error
    Validate() error
    String() string
}
```

**Validation checks:**
- ✓ Paths are not empty
- ✓ No directory traversal (`..`)
- ✓ Commands are not empty
- ✓ Search strings exist (for modify)

## Example LLM Response

Here's what a well-formed LLM response looks like:

```
I'll create a simple web server with proper structure.

<create_directory>
<path>cmd/server</path>
</create_directory>

<create_file>
<path>cmd/server/main.go</path>
<content>
package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello!")
    })
    http.ListenAndServe(":8080", nil)
}
</content>
</create_file>

<create_file>
<path>go.mod</path>
<content>
module webserver

go 1.21
</content>
</create_file>

<execute_command>
<command>go mod tidy</command>
<description>Download dependencies</description>
</execute_command>

This server will listen on port 8080. Here's how the HTTP handler works:

```go
// This is just an explanation, not an action
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello!")
})
```

The handler responds to all requests on the root path.

To run the server:
<execute_command>
<command>go run cmd/server/main.go</command>
<description>Start the server</description>
</execute_command>
```

**Notice:**
- Action tags for actual file creation/commands
- Regular markdown for explanations
- Clear separation of intent

## Architecture

```
User Request
    │
    ▼
Agent.SendMessage()
    │
    ├─> Build messages with system prompt
    ├─> Stream from Ollama
    ├─> Accumulate full response
    │
    ▼
ActionParser.Parse(response)
    │
    ├─> Regex extract <create_file>
    ├─> Regex extract <execute_command>
    ├─> Regex extract <create_directory>
    ├─> Regex extract <modify_file>
    └─> Regex extract <read_file>
    │
    ▼
Display Actions to User
    │
    ▼
(Optional) ExecuteActions()
    │
    ├─> Validate each action
    ├─> Execute in sequence
    └─> Report results
```

## Advantages of This Approach

1. **Unambiguous Intent**: No guessing if code is an example or a directive
2. **Structured Parsing**: Regex-based extraction is reliable
3. **Safety**: Validation prevents malicious paths/commands
4. **Flexibility**: Can disable auto-execution for manual review
5. **Extensibility**: Easy to add new action types
6. **MCP-Like**: Similar to how Claude/GitHub Copilot handle tools

## Adding New Action Types

To add a new action type:

1. **Define the struct** in `actions.go`:
```go
type MyNewAction struct {
    Field1 string
    Field2 string
}
```

2. **Implement the Action interface**:
```go
func (a *MyNewAction) Execute(ctx context.Context, workDir string) error { ... }
func (a *MyNewAction) Validate() error { ... }
func (a *MyNewAction) String() string { ... }
```

3. **Add regex pattern** to `ActionParser`:
```go
myNewActionRegex: regexp.MustCompile(`<my_new_action>...`),
```

4. **Update Parse()** method to extract it:
```go
for _, match := range p.myNewActionRegex.FindAllStringSubmatch(response, -1) {
    actions = append(actions, &MyNewAction{...})
}
```

5. **Update system prompt** to document the new action

## Comparison: Action Tags vs Heuristics

| Aspect | Action Tags | Heuristic Parsing |
|--------|-------------|-------------------|
| **Clarity** | ✅ Explicit intent | ❌ Ambiguous |
| **Reliability** | ✅ Regex-based | ❌ Fragile patterns |
| **LLM Training** | Requires prompt | Works out-of-box |
| **Extensibility** | ✅ Easy to add actions | ❌ Hard to extend |
| **Safety** | ✅ Validated actions | ⚠️ Risky |
| **Example Code** | ✅ Can show examples | ❌ Confused with actions |

## Future Enhancements

- [ ] Add `/execute` command for manual action execution
- [ ] Store last parsed actions for review
- [ ] Add action confirmation prompts
- [ ] Implement dry-run mode
- [ ] Add action undo/rollback
- [ ] Feed `read_file` content back to LLM context
- [ ] Support action dependencies/ordering
- [ ] Add git integration actions (commit, push, etc.)
- [ ] Support templating in file content
- [ ] Add search/replace with regex support

## Testing

Run tests:
```bash
go test ./internal/agent/... -v
```

Tests cover:
- Action parsing from LLM responses
- File creation with subdirectories
- Path validation (security)
- Modify file operations
- All action types

## Security Considerations

1. **Path Validation**: Prevents `..` directory traversal
2. **Working Directory**: All operations scoped to workdir
3. **Command Validation**: No shell injection (uses `exec.Command`)
4. **Manual Approval**: Auto-execution disabled by default
5. **No Absolute Paths**: Actions use relative paths only

## Conclusion

The action tag system provides a **robust, unambiguous, and safe** way for the LLM to communicate its intentions to the agent. It's inspired by MCP (Model Context Protocol) and similar to how production coding agents like GitHub Copilot Workspace operate.

The key insight: **Don't make the agent guess — make the LLM be explicit.**
