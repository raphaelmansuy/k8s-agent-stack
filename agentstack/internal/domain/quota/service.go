// Package quota provides resource quota and rate limiting functionality.
package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for quota data access.
type Repository interface {
	GetQuota(ctx context.Context, teamID string, quotaType QuotaType) (*Quota, error)
	GetQuotas(ctx context.Context, teamID string) ([]Quota, error)
	SetQuota(ctx context.Context, quota *Quota) error
	GetUsage(ctx context.Context, quotaID string) (*Usage, error)
	IncrementUsage(ctx context.Context, quotaID string, amount int64) error
}

// Cache defines the interface for rate limit caching.
type Cache interface {
	Get(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// Service provides quota functionality.
type Service struct {
	repo  Repository
	cache Cache
}

// NewService creates a new quota service.
func NewService(repo Repository, cache Cache) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
	}
}

// CheckQuota verifies if an action would exceed quota.
func (s *Service) CheckQuota(ctx context.Context, req CheckQuotaRequest) (*CheckQuotaResult, error) {
	// Get quota definition
	quota, err := s.repo.GetQuota(ctx, req.TeamID, req.Type)
	if err != nil {
		// No quota defined = unlimited
		return &CheckQuotaResult{Allowed: true, Limit: -1}, nil
	}

	// Unlimited quota
	if IsUnlimited(quota.Limit) {
		return &CheckQuotaResult{Allowed: true, Limit: -1}, nil
	}

	// Get current usage
	var current int64
	if quota.Period != "" && s.cache != nil {
		// Rate-limited quota - use cache
		current, _ = s.getRateLimitUsage(ctx, quota)
	} else {
		// Absolute quota - use DB
		usage, err := s.repo.GetUsage(ctx, quota.ID)
		if err == nil {
			current = usage.Current
		}
	}

	allowed := current+req.Amount <= quota.Limit

	return &CheckQuotaResult{
		Allowed:   allowed,
		Current:   current,
		Limit:     quota.Limit,
		Remaining: quota.Limit - current,
	}, nil
}

// IncrementUsage increases quota usage.
func (s *Service) IncrementUsage(ctx context.Context, req IncrementUsageRequest) error {
	quota, err := s.repo.GetQuota(ctx, req.TeamID, req.Type)
	if err != nil {
		return nil // No quota = no tracking needed
	}

	if quota.Period != "" && s.cache != nil {
		// Rate-limited quota - use cache with TTL
		return s.incrementRateLimitUsage(ctx, quota, req.Amount)
	}

	// Absolute quota - use DB
	return s.repo.IncrementUsage(ctx, quota.ID, req.Amount)
}

// DecrementUsage decreases quota usage (e.g., when deleting a resource).
func (s *Service) DecrementUsage(ctx context.Context, req IncrementUsageRequest) error {
	return s.IncrementUsage(ctx, IncrementUsageRequest{
		TeamID:    req.TeamID,
		ProjectID: req.ProjectID,
		Type:      req.Type,
		Amount:    -req.Amount,
	})
}

func (s *Service) getRateLimitUsage(ctx context.Context, quota *Quota) (int64, error) {
	key := s.rateLimitKey(quota)
	return s.cache.Get(ctx, key)
}

func (s *Service) incrementRateLimitUsage(ctx context.Context, quota *Quota, amount int64) error {
	key := s.rateLimitKey(quota)

	// Increment
	_, err := s.cache.IncrBy(ctx, key, amount)
	if err != nil {
		return err
	}

	// Set TTL if not set
	ttl, _ := s.cache.TTL(ctx, key)
	if ttl < 0 {
		_ = s.cache.Expire(ctx, key, s.periodToDuration(quota.Period))
	}

	return nil
}

func (s *Service) rateLimitKey(quota *Quota) string {
	// Include period start time in key for automatic reset
	periodStart := s.periodStart(quota.Period)
	return fmt.Sprintf("quota:%s:%s:%d", quota.ID, quota.Type, periodStart.Unix())
}

func (s *Service) periodStart(period string) time.Time {
	now := time.Now().UTC()
	switch period {
	case "hour":
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	case "day":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case "month":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return now
	}
}

func (s *Service) periodToDuration(period string) time.Duration {
	switch period {
	case "hour":
		return time.Hour
	case "day":
		return 24 * time.Hour
	case "month":
		return 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

// GetUsageSummary returns quota usage for a team.
func (s *Service) GetUsageSummary(ctx context.Context, teamID string) (*UsageSummary, error) {
	quotas, err := s.repo.GetQuotas(ctx, teamID)
	if err != nil {
		return nil, err
	}

	items := make([]UsageItem, 0, len(quotas))
	for _, quota := range quotas {
		var current int64
		if quota.Period != "" && s.cache != nil {
			current, _ = s.getRateLimitUsage(ctx, &quota)
		} else {
			if usage, err := s.repo.GetUsage(ctx, quota.ID); err == nil {
				current = usage.Current
			}
		}

		item := UsageItem{
			Type:      quota.Type,
			Current:   current,
			Limit:     quota.Limit,
			Period:    quota.Period,
			Remaining: quota.Limit - current,
		}

		if quota.Limit > 0 {
			item.Percentage = float64(current) / float64(quota.Limit) * 100
		}

		items = append(items, item)
	}

	return &UsageSummary{
		TeamID: teamID,
		Items:  items,
	}, nil
}

// SetQuota creates or updates a quota.
func (s *Service) SetQuota(ctx context.Context, quota *Quota) error {
	if quota.ID == "" {
		quota.ID = "quota_" + uuid.New().String()[:8]
	}
	quota.UpdatedAt = time.Now()
	if quota.CreatedAt.IsZero() {
		quota.CreatedAt = time.Now()
	}
	return s.repo.SetQuota(ctx, quota)
}

// ApplyPlan applies a plan's quotas to a team.
func (s *Service) ApplyPlan(ctx context.Context, teamID, planID string) error {
	plan := GetPlan(planID)
	if plan == nil {
		return fmt.Errorf("plan not found: %s", planID)
	}

	// Apply absolute quotas
	for quotaType, limit := range plan.Quotas {
		quota := &Quota{
			TeamID: teamID,
			Type:   quotaType,
			Limit:  limit,
		}
		if err := s.SetQuota(ctx, quota); err != nil {
			return fmt.Errorf("failed to set quota %s: %w", quotaType, err)
		}
	}

	// Apply period quotas
	for quotaType, pq := range plan.PeriodQuotas {
		quota := &Quota{
			TeamID: teamID,
			Type:   quotaType,
			Limit:  pq.Limit,
			Period: pq.Period,
		}
		if err := s.SetQuota(ctx, quota); err != nil {
			return fmt.Errorf("failed to set quota %s: %w", quotaType, err)
		}
	}

	return nil
}
