package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Action represents an executable action parsed from LLM output
type Action interface {
	Execute(ctx context.Context, workDir string) error
	Validate() error
	String() string
}

// CreateFileAction represents a file creation action
type CreateFileAction struct {
	Path    string
	Content string
}

func (a *CreateFileAction) Execute(ctx context.Context, workDir string) error {
	fullPath := filepath.Join(workDir, a.Path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(a.Content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

func (a *CreateFileAction) Validate() error {
	if a.Path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	// Basic path validation - prevent directory traversal attacks
	if strings.Contains(a.Path, "..") {
		return fmt.Errorf("file path cannot contain '..'")
	}
	return nil
}

func (a *CreateFileAction) String() string {
	return fmt.Sprintf("CREATE_FILE: %s (%d bytes)", a.Path, len(a.Content))
}

// ExecuteCommandAction represents a shell command execution
type ExecuteCommandAction struct {
	Command     string
	Description string
}

func (a *ExecuteCommandAction) Execute(ctx context.Context, workDir string) error {
	// Parse command into parts
	parts := strings.Fields(a.Command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func (a *ExecuteCommandAction) Validate() error {
	if a.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	return nil
}

func (a *ExecuteCommandAction) String() string {
	desc := a.Description
	if desc == "" {
		desc = "no description"
	}
	return fmt.Sprintf("EXECUTE_COMMAND: %s (%s)", a.Command, desc)
}

// CreateDirectoryAction represents a directory creation action
type CreateDirectoryAction struct {
	Path string
}

func (a *CreateDirectoryAction) Execute(ctx context.Context, workDir string) error {
	fullPath := filepath.Join(workDir, a.Path)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
	}
	return nil
}

func (a *CreateDirectoryAction) Validate() error {
	if a.Path == "" {
		return fmt.Errorf("directory path cannot be empty")
	}
	if strings.Contains(a.Path, "..") {
		return fmt.Errorf("directory path cannot contain '..'")
	}
	return nil
}

func (a *CreateDirectoryAction) String() string {
	return fmt.Sprintf("CREATE_DIRECTORY: %s", a.Path)
}

// ModifyFileAction represents a file modification action
type ModifyFileAction struct {
	Path    string
	Search  string
	Replace string
}

func (a *ModifyFileAction) Execute(ctx context.Context, workDir string) error {
	fullPath := filepath.Join(workDir, a.Path)

	// Read existing file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	// Replace content
	newContent := strings.Replace(string(content), a.Search, a.Replace, 1)
	if newContent == string(content) {
		return fmt.Errorf("search string not found in file %s", fullPath)
	}

	// Write back
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

func (a *ModifyFileAction) Validate() error {
	if a.Path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	if a.Search == "" {
		return fmt.Errorf("search string cannot be empty")
	}
	return nil
}

func (a *ModifyFileAction) String() string {
	return fmt.Sprintf("MODIFY_FILE: %s", a.Path)
}

// ReadFileAction represents a file read request (returns content to LLM context)
type ReadFileAction struct {
	Path string
}

func (a *ReadFileAction) Execute(ctx context.Context, workDir string) error {
	fullPath := filepath.Join(workDir, a.Path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	// Print content for now - in a real implementation, this would be
	// added back to the LLM context
	fmt.Printf("\n=== Content of %s ===\n%s\n=== End ===\n\n", a.Path, string(content))
	return nil
}

func (a *ReadFileAction) Validate() error {
	if a.Path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	return nil
}

func (a *ReadFileAction) String() string {
	return fmt.Sprintf("READ_FILE: %s", a.Path)
}

// ActionParser parses LLM output to extract action tags
type ActionParser struct {
	createFileRegex     *regexp.Regexp
	executeCommandRegex *regexp.Regexp
	createDirRegex      *regexp.Regexp
	modifyFileRegex     *regexp.Regexp
	readFileRegex       *regexp.Regexp
	jsonBlockRegex      *regexp.Regexp
	fencedCodeRegex     *regexp.Regexp
}

// NewActionParser creates a new action parser
func NewActionParser() *ActionParser {
	return &ActionParser{
		createFileRegex:     regexp.MustCompile(`(?s)<create_file>\s*<path>(.*?)</path>\s*<content>(.*?)</content>\s*</create_file>`),
		executeCommandRegex: regexp.MustCompile(`(?s)<execute_command>\s*<command>(.*?)</command>(?:\s*<description>(.*?)</description>)?\s*</execute_command>`),
		createDirRegex:      regexp.MustCompile(`<create_directory>\s*<path>(.*?)</path>\s*</create_directory>`),
		modifyFileRegex:     regexp.MustCompile(`(?s)<modify_file>\s*<path>(.*?)</path>\s*<search>(.*?)</search>\s*<replace>(.*?)</replace>\s*</modify_file>`),
		readFileRegex:       regexp.MustCompile(`<read_file>\s*<path>(.*?)</path>\s*</read_file>`),
		// Matches fenced code blocks containing JSON: ```json {...} ``` or ``` {...} ```
		jsonBlockRegex:  regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\}|\\[.*?\\])\\s*```"),
		fencedCodeRegex: regexp.MustCompile("(?s)```(\\w+)?\\s*(.*?)\\s*```"),
	}
}

// Parse extracts all actions from the LLM response
func (p *ActionParser) Parse(response string) []Action {
	var actions []Action

	// First, attempt to find fenced JSON blocks and parse them. This lets
	// models return structured JSON instead of XML-style tags.
	for _, match := range p.jsonBlockRegex.FindAllStringSubmatch(response, -1) {
		if len(match) >= 2 {
			block := strings.TrimSpace(match[1])
			var parsed interface{}
			if err := json.Unmarshal([]byte(block), &parsed); err == nil {
				// handle array of file objects
				switch v := parsed.(type) {
				case []interface{}:
					for _, item := range v {
						if obj, ok := item.(map[string]interface{}); ok {
							// common keys: path/name and content
							path := ""
							content := ""
							if pth, ok := obj["path"].(string); ok {
								path = pth
							} else if nm, ok := obj["name"].(string); ok {
								path = nm
							}
							if cnt, ok := obj["content"].(string); ok {
								content = cnt
							} else if cnt, ok := obj["body"].(string); ok {
								content = cnt
							}
							if path != "" && content != "" {
								actions = append(actions, &CreateFileAction{Path: strings.TrimSpace(path), Content: content})
							}
						}
					}
				case map[string]interface{}:
					// look for top-level keys like "create_file", "create_files", "files"
					for key, val := range v {
						lk := strings.ToLower(key)
						if lk == "create_file" {
							if obj, ok := val.(map[string]interface{}); ok {
								path := ""
								content := ""
								if pth, ok := obj["path"].(string); ok {
									path = pth
								}
								if cnt, ok := obj["content"].(string); ok {
									content = cnt
								}
								if path != "" && content != "" {
									actions = append(actions, &CreateFileAction{Path: strings.TrimSpace(path), Content: content})
								}
							}
						} else if lk == "create_files" || lk == "files" {
							if arr, ok := val.([]interface{}); ok {
								for _, item := range arr {
									if obj, ok := item.(map[string]interface{}); ok {
										path := ""
										content := ""
										if pth, ok := obj["path"].(string); ok {
											path = pth
										} else if nm, ok := obj["name"].(string); ok {
											path = nm
										}
										if cnt, ok := obj["content"].(string); ok {
											content = cnt
										} else if cnt, ok := obj["body"].(string); ok {
											content = cnt
										}
										if path != "" && content != "" {
											actions = append(actions, &CreateFileAction{Path: strings.TrimSpace(path), Content: content})
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Next, detect generic fenced code blocks (```html, ```js, etc.) and
	// infer filenames when none are explicitly provided. This handles
	// model outputs that return code but omit filenames.
	nameCount := map[string]int{}
	for _, match := range p.fencedCodeRegex.FindAllStringSubmatch(response, -1) {
		if len(match) < 3 {
			continue
		}
		lang := strings.ToLower(strings.TrimSpace(match[1]))
		body := match[2]

		// Try to extract filename from common comment patterns
		var filename string
		// HTML comment: <!-- filename: index.html -->
		htmlFilenameRe := regexp.MustCompile(`(?i)<!--\s*filename\s*:\s*(\S+)\s*-->|<!--\s*file\s*:\s*(\S+)\s*-->`)
		// Line comment: // filename: app.js or # filename: foo
		lineCommentRe := regexp.MustCompile(`(?m)^[ \t]*(?:\/\/|#)\s*filename\s*:\s*(\S+)`)
		// Block comment: /* filename: app.js */
		blockCommentRe := regexp.MustCompile(`(?s)/\*\s*filename\s*:\s*(\S+)\s*\*/`)

		if lang == "html" {
			if m := htmlFilenameRe.FindStringSubmatch(body); m != nil {
				if m[1] != "" {
					filename = m[1]
				} else if m[2] != "" {
					filename = m[2]
				}
			}
		}
		if filename == "" {
			if m := lineCommentRe.FindStringSubmatch(body); m != nil {
				filename = m[1]
			}
		}
		if filename == "" {
			if m := blockCommentRe.FindStringSubmatch(body); m != nil {
				filename = m[1]
			}
		}

		// If still empty, pick a sensible default based on language or content
		if filename == "" {
			switch lang {
			case "html":
				// if body looks like a full HTML page, use index.html
				if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
					filename = "index.html"
				} else {
					filename = "index.html"
				}
			case "css":
				filename = "style.css"
			case "js", "javascript":
				filename = "script.js"
			case "ts":
				filename = "script.ts"
			case "json":
				filename = "data.json"
			case "md", "markdown":
				filename = "README.md"
			case "go":
				filename = "main.go"
			case "py":
				filename = "main.py"
			case "sh", "bash":
				filename = "run.sh"
			default:
				// no language or unknown: use .txt
				filename = "file.txt"
			}
		}

		// Sanitize filename (basic) and ensure uniqueness
		filename = strings.TrimSpace(filename)
		filename = strings.ReplaceAll(filename, "..", "")
		base := filename
		if cnt := nameCount[base]; cnt > 0 {
			ext := filepath.Ext(base)
			nameOnly := strings.TrimSuffix(base, ext)
			filename = fmt.Sprintf("%s.%d%s", nameOnly, cnt, ext)
		}
		nameCount[base]++

		// Emit create file action
		actions = append(actions, &CreateFileAction{Path: filename, Content: body})
	}

	// Parse create_file actions
	for _, match := range p.createFileRegex.FindAllStringSubmatch(response, -1) {
		if len(match) >= 3 {
			actions = append(actions, &CreateFileAction{
				Path:    strings.TrimSpace(match[1]),
				Content: strings.TrimSpace(match[2]),
			})
		}
	}

	// Parse execute_command actions
	for _, match := range p.executeCommandRegex.FindAllStringSubmatch(response, -1) {
		if len(match) >= 2 {
			description := ""
			if len(match) >= 3 {
				description = strings.TrimSpace(match[2])
			}
			actions = append(actions, &ExecuteCommandAction{
				Command:     strings.TrimSpace(match[1]),
				Description: description,
			})
		}
	}

	// Parse create_directory actions
	for _, match := range p.createDirRegex.FindAllStringSubmatch(response, -1) {
		if len(match) >= 2 {
			actions = append(actions, &CreateDirectoryAction{
				Path: strings.TrimSpace(match[1]),
			})
		}
	}

	// Parse modify_file actions
	for _, match := range p.modifyFileRegex.FindAllStringSubmatch(response, -1) {
		if len(match) >= 4 {
			actions = append(actions, &ModifyFileAction{
				Path:    strings.TrimSpace(match[1]),
				Search:  strings.TrimSpace(match[2]),
				Replace: strings.TrimSpace(match[3]),
			})
		}
	}

	// Parse read_file actions
	for _, match := range p.readFileRegex.FindAllStringSubmatch(response, -1) {
		if len(match) >= 2 {
			actions = append(actions, &ReadFileAction{
				Path: strings.TrimSpace(match[1]),
			})
		}
	}

	return actions
}

// ExecuteActions executes a list of actions in order
func ExecuteActions(ctx context.Context, actions []Action, workDir string) error {
	var failed []error

	for i, action := range actions {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(actions), action.String())

		// Validate
		if err := action.Validate(); err != nil {
			// Log and continue with next action
			fmt.Printf("✖ Validation failed for action %d: %v\n", i+1, err)
			failed = append(failed, fmt.Errorf("validation failed for action %d: %w", i+1, err))
			continue
		}

		// Execute
		if err := action.Execute(ctx, workDir); err != nil {
			// Log and continue with next action
			fmt.Printf("✖ Execution failed for action %d: %v\n", i+1, err)
			failed = append(failed, fmt.Errorf("execution failed for action %d: %w", i+1, err))
			continue
		}

		fmt.Printf("✓ Completed\n")
	}

	if len(failed) > 0 {
		return fmt.Errorf("completed with %d failure(s)", len(failed))
	}

	return nil
}
