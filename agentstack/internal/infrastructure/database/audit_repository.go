package database

import (
	"context"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

type AuditRepository struct {
	queries *db.Queries
}

func NewAuditRepository(queries *db.Queries) *AuditRepository {
	return &AuditRepository{queries: queries}
}

func (r *AuditRepository) Write(ctx context.Context, event *audit.Event) error {
	var ipAddr *netip.Addr
	if event.IPAddress != "" {
		if addr, err := netip.ParseAddr(event.IPAddress); err == nil {
			ipAddr = &addr
		}
	}

	_, err := r.queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		TeamID:       event.TeamID,
		UserID:       pgtype.Text{String: event.ActorID, Valid: event.ActorID != ""},
		Action:       event.Action,
		ResourceType: event.Resource,
		ResourceID:   event.ResourceID,
		Changes:      event.Details,
		IpAddress:    ipAddr,
		UserAgent:    pgtype.Text{String: event.UserAgent, Valid: event.UserAgent != ""},
	})
	return err
}

func (r *AuditRepository) Query(ctx context.Context, filter audit.Filter) ([]audit.Event, error) {
	logs, err := r.queries.ListAuditLogs(ctx, db.ListAuditLogsParams{
		TeamID: filter.TeamID,
		Limit:  int32(filter.Limit),
		Offset: int32(filter.Offset),
	})
	if err != nil {
		return nil, err
	}

	events := make([]audit.Event, len(logs))
	for i, log := range logs {
		var ipAddr string
		if log.IpAddress != nil {
			ipAddr = log.IpAddress.String()
		}

		events[i] = audit.Event{
			ID:         log.ID,
			TeamID:     log.TeamID,
			ActorID:    log.UserID.String,
			Action:     log.Action,
			Resource:   log.ResourceType,
			ResourceID: log.ResourceID,
			Details:    log.Changes,
			IPAddress:  ipAddr,
			UserAgent:  log.UserAgent.String,
			Timestamp:  log.CreatedAt,
		}
	}

	return events, nil
}
