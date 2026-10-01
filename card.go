package cryptures

// CardService groups the card-domain operations: virtual card lifecycle,
// the project's shared USD balance, card tags, reports, and webhooks.
//
// Once a card request clears Cryptures' own validation and ownership checks,
// the card issuer's answer is forwarded verbatim, including errors. Such an
// error is still returned as an *APIError, but its body is the issuer's own
// {"status":"failure","message","code"} shape; use APIError.IsProviderError
// to tell the two apart.
type CardService struct {
	// Cards covers card creation, funding, withdrawal, PINs, blocking,
	// termination, transactions, per-card tags, and the product catalog.
	Cards *CardCardsService
	// Balance covers the project's shared USD balance and its ledger.
	Balance *CardBalanceService
	// Tags covers the project's card tags.
	Tags *CardTagsService
	// Reports covers card funding/withdrawal reports.
	Reports *CardReportsService
	// Webhooks covers card-domain webhook registration.
	Webhooks *CardWebhooksService
}

// StatusMessage is the minimal {status, message} envelope some card
// operations return.
type StatusMessage struct {
	// Status is "success" on success.
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// CardTag is a tag attached to a card.
type CardTag struct {
	TagID string `json:"tag_id"`
	Name  string `json:"name"`
	// Color is a hex color, e.g. "#4F46E5".
	Color string `json:"color"`
}
