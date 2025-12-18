// Package moderation provides content moderation clients for safety checking.
package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/safety"
)

// OpenAIModerator implements the safety.Moderator interface using OpenAI's moderation API.
type OpenAIModerator struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewOpenAIModerator creates a new OpenAI moderation client.
func NewOpenAIModerator(apiKey string) *OpenAIModerator {
	return &OpenAIModerator{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.openai.com",
	}
}

// NewOpenAIModeratorWithHTTP creates a new OpenAI moderation client with custom HTTP client.
func NewOpenAIModeratorWithHTTP(apiKey string, httpClient *http.Client, baseURL string) *OpenAIModerator {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAIModerator{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// Classify classifies text for safety using OpenAI's moderation API.
func (m *OpenAIModerator) Classify(ctx context.Context, text string) (*safety.Classification, error) {
	if text == "" {
		return &safety.Classification{}, nil
	}

	req := moderationRequest{
		Input: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		m.baseURL+"/v1/moderations",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("moderation API returned %d: %s", resp.StatusCode, errResp.Error.Message)
	}

	var result moderationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no moderation results returned")
	}

	scores := result.Results[0].CategoryScores

	return &safety.Classification{
		Hate:            scores.Hate,
		HateWithThreat:  scores.HateThreatening,
		SelfHarm:        scores.SelfHarm,
		Sexual:          scores.Sexual,
		Violence:        scores.Violence,
		ViolenceGraphic: scores.ViolenceGraphic,
	}, nil
}

// Request/Response types for OpenAI Moderation API

type moderationRequest struct {
	Input string `json:"input"`
	Model string `json:"model,omitempty"`
}

type moderationResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Results []moderationResult `json:"results"`
}

type moderationResult struct {
	Flagged        bool                  `json:"flagged"`
	Categories     moderationCategories  `json:"categories"`
	CategoryScores moderationScores      `json:"category_scores"`
}

type moderationCategories struct {
	Hate            bool `json:"hate"`
	HateThreatening bool `json:"hate/threatening"`
	Harassment      bool `json:"harassment"`
	HarassmentThr   bool `json:"harassment/threatening"`
	SelfHarm        bool `json:"self-harm"`
	SelfHarmIntent  bool `json:"self-harm/intent"`
	SelfHarmInstr   bool `json:"self-harm/instructions"`
	Sexual          bool `json:"sexual"`
	SexualMinors    bool `json:"sexual/minors"`
	Violence        bool `json:"violence"`
	ViolenceGraphic bool `json:"violence/graphic"`
}

type moderationScores struct {
	Hate            float64 `json:"hate"`
	HateThreatening float64 `json:"hate/threatening"`
	Harassment      float64 `json:"harassment"`
	HarassmentThr   float64 `json:"harassment/threatening"`
	SelfHarm        float64 `json:"self-harm"`
	SelfHarmIntent  float64 `json:"self-harm/intent"`
	SelfHarmInstr   float64 `json:"self-harm/instructions"`
	Sexual          float64 `json:"sexual"`
	SexualMinors    float64 `json:"sexual/minors"`
	Violence        float64 `json:"violence"`
	ViolenceGraphic float64 `json:"violence/graphic"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NoOpModerator is a no-operation moderator for testing or when moderation is disabled.
type NoOpModerator struct{}

// NewNoOpModerator creates a new no-op moderator.
func NewNoOpModerator() *NoOpModerator {
	return &NoOpModerator{}
}

// Classify always returns an empty classification (all zeros).
func (m *NoOpModerator) Classify(ctx context.Context, text string) (*safety.Classification, error) {
	return &safety.Classification{}, nil
}

// MockModerator is a mock moderator for testing.
type MockModerator struct {
	Classification *safety.Classification
	Error          error
}

// NewMockModerator creates a new mock moderator.
func NewMockModerator(classification *safety.Classification, err error) *MockModerator {
	return &MockModerator{
		Classification: classification,
		Error:          err,
	}
}

// Classify returns the configured classification or error.
func (m *MockModerator) Classify(ctx context.Context, text string) (*safety.Classification, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Classification != nil {
		return m.Classification, nil
	}
	return &safety.Classification{}, nil
}
