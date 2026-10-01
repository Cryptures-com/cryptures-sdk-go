package cryptures

import (
	"context"
	"net/http"
	"time"
)

// CardReportsService produces card funding/withdrawal reports.
type CardReportsService struct {
	client *Client
}

// ReportSummaryParams selects the report range. Zero values use the API's
// defaults (the trailing 12 months).
type ReportSummaryParams struct {
	// From is the inclusive start of the range. Defaults to 12 months ago.
	From time.Time
	// To is the exclusive end of the range. Defaults to now. Must be later
	// than From.
	To time.Time
}

// ReportSummary summarizes the project's card funding and withdrawal
// activity over a date range. Only completed ledger entries are counted.
type ReportSummary struct {
	// Range is the range actually applied, re-serialized as canonical UTC
	// RFC 3339 strings; compare parsed instants, not strings.
	Range        ReportRange        `json:"range"`
	Totals       ReportTotals       `json:"totals"`
	Unattributed ReportUnattributed `json:"unattributed"`
	ByMonth      []ReportMonth      `json:"by_month"`
	// ByType is sorted by NetUSD, descending.
	ByType []ReportProduct `json:"by_type"`
	// ByTag is sorted by NetUSD, descending. A card counts toward every tag
	// it has, so tag totals can overlap.
	ByTag    []ReportTag    `json:"by_tag"`
	Untagged ReportUntagged `json:"untagged"`
	// ByCard is sorted by NetUSD, descending.
	ByCard []ReportCard `json:"by_card"`
}

// ReportRange is the applied report range.
type ReportRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ReportTotals are the overall report totals.
type ReportTotals struct {
	FundedUSD    float64 `json:"funded_usd"`
	WithdrawnUSD float64 `json:"withdrawn_usd"`
	NetUSD       float64 `json:"net_usd"`
	// CardCount counts distinct card references on the matching ledger rows,
	// which is not the same as real cards; use len(ByCard) for "how many real
	// cards had activity".
	CardCount int64 `json:"card_count"`
}

// ReportUnattributed is funding not attributable to a single card.
type ReportUnattributed struct {
	FundedUSD float64 `json:"funded_usd"`
}

// ReportMonth is one month of the report.
type ReportMonth struct {
	// Month is "YYYY-MM".
	Month        string  `json:"month"`
	FundedUSD    float64 `json:"funded_usd"`
	WithdrawnUSD float64 `json:"withdrawn_usd"`
	NetUSD       float64 `json:"net_usd"`
}

// ReportProduct is the report breakdown for one product.
type ReportProduct struct {
	ProductCode  string  `json:"product_code"`
	FundedUSD    float64 `json:"funded_usd"`
	WithdrawnUSD float64 `json:"withdrawn_usd"`
	NetUSD       float64 `json:"net_usd"`
	CardCount    int64   `json:"card_count"`
}

// ReportTag is the report breakdown for one tag.
type ReportTag struct {
	TagID        string  `json:"tag_id"`
	Name         string  `json:"name"`
	Color        string  `json:"color"`
	FundedUSD    float64 `json:"funded_usd"`
	WithdrawnUSD float64 `json:"withdrawn_usd"`
	NetUSD       float64 `json:"net_usd"`
	CardCount    int64   `json:"card_count"`
}

// ReportUntagged is the report breakdown for cards without tags.
type ReportUntagged struct {
	FundedUSD    float64 `json:"funded_usd"`
	WithdrawnUSD float64 `json:"withdrawn_usd"`
	NetUSD       float64 `json:"net_usd"`
	CardCount    int64   `json:"card_count"`
}

// ReportCard is the report breakdown for one card.
type ReportCard struct {
	CardID    string `json:"card_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// LastFour may be nil; always nil-check it.
	LastFour     *string   `json:"last_four"`
	ProductCode  string    `json:"product_code"`
	Tags         []CardTag `json:"tags"`
	FundedUSD    float64   `json:"funded_usd"`
	WithdrawnUSD float64   `json:"withdrawn_usd"`
	NetUSD       float64   `json:"net_usd"`
}

// Summary returns the card funding/withdrawal report for a date range.
// params may be nil.
func (s *CardReportsService) Summary(ctx context.Context, params *ReportSummaryParams, opts ...RequestOption) (*ReportSummary, error) {
	q := newQuery()
	if params != nil {
		if !params.From.IsZero() {
			q.str("from", params.From.Format(time.RFC3339Nano))
		}
		if !params.To.IsZero() {
			q.str("to", params.To.Format(time.RFC3339Nano))
		}
	}
	return doJSON[ReportSummary](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/card/reports/summary",
		query:     q.values(),
		retryable: true,
	}, opts)
}
