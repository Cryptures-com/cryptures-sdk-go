package cryptures

import (
	"context"
	"errors"
	"net/http"
)

// ComplianceMonitoringService manages ongoing AML monitoring. Monitoring
// belongs to a person or company (named by entity kind and your external
// user id), not to a single session.
type ComplianceMonitoringService struct {
	client *Client
}

// Monitoring entity kinds.
const (
	EntityKindUser     = "user"
	EntityKindBusiness = "business"
)

// MonitoringRequest names the entity to monitor.
type MonitoringRequest struct {
	// EntityKind is EntityKindUser (a KYC or person screening) or
	// EntityKindBusiness (a KYB or company screening).
	EntityKind string `json:"entity_kind"`
	// ExternalUserID is your own identifier for the person or company, as
	// used when its session or screening was created.
	ExternalUserID string `json:"external_user_id"`
}

// MonitoringSubscription is an entity's monitoring subscription.
type MonitoringSubscription struct {
	EntityKind     string `json:"entity_kind"`
	ExternalUserID string `json:"external_user_id"`
	// Status is "active", "cancelled", or "suspended_insufficient_balance"
	// (a yearly renewal could not be paid; enable again after topping up).
	Status string `json:"status"`
	// EnabledAt is the start of the current paid year.
	EnabledAt string `json:"enabled_at"`
	// NextRenewalAt is when the next year is charged.
	NextRenewalAt string  `json:"next_renewal_at"`
	CancelledAt   *string `json:"cancelled_at"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// Enable places a person or company under ongoing AML monitoring, charging
// one year up front. Enabling an already-active entity is idempotent (no
// second charge). The entity needs at least one completed, billed screening
// first.
func (s *ComplianceMonitoringService) Enable(ctx context.Context, req *MonitoringRequest, opts ...RequestOption) (*MonitoringSubscription, error) {
	if req == nil {
		return nil, errors.New("cryptures: Monitoring.Enable: nil request")
	}
	return doJSON[MonitoringSubscription](ctx, s.client, &requestSpec{
		method:    http.MethodPost,
		path:      "/api/v1/compliance/monitoring",
		body:      req,
		retryable: true, // documented as idempotent for an active entity
	}, opts)
}

// Disable stops ongoing monitoring for an entity. The year already paid is
// not refunded. Repeating the call is safe.
func (s *ComplianceMonitoringService) Disable(ctx context.Context, entityKind, externalUserID string, opts ...RequestOption) error {
	if entityKind == "" || externalUserID == "" {
		return errors.New("cryptures: Monitoring.Disable: entityKind and externalUserID are required")
	}
	return doNoContent(ctx, s.client, &requestSpec{
		method:    http.MethodDelete,
		path:      "/api/v1/compliance/monitoring",
		query:     newQuery().str("entity_kind", entityKind).str("external_user_id", externalUserID).values(),
		retryable: true,
	}, opts)
}

// ListMonitoringParams filters and pages List.
type ListMonitoringParams struct {
	// Status is "active", "cancelled", or "suspended_insufficient_balance".
	Status string
	// Limit is the page size, 1-200. Defaults to 50.
	Limit *int64
	// Cursor is the NextCursor from the previous page.
	Cursor string
}

// MonitoringList is one page of monitoring subscriptions.
type MonitoringList struct {
	Subscriptions []MonitoringSubscription `json:"subscriptions"`
	// NextCursor is passed as Cursor to fetch the next page. It is empty on
	// the last page.
	NextCursor string `json:"next_cursor"`
}

func (p *ListMonitoringParams) query(cursor string) *queryBuilder {
	q := newQuery()
	if p != nil {
		q.str("status", p.Status).intPtr("limit", p.Limit)
	}
	return q.str("cursor", cursor)
}

// List returns one page of monitoring subscriptions, newest first, including
// cancelled and suspended ones. params may be nil. Use ListAutoPaging to walk
// every page.
func (s *ComplianceMonitoringService) List(ctx context.Context, params *ListMonitoringParams, opts ...RequestOption) (*MonitoringList, error) {
	cursor := ""
	if params != nil {
		cursor = params.Cursor
	}
	return doJSON[MonitoringList](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/compliance/monitoring",
		query:     params.query(cursor).values(),
		retryable: true,
	}, opts)
}

// ListAutoPaging returns an iterator over every monitoring subscription
// matching params, fetching pages as needed.
func (s *ComplianceMonitoringService) ListAutoPaging(ctx context.Context, params *ListMonitoringParams, opts ...RequestOption) *Iter[MonitoringSubscription] {
	start := ""
	if params != nil {
		start = params.Cursor
	}
	first := true
	return newIter(ctx, func(ctx context.Context, cursor string) ([]MonitoringSubscription, string, error) {
		if first {
			cursor, first = start, false
		}
		page, err := doJSON[MonitoringList](ctx, s.client, &requestSpec{
			method:    http.MethodGet,
			path:      "/api/v1/compliance/monitoring",
			query:     params.query(cursor).values(),
			retryable: true,
		}, opts)
		if err != nil {
			return nil, "", err
		}
		return page.Subscriptions, page.NextCursor, nil
	})
}
