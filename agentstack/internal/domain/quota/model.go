// Package quota provides resource quota and rate limiting functionality.
package quota

import "time"

// QuotaType defines the type of resource being limited.
type QuotaType string

const (
	QuotaAgents          QuotaType = "agents"
	QuotaDeployments     QuotaType = "deployments"
	QuotaAPIRequests     QuotaType = "api_requests"
	QuotaChatMessages    QuotaType = "chat_messages"
	QuotaTokens          QuotaType = "tokens"
	QuotaStorage         QuotaType = "storage_bytes"
	QuotaConcurrentChats QuotaType = "concurrent_chats"
)

// Quota defines limits for a team or project.
type Quota struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	ProjectID string    `json:"project_id,omitempty"`
	Type      QuotaType `json:"type"`
	Limit     int64     `json:"limit"`
	Period    string    `json:"period,omitempty"` // "hour", "day", "month" for rate limits
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Usage tracks current consumption.
type Usage struct {
	QuotaID   string    `json:"quota_id"`
	Current   int64     `json:"current"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Plan defines a set of quotas for a subscription tier.
type Plan struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Description  string                    `json:"description"`
	Quotas       map[QuotaType]int64       `json:"quotas"`
	PeriodQuotas map[QuotaType]PeriodQuota `json:"period_quotas"`
	Price        float64                   `json:"price"`
	IsActive     bool                      `json:"is_active"`
}

// PeriodQuota defines a quota with a time period.
type PeriodQuota struct {
	Limit  int64  `json:"limit"`
	Period string `json:"period"`
}

// CheckQuotaRequest represents a request to check quota.
type CheckQuotaRequest struct {
	TeamID    string
	ProjectID string
	Type      QuotaType
	Amount    int64
}

// CheckQuotaResult contains the result of a quota check.
type CheckQuotaResult struct {
	Allowed   bool  `json:"allowed"`
	Current   int64 `json:"current"`
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
}

// IncrementUsageRequest represents a request to increment usage.
type IncrementUsageRequest struct {
	TeamID    string
	ProjectID string
	Type      QuotaType
	Amount    int64
}

// UsageSummary contains quota usage for a team.
type UsageSummary struct {
	TeamID string      `json:"team_id"`
	Items  []UsageItem `json:"items"`
}

// UsageItem represents a single quota usage item.
type UsageItem struct {
	Type       QuotaType `json:"type"`
	Current    int64     `json:"current"`
	Limit      int64     `json:"limit"`
	Period     string    `json:"period,omitempty"`
	Remaining  int64     `json:"remaining"`
	Percentage float64   `json:"percentage"`
}

// DefaultPlans are the pre-defined subscription plans.
var DefaultPlans = []Plan{
	{
		ID:          "plan_free",
		Name:        "Free",
		Description: "For personal projects and experimentation",
		Quotas: map[QuotaType]int64{
			QuotaAgents:          3,
			QuotaDeployments:     10,
			QuotaStorage:         100 * 1024 * 1024, // 100MB
			QuotaConcurrentChats: 5,
		},
		PeriodQuotas: map[QuotaType]PeriodQuota{
			QuotaAPIRequests:  {Limit: 1000, Period: "hour"},
			QuotaChatMessages: {Limit: 500, Period: "day"},
			QuotaTokens:       {Limit: 100000, Period: "month"},
		},
		Price:    0,
		IsActive: true,
	},
	{
		ID:          "plan_pro",
		Name:        "Pro",
		Description: "For professional teams",
		Quotas: map[QuotaType]int64{
			QuotaAgents:          25,
			QuotaDeployments:     100,
			QuotaStorage:         10 * 1024 * 1024 * 1024, // 10GB
			QuotaConcurrentChats: 50,
		},
		PeriodQuotas: map[QuotaType]PeriodQuota{
			QuotaAPIRequests:  {Limit: 50000, Period: "hour"},
			QuotaChatMessages: {Limit: 10000, Period: "day"},
			QuotaTokens:       {Limit: 5000000, Period: "month"},
		},
		Price:    99,
		IsActive: true,
	},
	{
		ID:          "plan_enterprise",
		Name:        "Enterprise",
		Description: "For large organizations",
		Quotas: map[QuotaType]int64{
			QuotaAgents:          -1, // Unlimited
			QuotaDeployments:     -1,
			QuotaStorage:         -1,
			QuotaConcurrentChats: -1,
		},
		PeriodQuotas: map[QuotaType]PeriodQuota{
			QuotaAPIRequests:  {Limit: -1, Period: "hour"},
			QuotaChatMessages: {Limit: -1, Period: "day"},
			QuotaTokens:       {Limit: -1, Period: "month"},
		},
		Price:    0, // Custom pricing
		IsActive: true,
	},
}

// GetPlan returns a plan by ID.
func GetPlan(id string) *Plan {
	for _, p := range DefaultPlans {
		if p.ID == id {
			plan := p
			return &plan
		}
	}
	return nil
}

// IsUnlimited returns true if the limit represents unlimited.
func IsUnlimited(limit int64) bool {
	return limit < 0
}
