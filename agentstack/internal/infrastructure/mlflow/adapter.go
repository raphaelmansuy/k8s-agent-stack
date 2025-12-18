package mlflow

import (
	"context"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
)

type MLflowAdapter struct {
	client *Client
}

func NewMLflowAdapter(client *Client) *MLflowAdapter {
	return &MLflowAdapter{
		client: client,
	}
}

func (a *MLflowAdapter) GetOrCreateExperiment(ctx context.Context, name string) (string, error) {
	return a.client.GetOrCreateExperiment(ctx, name)
}

func (a *MLflowAdapter) StartRun(ctx context.Context, experimentID, runName string, tags map[string]string) (evaluation.MLflowRun, error) {
	run, err := a.client.StartRun(ctx, experimentID, runName, tags)
	if err != nil {
		return nil, err
	}
	return &RunAdapter{run: run}, nil
}

type RunAdapter struct {
	run *Run
}

func (a *RunAdapter) LogParam(ctx context.Context, key, value string) error {
	return a.run.LogParam(ctx, key, value)
}

func (a *RunAdapter) LogMetric(ctx context.Context, key string, value float64, step int64) error {
	return a.run.LogMetric(ctx, key, value, step)
}

func (a *RunAdapter) LogBatch(ctx context.Context, params map[string]string, metrics map[string]float64) error {
	return a.run.LogBatch(ctx, params, metrics)
}

func (a *RunAdapter) End(ctx context.Context, status string) error {
	return a.run.End(ctx, RunStatus(status))
}

// Ensure MLflowAdapter implements evaluation.MLflowClient
var _ evaluation.MLflowClient = (*MLflowAdapter)(nil)
