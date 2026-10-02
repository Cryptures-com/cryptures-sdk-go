package cryptures

import (
	"context"
	"errors"
	"net/http"
)

// CardCardsService manages virtual cards.
//
// Operations that move money or change card state (Create, Fund, Withdraw,
// SetPIN, Block, Unblock, Terminate) are never retried automatically: an
// ambiguous failure may already have debited the balance or reached the
// issuer. Reconcile with Card.Balance.ListTransactions or Cards.Get before
// repeating them.
type CardCardsService struct {
	client *Client
}

// Card is a virtual card.
type Card struct {
	// CardID identifies the card on every other card operation.
	CardID string `json:"card_id"`
	// ProductCode is the product the card was issued under.
	ProductCode string `json:"product_code,omitempty"`
	// Brand is the card network, e.g. "visa" or "mastercard".
	Brand string `json:"brand,omitempty"`
	// Type is always "virtual".
	Type string `json:"type,omitempty"`
	// Currency is the ISO currency code, "USD" today.
	Currency string `json:"currency,omitempty"`
	// Status is e.g. "active", "blocked", "terminated".
	Status     string `json:"status"`
	NameOnCard string `json:"name_on_card,omitempty"`
	// Email is a system-generated placeholder, not the real cardholder email;
	// use List to read the real email on file.
	Email string `json:"email,omitempty"`
	// LastFour is the card's last four digits.
	LastFour string `json:"last_four,omitempty"`
	// ExpiryMonth is two digits, e.g. "08".
	ExpiryMonth string `json:"expiry_month,omitempty"`
	// ExpiryYear is four digits, e.g. "2031".
	ExpiryYear string `json:"expiry_year,omitempty"`
	// Balance is the card's own balance (distinct from the project balance).
	Balance *CardBalance `json:"balance,omitempty"`
	// CardNumber is the full PAN.
	CardNumber string `json:"card_number,omitempty"`
	// CVV is the card verification value.
	CVV string `json:"cvv,omitempty"`
	// CardNumberAlt carries the same PAN under the API's second key
	// "cardnumber" (no underscore).
	CardNumberAlt string `json:"cardnumber,omitempty"`
	// ExpireDate is the expiry pre-combined as "MM/YY".
	ExpireDate string `json:"expiredate,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	// Tags are the card's current tags. On Create they appear only when tags
	// were sent in the request.
	Tags []CardTag `json:"tags,omitempty"`
}

// CardBalance is a card's own balance.
type CardBalance struct {
	// Amount is in micro-units (divide by 1,000,000 for a display value).
	Amount int64 `json:"amount"`
	// DisplayAmount is the same balance as a plain decimal.
	DisplayAmount float64 `json:"display_amount"`
	Currency      string  `json:"currency"`
}

// CardResponse wraps a card in the API's {status, message, data} envelope.
type CardResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    Card   `json:"data"`
}

// CreateCardRequest creates a virtual card.
type CreateCardRequest struct {
	// ProductCode is one of the codes returned by ListProducts.
	ProductCode string `json:"product_code"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	// Email is the real cardholder email, retrievable later via List.
	Email string `json:"email"`
	// InitialLoad is the USD amount to load. Required for every product
	// except those whose CardProduct.DisallowInitialLoad is true (e.g.
	// us_493_visa_atm), which must leave it nil.
	InitialLoad *float64 `json:"initial_load,omitempty"`
	// Tags optionally attaches up to 10 tags by name (1-50 characters each),
	// matched case-insensitively and auto-created when new.
	Tags []string `json:"tags,omitempty"`
}

// Create creates a new virtual card. It atomically debits the project's
// shared USD balance (initial load plus issuance, top-up, and annual fees);
// a 402 insufficient_balance *APIError means nothing was charged and no card
// was created. Never retried automatically.
func (s *CardCardsService) Create(ctx context.Context, req *CreateCardRequest, opts ...RequestOption) (*CardResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: Cards.Create: nil request")
	}
	return doJSON[CardResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   "/api/v1/card/cards/create",
		body:   req,
	}, opts)
}

// Get fetches a card by id. A card not owned by the calling project returns
// a 403 forbidden_card *APIError; a 404 means the issuer no longer has a card
// Cryptures knows is yours.
func (s *CardCardsService) Get(ctx context.Context, cardID string, opts ...RequestOption) (*CardResponse, error) {
	return doJSON[CardResponse](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/card/cards/%s", cardID),
		route:     "/api/v1/card/cards/{card_id}",
		retryable: true,
	}, opts)
}

type setPINRequest struct {
	PIN string `json:"pin"`
}

// SetPIN sets a 6-digit PIN on a card. Only available for the
// us_493_visa_atm product; other products return a 400
// unsupported_for_product *APIError.
func (s *CardCardsService) SetPIN(ctx context.Context, cardID, pin string, opts ...RequestOption) (*StatusMessage, error) {
	return doJSON[StatusMessage](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   pathf("/api/v1/card/cards/%s/pin", cardID),
		route:  "/api/v1/card/cards/{card_id}/pin",
		body:   setPINRequest{PIN: pin},
	}, opts)
}

// CardAmountResponse is returned by Fund and Withdraw.
type CardAmountResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Data    CardAmountData `json:"data"`
}

// CardAmountData is the confirmed amount of a fund or withdrawal.
type CardAmountData struct {
	CardID string `json:"card_id"`
	// Amount is in micro-units (divide by 1,000,000 for USD).
	Amount int64 `json:"amount"`
	// DisplayAmount is the same amount as a plain USD decimal.
	DisplayAmount float64 `json:"display_amount"`
}

type cardAmountRequest struct {
	Amount float64 `json:"amount"`
}

// Fund adds amount USD to a card. The project's balance is debited amount
// plus a top-up fee first; a 402 insufficient_balance *APIError means
// nothing was charged. Never retried automatically.
func (s *CardCardsService) Fund(ctx context.Context, cardID string, amount float64, opts ...RequestOption) (*CardAmountResponse, error) {
	if amount <= 0 {
		return nil, errors.New("cryptures: Cards.Fund: amount must be positive")
	}
	return doJSON[CardAmountResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   pathf("/api/v1/card/cards/%s/fund", cardID),
		route:  "/api/v1/card/cards/{card_id}/fund",
		body:   cardAmountRequest{Amount: amount},
	}, opts)
}

// Withdraw withdraws amount USD from a card back to the project's balance.
// Never retried automatically.
func (s *CardCardsService) Withdraw(ctx context.Context, cardID string, amount float64, opts ...RequestOption) (*CardAmountResponse, error) {
	if amount <= 0 {
		return nil, errors.New("cryptures: Cards.Withdraw: amount must be positive")
	}
	return doJSON[CardAmountResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   pathf("/api/v1/card/cards/%s/withdraw", cardID),
		route:  "/api/v1/card/cards/{card_id}/withdraw",
		body:   cardAmountRequest{Amount: amount},
	}, opts)
}

// CardStatusResponse is returned by Terminate, Block, and Unblock.
type CardStatusResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    *CardStatusData `json:"data,omitempty"`
}

// CardStatusData is a card's new status.
type CardStatusData struct {
	CardID string `json:"card_id"`
	// Status is "terminated", "blocked", or "active".
	Status string `json:"status"`
}

// Terminate permanently terminates a card.
func (s *CardCardsService) Terminate(ctx context.Context, cardID string, opts ...RequestOption) (*CardStatusResponse, error) {
	return s.changeStatus(ctx, cardID, "terminate", opts)
}

// Block freezes a card.
func (s *CardCardsService) Block(ctx context.Context, cardID string, opts ...RequestOption) (*CardStatusResponse, error) {
	return s.changeStatus(ctx, cardID, "block", opts)
}

// Unblock unfreezes a previously blocked card.
func (s *CardCardsService) Unblock(ctx context.Context, cardID string, opts ...RequestOption) (*CardStatusResponse, error) {
	return s.changeStatus(ctx, cardID, "unblock", opts)
}

func (s *CardCardsService) changeStatus(ctx context.Context, cardID, action string, opts []RequestOption) (*CardStatusResponse, error) {
	return doJSON[CardStatusResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   pathf("/api/v1/card/cards/%s/", cardID) + action,
		route:  "/api/v1/card/cards/{card_id}/" + action,
	}, opts)
}

// CardTransactionsParams pages through ListTransactions.
type CardTransactionsParams struct {
	// PageNum is the page number, forwarded to the issuer verbatim. It is the
	// only pagination parameter; there is no page size.
	PageNum string
}

// CardTransactionsResponse is a page of a card's transaction history. The
// issuer's body is forwarded verbatim; the API reference describes this
// shape as its best current model.
type CardTransactionsResponse struct {
	Status  string               `json:"status"`
	Message string               `json:"message,omitempty"`
	Data    CardTransactionsData `json:"data"`
}

// CardTransactionsData holds the transactions and pagination state.
type CardTransactionsData struct {
	Transactions []CardTransaction    `json:"transactions"`
	Pagination   CardTransactionsPage `json:"pagination"`
}

// CardTransaction is one card transaction.
type CardTransaction struct {
	ID     string `json:"id"`
	CardID string `json:"card_id,omitempty"`
	// Type is e.g. "authorization".
	Type string `json:"type"`
	// Status is e.g. "completed".
	Status string `json:"status"`
	// Amount is in micro-units (divide by 1,000,000 for USD).
	Amount        int64   `json:"amount"`
	DisplayAmount float64 `json:"display_amount"`
	Currency      string  `json:"currency"`
	MerchantName  string  `json:"merchant_name,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// CardTransactionsPage is the pagination state of a transactions page.
type CardTransactionsPage struct {
	Type     string `json:"type,omitempty"`
	PageNum  int64  `json:"page_num"`
	PageSize int64  `json:"page_size"`
	Total    int64  `json:"total"`
	HasMore  bool   `json:"has_more"`
}

// ListTransactions returns a card's transaction history. params may be nil.
func (s *CardCardsService) ListTransactions(ctx context.Context, cardID string, params *CardTransactionsParams, opts ...RequestOption) (*CardTransactionsResponse, error) {
	q := newQuery()
	if params != nil {
		q.str("pageNum", params.PageNum)
	}
	return doJSON[CardTransactionsResponse](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/card/cards/%s/transactions", cardID),
		route:     "/api/v1/card/cards/{card_id}/transactions",
		query:     q.values(),
		retryable: true,
	}, opts)
}

// CardList is every card belonging to the calling project.
type CardList struct {
	Data []CardSummary `json:"data"`
}

// CardSummary is one card in a CardList.
type CardSummary struct {
	CardID      string `json:"card_id"`
	ProductCode string `json:"product_code"`
	// Email is the real cardholder email supplied at creation.
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Status    string `json:"status"`
	// LastFour may be nil; always nil-check it.
	LastFour *string `json:"last_four"`
	// Label is reserved for future use and always nil today.
	Label     *string   `json:"label"`
	CreatedAt string    `json:"created_at"`
	Tags      []CardTag `json:"tags"`
}

// List returns every card belonging to the calling project, newest first.
func (s *CardCardsService) List(ctx context.Context, opts ...RequestOption) (*CardList, error) {
	return doJSON[CardList](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/card/cards",
		retryable: true,
	}, opts)
}

// CardTags is a card's complete tag list.
type CardTags struct {
	Tags []CardTag `json:"tags"`
}

type setCardTagsRequest struct {
	Tags []string `json:"tags"`
}

// SetTags replaces a card's entire tag list with tags (at most 10 names,
// each 1-50 characters). This is a full replace, not a merge: names missing
// from tags are removed, and an empty or nil slice clears every tag. Names
// that collide case-insensitively are collapsed, so read the returned list
// rather than assuming it equals the input.
func (s *CardCardsService) SetTags(ctx context.Context, cardID string, tags []string, opts ...RequestOption) (*CardTags, error) {
	if tags == nil {
		tags = []string{}
	}
	return doJSON[CardTags](ctx, s.client, &requestSpec{
		method:    http.MethodPut,
		path:      pathf("/api/v1/card/cards/%s/tags", cardID),
		route:     "/api/v1/card/cards/{card_id}/tags",
		body:      setCardTagsRequest{Tags: tags},
		retryable: true, // full replace, idempotent
	}, opts)
}

// CardProducts is the catalog of card products available for issuance.
type CardProducts struct {
	Products []CardProduct `json:"products"`
}

// CardProduct is one product in the catalog.
type CardProduct struct {
	// ProductCode is the value to pass as CreateCardRequest.ProductCode.
	ProductCode string `json:"product_code"`
	DisplayName string `json:"display_name"`
	// MinLoadUSD is nil when the product has no minimum.
	MinLoadUSD *float64 `json:"min_load_usd"`
	// MaxLoadUSD is nil when the product has no maximum.
	MaxLoadUSD *float64 `json:"max_load_usd"`
	// IssuanceFeeUSD is charged once at card creation.
	IssuanceFeeUSD float64 `json:"issuance_fee_usd"`
	// FundFeeFlatUSD is added to every funding operation.
	FundFeeFlatUSD float64 `json:"fund_fee_flat_usd"`
	// FundFeePct is a fraction (0-1) applied to every funding amount.
	FundFeePct float64 `json:"fund_fee_pct"`
	// AnnualFeeUSD is nil when the product has none.
	AnnualFeeUSD *float64 `json:"annual_fee_usd"`
	// DisallowInitialLoad means Create must omit InitialLoad.
	DisallowInitialLoad bool `json:"disallow_initial_load"`
}

// ListProducts returns the active card product catalog. Fetch it rather
// than hard-coding product codes.
func (s *CardCardsService) ListProducts(ctx context.Context, opts ...RequestOption) (*CardProducts, error) {
	return doJSON[CardProducts](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/card/products",
		retryable: true,
	}, opts)
}
