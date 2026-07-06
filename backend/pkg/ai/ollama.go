// Package ai is a thin client for a self-hosted Ollama server, used for
// on-premise embeddings (semantic search) and generation (ask-your-files RAG).
// Everything stays on your infrastructure — no data leaves the network.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// Client talks to an Ollama HTTP API.
type Client struct {
	base       string
	embedModel string
	chatModel  string
	http       *http.Client
}

// New builds an Ollama client.
func New(baseURL, embedModel, chatModel string) *Client {
	return &Client{
		base:       strings.TrimRight(baseURL, "/"),
		embedModel: embedModel,
		chatModel:  chatModel,
		http:       &http.Client{Timeout: 120 * time.Second},
	}
}

// Embed returns the embedding vector for a piece of text.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": c.embedModel, "prompt": text})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embeddings: %s", resp.Status)
	}
	var out struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned an empty embedding")
	}
	return out.Embedding, nil
}

// Generate returns a completion for a prompt (non-streaming).
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{"model": c.chatModel, "prompt": prompt, "stream": false})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama generate: %s", resp.Status)
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Response), nil
}

// Cosine returns the cosine similarity of two equal-length vectors.
func Cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}
