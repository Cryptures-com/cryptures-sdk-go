package cryptures

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const testAPIKey = "test_key_123"

// recordedRequest is what the mock API server saw.
type recordedRequest struct {
	Method string
	Path   string // escaped path, exactly as sent
	Query  url.Values
	Header http.Header
	Body   []byte
}

// mockServer is an httptest server that records every request and answers
// with a scripted sequence of responses (the last one repeats).
type mockServer struct {
	*httptest.Server
	mu        sync.Mutex
	requests  []recordedRequest
	responses []mockResponse
}

type mockResponse struct {
	status      int
	body        string
	header      map[string]string
	contentType string
}

func newMockServer(t *testing.T, responses ...mockResponse) *mockServer {
	t.Helper()
	m := &mockServer{responses: responses}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m.mu.Lock()
		m.requests = append(m.requests, recordedRequest{
			Method: r.Method,
			Path:   r.URL.EscapedPath(),
			Query:  r.URL.Query(),
			Header: r.Header.Clone(),
			Body:   body,
		})
		idx := len(m.requests) - 1
		if idx >= len(m.responses) {
			idx = len(m.responses) - 1
		}
		resp := mockResponse{status: http.StatusOK, body: "{}"}
		if idx >= 0 {
			resp = m.responses[idx]
		}
		m.mu.Unlock()

		ct := resp.contentType
		if ct == "" {
			ct = "application/json"
		}
		w.Header().Set("Content-Type", ct)
		for k, v := range resp.header {
			w.Header().Set(k, v)
		}
		status := resp.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, resp.body)
	}))
	t.Cleanup(m.Close)
	return m
}

func (m *mockServer) recorded() []recordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]recordedRequest(nil), m.requests...)
}

func (m *mockServer) client(opts ...ClientOption) *Client {
	base := []ClientOption{WithBaseURL(m.URL), withRetryBaseDelay(time.Millisecond)}
	return NewClient(testAPIKey, append(base, opts...)...)
}

// endpointCase is one row of a table-driven endpoint test: it calls one SDK
// method against the mock server, then asserts the exact HTTP request that
// went out and the parsed response that came back.
type endpointCase struct {
	name string
	call func(ctx context.Context, c *Client) (any, error)

	// Expected request.
	method string
	path   string
	query  url.Values // nil means "no query string"
	body   string     // expected JSON body; "" means no body
	header map[string]string

	// Scripted response.
	status      int
	response    string
	contentType string

	// check asserts on the parsed result.
	check func(t *testing.T, got any)
}

func runEndpointCases(t *testing.T, cases []endpointCase) {
	t.Helper()
	seen := map[string]bool{}
	for _, tc := range cases {
		tc := tc
		if seen[tc.name] {
			t.Fatalf("duplicate case name %q", tc.name)
		}
		seen[tc.name] = true
		t.Run(tc.name, func(t *testing.T) {
			srv := newMockServer(t, mockResponse{status: tc.status, body: tc.response, contentType: tc.contentType})
			got, err := tc.call(context.Background(), srv.client())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			reqs := srv.recorded()
			if len(reqs) != 1 {
				t.Fatalf("expected exactly 1 request, got %d", len(reqs))
			}
			r := reqs[0]
			if r.Method != tc.method {
				t.Errorf("method = %s, want %s", r.Method, tc.method)
			}
			if r.Path != tc.path {
				t.Errorf("path = %s, want %s", r.Path, tc.path)
			}
			wantQuery := tc.query
			if wantQuery == nil {
				wantQuery = url.Values{}
			}
			if !reflect.DeepEqual(r.Query, wantQuery) {
				t.Errorf("query = %v, want %v", r.Query, wantQuery)
			}
			if got := r.Header.Get("x-api-key"); got != testAPIKey {
				t.Errorf("x-api-key = %q, want %q", got, testAPIKey)
			}
			if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "cryptures-sdk-go/") {
				t.Errorf("User-Agent = %q", ua)
			}
			for k, v := range tc.header {
				if got := r.Header.Get(k); got != v {
					t.Errorf("header %s = %q, want %q", k, got, v)
				}
			}
			if tc.body == "" {
				if len(r.Body) != 0 {
					t.Errorf("expected no request body, got %s", r.Body)
				}
			} else {
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
				assertJSONEqual(t, r.Body, []byte(tc.body))
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// assertJSONEqual compares two JSON documents semantically.
func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("invalid expected JSON %q: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("JSON mismatch\n got: %s\nwant: %s", got, want)
	}
}

func eq[T any](t *testing.T, label string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %#v, want %#v", label, got, want)
	}
}

func q(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Add(kv[i], kv[i+1])
	}
	return v
}

func ctxBG() context.Context { return context.Background() }

var errorsAs = errors.As
