package cryptures

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("k")
	eq(t, "base url", c.BaseURL(), DefaultBaseURL)
	eq(t, "base url value", c.BaseURL(), "https://api.cryptures.com")
	eq(t, "max retries", c.maxRetries, DefaultMaxRetries)
	eq(t, "timeout", c.httpClient.Timeout, DefaultTimeout)
	if c.Blockchain.Data == nil || c.Blockchain.Operations == nil || c.Blockchain.Wallet == nil ||
		c.Blockchain.Contracts == nil || c.Blockchain.Fee == nil || c.Blockchain.Lookups == nil ||
		c.Blockchain.NFT == nil || c.Blockchain.Storage == nil {
		t.Error("blockchain services not wired")
	}
	if c.Card.Cards == nil || c.Card.Balance == nil || c.Card.Tags == nil || c.Card.Reports == nil || c.Card.Webhooks == nil {
		t.Error("card services not wired")
	}
	if c.Compliance.Sessions == nil || c.Compliance.Screening == nil || c.Compliance.Monitoring == nil || c.Compliance.Webhooks == nil {
		t.Error("compliance services not wired")
	}
}

func TestClientOptions(t *testing.T) {
	t.Run("WithBaseURL trims trailing slash", func(t *testing.T) {
		eq(t, "base", NewClient("k", WithBaseURL("https://sandbox.example.com/")).BaseURL(), "https://sandbox.example.com")
	})
	t.Run("WithTimeout does not mutate a caller's http.Client", func(t *testing.T) {
		hc := &http.Client{Timeout: time.Minute}
		c := NewClient("k", WithHTTPClient(hc), WithTimeout(5*time.Second))
		eq(t, "client timeout", c.httpClient.Timeout, 5*time.Second)
		eq(t, "caller timeout", hc.Timeout, time.Minute)

		// Option order must not matter.
		c = NewClient("k", WithTimeout(7*time.Second), WithHTTPClient(hc))
		eq(t, "client timeout (reversed order)", c.httpClient.Timeout, 7*time.Second)
		eq(t, "caller timeout (reversed order)", hc.Timeout, time.Minute)

		// Without WithTimeout, a caller's client is used as-is.
		eq(t, "untouched", NewClient("k", WithHTTPClient(hc)).httpClient, hc)
	})
	t.Run("WithMaxRetries clamps negatives", func(t *testing.T) {
		eq(t, "retries", NewClient("k", WithMaxRetries(-4)).maxRetries, 0)
		eq(t, "retries", NewClient("k", WithMaxRetries(7)).maxRetries, 7)
	})
	t.Run("WithHTTPClient nil is ignored", func(t *testing.T) {
		if NewClient("k", WithHTTPClient(nil)).httpClient == nil {
			t.Error("http client must not be nil")
		}
	})
	t.Run("WithHTTPClient is used for requests", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{body: `{"balance":"1"}`})
		var used int32
		hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			atomic.AddInt32(&used, 1)
			return http.DefaultTransport.RoundTrip(r)
		})}
		if _, err := srv.client(WithHTTPClient(hc)).Blockchain.Data.GetBalance(ctxBG(), "ETH", "0x1"); err != nil {
			t.Fatal(err)
		}
		eq(t, "transport used", atomic.LoadInt32(&used), int32(1))
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestHeaders(t *testing.T) {
	srv := newMockServer(t, mockResponse{body: `{"balance":"1"}`})
	_, err := srv.client().Blockchain.Data.GetBalance(ctxBG(), "ETH", "0x1",
		WithHeader("X-Trace", "abc"),
		WithHeader("x-api-key", "attempted-override"),
	)
	if err != nil {
		t.Fatal(err)
	}
	h := srv.recorded()[0].Header
	eq(t, "trace", h.Get("X-Trace"), "abc")
	eq(t, "api key cannot be overridden", h.Values("x-api-key"), []string{testAPIKey})
	eq(t, "accept", h.Get("Accept"), "application/json")
	eq(t, "user agent", h.Get("User-Agent"), "cryptures-sdk-go/"+Version)
	eq(t, "no content type on GET", h.Get("Content-Type"), "")
}

func TestPathSegmentsAreEscaped(t *testing.T) {
	srv := newMockServer(t, mockResponse{body: `{"address":"a"}`})
	_, err := srv.client().Blockchain.Wallet.DeriveAddress(ctxBG(), "EGLD", "word one/../two?x", 0)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "path", srv.recorded()[0].Path, "/api/v1/blockchain/wallet/EGLD/address/word%20one%2F..%2Ftwo%3Fx/0")
}

func TestAPIErrorMapping(t *testing.T) {
	t.Run("standard envelope", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 402, body: `{"error":{"code":"insufficient_balance","message":"Balance too low","requestId":"req_abc"}}`})
		_, err := srv.client().Card.Cards.Fund(ctxBG(), "card_1", 10)
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T: %v", err, err)
		}
		eq(t, "status", apiErr.StatusCode, 402)
		eq(t, "code", apiErr.Code, "insufficient_balance")
		eq(t, "message", apiErr.Message, "Balance too low")
		eq(t, "request id", apiErr.RequestID, "req_abc")
		eq(t, "provider", apiErr.IsProviderError(), false)
		eq(t, "string", apiErr.Error(), "cryptures: HTTP 402 insufficient_balance: Balance too low (request_id=req_abc)")
	})
	t.Run("card issuer pass-through", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 404, body: `{"status":"failure","message":"Card not found.","code":"CARD_NOT_FOUND"}`, header: map[string]string{"X-Request-ID": "req_hdr"}})
		_, err := srv.client().Card.Cards.Get(ctxBG(), "card_gone")
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %v", err)
		}
		eq(t, "code", apiErr.Code, "CARD_NOT_FOUND")
		eq(t, "message", apiErr.Message, "Card not found.")
		eq(t, "request id from header", apiErr.RequestID, "req_hdr")
		eq(t, "provider", apiErr.IsProviderError(), true)
	})
	t.Run("rate-not-found 403", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 403, body: `{"statusCode":403,"errorCode":"rate.not.found","message":"No USD, XRP currency rates."}`})
		_, err := srv.client().Blockchain.Data.GetExchangeRate(ctxBG(), "XRP", nil)
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %v", err)
		}
		eq(t, "code", apiErr.Code, "rate.not.found")
		eq(t, "message", apiErr.Message, "No USD, XRP currency rates.")
	})
	t.Run("non-JSON body", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 418, body: `<html>teapot</html>`, contentType: "text/html"})
		_, err := srv.client().Blockchain.Data.GetBalance(ctxBG(), "ETH", "0x1")
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %v", err)
		}
		eq(t, "message", apiErr.Message, "I'm a teapot")
		eq(t, "body", string(apiErr.Body), `<html>teapot</html>`)
	})
	t.Run("metadata captured on error", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 404, body: `{"error":{"code":"session_not_found","message":"m","requestId":"r1"}}`, header: map[string]string{"X-Request-ID": "r1"}})
		var meta ResponseMetadata
		_, err := srv.client().Compliance.Sessions.Get(ctxBG(), "nope", WithResponseMetadata(&meta))
		if err == nil {
			t.Fatal("expected error")
		}
		eq(t, "status", meta.StatusCode, 404)
		eq(t, "request id", meta.RequestID, "r1")
	})
}

func TestDecodeErrorOnMalformedSuccessBody(t *testing.T) {
	srv := newMockServer(t, mockResponse{body: `{"txId":`})
	_, err := srv.client().Blockchain.Operations.Broadcast(ctxBG(), "BTC", &BroadcastRequest{TxData: "00"})
	if err == nil || !strings.Contains(err.Error(), "decoding") {
		t.Fatalf("expected a decoding error, got %v", err)
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Error("a decoding failure must not be reported as an APIError")
	}
}

func TestNilContext(t *testing.T) {
	c := NewClient("k")
	var nilCtx context.Context
	_, err := c.Blockchain.Data.GetBalance(nilCtx, "ETH", "0x1")
	if err == nil {
		t.Fatal("expected an error for a nil context")
	}
}

func TestRetries(t *testing.T) {
	t.Run("retries 5xx then succeeds", func(t *testing.T) {
		srv := newMockServer(t,
			mockResponse{status: 503, body: `{"error":{"code":"upstream_backoff","message":"m","requestId":"r"}}`},
			mockResponse{status: 502, body: `{}`},
			mockResponse{status: 200, body: `{"balance":"7"}`},
		)
		var meta ResponseMetadata
		got, err := srv.client().Blockchain.Data.GetBalance(ctxBG(), "ETH", "0x1", WithResponseMetadata(&meta))
		if err != nil {
			t.Fatal(err)
		}
		s, _ := got.AsSimple()
		eq(t, "balance", s.Balance, "7")
		eq(t, "requests", len(srv.recorded()), 3)
		eq(t, "attempts", meta.Attempts, 3)
	})
	t.Run("gives up after MaxRetries", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 500, body: `{"error":{"code":"internal","message":"boom","requestId":"r"}}`})
		_, err := srv.client(WithMaxRetries(2)).Card.Balance.Get(ctxBG())
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %v", err)
		}
		eq(t, "status", apiErr.StatusCode, 500)
		eq(t, "requests", len(srv.recorded()), 3)
	})
	t.Run("never retries 4xx", func(t *testing.T) {
		for _, status := range []int{400, 401, 403, 404, 409, 429} {
			srv := newMockServer(t, mockResponse{status: status, body: `{"error":{"code":"x","message":"m","requestId":"r"}}`})
			_, err := srv.client().Card.Balance.Get(ctxBG())
			if err == nil {
				t.Fatalf("status %d: expected error", status)
			}
			eq(t, "requests", len(srv.recorded()), 1)
		}
	})
	t.Run("WithMaxRetries(0) disables retries", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 503, body: `{}`})
		_, _ = srv.client(WithMaxRetries(0)).Card.Balance.Get(ctxBG())
		eq(t, "requests", len(srv.recorded()), 1)
	})
	t.Run("does not retry side-effecting requests", func(t *testing.T) {
		calls := map[string]func(c *Client) error{
			"Operations.Send": func(c *Client) error {
				_, err := c.Blockchain.Operations.Send(ctxBG(), "ETH", &SendEVMRequest{Currency: "ETH", Amount: "1", To: "0x", FromPrivateKey: "k"})
				return err
			},
			"Operations.Broadcast": func(c *Client) error {
				_, err := c.Blockchain.Operations.Broadcast(ctxBG(), "BTC", &BroadcastRequest{TxData: "00"})
				return err
			},
			"Contracts.DeployToken": func(c *Client) error {
				_, err := c.Blockchain.Contracts.DeployToken(ctxBG(), &DeployTokenRequest{Chain: "ETH"})
				return err
			},
			"Cards.Create": func(c *Client) error {
				_, err := c.Card.Cards.Create(ctxBG(), &CreateCardRequest{ProductCode: "p"})
				return err
			},
			"Cards.Fund": func(c *Client) error {
				_, err := c.Card.Cards.Fund(ctxBG(), "c", 1)
				return err
			},
			"Screening.AMLCheck without key": func(c *Client) error {
				_, err := c.Compliance.Screening.AMLCheck(ctxBG(), &AMLCheckRequest{ExternalUserID: "u", FullName: "n"})
				return err
			},
		}
		for name, call := range calls {
			srv := newMockServer(t, mockResponse{status: 500, body: `{"error":{"code":"internal","message":"timeout","requestId":"r"}}`})
			if err := call(srv.client()); err == nil {
				t.Fatalf("%s: expected error", name)
			}
			if n := len(srv.recorded()); n != 1 {
				t.Errorf("%s: sent %d requests, want exactly 1 (no automatic retry)", name, n)
			}
		}
	})
	t.Run("idempotency key makes a POST retry-safe", func(t *testing.T) {
		srv := newMockServer(t,
			mockResponse{status: 502, body: `{"error":{"code":"upstream_error","message":"m","requestId":"r"}}`},
			mockResponse{status: 201, body: `{"check_id":"chk","address":"a","chain":"ETH","result":{"severity":"LOW","sanctions_hit":false,"risk_factors":[]},"screened_at":"x"}`},
		)
		got, err := srv.client().Compliance.Screening.CreateWalletScreening(ctxBG(), &WalletScreeningRequest{Address: "a", Chain: "ETH", IdempotencyKey: "key-1"})
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "check", got.CheckID, "chk")
		reqs := srv.recorded()
		eq(t, "requests", len(reqs), 2)
		for _, r := range reqs {
			eq(t, "key", r.Header.Get("Idempotency-Key"), "key-1")
			assertJSONEqual(t, r.Body, []byte(`{"address":"a","chain":"ETH"}`))
		}
	})
	t.Run("retries network errors", func(t *testing.T) {
		var n int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&n, 1) == 1 {
				hj, ok := w.(http.Hijacker)
				if !ok {
					t.Fatal("hijacking not supported")
				}
				conn, _, _ := hj.Hijack()
				_ = conn.Close() // drop the connection without a response
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"balance_usd":1,"updated_at":null}`))
		}))
		defer srv.Close()
		c := NewClient(testAPIKey, WithBaseURL(srv.URL), withRetryBaseDelay(time.Millisecond))
		got, err := c.Card.Balance.Get(ctxBG())
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "balance", got.BalanceUSD, 1.0)
		eq(t, "attempts", atomic.LoadInt32(&n), int32(2))
	})
	t.Run("network error without retries is returned", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := ln.Addr().String()
		_ = ln.Close() // nothing listens here any more
		c := NewClient(testAPIKey, WithBaseURL("http://"+addr), WithMaxRetries(1), withRetryBaseDelay(time.Millisecond))
		_, err = c.Card.Balance.Get(ctxBG())
		if err == nil {
			t.Fatal("expected a connection error")
		}
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			t.Error("a transport failure must not be reported as an APIError")
		}
	})
	t.Run("context cancellation stops the backoff", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 503, body: `{}`})
		c := srv.client(withRetryBaseDelay(10 * time.Second))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err := c.Card.Balance.Get(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("cancellation took %v", elapsed)
		}
	})
}

func TestBackoff(t *testing.T) {
	c := NewClient("k")
	for attempt := 0; attempt < 4; attempt++ {
		full := defaultRetryBaseDelay << uint(attempt)
		for i := 0; i < 50; i++ {
			d := c.backoff(attempt, "")
			if d < full/2 || d > full {
				t.Fatalf("attempt %d: delay %v outside [%v, %v]", attempt, d, full/2, full)
			}
		}
	}
	if d := c.backoff(100, ""); d > maxRetryDelay || d < maxRetryDelay/2 {
		t.Errorf("large attempt delay %v not capped", d)
	}
	eq(t, "retry-after honored", c.backoff(0, "3"), 3*time.Second)
	if d := c.backoff(0, "3600"); d > time.Second {
		t.Errorf("retry-after beyond the cap must be ignored, got %v", d)
	}
	if d := c.backoff(0, "soon"); d > defaultRetryBaseDelay {
		t.Errorf("unparseable retry-after must be ignored, got %v", d)
	}
	eq(t, "zero base", NewClient("k", withRetryBaseDelay(0)).backoff(2, ""), time.Duration(0))
}
