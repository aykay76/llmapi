# Action System Architecture

## Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         User                                 │
│  "Create a web server in Go with health check endpoint"     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Agent (REPL)                              │
│  • Adds user message to conversation history                │
│  • Prepends system prompt with action instructions          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  Ollama LLM (qwen-coder)                     │
│  Generates response with explicit action tags:              │
│                                                              │
│  <create_file>                                               │
│    <path>main.go</path>                                      │
│    <content>package main...</content>                        │
│  </create_file>                                              │
│                                                              │
│  <execute_command>                                           │
│    <command>go mod tidy</command>                            │
│  </execute_command>                                          │
│                                                              │
│  Regular markdown for explanations only.                    │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              ActionParser.Parse(response)                    │
│                                                              │
│  Regex patterns extract:                                    │
│  • <create_file> → CreateFileAction                         │
│  • <execute_command> → ExecuteCommandAction                 │
│  • <create_directory> → CreateDirectoryAction               │
│  • <modify_file> → ModifyFileAction                         │
│  • <read_file> → ReadFileAction                             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           Display Detected Actions to User                  │
│                                                              │
│  📋 Detected 2 action(s):                                   │
│    1. CREATE_FILE: main.go (234 bytes)                      │
│    2. EXECUTE_COMMAND: go mod tidy                          │
│                                                              │
│  💡 Tip: Use /auto on to auto-execute                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                ┌────────┴────────┐
                │                 │
        Auto Enabled?      Auto Disabled?
                │                 │
                ▼                 ▼
    ┌──────────────────┐   ┌──────────────────┐
    │ Execute Actions  │   │  Wait for /execute│
    │  Automatically   │   │  (future feature) │
    └────────┬─────────┘   └──────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│            ExecuteActions(ctx, actions, workDir)             │
│                                                              │
│  For each action:                                            │
│    1. Validate() - Check paths, commands                    │
│    2. Execute() - Run action in workDir                     │
│    3. Report status                                         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   File System / Shell                        │
│                                                              │
│  ✓ main.go created                                          │
│  ✓ go mod tidy executed                                     │
│                                                              │
│  ✅ All actions completed successfully                       │
└─────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Why XML-Style Tags?

**Alternatives considered:**
- JSON blocks: `{"action": "create_file", ...}`
- Custom syntax: `[CREATE_FILE:main.go]`
- Markdown headers: `### CREATE_FILE: main.go`

**Why XML won:**
- ✅ Natural for LLMs (trained on HTML/XML)
- ✅ Supports multi-line content easily
- ✅ Clear nesting structure
- ✅ Human-readable
- ✅ Distinct from markdown (no confusion)

### 2. Why System Prompt Instructions?

**Alternative:** Train the model on action format

**Why prompt-based won:**
- ✅ Works with any model (no fine-tuning)
- ✅ Easy to update/modify
- ✅ Transparent to users
- ✅ Can be versioned
- ✅ Model-agnostic

### 3. Why Separate Actions from Explanations?

**Problem:** LLMs naturally want to show examples

**Solution:** 
- `<create_file>` tags = "DO THIS"
- ` ```language ` blocks = "HERE'S AN EXAMPLE"

This allows the LLM to:
1. Create actual files
2. Show code examples for explanation
3. Provide context without triggering actions

### 4. Why Auto-Execute Toggle?

**Safety:** Off by default prevents:
- Accidental file overwrites
- Unintended command execution
- Malicious action execution

**Convenience:** On allows:
- Rapid prototyping
- Automated workflows
- Trusted prompt scenarios

## Comparison to Other Systems

### GitHub Copilot Workspace

```
Copilot:  "I'll create these files..."
          [Plan shown to user]
          [User approves]
          [Files created]

Our Agent: "I'll create these files..."
           <create_file>...</create_file>
           [Actions shown to user]
           [Auto-execute or manual]
           [Files created]
```

**Similar philosophy:** Explicit, reviewable actions

### Claude MCP (Model Context Protocol)

```
MCP:       Tool definitions sent to LLM
           LLM returns tool_use blocks
           Server executes tools

Our Agent: Action definitions in system prompt
           LLM returns action tags
           Agent executes actions
```

**Similar architecture:** Structured tool/action calling

### Cursor / Windsurf

```
Cursor:    LLM generates diffs
           Applies to open files
           User reviews changes

Our Agent: LLM generates full files
           Creates from scratch
           User reviews actions
```

**Different approach:** We build from scratch, they modify existing

## Security Model

```
┌─────────────────────────────────────────────────────────────┐
│                     Security Layers                          │
└─────────────────────────────────────────────────────────────┘

Layer 1: Path Validation
   • No ".." directory traversal
   • Relative paths only
   • Path must not be empty

Layer 2: Working Directory Scope
   • All actions scoped to workDir
   • No access outside workDir
   • User explicitly sets workDir

Layer 3: Command Execution
   • Uses exec.Command (no shell injection)
   • No piping or chaining
   • Command validation

Layer 4: Manual Approval (default)
   • Auto-execute disabled by default
   • User must /auto on to enable
   • Actions shown before execution

Layer 5: Action Validation
   • Each action validates itself
   • Fails fast on invalid input
   • Clear error messages
```

## Future Extensions

### Planned Actions

```xml
<git_commit>
  <message>Initial commit</message>
  <files>main.go go.mod</files>
</git_commit>

<git_push>
  <remote>origin</remote>
  <branch>main</branch>
</git_push>

<run_tests>
  <package>./...</package>
</run_tests>

<search_replace>
  <path>main.go</path>
  <regex>func (\w+)\(\)</regex>
  <replace>func $1(ctx context.Context)</replace>
</search_replace>

<template>
  <path>config.yaml</path>
  <template_file>templates/config.tmpl</template_file>
  <vars>
    <var name="port">8080</var>
    <var name="host">localhost</var>
  </vars>
</template>
```

### Planned Features

- **Action History**: Review previous actions
- **Undo/Rollback**: Revert actions
- **Dry Run Mode**: Show what would happen
- **Action Dependencies**: Order constraints
- **Conditional Execution**: If/then logic
- **Action Streaming**: Execute during generation
- **Multi-threaded Execution**: Parallel actions
- **Action Plugins**: User-defined actions

## Performance Characteristics

```
Action Parsing:  O(n) where n = response length
Action Execution: O(m) where m = number of actions
File Creation:   ~1-5ms per file
Directory Creation: ~1ms per directory
Command Execution: Variable (depends on command)

Memory Usage:
  • Response buffer: ~1-10KB
  • Action list: ~1KB per action
  • File content: As needed

Concurrency:
  • Parsing: Single-threaded
  • Execution: Sequential (future: parallel)
  • Streaming: Concurrent with parsing
```
