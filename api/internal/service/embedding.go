package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/marcoshack/taskwondo/internal/model"
)

// EmbeddingService communicates with Ollama to generate text embeddings.
type EmbeddingService struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewEmbeddingService creates a new EmbeddingService.
// If baseURL is empty, all methods return ErrEmbeddingUnavailable.
func NewEmbeddingService(baseURL, modelName string) *EmbeddingService {
	return &EmbeddingService{
		baseURL: baseURL,
		model:   modelName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ollamaEmbedRequest is the request body for Ollama's /api/embed endpoint.
type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"` // string or []string
}

// ollamaEmbedResponse is the response from Ollama's /api/embed endpoint.
type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// ollamaTagsResponse is the response from Ollama's /api/tags endpoint.
type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name string `json:"name"`
}

// Embed generates a single embedding for the given text.
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	if s.baseURL == "" {
		return nil, model.ErrEmbeddingUnavailable
	}

	reqBody := ollamaEmbedRequest{
		Model: s.model,
		Input: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrEmbeddingUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: ollama returned %d: %s", model.ErrEmbeddingUnavailable, resp.StatusCode, string(respBody))
	}

	var result ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding embed response: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("%w: no embeddings returned", model.ErrEmbeddingUnavailable)
	}

	return result.Embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts in a single request.
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if s.baseURL == "" {
		return nil, model.ErrEmbeddingUnavailable
	}

	reqBody := ollamaEmbedRequest{
		Model: s.model,
		Input: texts,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling embed batch request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating embed batch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrEmbeddingUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: ollama returned %d: %s", model.ErrEmbeddingUnavailable, resp.StatusCode, string(respBody))
	}

	var result ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding embed batch response: %w", err)
	}

	return result.Embeddings, nil
}

// Probe checks whether Ollama is reachable and has the expected model loaded.
func (s *EmbeddingService) Probe(ctx context.Context) (bool, error) {
	if s.baseURL == "" {
		return false, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/tags", nil)
	if err != nil {
		return false, fmt.Errorf("creating probe request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, nil // Ollama not reachable — not an error, just unavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return false, nil
	}

	for _, m := range tags.Models {
		if m.Name == s.model || m.Name == s.model+":latest" {
			return true, nil
		}
	}

	return false, nil
}
