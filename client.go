package cryptures

import (
	"net/http"
	"strings"
	"time"
)

const (
	// Version is the version of this SDK. It is sent in the User-Agent header.
	Version = "0.1.1"

	// DefaultBaseURL is the production Cryptures API endpoint.
	DefaultBaseURL = "https://api.cryptures.com"

	// DefaultTimeout is the per-attempt HTTP timeout used when no custom
	// timeout or *http.Client is supplied.
	DefaultTimeout = 30 * time.Second

	// DefaultMaxRetries is the number of times a retry-safe request is retried
	// after a network error or a 5xx response.
	DefaultMaxRetries = 3

	defaultRetryBaseDelay = 250 * time.Millisecond
	maxRetryDelay         = 30 * time.Second
	userAgent             = "cryptures-sdk-go/" + Version
)

// Client is the entry point to the Cryptures API. Create one with NewClient
// and reuse it: a Client is safe for concurrent use by multiple goroutines.
//
// Operations are grouped by API domain, mirroring the structure of the API
// reference at https://docs.cryptures.com/:
//
//	client.Blockchain.Data       // balances, history, market data, rates
//	client.Blockchain.Operations // send, broadcast, JSON-RPC passthrough
//	client.Card.Cards            // virtual card lifecycle
//	client.Compliance.Sessions   // KYC / KYB verification sessions
type Client struct {
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	timeout        *time.Duration
	maxRetries     int
	retryBaseDelay time.Duration

	// Blockchain groups the blockchain-domain operations.
	Blockchain *BlockchainService
	// Card groups the card-domain operations.
	Card *CardService
	// Compliance groups the compliance-domain operations.
	Compliance *ComplianceService
}

// ClientOption configures a Client. Pass options to NewClient.
type ClientOption func(*Client)

// WithBaseURL overrides the API base URL (default https://api.cryptures.com).
// A trailing slash is ignored.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithTimeout sets the per-attempt HTTP timeout (default 30s). It applies to
// the default HTTP client, or to a copy of the client given to
// WithHTTPClient, regardless of option order; the caller's *http.Client is
// never mutated. Use the context passed to each method to bound the total
// time across retries.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.timeout = &timeout }
}

// WithMaxRetries sets how many times a retry-safe request is retried after a
// network error or 5xx response (default 3). Zero disables retries; negative
// values are treated as zero.
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		if maxRetries < 0 {
			maxRetries = 0
		}
		c.maxRetries = maxRetries
	}
}

// WithHTTPClient supplies the *http.Client used for every request, for
// example to configure a proxy, custom transport, or instrumentation. A nil
// client is ignored.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// withRetryBaseDelay is used by tests to keep retry backoff fast.
func withRetryBaseDelay(d time.Duration) ClientOption {
	return func(c *Client) { c.retryBaseDelay = d }
}

// NewClient returns a Client authenticated with the given project API key,
// which is sent as the x-api-key header on every request.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:         apiKey,
		baseURL:        DefaultBaseURL,
		httpClient:     &http.Client{Timeout: DefaultTimeout},
		maxRetries:     DefaultMaxRetries,
		retryBaseDelay: defaultRetryBaseDelay,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	if c.timeout != nil {
		hc := *c.httpClient
		hc.Timeout = *c.timeout
		c.httpClient = &hc
	}

	c.Blockchain = &BlockchainService{
		Data:       &BlockchainDataService{client: c},
		Operations: &BlockchainOperationsService{client: c},
		Wallet:     &BlockchainWalletService{client: c},
		Contracts:  &BlockchainContractsService{client: c},
		Fee:        &BlockchainFeeService{client: c},
		Lookups:    &BlockchainLookupsService{client: c},
		NFT:        &BlockchainNFTService{client: c},
		Storage:    &BlockchainStorageService{client: c},
	}
	c.Card = &CardService{
		Cards:    &CardCardsService{client: c},
		Balance:  &CardBalanceService{client: c},
		Tags:     &CardTagsService{client: c},
		Reports:  &CardReportsService{client: c},
		Webhooks: &CardWebhooksService{client: c},
	}
	c.Compliance = &ComplianceService{
		Sessions:   &ComplianceSessionsService{client: c},
		Screening:  &ComplianceScreeningService{client: c},
		Monitoring: &ComplianceMonitoringService{client: c},
		Webhooks:   &ComplianceWebhooksService{client: c},
	}
	return c
}

// BaseURL returns the base URL this client sends requests to.
func (c *Client) BaseURL() string { return c.baseURL }
