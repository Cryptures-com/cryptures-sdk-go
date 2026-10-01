package cryptures

import (
	"context"
	"errors"
	"net/http"
)

// ComplianceScreeningService runs standalone AML screenings and blockchain
// wallet screenings. Both are billed per successful call.
//
// Set IdempotencyKey on the request to make retries safe: a repeated call
// with the same key and request returns the original result (HTTP 200,
// Idempotent-Replay: true; see WithResponseMetadata) without screening or
// charging again. With a key set, the SDK also retries network errors and
// 5xx responses automatically; without one, every call is a separate
// screening and a separate charge, so it is never retried.
type ComplianceScreeningService struct {
	client *Client
}

// AMLCheckRequest identifies a person or company to screen.
type AMLCheckRequest struct {
	// ExternalUserID is your own identifier for the subject: at most 200
	// characters, no colon, no control characters.
	ExternalUserID string `json:"external_user_id"`
	// FullName is the person or company name, at most 200 characters.
	FullName string `json:"full_name"`
	// DateOfBirth is YYYY-MM-DD. Optional; improves matching for people.
	DateOfBirth string `json:"date_of_birth,omitempty"`
	// Nationality is an ISO 3166-1 alpha-2 country code, e.g. "ES".
	Nationality string `json:"nationality,omitempty"`
	// EntityType is "person" (default) or "company".
	EntityType string `json:"entity_type,omitempty"`

	// IdempotencyKey is sent as the Idempotency-Key header (1-128 printable
	// characters). It is not part of the JSON body.
	IdempotencyKey string `json:"-"`
}

// AMLCheck is the result of a standalone AML screening.
type AMLCheck struct {
	// CheckID is the related_reference of the screening's ledger row.
	CheckID string `json:"check_id"`
	// SessionID is the stored aml_standalone session, readable with
	// Sessions.Get. Nil only if none was returned.
	SessionID      *string `json:"session_id"`
	ExternalUserID string  `json:"external_user_id"`
	// Status is the outcome, e.g. "Approved", "In Review", "Declined".
	Status    string    `json:"status"`
	Result    AMLResult `json:"result"`
	CreatedAt string    `json:"created_at"`
}

// AMLResult is the normalized result of an AML screening.
type AMLResult struct {
	Features []string              `json:"features"`
	Results  []VerificationResult  `json:"results"`
	Warnings []VerificationWarning `json:"warnings"`
}

// AMLCheck screens a person or company against sanctions, PEP, and
// adverse-media lists and returns the result synchronously. A 402
// insufficient_balance *APIError means nothing was screened or charged.
func (s *ComplianceScreeningService) AMLCheck(ctx context.Context, req *AMLCheckRequest, opts ...RequestOption) (*AMLCheck, error) {
	if req == nil {
		return nil, errors.New("cryptures: Screening.AMLCheck: nil request")
	}
	return doJSON[AMLCheck](ctx, s.client, &requestSpec{
		method:         http.MethodPost,
		path:           "/api/v1/compliance/aml/checks",
		body:           req,
		idempotencyKey: req.IdempotencyKey,
	}, opts)
}

// WalletScreeningRequest identifies a blockchain address to screen.
type WalletScreeningRequest struct {
	// Address is 1-128 characters: letters, digits, and ". _ : -".
	Address string `json:"address"`
	// Chain is one of BTC, ETH, BNB, MATIC, TRON (TRX is an alias), LTC,
	// DOGE, SOL, XRP, BCH. Case-insensitive.
	Chain string `json:"chain"`

	// IdempotencyKey is sent as the Idempotency-Key header (1-128 printable
	// characters). It is not part of the JSON body.
	IdempotencyKey string `json:"-"`
}

// WalletScreening is the result of a wallet screening.
type WalletScreening struct {
	CheckID string `json:"check_id"`
	// Address is as you sent it.
	Address string `json:"address"`
	// Chain is the upper-case chain code (TRX is reported as TRON).
	Chain      string                `json:"chain"`
	Result     WalletScreeningResult `json:"result"`
	ScreenedAt string                `json:"screened_at"`
}

// WalletScreeningResult is a wallet's risk assessment.
type WalletScreeningResult struct {
	// Severity is one of "UNKNOWN", "LOW", "MEDIUM", "HIGH", "CRITICAL".
	Severity string `json:"severity"`
	// RiskScore is 0-100, nil when not available.
	RiskScore    *int64 `json:"risk_score,omitempty"`
	SanctionsHit bool   `json:"sanctions_hit"`
	// PEPCounterparty is nil when not available.
	PEPCounterparty *bool `json:"pep_counterparty,omitempty"`
	// DominantRiskCategory is empty when not available.
	DominantRiskCategory string `json:"dominant_risk_category,omitempty"`
	// RiskFactors lists up to 25 exposure factors.
	RiskFactors []RiskFactor `json:"risk_factors"`
}

// RiskFactor is one exposure factor. Each field is unset when not available.
type RiskFactor struct {
	// Category is a token such as "mixer", "sanctions", "darknet_market",
	// "ransomware", "scam", "exchange", or "other".
	Category string `json:"category,omitempty"`
	// Direction is "incoming" or "outgoing".
	Direction string `json:"direction,omitempty"`
	// ExposureType is "direct" or "indirect".
	ExposureType string `json:"exposure_type,omitempty"`
	// Percentage is the share of exposure, 0-100.
	Percentage *float64 `json:"percentage,omitempty"`
	IsHighRisk *bool    `json:"is_high_risk,omitempty"`
}

// CreateWalletScreening screens a public blockchain address for risk
// exposure (sanctions, mixers, illicit actors, and similar) and returns the
// result synchronously.
func (s *ComplianceScreeningService) CreateWalletScreening(ctx context.Context, req *WalletScreeningRequest, opts ...RequestOption) (*WalletScreening, error) {
	if req == nil {
		return nil, errors.New("cryptures: Screening.CreateWalletScreening: nil request")
	}
	return doJSON[WalletScreening](ctx, s.client, &requestSpec{
		method:         http.MethodPost,
		path:           "/api/v1/compliance/wallet-screenings",
		body:           req,
		idempotencyKey: req.IdempotencyKey,
	}, opts)
}

// GetWalletScreening returns a stored wallet screening by its check id,
// exactly as it was returned when it ran. Reading is free.
func (s *ComplianceScreeningService) GetWalletScreening(ctx context.Context, checkID string, opts ...RequestOption) (*WalletScreening, error) {
	return doJSON[WalletScreening](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/compliance/wallet-screenings/%s", checkID),
		retryable: true,
	}, opts)
}
