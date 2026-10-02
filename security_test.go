package cryptures

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// A real-looking BIP-39 mnemonic. None of its words appear in any API path
// template, so finding one in an error string means the secret leaked.
const leakMnemonic = "abandon ability able about above absent absorb abstract absurd abuse access accident"

// leakAPIKey is a realistic-looking API key that must never appear in an error.
const leakAPIKey = "cr_live_5f2b9c1d7e8a4b6c9d0e1f2a3b4c5d6e"

// unreachableClient returns a client whose base URL points at a port nothing
// is listening on, so every request fails with a connection error.
func unreachableClient(t *testing.T, opts ...ClientOption) *Client {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL
	srv.Close()
	all := append([]ClientOption{WithBaseURL(base), withRetryBaseDelay(time.Millisecond), WithMaxRetries(0)}, opts...)
	return NewClient(leakAPIKey, all...)
}

func assertNoSecret(t *testing.T, err error, secrets ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	msg := err.Error()
	for _, s := range secrets {
		if strings.Contains(msg, s) {
			t.Errorf("error string leaks %q:\n%s", s, msg)
		}
	}
	for _, word := range strings.Fields(leakMnemonic) {
		if strings.Contains(msg, word) {
			t.Errorf("error string leaks mnemonic word %q:\n%s", word, msg)
		}
	}
	if strings.Contains(msg, "?") {
		t.Errorf("error string contains a query string:\n%s", msg)
	}
}

func TestTransportErrorDoesNotLeakMnemonic(t *testing.T) {
	t.Run("Wallet.Generate with a mnemonic (query string)", func(t *testing.T) {
		c := unreachableClient(t)
		_, err := c.Blockchain.Wallet.Generate(ctxBG(), "ETH", &GenerateWalletParams{Mnemonic: leakMnemonic})
		assertNoSecret(t, err, "mnemonic=", url.QueryEscape(leakMnemonic), leakAPIKey)
		if !strings.HasPrefix(err.Error(), "cryptures: GET /api/v1/blockchain/wallet/{chain}: ") {
			t.Errorf("unexpected error string: %s", err)
		}

		// The error still unwraps to a *url.Error (now redacted) and a net.Error.
		var urlErr *url.Error
		if !errors.As(err, &urlErr) {
			t.Fatalf("expected the error to unwrap to *url.Error, got %T", err)
		}
		eq(t, "url.Error.URL", urlErr.URL, "/api/v1/blockchain/wallet/{chain}")
		assertNoSecret(t, urlErr)
		var netErr net.Error
		if !errors.As(err, &netErr) {
			t.Fatalf("expected the error to unwrap to net.Error, got %T", err)
		}
	})

	t.Run("Wallet.DeriveAddress with an EGLD mnemonic (path segment)", func(t *testing.T) {
		c := unreachableClient(t)
		_, err := c.Blockchain.Wallet.DeriveAddress(ctxBG(), "EGLD", leakMnemonic, 0)
		assertNoSecret(t, err, url.PathEscape(leakMnemonic), leakAPIKey)
		if !strings.HasPrefix(err.Error(), "cryptures: GET /api/v1/blockchain/wallet/{chain}/address/{xpub}/{index}: ") {
			t.Errorf("unexpected error string: %s", err)
		}
	})

	t.Run("retried requests are redacted too", func(t *testing.T) {
		c := unreachableClient(t, WithMaxRetries(2))
		_, err := c.Blockchain.Wallet.DeriveAddress(ctxBG(), "EGLD", leakMnemonic, 3)
		assertNoSecret(t, err, url.PathEscape(leakMnemonic), leakAPIKey)
	})

	t.Run("context deadline", func(t *testing.T) {
		// A server that never answers, so the request ends on the context.
		block := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
		defer srv.Close()
		defer close(block)
		c := NewClient(leakAPIKey, WithBaseURL(srv.URL), WithMaxRetries(0))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := c.Blockchain.Wallet.Generate(ctx, "ETH", &GenerateWalletParams{Mnemonic: leakMnemonic})
		assertNoSecret(t, err, "mnemonic=", leakAPIKey)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected errors.Is(err, context.DeadlineExceeded), got %v", err)
		}
	})

	t.Run("client timeout", func(t *testing.T) {
		block := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
		defer srv.Close()
		defer close(block)
		c := NewClient(leakAPIKey, WithBaseURL(srv.URL), WithMaxRetries(0), WithTimeout(50*time.Millisecond))
		_, err := c.Blockchain.Wallet.DeriveAddress(ctxBG(), "EGLD", leakMnemonic, 0)
		assertNoSecret(t, err, url.PathEscape(leakMnemonic), leakAPIKey)
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Errorf("expected a timeout net.Error, got %v", err)
		}
	})
}

// TestTransportErrorsNameOnlyTheRouteTemplate runs every endpoint test case
// against an unreachable server and checks that each error names the
// documented path template, never the actual path or query string.
func TestTransportErrorsNameOnlyTheRouteTemplate(t *testing.T) {
	ops := loadOperations(t)
	templates := map[string]bool{}
	for _, op := range ops {
		templates[op.method+" "+op.template] = true
	}
	c := unreachableClient(t)
	for _, table := range allEndpointTables() {
		for _, tc := range table {
			_, err := tc.call(ctxBG(), c)
			if err == nil {
				t.Errorf("%s: expected a connection error", tc.name)
				continue
			}
			msg := err.Error()
			rest, ok := strings.CutPrefix(msg, "cryptures: ")
			op, _, found := strings.Cut(rest, ": ")
			if !ok || !found || !templates[op] {
				t.Errorf("%s: error does not start with a documented METHOD /template: %s", tc.name, msg)
			}
			if strings.Contains(msg, "?") {
				t.Errorf("%s: error contains a query string: %s", tc.name, msg)
			}
			if !strings.Contains(op, "{") && op != tc.method+" "+tc.path {
				t.Errorf("%s: route %q does not match the request path %q", tc.name, op, tc.path)
			}
		}
	}
}

func allEndpointTables() [][]endpointCase {
	return [][]endpointCase{
		blockchainDataCases, blockchainOperationsCases, blockchainWalletCases, blockchainContractsCases,
		blockchainFeeCases, blockchainLookupsCases, blockchainNFTCases, cardCases, complianceCases,
	}
}

// retryMatrix is whether each operation is retried automatically after a 5xx
// or a network error (without an Idempotency-Key). It is identical in the Go,
// JS, and Python SDKs.
var retryMatrix = map[string]bool{
	// Never retried: repeating them could duplicate a broadcast, a charge, a
	// money movement, or another irreversible side effect.
	"tx.send":                            false,
	"tx.broadcast":                       false,
	"rpc.gateway":                        false, // may carry eth_sendRawTransaction
	"contract.token.deploy":              false,
	"contract.token.mint":                false,
	"contract.token.burn":                false,
	"storage.ipfs.upload":                false, // billed
	"wallet.generate":                    false, // a retry returns a different wallet
	"card.create":                        false,
	"card.fund":                          false,
	"card.withdraw":                      false,
	"card.setpin":                        false,
	"card.block":                         false,
	"card.unblock":                       false,
	"card.terminate":                     false,
	"card.tags.create":                   false,
	"compliance.session.create":          false,
	"compliance.aml.check":               false, // retried only with an Idempotency-Key
	"compliance.wallet_screening.create": false, // retried only with an Idempotency-Key

	// Retried: reads, and writes that are idempotent.
	"address.derive":                     true,
	"balance-history.get":                true,
	"balance.batch":                      true,
	"balance.check":                      true,
	"block.get":                          true,
	"block.latest":                       true,
	"card.balance.get":                   true,
	"card.balance.transactions":          true,
	"card.get":                           true,
	"card.list":                          true,
	"card.products.list":                 true,
	"card.reports.summary":               true,
	"card.tags.delete":                   true,
	"card.tags.list":                     true,
	"card.tags.set":                      true,
	"card.tags.update":                   true,
	"card.transactions":                  true,
	"card.webhooks.register":             true,
	"card.webhooks.status":               true,
	"compliance.monitoring.disable":      true,
	"compliance.monitoring.enable":       true,
	"compliance.monitoring.list":         true,
	"compliance.presets.list":            true,
	"compliance.session.delete":          true,
	"compliance.session.documents.get":   true,
	"compliance.session.get":             true,
	"compliance.session.report.create":   true,
	"compliance.session.report.download": true,
	"compliance.session.status.update":   true,
	"compliance.sessions.list":           true,
	"compliance.wallet_screening.get":    true,
	"compliance.webhooks.register":       true,
	"compliance.webhooks.status":         true,
	"exchange.rate":                      true,
	"exchange.rate.batch":                true,
	"exchange.rate.contract":             true,
	"fee.gas":                            true,
	"fee.get":                            true,
	"market.assets":                      true,
	"market.coin.info":                   true,
	"market.coin.markets":                true,
	"market.coin.ohlcv":                  true,
	"market.coin.social":                 true,
	"market.exchanges":                   true,
	"market.exchanges.single":            true,
	"market.global":                      true,
	"market.movers":                      true,
	"market.tickers":                     true,
	"market.tickers.single":              true,
	"nft.collection.get":                 true,
	"nft.owner.get":                      true,
	"portfolio.get":                      true,
	"privatekey.derive":                  true,
	"security.address-check":             true,
	"sentiment.fear-greed":               true,
	"token.transfers":                    true,
	"tokens.get":                         true,
	"tx.hash":                            true,
	"tx.history":                         true,
	"utxo.batch":                         true,
	"utxo.list":                          true,
}

// TestRetryMatrix proves, operation by operation, which calls are retried
// after a 5xx: each one is answered 500 twice and must make 2 attempts (one
// retry) when retryable, exactly 1 otherwise.
func TestRetryMatrix(t *testing.T) {
	ops := loadOperations(t)
	if len(retryMatrix) != len(ops) {
		t.Fatalf("retryMatrix has %d entries, the API has %d operations", len(retryMatrix), len(ops))
	}
	for _, op := range ops {
		if _, ok := retryMatrix[op.id]; !ok {
			t.Errorf("retryMatrix is missing %s", op.id)
		}
	}

	opFor := func(method, path string) string {
		var best *operation
		for i := range ops {
			op := &ops[i]
			if op.method == method && op.re.MatchString(path) && (best == nil || op.params < best.params) {
				best = op
			}
		}
		if best == nil {
			t.Fatalf("%s %s matches no operation", method, path)
		}
		return best.id
	}

	type call struct {
		name, method, path string
		fn                 func(ctx context.Context, c *Client) (any, error)
		idempotencyKey     bool
	}
	var calls []call
	for _, table := range allEndpointTables() {
		for _, tc := range table {
			calls = append(calls, call{tc.name, tc.method, tc.path, tc.call, tc.header["Idempotency-Key"] != ""})
		}
	}
	calls = append(calls,
		call{"Storage.UploadIPFS", "POST", "/api/v1/blockchain/storage/ipfs", func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Storage.UploadIPFS(ctx, "a.txt", strings.NewReader("x"))
		}, false},
		call{"Sessions.GetDocument", "GET", "/api/v1/compliance/sessions/s1/documents/front_image", func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.GetDocument(ctx, "s1", DocumentFrontImage, nil)
		}, false},
		call{"Sessions.DownloadReport", "GET", "/api/v1/compliance/sessions/s1/report", func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.DownloadReport(ctx, "s1")
		}, false},
		call{"Screening.AMLCheck without key", "POST", "/api/v1/compliance/aml/checks", func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Screening.AMLCheck(ctx, &AMLCheckRequest{ExternalUserID: "u1", FullName: "Jane Doe"})
		}, false},
		call{"Screening.CreateWalletScreening without key", "POST", "/api/v1/compliance/wallet-screenings", func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Screening.CreateWalletScreening(ctx, &WalletScreeningRequest{Address: "0x1", Chain: "ETH"})
		}, false},
	)

	seen := map[string]bool{}
	for _, cl := range calls {
		id := opFor(cl.method, cl.path)
		seen[id] = true
		srv := newMockServer(t, mockResponse{status: 500, body: `{"error":{"code":"internal","message":"boom"}}`})
		_, _ = cl.fn(ctxBG(), srv.client(WithMaxRetries(1)))
		retryable := retryMatrix[id] || cl.idempotencyKey
		want := 1
		if retryable {
			want = 2
		}
		if got := len(srv.recorded()); got != want {
			t.Errorf("%s (%s): %d attempts after a 500, want %d (retryable=%v)", cl.name, id, got, want, retryable)
		}
	}
	for _, op := range ops {
		if !seen[op.id] {
			t.Errorf("no call exercised %s", op.id)
		}
	}
}

func TestAPIErrorSanitizesServerStrings(t *testing.T) {
	long := strings.Repeat("x", 5000)
	srv := newMockServer(t, mockResponse{
		status: 400,
		body:   `{"error":{"code":"bad\r\ncode","message":"line one\r\nINFO forged log line\u2028` + long + `","requestId":"req\n1"}}`,
	})
	_, err := srv.client().Card.Balance.Get(ctxBG())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	for _, s := range []string{err.Error(), apiErr.Message, apiErr.Code, apiErr.RequestID} {
		if strings.ContainsAny(s, "\r\n\u2028") {
			t.Errorf("control characters survived: %q", s)
		}
	}
	if !strings.HasPrefix(apiErr.Message, "line one  INFO forged log line ") {
		t.Errorf("message = %q", apiErr.Message[:40])
	}
	if !strings.HasSuffix(apiErr.Message, truncatedSuffix) {
		t.Errorf("long message was not marked as truncated")
	}
	if n := len([]rune(strings.TrimSuffix(apiErr.Message, truncatedSuffix))); n != maxErrorFieldRunes {
		t.Errorf("message length = %d runes, want %d", n, maxErrorFieldRunes)
	}
	eq(t, "code", apiErr.Code, "bad  code")
	eq(t, "request id", apiErr.RequestID, "req 1")
	// The raw body is untouched.
	if !strings.Contains(string(apiErr.Body), long) {
		t.Error("Body was modified")
	}
}
