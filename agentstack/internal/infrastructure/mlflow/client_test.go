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

package mlflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:5000")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "http://localhost:5000" {
		t.Errorf("expected baseURL 'http://localhost:5000', got '%s'", client.baseURL)
	}
}

func TestCreateExperiment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/experiments/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"experiment_id": "exp-123",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	expID, err := client.CreateExperiment(context.Background(), "test-experiment", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expID != "exp-123" {
		t.Errorf("expected experiment ID 'exp-123', got '%s'", expID)
	}
}

func TestGetOrCreateExperiment_ExistingExperiment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.0/mlflow/experiments/get-by-name" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"experiment": map[string]interface{}{
					"experiment_id": "existing-exp-456",
					"name":          "test-experiment",
				},
			})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	expID, err := client.GetOrCreateExperiment(context.Background(), "test-experiment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expID != "existing-exp-456" {
		t.Errorf("expected experiment ID 'existing-exp-456', got '%s'", expID)
	}
}

func TestGetOrCreateExperiment_NewExperiment(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/api/2.0/mlflow/experiments/get-by-name" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
			})
			return
		}
		if r.URL.Path == "/api/2.0/mlflow/experiments/create" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"experiment_id": "new-exp-789",
			})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	expID, err := client.GetOrCreateExperiment(context.Background(), "new-experiment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expID != "new-exp-789" {
		t.Errorf("expected experiment ID 'new-exp-789', got '%s'", expID)
	}
}

func TestStartRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/runs/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["experiment_id"] != "exp-123" {
			t.Errorf("expected experiment_id 'exp-123', got '%v'", req["experiment_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"run": map[string]interface{}{
				"info": map[string]interface{}{
					"run_id":        "run-456",
					"experiment_id": "exp-123",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	run, err := client.StartRun(context.Background(), "exp-123", "test-run", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.RunID != "run-456" {
		t.Errorf("expected run ID 'run-456', got '%s'", run.RunID)
	}
	if run.ExperimentID != "exp-123" {
		t.Errorf("expected experiment ID 'exp-123', got '%s'", run.ExperimentID)
	}
}

func TestRun_LogParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/runs/log-parameter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["run_id"] != "run-123" {
			t.Errorf("expected run_id 'run-123', got '%v'", req["run_id"])
		}
		if req["key"] != "test-param" {
			t.Errorf("expected key 'test-param', got '%v'", req["key"])
		}
		if req["value"] != "test-value" {
			t.Errorf("expected value 'test-value', got '%v'", req["value"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	run := &Run{
		RunID:        "run-123",
		ExperimentID: "exp-123",
		client:       client,
	}

	err := run.LogParam(context.Background(), "test-param", "test-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_LogMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/runs/log-metric" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["run_id"] != "run-123" {
			t.Errorf("expected run_id 'run-123', got '%v'", req["run_id"])
		}
		if req["key"] != "accuracy" {
			t.Errorf("expected key 'accuracy', got '%v'", req["key"])
		}
		if req["value"] != 0.95 {
			t.Errorf("expected value 0.95, got '%v'", req["value"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	run := &Run{
		RunID:        "run-123",
		ExperimentID: "exp-123",
		client:       client,
	}

	err := run.LogMetric(context.Background(), "accuracy", 0.95, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_LogBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/runs/log-batch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["run_id"] != "run-123" {
			t.Errorf("expected run_id 'run-123', got '%v'", req["run_id"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	run := &Run{
		RunID:        "run-123",
		ExperimentID: "exp-123",
		client:       client,
	}

	err := run.LogBatch(context.Background(),
		map[string]string{"param1": "value1"},
		map[string]float64{"metric1": 0.5},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_End(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/runs/update" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["run_id"] != "run-123" {
			t.Errorf("expected run_id 'run-123', got '%v'", req["run_id"])
		}
		if req["status"] != "FINISHED" {
			t.Errorf("expected status 'FINISHED', got '%v'", req["status"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	run := &Run{
		RunID:        "run-123",
		ExperimentID: "exp-123",
		client:       client,
	}

	err := run.End(context.Background(), RunStatusFinished)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTagsToList(t *testing.T) {
	tags := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	list := tagsToList(tags)
	if len(list) != 2 {
		t.Errorf("expected 2 tags, got %d", len(list))
	}

	// Verify structure (order may vary)
	foundKeys := make(map[string]bool)
	for _, tag := range list {
		foundKeys[tag["key"]] = true
	}
	if !foundKeys["key1"] || !foundKeys["key2"] {
		t.Error("expected both keys in list")
	}
}

func TestTagsToList_Nil(t *testing.T) {
	list := tagsToList(nil)
	if list != nil {
		t.Errorf("expected nil, got %v", list)
	}
}
