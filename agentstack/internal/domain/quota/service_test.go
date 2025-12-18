package quota

import (
	"context"
	"testing"
)

// mockRepository implements Repository for testing
type mockRepository struct {
	quotas map[string]*Quota
	usage  map[string]int64
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		quotas: make(map[string]*Quota),
		usage:  make(map[string]int64),
	}
}

func (m *mockRepository) GetQuota(ctx context.Context, teamID string, quotaType QuotaType) (*Quota, error) {
	key := teamID + ":" + string(quotaType)
	if q, ok := m.quotas[key]; ok {
		return q, nil
	}
	return nil, context.DeadlineExceeded
}

func (m *mockRepository) GetQuotas(ctx context.Context, teamID string) ([]Quota, error) {
	var quotas []Quota
	for _, q := range m.quotas {
		if q.TeamID == teamID {
			quotas = append(quotas, *q)
		}
	}
	return quotas, nil
}

func (m *mockRepository) SetQuota(ctx context.Context, quota *Quota) error {
	key := quota.TeamID + ":" + string(quota.Type)
	m.quotas[key] = quota
	return nil
}

func (m *mockRepository) GetUsage(ctx context.Context, quotaID string) (*Usage, error) {
	current := m.usage[quotaID]
	return &Usage{QuotaID: quotaID, Current: current}, nil
}

func (m *mockRepository) IncrementUsage(ctx context.Context, quotaID string, amount int64) error {
	m.usage[quotaID] += amount
	return nil
}

func TestCheckQuota(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	// Set up quota
	repo.quotas["team1:agents"] = &Quota{
		ID:     "q1",
		TeamID: "team1",
		Type:   QuotaAgents,
		Limit:  5,
	}
	repo.usage["q1"] = 3

	tests := []struct {
		name string
		req  CheckQuotaRequest
		want bool
	}{
		{
			name: "within quota",
			req: CheckQuotaRequest{
				TeamID: "team1",
				Type:   QuotaAgents,
				Amount: 1,
			},
			want: true,
		},
		{
			name: "at limit",
			req: CheckQuotaRequest{
				TeamID: "team1",
				Type:   QuotaAgents,
				Amount: 2,
			},
			want: true,
		},
		{
			name: "exceeds quota",
			req: CheckQuotaRequest{
				TeamID: "team1",
				Type:   QuotaAgents,
				Amount: 3,
			},
			want: false,
		},
		{
			name: "no quota defined (unlimited)",
			req: CheckQuotaRequest{
				TeamID: "team2",
				Type:   QuotaAgents,
				Amount: 100,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CheckQuota(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Allowed != tt.want {
				t.Errorf("got allowed=%v, want %v", result.Allowed, tt.want)
			}
		})
	}
}

func TestUnlimitedQuota(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	// Set up unlimited quota
	repo.quotas["team1:agents"] = &Quota{
		ID:     "q1",
		TeamID: "team1",
		Type:   QuotaAgents,
		Limit:  -1, // Unlimited
	}

	result, err := svc.CheckQuota(context.Background(), CheckQuotaRequest{
		TeamID: "team1",
		Type:   QuotaAgents,
		Amount: 1000000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("unlimited quota should allow any amount")
	}
}

func TestIncrementUsage(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	// Set up quota
	repo.quotas["team1:agents"] = &Quota{
		ID:     "q1",
		TeamID: "team1",
		Type:   QuotaAgents,
		Limit:  10,
	}

	err := svc.IncrementUsage(context.Background(), IncrementUsageRequest{
		TeamID: "team1",
		Type:   QuotaAgents,
		Amount: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.usage["q1"] != 3 {
		t.Errorf("got usage=%d, want 3", repo.usage["q1"])
	}
}

func TestGetUsageSummary(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	// Set up quotas
	repo.quotas["team1:agents"] = &Quota{
		ID:     "q1",
		TeamID: "team1",
		Type:   QuotaAgents,
		Limit:  10,
	}
	repo.quotas["team1:deployments"] = &Quota{
		ID:     "q2",
		TeamID: "team1",
		Type:   QuotaDeployments,
		Limit:  20,
	}
	repo.usage["q1"] = 5
	repo.usage["q2"] = 10

	summary, err := svc.GetUsageSummary(context.Background(), "team1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary.Items) != 2 {
		t.Errorf("got %d items, want 2", len(summary.Items))
	}
}

func TestApplyPlan(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	err := svc.ApplyPlan(context.Background(), "team1", "plan_free")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that quotas were created
	quota, err := repo.GetQuota(context.Background(), "team1", QuotaAgents)
	if err != nil {
		t.Fatalf("quota should exist: %v", err)
	}
	if quota.Limit != 3 {
		t.Errorf("got limit=%d, want 3", quota.Limit)
	}
}

func TestGetPlan(t *testing.T) {
	plan := GetPlan("plan_free")
	if plan == nil {
		t.Fatal("plan_free should exist")
	}
	if plan.Name != "Free" {
		t.Errorf("got name=%s, want Free", plan.Name)
	}

	plan = GetPlan("nonexistent")
	if plan != nil {
		t.Error("nonexistent plan should return nil")
	}
}

func TestIsUnlimited(t *testing.T) {
	if !IsUnlimited(-1) {
		t.Error("-1 should be unlimited")
	}
	if IsUnlimited(0) {
		t.Error("0 should not be unlimited")
	}
	if IsUnlimited(100) {
		t.Error("100 should not be unlimited")
	}
}
