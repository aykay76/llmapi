package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestActionParser_Parse(t *testing.T) {
	parser := NewActionParser()

	response := `
I'll create a simple HTTP server for you.

<create_file>
<path>main.go</path>
<content>
package main

func main() {
    println("Hello")
}
</content>
</create_file>

<create_directory>
<path>pkg/models</path>
</create_directory>

<execute_command>
<command>go mod init myproject</command>
<description>Initialize Go module</description>
</execute_command>

This is the basic structure you need.
`

	actions := parser.Parse(response)

	if len(actions) != 3 {
		t.Fatalf("Expected 3 actions, got %d", len(actions))
	}

	// Find actions by type (order may vary based on regex matching order)
	var createFile *CreateFileAction
	var createDir *CreateDirectoryAction
	var execCmd *ExecuteCommandAction

	for _, action := range actions {
		switch a := action.(type) {
		case *CreateFileAction:
			createFile = a
		case *CreateDirectoryAction:
			createDir = a
		case *ExecuteCommandAction:
			execCmd = a
		}
	}

	// Check CreateFileAction
	if createFile == nil {
		t.Error("Expected CreateFileAction not found")
	} else {
		if createFile.Path != "main.go" {
			t.Errorf("Expected path 'main.go', got '%s'", createFile.Path)
		}
		if len(createFile.Content) == 0 {
			t.Error("Expected non-empty content")
		}
	}

	// Check CreateDirectoryAction
	if createDir == nil {
		t.Error("Expected CreateDirectoryAction not found")
	} else {
		if createDir.Path != "pkg/models" {
			t.Errorf("Expected path 'pkg/models', got '%s'", createDir.Path)
		}
	}

	// Check ExecuteCommandAction
	if execCmd == nil {
		t.Error("Expected ExecuteCommandAction not found")
	} else {
		if execCmd.Command != "go mod init myproject" {
			t.Errorf("Expected command 'go mod init myproject', got '%s'", execCmd.Command)
		}
		if execCmd.Description != "Initialize Go module" {
			t.Errorf("Expected description 'Initialize Go module', got '%s'", execCmd.Description)
		}
	}
}

func TestCreateFileAction_Execute(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	action := &CreateFileAction{
		Path:    "test.txt",
		Content: "Hello, World!",
	}

	// Execute action
	ctx := context.Background()
	if err := action.Execute(ctx, tmpDir); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, "test.txt")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", string(content))
	}
}

func TestCreateFileAction_ExecuteWithSubdirectory(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	action := &CreateFileAction{
		Path:    "pkg/models/user.go",
		Content: "package models",
	}

	// Execute action
	ctx := context.Background()
	if err := action.Execute(ctx, tmpDir); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, "pkg/models/user.go")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "package models" {
		t.Errorf("Expected content 'package models', got '%s'", string(content))
	}
}

func TestCreateFileAction_Validate(t *testing.T) {
	tests := []struct {
		name      string
		action    *CreateFileAction
		expectErr bool
	}{
		{
			name: "valid action",
			action: &CreateFileAction{
				Path:    "test.txt",
				Content: "content",
			},
			expectErr: false,
		},
		{
			name: "empty path",
			action: &CreateFileAction{
				Path:    "",
				Content: "content",
			},
			expectErr: true,
		},
		{
			name: "path with ..",
			action: &CreateFileAction{
				Path:    "../etc/passwd",
				Content: "malicious",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error=%v, got error=%v", tt.expectErr, err)
			}
		})
	}
}

func TestCreateDirectoryAction_Execute(t *testing.T) {
	tmpDir := t.TempDir()

	action := &CreateDirectoryAction{
		Path: "pkg/models/user",
	}

	ctx := context.Background()
	if err := action.Execute(ctx, tmpDir); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify directory exists
	fullPath := filepath.Join(tmpDir, "pkg/models/user")
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected path to be a directory")
	}
}

func TestModifyFileAction_Execute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	testFile := filepath.Join(tmpDir, "test.txt")
	initialContent := "Hello, World!\nThis is a test."
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Modify file
	action := &ModifyFileAction{
		Path:    "test.txt",
		Search:  "World",
		Replace: "Go",
	}

	ctx := context.Background()
	if err := action.Execute(ctx, tmpDir); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "Hello, Go!\nThis is a test."
	if string(content) != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, string(content))
	}
}

func TestActionParser_ParseModifyFile(t *testing.T) {
	parser := NewActionParser()

	response := `
<modify_file>
<path>main.go</path>
<search>
func old() {
    return 1
}
</search>
<replace>
func new() {
    return 2
}
</replace>
</modify_file>
`

	actions := parser.Parse(response)

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	if modify, ok := actions[0].(*ModifyFileAction); ok {
		if modify.Path != "main.go" {
			t.Errorf("Expected path 'main.go', got '%s'", modify.Path)
		}
	} else {
		t.Errorf("Expected ModifyFileAction, got %T", actions[0])
	}
}
