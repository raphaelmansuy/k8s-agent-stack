/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raphaelmansuy/agentstack/internal/domain/safety"
)

func TestNewOpenAIModerator(t *testing.T) {
	moderator := NewOpenAIModerator("test-api-key")
	if moderator == nil {
		t.Fatal("expected non-nil moderator")
	}
	if moderator.apiKey != "test-api-key" {
		t.Errorf("expected apiKey 'test-api-key', got '%s'", moderator.apiKey)
	}
	if moderator.baseURL != "https://api.openai.com" {
		t.Errorf("expected default baseURL, got '%s'", moderator.baseURL)
	}
}

func TestOpenAIModerator_Classify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/moderations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		if req["input"] != "test content" {
			t.Errorf("expected input 'test content', got '%v'", req["input"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "modr-123",
			"model": "text-moderation-latest",
			"results": []map[string]interface{}{
				{
					"flagged": false,
					"categories": map[string]bool{
						"hate":     false,
						"violence": false,
					},
					"category_scores": map[string]float64{
						"hate":             0.1,
						"hate/threatening": 0.05,
						"self-harm":        0.02,
						"sexual":           0.01,
						"violence":         0.15,
						"violence/graphic": 0.03,
					},
				},
			},
		})
	}))
	defer server.Close()

	moderator := NewOpenAIModeratorWithHTTP("test-key", server.Client(), server.URL)

	classification, err := moderator.Classify(context.Background(), "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if classification.Hate != 0.1 {
		t.Errorf("expected Hate 0.1, got %f", classification.Hate)
	}
	if classification.Violence != 0.15 {
		t.Errorf("expected Violence 0.15, got %f", classification.Violence)
	}
	if classification.HateWithThreat != 0.05 {
		t.Errorf("expected HateWithThreat 0.05, got %f", classification.HateWithThreat)
	}
}

func TestOpenAIModerator_Classify_EmptyText(t *testing.T) {
	moderator := NewOpenAIModerator("test-key")

	classification, err := moderator.Classify(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty text should return zero scores
	if classification.Hate != 0 {
		t.Errorf("expected Hate 0 for empty text, got %f", classification.Hate)
	}
}

func TestOpenAIModerator_Classify_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
				"code":    "invalid_api_key",
			},
		})
	}))
	defer server.Close()

	moderator := NewOpenAIModeratorWithHTTP("bad-key", server.Client(), server.URL)

	_, err := moderator.Classify(context.Background(), "test content")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestOpenAIModerator_Classify_NoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "modr-123",
			"results": []interface{}{},
		})
	}))
	defer server.Close()

	moderator := NewOpenAIModeratorWithHTTP("test-key", server.Client(), server.URL)

	_, err := moderator.Classify(context.Background(), "test content")
	if err == nil {
		t.Fatal("expected error for empty results")
	}
}

func TestNoOpModerator(t *testing.T) {
	moderator := NewNoOpModerator()

	classification, err := moderator.Classify(context.Background(), "any text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All scores should be zero
	if classification.Hate != 0 {
		t.Errorf("expected Hate 0, got %f", classification.Hate)
	}
	if classification.Violence != 0 {
		t.Errorf("expected Violence 0, got %f", classification.Violence)
	}
}

func TestMockModerator_WithClassification(t *testing.T) {
	expected := &safety.Classification{
		Hate:     0.5,
		Violence: 0.3,
	}
	moderator := NewMockModerator(expected, nil)

	classification, err := moderator.Classify(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if classification.Hate != 0.5 {
		t.Errorf("expected Hate 0.5, got %f", classification.Hate)
	}
}

func TestMockModerator_WithError(t *testing.T) {
	expectedErr := errors.New("mock error")
	moderator := NewMockModerator(nil, expectedErr)

	_, err := moderator.Classify(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "mock error" {
		t.Errorf("expected 'mock error', got '%s'", err.Error())
	}
}

func TestMockModerator_Default(t *testing.T) {
	moderator := NewMockModerator(nil, nil)

	classification, err := moderator.Classify(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty classification
	if classification.Hate != 0 {
		t.Errorf("expected Hate 0, got %f", classification.Hate)
	}
}
