// Package mlflow provides a Go client for the MLflow REST API.
// MLflow is used for experiment tracking, model evaluation, and metrics logging.
package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client wraps the MLflow REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new MLflow client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithHTTP creates a new MLflow client with a custom HTTP client.
func NewClientWithHTTP(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// CreateExperiment creates a new MLflow experiment.
func (c *Client) CreateExperiment(ctx context.Context, name string, tags map[string]string) (string, error) {
	req := map[string]interface{}{
		"name": name,
		"tags": tagsToList(tags),
	}

	resp, err := c.post(ctx, "/api/2.0/mlflow/experiments/create", req)
	if err != nil {
		return "", err
	}

	experimentID, ok := resp["experiment_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid experiment_id in response")
	}

	return experimentID, nil
}

// GetOrCreateExperiment gets an existing experiment by name or creates a new one.
func (c *Client) GetOrCreateExperiment(ctx context.Context, name string) (string, error) {
	// Try to get existing experiment
	resp, err := c.get(ctx, "/api/2.0/mlflow/experiments/get-by-name", map[string]string{
		"experiment_name": name,
	})
	if err == nil {
		exp, ok := resp["experiment"].(map[string]interface{})
		if ok {
			if expID, ok := exp["experiment_id"].(string); ok {
				return expID, nil
			}
		}
	}

	// Create new experiment
	return c.CreateExperiment(ctx, name, nil)
}

// GetExperiment retrieves an experiment by ID.
func (c *Client) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	resp, err := c.get(ctx, "/api/2.0/mlflow/experiments/get", map[string]string{
		"experiment_id": experimentID,
	})
	if err != nil {
		return nil, err
	}

	exp, ok := resp["experiment"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid experiment in response")
	}

	return parseExperiment(exp), nil
}

// StartRun creates a new run in an experiment.
func (c *Client) StartRun(ctx context.Context, experimentID, runName string, tags map[string]string) (*Run, error) {
	req := map[string]interface{}{
		"experiment_id": experimentID,
		"run_name":      runName,
		"start_time":    time.Now().UnixMilli(),
		"tags":          tagsToList(tags),
	}

	resp, err := c.post(ctx, "/api/2.0/mlflow/runs/create", req)
	if err != nil {
		return nil, err
	}

	runData, ok := resp["run"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid run in response")
	}

	runInfo, ok := runData["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid run info in response")
	}

	runID, ok := runInfo["run_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid run_id in response")
	}

	return &Run{
		RunID:        runID,
		ExperimentID: experimentID,
		client:       c,
	}, nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*RunInfo, error) {
	resp, err := c.get(ctx, "/api/2.0/mlflow/runs/get", map[string]string{
		"run_id": runID,
	})
	if err != nil {
		return nil, err
	}

	runData, ok := resp["run"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid run in response")
	}

	return parseRunInfo(runData), nil
}

// SearchRuns searches for runs matching a filter.
func (c *Client) SearchRuns(ctx context.Context, experimentIDs []string, filter string, maxResults int) ([]*RunInfo, error) {
	req := map[string]interface{}{
		"experiment_ids": experimentIDs,
		"filter_string":  filter,
		"max_results":    maxResults,
	}

	resp, err := c.post(ctx, "/api/2.0/mlflow/runs/search", req)
	if err != nil {
		return nil, err
	}

	runsData, ok := resp["runs"].([]interface{})
	if !ok {
		return []*RunInfo{}, nil
	}

	runs := make([]*RunInfo, 0, len(runsData))
	for _, r := range runsData {
		if runMap, ok := r.(map[string]interface{}); ok {
			runs = append(runs, parseRunInfo(runMap))
		}
	}

	return runs, nil
}

// Run represents an active MLflow run.
type Run struct {
	RunID        string
	ExperimentID string
	client       *Client
}

// LogParam logs a parameter to the run.
func (r *Run) LogParam(ctx context.Context, key, value string) error {
	req := map[string]interface{}{
		"run_id": r.RunID,
		"key":    key,
		"value":  value,
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/log-parameter", req)
	return err
}

// LogMetric logs a metric to the run.
func (r *Run) LogMetric(ctx context.Context, key string, value float64, step int64) error {
	req := map[string]interface{}{
		"run_id":    r.RunID,
		"key":       key,
		"value":     value,
		"timestamp": time.Now().UnixMilli(),
		"step":      step,
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/log-metric", req)
	return err
}

// LogBatch logs multiple parameters and metrics in a single call.
func (r *Run) LogBatch(ctx context.Context, params map[string]string, metrics map[string]float64) error {
	paramList := make([]map[string]string, 0, len(params))
	for k, v := range params {
		paramList = append(paramList, map[string]string{"key": k, "value": v})
	}

	metricList := make([]map[string]interface{}, 0, len(metrics))
	ts := time.Now().UnixMilli()
	for k, v := range metrics {
		metricList = append(metricList, map[string]interface{}{
			"key":       k,
			"value":     v,
			"timestamp": ts,
			"step":      0,
		})
	}

	req := map[string]interface{}{
		"run_id":  r.RunID,
		"params":  paramList,
		"metrics": metricList,
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/log-batch", req)
	return err
}

// SetTag sets a tag on the run.
func (r *Run) SetTag(ctx context.Context, key, value string) error {
	req := map[string]interface{}{
		"run_id": r.RunID,
		"key":    key,
		"value":  value,
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/set-tag", req)
	return err
}

// End ends the run with a status.
func (r *Run) End(ctx context.Context, status RunStatus) error {
	req := map[string]interface{}{
		"run_id":   r.RunID,
		"status":   string(status),
		"end_time": time.Now().UnixMilli(),
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/update", req)
	return err
}

// RunStatus represents the status of a run.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusScheduled RunStatus = "SCHEDULED"
	RunStatusFinished  RunStatus = "FINISHED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusKilled    RunStatus = "KILLED"
)

// Experiment represents an MLflow experiment.
type Experiment struct {
	ExperimentID     string
	Name             string
	ArtifactLocation string
	LifecycleStage   string
	Tags             map[string]string
}

// RunInfo represents information about a run.
type RunInfo struct {
	RunID        string
	RunName      string
	ExperimentID string
	Status       RunStatus
	StartTime    int64
	EndTime      int64
	Params       map[string]string
	Metrics      map[string]float64
	Tags         map[string]string
}

// HTTP helper methods

func (c *Client) get(ctx context.Context, path string, params map[string]string) (map[string]interface{}, error) {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Set(k, v)
		}
		reqURL += "?" + values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	return c.do(req)
}

func (c *Client) post(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req)
}

func (c *Client) do(req *http.Request) (map[string]interface{}, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("MLflow API error: %s - %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
	}

	return result, nil
}

// Helper functions

func tagsToList(tags map[string]string) []map[string]string {
	if tags == nil {
		return nil
	}
	list := make([]map[string]string, 0, len(tags))
	for k, v := range tags {
		list = append(list, map[string]string{"key": k, "value": v})
	}
	return list
}

func parseExperiment(data map[string]interface{}) *Experiment {
	exp := &Experiment{}

	if v, ok := data["experiment_id"].(string); ok {
		exp.ExperimentID = v
	}
	if v, ok := data["name"].(string); ok {
		exp.Name = v
	}
	if v, ok := data["artifact_location"].(string); ok {
		exp.ArtifactLocation = v
	}
	if v, ok := data["lifecycle_stage"].(string); ok {
		exp.LifecycleStage = v
	}

	exp.Tags = parseTags(data["tags"])
	return exp
}

func parseRunInfo(data map[string]interface{}) *RunInfo {
	run := &RunInfo{
		Params:  make(map[string]string),
		Metrics: make(map[string]float64),
		Tags:    make(map[string]string),
	}

	if info, ok := data["info"].(map[string]interface{}); ok {
		if v, ok := info["run_id"].(string); ok {
			run.RunID = v
		}
		if v, ok := info["run_name"].(string); ok {
			run.RunName = v
		}
		if v, ok := info["experiment_id"].(string); ok {
			run.ExperimentID = v
		}
		if v, ok := info["status"].(string); ok {
			run.Status = RunStatus(v)
		}
		if v, ok := info["start_time"].(float64); ok {
			run.StartTime = int64(v)
		}
		if v, ok := info["end_time"].(float64); ok {
			run.EndTime = int64(v)
		}
	}

	if dataSection, ok := data["data"].(map[string]interface{}); ok {
		// Parse params
		if params, ok := dataSection["params"].([]interface{}); ok {
			for _, p := range params {
				if param, ok := p.(map[string]interface{}); ok {
					key, _ := param["key"].(string)
					value, _ := param["value"].(string)
					if key != "" {
						run.Params[key] = value
					}
				}
			}
		}

		// Parse metrics
		if metrics, ok := dataSection["metrics"].([]interface{}); ok {
			for _, m := range metrics {
				if metric, ok := m.(map[string]interface{}); ok {
					key, _ := metric["key"].(string)
					value, _ := metric["value"].(float64)
					if key != "" {
						run.Metrics[key] = value
					}
				}
			}
		}

		// Parse tags
		run.Tags = parseTags(dataSection["tags"])
	}

	return run
}

func parseTags(data interface{}) map[string]string {
	tags := make(map[string]string)
	if tagList, ok := data.([]interface{}); ok {
		for _, t := range tagList {
			if tag, ok := t.(map[string]interface{}); ok {
				key, _ := tag["key"].(string)
				value, _ := tag["value"].(string)
				if key != "" {
					tags[key] = value
				}
			}
		}
	}
	return tags
}
