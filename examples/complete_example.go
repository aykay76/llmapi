// Complete Example - Demonstrates basic Ollama API usage with streaming
//
// Run this example with:
//
//	go run examples/complete_example.go
//
// Note: This file cannot be compiled together with agent_example.go
// as they both have main functions. Run them separately.
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aykay76/llmapi/pkg/ollama"
)

func main() {
	// Create a new Ollama client
	client := ollama.NewClient("http://localhost:11434")

	// List available models
	fmt.Println("Listing available models...")
	models, err := client.ListModels()
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}
	for _, model := range models.Models {
		fmt.Printf("- %s (Modified: %s, Size: %d bytes)\n",
			model.Name, model.ModifiedAt.Format("2006-01-02 15:04:05"), model.Size)
	}
	fmt.Println()

	// --- Non-blocking streaming example (use a WaitGroup to wait for completion) ---
	fmt.Println("\nStarting async stream (will wait for completion)...")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.SetTimeout(0) // 0 disables http.Client timeout; use context for control
		// small delay to show concurrent behavior
		time.Sleep(100 * time.Millisecond)
		err := client.StreamGenerate(&ollama.GenerateRequest{
			Model: "qwen3-coder:30b",
			System: `You are an AI coding assistant, powered by LLM technology. You operate in an IDE-like environment and are pair programming with a USER to solve their coding task.

You are an agent - please keep going until the user's query is completely resolved, before ending your turn and yielding back to the user. Only terminate your turn when you are sure that the problem is solved. Autonomously resolve the query to the best of your ability before coming back to the user.

Your main goal is to follow the USER's instructions at each message.

COMMUNICATION RULES:
- Always ensure only relevant sections (code snippets, tables, commands, or structured data) are formatted in valid Markdown with proper fencing.
- Avoid wrapping the entire message in a single code block. Use Markdown only where semantically correct.
- Use backticks for file, directory, function, and class names.
- When communicating with the user, optimize your writing for clarity and skimmability.
- Do not add narration comments inside code just to explain actions.
- Refer to code changes as "edits" not "patches".
- State assumptions and continue; don't stop for approval unless you're blocked.

CODE STYLE RULES:
- Write code that is clear and readable
- Use meaningful variable and function names
- Follow language-specific best practices and conventions
- Add appropriate comments and documentation
- Handle errors appropriately
- Write testable and maintainable code

Your goal is to help the user write better code and solve their programming problems effectively.`,
			Prompt: "Write a program in Go that can interact with Ollama using APIs. Follow best practices in terms of Go project structure and ensure SOLID principles and testing.",
		}, func(chunk string) error {
			fmt.Print(chunk)
			return nil
		})
		if err != nil {
			log.Printf("Async stream error: %v", err)
		}
	}()

	// Wait until streaming goroutine completes before exiting
	wg.Wait()
}
