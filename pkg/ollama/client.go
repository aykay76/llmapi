package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents an Ollama API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Ollama API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetTimeout sets the underlying HTTP client's timeout. Use 0 to disable the
// client-side timeout (recommended for long-lived streaming connections) and
// control cancellation via context.Context instead.
func (c *Client) SetTimeout(d time.Duration) {
	c.httpClient.Timeout = d
}

// Common Types

// ModelConfig represents the model configuration
type ModelConfig struct {
	NumCtx      int      `json:"num_ctx,omitempty"`     // Maximum context size
	NumGQA      int      `json:"num_gqa,omitempty"`     // Number of GQA groups
	NumGPU      int      `json:"num_gpu,omitempty"`     // Number of layers to send to GPU
	NumThread   int      `json:"num_thread,omitempty"`  // Number of threads to use
	Temperature float64  `json:"temperature,omitempty"` // Temperature for sampling
	TopK        int      `json:"top_k,omitempty"`       // Top-k for sampling
	TopP        float64  `json:"top_p,omitempty"`       // Top-p for sampling
	StopWords   []string `json:"stop,omitempty"`        // Stop words for text generation
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat API Types

// ChatRequest represents a request to the chat API
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
	Format   string        `json:"format,omitempty"`
	Options  *ModelConfig  `json:"options,omitempty"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

// Generate API Types

// GenerateRequest represents a request to the generate API
type GenerateRequest struct {
	Model    string       `json:"model"`
	Prompt   string       `json:"prompt"`
	System   string       `json:"system,omitempty"`
	Template string       `json:"template,omitempty"`
	Context  []int        `json:"context,omitempty"`
	Stream   bool         `json:"stream,omitempty"`
	Raw      bool         `json:"raw,omitempty"`
	Format   string       `json:"format,omitempty"`
	Options  *ModelConfig `json:"options,omitempty"`
}

// GenerateResponse represents a response from the generate API
type GenerateResponse struct {
	Model              string `json:"model"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	CreatedAt          string `json:"created_at"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// Embeddings API Types

// EmbeddingsRequest represents a request to create embeddings
type EmbeddingsRequest struct {
	Model   string       `json:"model"`
	Prompt  string       `json:"prompt"`
	Options *ModelConfig `json:"options,omitempty"`
}

// EmbeddingsResponse represents a response from the embeddings API
type EmbeddingsResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Model Management Types

// ListModelsResponse represents the response from listing models
type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	Name       string       `json:"name"`
	ModifiedAt time.Time    `json:"modified_at"`
	Size       int64        `json:"size"`
	Digest     string       `json:"digest"`
	Details    ModelDetails `json:"details"`
}

// ModelDetails represents detailed information about a model
type ModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ShowModelResponse represents the response from showing model details
type ShowModelResponse struct {
	License    string       `json:"license"`
	ModelFile  string       `json:"modelfile"`
	Parameters string       `json:"parameters"`
	Template   string       `json:"template"`
	System     string       `json:"system"`
	Details    ModelDetails `json:"details"`
}

// CopyModelRequest represents a request to copy a model
type CopyModelRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// DeleteModelRequest represents a request to delete a model
type DeleteModelRequest struct {
	Name string `json:"name"`
}

// PullModelRequest represents a request to pull a model
type PullModelRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
}

// PullModelResponse represents the response from pulling a model
type PullModelResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// PushModelRequest represents a request to push a model
type PushModelRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
}

// CreateModelRequest represents a request to create a model
type CreateModelRequest struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	ModelFile string `json:"modelfile"`
}

// Chat API Methods

// CreateChatCompletion sends a chat completion request to the Ollama API
func (c *Client) CreateChatCompletion(req *ChatRequest) (*ChatResponse, error) {
	resp := &ChatResponse{}
	err := c.sendRequest(http.MethodPost, "/api/chat", req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Generate API Methods

// CreateGeneration sends a generate request to the Ollama API
func (c *Client) CreateGeneration(req *GenerateRequest) (*GenerateResponse, error) {
	resp := &GenerateResponse{}
	err := c.sendRequest(http.MethodPost, "/api/generate", req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// StreamGenerate sends a generate request and streams partial responses via the
// provided callback. The callback is invoked for each chunk (partial text).
// If the callback returns an error, streaming stops and that error is returned.
func (c *Client) StreamGenerate(req *GenerateRequest, onChunk func(string) error) error {
	// Ensure streaming is enabled
	req.Stream = true

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// debug output req
	fmt.Println(req)

	url := c.baseURL + "/api/generate"
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// If server returned non-200, read body and return error
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read streaming body line-by-line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to parse JSON chunk, but accept raw text as fallback
		var chunk struct {
			Response string `json:"response"`
			Delta    string `json:"delta"`
			Done     bool   `json:"done"`
			Error    string `json:"error"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err == nil {
			// prefer Delta if present
			part := chunk.Response
			if part == "" {
				part = chunk.Delta
			}
			if chunk.Error != "" {
				resp.Body.Close()
				return fmt.Errorf("stream error: %s", chunk.Error)
			}
			if part != "" {
				if err := onChunk(part); err != nil {
					resp.Body.Close()
					return err
				}
			}
			if chunk.Done {
				resp.Body.Close()
				return nil
			}
			continue
		}

		// Not JSON — treat as raw chunk
		if err := onChunk(line); err != nil {
			resp.Body.Close()
			return err
		}
	}

	// scanner finished — check error
	if err := scanner.Err(); err != nil {
		resp.Body.Close()
		return fmt.Errorf("error reading stream: %w", err)
	}

	resp.Body.Close()
	return nil
}

// StreamGenerateWithContext streams generate responses and accepts a
// context.Context so the caller can control cancellation and deadlines.
func (c *Client) StreamGenerateWithContext(ctx context.Context, reqBody *GenerateRequest, onChunk func(string) error) error {
	reqBody.Stream = true

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk struct {
			Response string `json:"response"`
			Delta    string `json:"delta"`
			Done     bool   `json:"done"`
			Error    string `json:"error"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err == nil {
			part := chunk.Response
			if part == "" {
				part = chunk.Delta
			}
			if chunk.Error != "" {
				resp.Body.Close()
				return fmt.Errorf("stream error: %s", chunk.Error)
			}
			if part != "" {
				if err := onChunk(part); err != nil {
					resp.Body.Close()
					return err
				}
			}
			if chunk.Done {
				resp.Body.Close()
				return nil
			}
			continue
		}

		if err := onChunk(line); err != nil {
			resp.Body.Close()
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		resp.Body.Close()
		return fmt.Errorf("error reading stream: %w", err)
	}

	resp.Body.Close()
	return nil
}

// StreamChat streams chat responses similarly to StreamGenerate.
func (c *Client) StreamChat(req *ChatRequest, onChunk func(string) error) error {
	req.Stream = true

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/chat"
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk struct {
			Response string `json:"response"`
			Delta    string `json:"delta"`
			Done     bool   `json:"done"`
			Error    string `json:"error"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err == nil {
			part := chunk.Response
			if part == "" {
				part = chunk.Delta
			}
			if chunk.Error != "" {
				resp.Body.Close()
				return fmt.Errorf("stream error: %s", chunk.Error)
			}
			if part != "" {
				if err := onChunk(part); err != nil {
					resp.Body.Close()
					return err
				}
			}
			if chunk.Done {
				resp.Body.Close()
				return nil
			}
			continue
		}

		if err := onChunk(line); err != nil {
			resp.Body.Close()
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		resp.Body.Close()
		return fmt.Errorf("error reading stream: %w", err)
	}

	resp.Body.Close()
	return nil
}

// StreamChatWithContext streams chat responses and accepts a context for
// cancellation and deadline control.
func (c *Client) StreamChatWithContext(ctx context.Context, reqBody *ChatRequest, onChunk func(string) error) error {
	reqBody.Stream = true

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chunk struct {
			Response string `json:"response"`
			Delta    string `json:"delta"`
			Done     bool   `json:"done"`
			Error    string `json:"error"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err == nil {
			part := chunk.Response
			if part == "" {
				part = chunk.Delta
			}
			if chunk.Error != "" {
				resp.Body.Close()
				return fmt.Errorf("stream error: %s", chunk.Error)
			}
			if part != "" {
				if err := onChunk(part); err != nil {
					resp.Body.Close()
					return err
				}
			}
			if chunk.Done {
				resp.Body.Close()
				return nil
			}
			continue
		}

		if err := onChunk(line); err != nil {
			resp.Body.Close()
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		resp.Body.Close()
		return fmt.Errorf("error reading stream: %w", err)
	}

	resp.Body.Close()
	return nil
}

// Embeddings API Methods

// CreateEmbeddings sends an embeddings request to the Ollama API
func (c *Client) CreateEmbeddings(req *EmbeddingsRequest) (*EmbeddingsResponse, error) {
	resp := &EmbeddingsResponse{}
	err := c.sendRequest(http.MethodPost, "/api/embeddings", req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Model Management Methods

// ListModels lists all available models
func (c *Client) ListModels() (*ListModelsResponse, error) {
	resp := &ListModelsResponse{}
	err := c.sendRequest(http.MethodGet, "/api/tags", nil, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ShowModel shows details of a specific model
func (c *Client) ShowModel(name string) (*ShowModelResponse, error) {
	req := struct{ Name string }{Name: name}
	resp := &ShowModelResponse{}
	err := c.sendRequest(http.MethodPost, "/api/show", &req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CopyModel copies a model
func (c *Client) CopyModel(req *CopyModelRequest) error {
	return c.sendRequest(http.MethodPost, "/api/copy", req, nil)
}

// DeleteModel deletes a model
func (c *Client) DeleteModel(req *DeleteModelRequest) error {
	return c.sendRequest(http.MethodDelete, "/api/delete", req, nil)
}

// PullModel pulls a model from a registry
func (c *Client) PullModel(req *PullModelRequest) (*PullModelResponse, error) {
	resp := &PullModelResponse{}
	err := c.sendRequest(http.MethodPost, "/api/pull", req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PushModel pushes a model to a registry
func (c *Client) PushModel(req *PushModelRequest) error {
	return c.sendRequest(http.MethodPost, "/api/push", req, nil)
}

// CreateModel creates a new model
func (c *Client) CreateModel(req *CreateModelRequest) error {
	return c.sendRequest(http.MethodPost, "/api/create", req, nil)
}

// Helper Methods

// sendRequest is a helper function to send HTTP requests
func (c *Client) sendRequest(method, endpoint string, reqBody interface{}, response interface{}) error {
	var data []byte
	var err error

	if reqBody != nil {
		data, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	url := c.baseURL + endpoint
	var resp *http.Response

	if method == http.MethodPost {
		resp, err = c.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	} else {
		resp, err = c.httpClient.Get(url)
	}

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
