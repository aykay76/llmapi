package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aykay76/llmapi/internal/agent"
	"github.com/aykay76/llmapi/pkg/ollama"
)

func main() {
	// Command line flags
	ollamaURL := flag.String("url", "http://localhost:11434", "Ollama API URL")
	modelName := flag.String("model", "qwen3-coder:30b", "Model name to use")
	promptDir := flag.String("prompts", "prompts", "Directory containing system prompt files")
	systemPrompt := flag.String("system", "", "System prompt to use")
	flag.Parse()

	// Create Ollama client
	client := ollama.NewClient(*ollamaURL)
	client.SetTimeout(0) // Disable timeout for streaming

	// Create agent
	agentInstance := agent.NewAgent(client, *modelName)

	// Load system prompts from directory if specified
	if *promptDir != "" {
		if err := agentInstance.LoadSystemPromptDirectory(*promptDir); err != nil {
			log.Printf("Warning: failed to load prompt directory: %v", err)
		} else {
			fmt.Printf("✓ Loaded system prompts from: %s\n", *promptDir)
		}
	}

	// Set system prompt if specified
	if *systemPrompt != "" {
		agentInstance.SetSystemPrompt(*systemPrompt)
		fmt.Println("✓ System prompt set")
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown for SIGTERM only
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nReceived termination signal, shutting down...")
		cancel()
		os.Exit(0)
	}()

	// Start REPL
	if err := agentInstance.RunREPL(ctx); err != nil {
		log.Fatalf("REPL error: %v", err)
	}
}
