package cryptures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is returned for every non-2xx response from the Cryptures API.
// Inspect it with errors.As:
//
//	var apiErr *cryptures.APIError
//	if errors.As(err, &apiErr) {
//		log.Printf("status=%d code=%s request_id=%s", apiErr.StatusCode, apiErr.Code, apiErr.RequestID)
//	}
//
// Most failures use the API's standard envelope
// {"error":{"code","message","requestId"}}. A few operations document a
// different error body: card operations forward the card issuer's own body
// ({"status":"failure","message","code"}), and some blockchain data
// operations forward the data source's own validation body. APIError
// populates Code and Message from whichever shape is present (falling back
// to the X-Request-ID header for RequestID), and always keeps the raw body
// in Body so nothing is lost.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Code is the machine-readable error code, e.g. "forbidden_scope",
	// "insufficient_balance", "rate_limited". It may be empty when the body
	// carries no code.
	Code string
	// Message is the human-readable explanation.
	Message string
	// RequestID identifies the request in Cryptures' logs. Include it when
	// reporting an issue.
	RequestID string
	// Body is the raw response body.
	Body []byte
	// Header is the response header.
	Header http.Header
}

// Error implements the error interface.
func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "cryptures: HTTP %d", e.StatusCode)
	if e.Code != "" {
		fmt.Fprintf(&b, " %s", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, " (request_id=%s)", e.RequestID)
	}
	return b.String()
}

// IsProviderError reports whether the body is a card issuer error forwarded
// verbatim ({"status":"failure",...}) rather than the API's own standard
// error envelope.
func (e *APIError) IsProviderError() bool {
	var probe struct {
		Error  json.RawMessage `json:"error"`
		Status string          `json:"status"`
	}
	if json.Unmarshal(e.Body, &probe) != nil {
		return false
	}
	return len(probe.Error) == 0 && probe.Status == "failure"
}

func newAPIError(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Body:       body,
		Header:     resp.Header,
	}

	var envelope struct {
		Error *struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"requestId"`
		} `json:"error"`
		// Card issuer pass-through shape and data-source shapes.
		Code      string `json:"code"`
		ErrorCode string `json:"errorCode"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if envelope.Error != nil {
			apiErr.Code = envelope.Error.Code
			apiErr.Message = envelope.Error.Message
			apiErr.RequestID = envelope.Error.RequestID
		} else {
			apiErr.Code = envelope.Code
			if apiErr.Code == "" {
				apiErr.Code = envelope.ErrorCode
			}
			apiErr.Message = envelope.Message
		}
	}
	if apiErr.RequestID == "" {
		apiErr.RequestID = resp.Header.Get("X-Request-ID")
	}
	if apiErr.Message == "" && apiErr.Code == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}
