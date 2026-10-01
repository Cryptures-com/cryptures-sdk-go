package cryptures

import (
	"context"
	"net/http"
)

// CardBalanceService reads the project's shared USD balance: the single
// balance that card creation and funding debit and withdrawals credit, and
// the whole-account ledger behind it.
type CardBalanceService struct {
	client *Client
}

// ProjectBalance is the calling project's shared USD balance.
type ProjectBalance struct {
	// BalanceUSD is 0 for a project with no balance activity yet.
	BalanceUSD float64 `json:"balance_usd"`
	// UpdatedAt is the RFC 3339 time of the last balance change, or nil if never updated.
	UpdatedAt *string `json:"updated_at"`
}

// Get returns the calling project's shared USD balance.
func (s *CardBalanceService) Get(ctx context.Context, opts ...RequestOption) (*ProjectBalance, error) {
	return doJSON[ProjectBalance](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/card/balance",
		retryable: true,
	}, opts)
}

// BalanceTransactionsParams pages through ListTransactions.
type BalanceTransactionsParams struct {
	// Page is 1-indexed. Defaults to 1.
	Page *int64
	// Limit is the number of rows per page. Defaults to 50; values above 200
	// are clamped to 200.
	Limit *int64
}

// BalanceTransactions is a page of the project's balance ledger.
type BalanceTransactions struct {
	Data  []LedgerEntry `json:"data"`
	Page  int64         `json:"page"`
	Limit int64         `json:"limit"`
	// Total is the row count across all pages.
	Total int64 `json:"total"`
}

// LedgerEntry is one balance-ledger row.
type LedgerEntry struct {
	ID string `json:"id"`
	// Type says what the row is for, e.g. "card_create_debit",
	// "card_fund_debit", "card_withdraw_credit", "deposit_credit",
	// "compliance_aml_check_debit". New types can be added over time; treat
	// an unrecognized value as some other billable activity.
	Type string `json:"type"`
	// AmountUSD is the USD amount debited or credited.
	AmountUSD float64 `json:"amount_usd"`
	// RelatedReference is the id of whatever the row is about; what it
	// points at depends on Type. It can be nil.
	RelatedReference *string `json:"related_reference"`
	// Status is one of "pending", "completed", "reversed", "disputed".
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// ListTransactions returns the project's whole-account balance ledger,
// newest first. params may be nil.
func (s *CardBalanceService) ListTransactions(ctx context.Context, params *BalanceTransactionsParams, opts ...RequestOption) (*BalanceTransactions, error) {
	q := newQuery()
	if params != nil {
		q.intPtr("page", params.Page).intPtr("limit", params.Limit)
	}
	return doJSON[BalanceTransactions](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/card/balance/transactions",
		query:     q.values(),
		retryable: true,
	}, opts)
}
