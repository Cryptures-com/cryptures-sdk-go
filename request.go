package cryptures

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxErrorBodyBytes bounds how much of a non-2xx body is read into an
// APIError, so a misbehaving intermediary cannot exhaust memory.
const maxErrorBodyBytes = 1 << 20

// requestSpec describes one API call. Endpoint methods build a spec and hand
// it to the shared HTTP layer, which owns auth, retries, and error mapping.
type requestSpec struct {
	method string
	// path is the request path, with every caller-supplied segment already
	// escaped (see pathf).
	path string
	// route is the documented path template, e.g.
	// "/api/v1/blockchain/wallet/{chain}". It is what error messages show,
	// so that caller-supplied path segments (which can be secrets, like an
	// EGLD mnemonic passed to DeriveAddress) never end up in an error string.
	// It must be set whenever path is built with pathf; for a fixed path it
	// may be left empty and path is used.
	route string
	query url.Values
	// body is JSON-encoded when non-nil.
	body any
	// rawBody and contentType are used instead of body for non-JSON request
	// bodies (multipart uploads).
	rawBody     []byte
	contentType string
	// accept overrides the Accept header (default application/json).
	accept string
	// idempotencyKey is sent as the Idempotency-Key header when non-empty,
	// and makes an otherwise non-retryable request retry-safe.
	idempotencyKey string
	// retryable marks the request as safe to retry after a network error or
	// a 5xx response. Requests whose repetition could duplicate a side effect
	// (a broadcast, a balance debit, a card creation) leave this false.
	retryable bool
}

// RequestOption customizes a single API call. Every service method accepts
// zero or more RequestOptions as its final arguments.
type RequestOption func(*requestOptions)

type requestOptions struct {
	header   http.Header
	metadata *ResponseMetadata
}

// WithHeader sets an additional HTTP header on a single request. It cannot
// override the x-api-key header.
func WithHeader(key, value string) RequestOption {
	return func(o *requestOptions) {
		if o.header == nil {
			o.header = make(http.Header)
		}
		o.header.Set(key, value)
	}
}

// WithResponseMetadata captures metadata about the final HTTP response (after
// any retries) into m. It is populated for both successful and failed calls,
// as long as a response was received.
func WithResponseMetadata(m *ResponseMetadata) RequestOption {
	return func(o *requestOptions) { o.metadata = m }
}

// ResponseMetadata describes the HTTP response to a call. Obtain it with
// WithResponseMetadata.
type ResponseMetadata struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// RequestID is the X-Request-ID response header. Include it when
	// reporting an issue to Cryptures.
	RequestID string
	// Cache is the X-Relay-Cache response header: "HIT", "MISS", "STALE",
	// "BYPASS", or empty when the operation omits the header (treat empty as
	// "BYPASS").
	Cache string
	// IdempotentReplay is true when the response is a replay of an earlier
	// call made with the same Idempotency-Key (Idempotent-Replay: true).
	IdempotentReplay bool
	// Attempts is the number of HTTP attempts made, including retries.
	Attempts int
	// Header is the full response header.
	Header http.Header
}

// displayRoute is the path as error messages show it: the documented template
// (route) when the path has caller-supplied segments, else the fixed path.
func (s *requestSpec) displayRoute() string {
	if s.route != "" {
		return s.route
	}
	return s.path
}

// op names the call in error messages as "METHOD /route/{template}". It never
// includes the interpolated path or the query string, either of which can
// carry secrets (the mnemonic sent by Wallet.Generate, or the EGLD mnemonic
// placed in the path by Wallet.DeriveAddress).
func (s *requestSpec) op() string {
	return s.method + " " + s.displayRoute()
}

// transportError is returned when a request fails at the network level
// (connection refused, DNS or TLS failure, timeout, ...) before any HTTP
// response arrived.
//
// Its message names only the HTTP method and the documented path template,
// never the request URL, which can carry secrets in its query string or path.
// It unwraps to a *url.Error whose URL field holds that same path template, so
// errors.As with *url.Error or net.Error, and errors.Is with
// context.DeadlineExceeded, keep working.
type transportError struct {
	op  string
	err *url.Error
}

func (e *transportError) Error() string { return "cryptures: " + e.op + ": " + e.err.Err.Error() }

func (e *transportError) Unwrap() error { return e.err }

// newTransportError wraps an error from http.Client.Do without keeping the
// request URL, which *url.Error embeds in its message verbatim.
func newTransportError(spec *requestSpec, err error) error {
	redacted := &url.Error{Op: spec.method, URL: spec.displayRoute(), Err: err}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		redacted.Op = urlErr.Op
		redacted.Err = urlErr.Err
	}
	return &transportError{op: spec.op(), err: redacted}
}

// pathf formats an API path, escaping every argument as a single path
// segment so caller-supplied identifiers cannot alter the route. Commas are
// left literal: they are legal in a path segment and the API accepts
// comma-separated id lists in the path (e.g. market tickers "90,80").
func pathf(format string, segments ...string) string {
	args := make([]any, len(segments))
	for i, s := range segments {
		args[i] = strings.ReplaceAll(url.PathEscape(s), "%2C", ",")
	}
	return fmt.Sprintf(format, args...)
}

// doJSON performs the request and decodes a JSON response body into T.
func doJSON[T any](ctx context.Context, c *Client, spec *requestSpec, opts []RequestOption) (*T, error) {
	resp, err := c.send(ctx, spec, opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cryptures: reading %s response: %w", spec.op(), err)
	}
	out := new(T)
	if len(bytes.TrimSpace(body)) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, fmt.Errorf("cryptures: decoding %s response: %w", spec.op(), err)
	}
	return out, nil
}

// doNoContent performs a request whose successful response has no body.
func doNoContent(ctx context.Context, c *Client, spec *requestSpec, opts []RequestOption) error {
	resp, err := c.send(ctx, spec, opts)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.Body.Close()
}

// send executes the request with authentication, retries, and error mapping.
// On success it returns the response with its body unread; the caller must
// close it. Any non-2xx response is returned as an *APIError.
func (c *Client) send(ctx context.Context, spec *requestSpec, opts []RequestOption) (*http.Response, error) {
	if ctx == nil {
		return nil, errors.New("cryptures: nil context")
	}
	ro := &requestOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(ro)
		}
	}

	var payload []byte
	contentType := spec.contentType
	switch {
	case spec.rawBody != nil:
		payload = spec.rawBody
	case spec.body != nil:
		b, err := json.Marshal(spec.body)
		if err != nil {
			return nil, fmt.Errorf("cryptures: encoding %s request body: %w", spec.op(), err)
		}
		payload = b
		contentType = "application/json"
	}

	endpoint := c.baseURL + spec.path
	if len(spec.query) > 0 {
		endpoint += "?" + spec.query.Encode()
	}

	retryable := spec.retryable || spec.idempotencyKey != ""
	maxRetries := 0
	if retryable {
		maxRetries = c.maxRetries
	}

	for attempt := 0; ; attempt++ {
		req, err := c.newHTTPRequest(ctx, spec, endpoint, payload, contentType, ro)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, fmt.Errorf("cryptures: %s: %w", spec.op(), ctxErr)
			}
			if attempt < maxRetries {
				if werr := c.wait(ctx, attempt, nil); werr != nil {
					return nil, werr
				}
				continue
			}
			// Never wrap err directly: *url.Error's message embeds the full
			// request URL, query string and all.
			return nil, newTransportError(spec, err)
		}

		if resp.StatusCode >= 500 && attempt < maxRetries {
			retryAfter := resp.Header.Get("Retry-After")
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBodyBytes))
			_ = resp.Body.Close()
			if werr := c.wait(ctx, attempt, &retryAfter); werr != nil {
				return nil, werr
			}
			continue
		}

		recordMetadata(ro.metadata, resp, attempt+1)

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
			_ = resp.Body.Close()
			return nil, newAPIError(resp, body)
		}
		return resp, nil
	}
}

func (c *Client) newHTTPRequest(ctx context.Context, spec *requestSpec, endpoint string, payload []byte, contentType string, ro *requestOptions) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, spec.method, endpoint, body)
	if err != nil {
		// A URL parse failure is a *url.Error that embeds the full URL; keep
		// only its cause.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("cryptures: building %s request: %w", spec.op(), err)
	}
	for k, vs := range ro.header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	accept := spec.accept
	if accept == "" {
		accept = "application/json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if spec.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", spec.idempotencyKey)
	}
	req.Header.Set("x-api-key", c.apiKey)
	return req, nil
}

// wait sleeps before retry number attempt+1 using exponential backoff with
// jitter, honoring a Retry-After header (in seconds) when the server sends
// one. It returns early with an error if ctx is done.
func (c *Client) wait(ctx context.Context, attempt int, retryAfter *string) error {
	ra := ""
	if retryAfter != nil {
		ra = *retryAfter
	}
	t := time.NewTimer(c.backoff(attempt, ra))
	defer t.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("cryptures: waiting to retry: %w", ctx.Err())
	case <-t.C:
		return nil
	}
}

// backoff returns the delay before retry number attempt+1: the base delay
// doubled per attempt, with equal jitter (between half and all of it),
// capped at maxRetryDelay. A Retry-After header in seconds raises the delay
// when it asks for longer, up to the same cap.
func (c *Client) backoff(attempt int, retryAfter string) time.Duration {
	var delay time.Duration
	if c.retryBaseDelay > 0 {
		delay = maxRetryDelay
		if attempt < 30 {
			if d := c.retryBaseDelay << uint(attempt); d > 0 && d < maxRetryDelay {
				delay = d
			}
		}
		if half := int64(delay / 2); half > 0 {
			delay = time.Duration(half + rand.Int63n(half+1))
		}
	}
	if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 {
		if ra := time.Duration(secs) * time.Second; ra > delay && ra <= maxRetryDelay {
			delay = ra
		}
	}
	return delay
}

func recordMetadata(m *ResponseMetadata, resp *http.Response, attempts int) {
	if m == nil {
		return
	}
	*m = ResponseMetadata{
		StatusCode:       resp.StatusCode,
		RequestID:        resp.Header.Get("X-Request-ID"),
		Cache:            resp.Header.Get("X-Relay-Cache"),
		IdempotentReplay: resp.Header.Get("Idempotent-Replay") == "true",
		Attempts:         attempts,
		Header:           resp.Header,
	}
}

// queryBuilder accumulates optional query parameters, skipping unset values.
type queryBuilder struct{ v url.Values }

func newQuery() *queryBuilder { return &queryBuilder{v: url.Values{}} }

func (q *queryBuilder) str(key, value string) *queryBuilder {
	if value != "" {
		q.v.Set(key, value)
	}
	return q
}

func (q *queryBuilder) intPtr(key string, value *int64) *queryBuilder {
	if value != nil {
		q.v.Set(key, strconv.FormatInt(*value, 10))
	}
	return q
}

func (q *queryBuilder) boolPtr(key string, value *bool) *queryBuilder {
	if value != nil {
		q.v.Set(key, strconv.FormatBool(*value))
	}
	return q
}

func (q *queryBuilder) float(key string, value float64) *queryBuilder {
	q.v.Set(key, strconv.FormatFloat(value, 'f', -1, 64))
	return q
}

func (q *queryBuilder) values() url.Values {
	if len(q.v) == 0 {
		return nil
	}
	return q.v
}
