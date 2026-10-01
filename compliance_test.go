package cryptures

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestComplianceEndpoints(t *testing.T) { runEndpointCases(t, complianceCases) }

var complianceCases = []endpointCase{
	{
		name: "Sessions.Create",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.Create(ctx, &CreateSessionRequest{
				PresetID: "preset_af36", ExternalUserID: "user_42", Callback: "https://example.com/done", Language: "en",
				Metadata:        json.RawMessage(`{"plan":"pro"}`),
				ExpectedDetails: &ExpectedDetails{FirstName: "Jane", LastName: "Doe", DateOfBirth: "1990-01-01"},
				ContactDetails:  &ContactDetails{Email: "jane@example.com"},
			})
		},
		method:   "POST",
		path:     "/api/v1/compliance/sessions/create",
		body:     `{"preset_id":"preset_af36","external_user_id":"user_42","callback":"https://example.com/done","language":"en","metadata":{"plan":"pro"},"expected_details":{"first_name":"Jane","last_name":"Doe","date_of_birth":"1990-01-01"},"contact_details":{"email":"jane@example.com"}}`,
		status:   201,
		response: `{"session_id":"a1b2c3d4","url":"https://verify.example.com/session/xyz","status":"Not Started"}`,
		check: func(t *testing.T, got any) {
			eq(t, "session", *got.(*CreatedSession), CreatedSession{SessionID: "a1b2c3d4", URL: "https://verify.example.com/session/xyz", Status: "Not Started"})
		},
	},
	{
		name: "Sessions.Get",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.Get(ctx, "a1b2c3d4")
		},
		method:   "GET",
		path:     "/api/v1/compliance/sessions/a1b2c3d4",
		response: `{"session_id":"a1b2c3d4","status":"Approved","external_user_id":"user_42","kind":"kyc","features":["ID_VERIFICATION","LIVENESS","AML"],"results":[{"feature":"ID_VERIFICATION","status":"Approved","document_type":"Identity Card","name":"Jane Doe","date_of_birth":"1990-01-01","issuing_state":"ESP"},{"feature":"LIVENESS","status":"Approved","method":"ACTIVE_3D","score":95.4},{"feature":"AML","status":"Approved","total_hits":0,"entity_type":"person"}],"warnings":[{"feature":"AML","short_description":"x"}],"document_urls":{"portrait_image":"https://api.cryptures.com/api/v1/compliance/sessions/a1b2c3d4/documents/portrait_image"}}`,
		check: func(t *testing.T, got any) {
			r := got.(*Session)
			eq(t, "kind", r.Kind, "kyc")
			eq(t, "features", r.Features, []string{"ID_VERIFICATION", "LIVENESS", "AML"})
			eq(t, "doc type", r.Results[0].DocumentType, "Identity Card")
			eq(t, "score", *r.Results[1].Score, 95.4)
			eq(t, "hits", *r.Results[2].TotalHits, int64(0))
			eq(t, "warning", r.Warnings[0].ShortDescription, "x")
			eq(t, "doc url", r.DocumentURLs[DocumentPortraitImage], "https://api.cryptures.com/api/v1/compliance/sessions/a1b2c3d4/documents/portrait_image")
		},
	},
	{
		name: "Sessions.List",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.List(ctx, &ListSessionsParams{
				Status: "In Review", ExternalUserID: "user_42", CreatedAfter: "2026-09-01", CreatedBefore: "2026-10-01T00:00:00Z", Limit: Int64(25), Cursor: "c1",
			})
		},
		method:   "GET",
		path:     "/api/v1/compliance/sessions",
		query:    q("status", "In Review", "external_user_id", "user_42", "created_after", "2026-09-01", "created_before", "2026-10-01T00:00:00Z", "limit", "25", "cursor", "c1"),
		response: `{"sessions":[{"session_id":"s1","external_user_id":"user_42","preset_id":"p1","status":"In Review","kind":"kyc","created_at":"2026-09-18T10:51:32.000Z","updated_at":"2026-09-18T10:55:53.000Z"}],"next_cursor":"c2"}`,
		check: func(t *testing.T, got any) {
			r := got.(*SessionList)
			eq(t, "next", r.NextCursor, "c2")
			eq(t, "preset", r.Sessions[0].PresetID, "p1")
		},
	},
	{
		name: "Sessions.ListPresets",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.ListPresets(ctx)
		},
		method:   "GET",
		path:     "/api/v1/compliance/presets",
		response: `{"presets":[{"id":"preset_af36","label":"Standard KYC","kind":"kyc"},{"id":"preset_kyb","label":"KYB","kind":"kyb"}]}`,
		check: func(t *testing.T, got any) {
			eq(t, "preset", got.(*PresetList).Presets[1], Preset{ID: "preset_kyb", Label: "KYB", Kind: "kyb"})
		},
	},
	{
		name: "Sessions.UpdateStatus",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.UpdateStatus(ctx, "a1b2c3d4", &UpdateSessionStatusRequest{Status: "Approved", Comment: "Reviewed."})
		},
		method:   "PATCH",
		path:     "/api/v1/compliance/sessions/a1b2c3d4/status",
		body:     `{"status":"Approved","comment":"Reviewed."}`,
		response: `{"session_id":"a1b2c3d4","status":"Approved"}`,
		check: func(t *testing.T, got any) {
			eq(t, "update", *got.(*SessionStatusUpdate), SessionStatusUpdate{SessionID: "a1b2c3d4", Status: "Approved"})
		},
	},
	{
		name: "Sessions.Delete",
		call: func(ctx context.Context, c *Client) (any, error) {
			return nil, c.Compliance.Sessions.Delete(ctx, "a1b2c3d4", &DeleteSessionParams{PrivacyErasure: Bool(true)})
		},
		method: "DELETE",
		path:   "/api/v1/compliance/sessions/a1b2c3d4",
		query:  q("privacy_erasure", "true"),
		status: 204,
	},
	{
		name: "Sessions.CreateReport",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Sessions.CreateReport(ctx, "a1b2c3d4")
		},
		method:   "POST",
		path:     "/api/v1/compliance/sessions/a1b2c3d4/report",
		status:   201,
		response: `{"report_id":"rep_3d5f","session_id":"a1b2c3d4","generated_at":"2026-09-25T09:00:00.000Z","expires_at":"2026-10-02T09:00:00.000Z","size_bytes":18234,"download_url":"https://api.cryptures.com/api/v1/compliance/sessions/a1b2c3d4/report"}`,
		check: func(t *testing.T, got any) {
			r := got.(*Report)
			eq(t, "size", r.SizeBytes, int64(18234))
			eq(t, "expires", r.ExpiresAt, "2026-10-02T09:00:00.000Z")
		},
	},
	{
		name: "Screening.AMLCheck",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Screening.AMLCheck(ctx, &AMLCheckRequest{
				ExternalUserID: "user_42", FullName: "Jane Doe", DateOfBirth: "1990-01-01", Nationality: "ES", EntityType: "person", IdempotencyKey: "idem-1",
			})
		},
		method:   "POST",
		path:     "/api/v1/compliance/aml/checks",
		body:     `{"external_user_id":"user_42","full_name":"Jane Doe","date_of_birth":"1990-01-01","nationality":"ES","entity_type":"person"}`,
		header:   map[string]string{"Idempotency-Key": "idem-1"},
		status:   201,
		response: `{"check_id":"chk_0b5e","session_id":"a1b2c3d4","external_user_id":"user_42","status":"Approved","result":{"features":["AML"],"results":[{"feature":"AML","status":"Approved","total_hits":0,"entity_type":"person"}],"warnings":[]},"created_at":"2026-09-25T09:00:00.000Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*AMLCheck)
			eq(t, "check", r.CheckID, "chk_0b5e")
			eq(t, "session", *r.SessionID, "a1b2c3d4")
			eq(t, "entity", r.Result.Results[0].EntityType, "person")
		},
	},
	{
		name: "Screening.CreateWalletScreening",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Screening.CreateWalletScreening(ctx, &WalletScreeningRequest{Address: "0x0000", Chain: "ETH", IdempotencyKey: "idem-2"})
		},
		method:   "POST",
		path:     "/api/v1/compliance/wallet-screenings",
		body:     `{"address":"0x0000","chain":"ETH"}`,
		header:   map[string]string{"Idempotency-Key": "idem-2"},
		status:   201,
		response: `{"check_id":"chk_7c1d","address":"0x0000","chain":"ETH","result":{"severity":"LOW","risk_score":12,"sanctions_hit":false,"pep_counterparty":false,"dominant_risk_category":"exchange","risk_factors":[{"category":"exchange","direction":"outgoing","exposure_type":"direct","percentage":40,"is_high_risk":false}]},"screened_at":"2026-09-25T09:00:00.000Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*WalletScreening)
			eq(t, "severity", r.Result.Severity, "LOW")
			eq(t, "score", *r.Result.RiskScore, int64(12))
			eq(t, "pep", *r.Result.PEPCounterparty, false)
			eq(t, "pct", *r.Result.RiskFactors[0].Percentage, 40.0)
		},
	},
	{
		name: "Screening.GetWalletScreening",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Screening.GetWalletScreening(ctx, "chk_7c1d")
		},
		method:   "GET",
		path:     "/api/v1/compliance/wallet-screenings/chk_7c1d",
		response: `{"check_id":"chk_7c1d","address":"0x0000","chain":"ETH","result":{"severity":"LOW","sanctions_hit":false,"risk_factors":[]},"screened_at":"2026-09-25T09:00:00.000Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*WalletScreening)
			if r.Result.RiskScore != nil {
				t.Error("expected nil risk_score")
			}
			eq(t, "factors", r.Result.RiskFactors, []RiskFactor{})
		},
	},
	{
		name: "Monitoring.Enable",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Monitoring.Enable(ctx, &MonitoringRequest{EntityKind: EntityKindUser, ExternalUserID: "user_42"})
		},
		method:   "POST",
		path:     "/api/v1/compliance/monitoring",
		body:     `{"entity_kind":"user","external_user_id":"user_42"}`,
		status:   201,
		response: `{"entity_kind":"user","external_user_id":"user_42","status":"active","enabled_at":"2026-09-25T09:00:00.000Z","next_renewal_at":"2027-09-25T09:00:00.000Z","cancelled_at":null,"created_at":"2026-09-25T09:00:00.000Z","updated_at":"2026-09-25T09:00:00.000Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*MonitoringSubscription)
			eq(t, "status", r.Status, "active")
			eq(t, "renewal", r.NextRenewalAt, "2027-09-25T09:00:00.000Z")
			if r.CancelledAt != nil {
				t.Error("expected nil cancelled_at")
			}
		},
	},
	{
		name: "Monitoring.Disable",
		call: func(ctx context.Context, c *Client) (any, error) {
			return nil, c.Compliance.Monitoring.Disable(ctx, EntityKindBusiness, "acme_1")
		},
		method: "DELETE",
		path:   "/api/v1/compliance/monitoring",
		query:  q("entity_kind", "business", "external_user_id", "acme_1"),
		status: 204,
	},
	{
		name: "Monitoring.List",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Monitoring.List(ctx, &ListMonitoringParams{Status: "active", Limit: Int64(10), Cursor: "m1"})
		},
		method:   "GET",
		path:     "/api/v1/compliance/monitoring",
		query:    q("status", "active", "limit", "10", "cursor", "m1"),
		response: `{"subscriptions":[{"entity_kind":"user","external_user_id":"user_42","status":"active","enabled_at":"a","next_renewal_at":"b","cancelled_at":null,"created_at":"c","updated_at":"d"}],"next_cursor":null}`,
		check: func(t *testing.T, got any) {
			r := got.(*MonitoringList)
			eq(t, "next", r.NextCursor, "")
			eq(t, "len", len(r.Subscriptions), 1)
		},
	},
	{
		name: "Compliance.Webhooks.Register",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Webhooks.Register(ctx, "https://example.com/webhooks/compliance")
		},
		method:   "POST",
		path:     "/api/v1/compliance/webhooks/register",
		body:     `{"url":"https://example.com/webhooks/compliance"}`,
		response: `{"url":"https://example.com/webhooks/compliance","secret":"abc123"}`,
		check: func(t *testing.T, got any) {
			eq(t, "secret", got.(*WebhookRegistration).Secret, "abc123")
		},
	},
	{
		name: "Compliance.Webhooks.Status",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Compliance.Webhooks.Status(ctx)
		},
		method:   "GET",
		path:     "/api/v1/compliance/webhooks/status",
		response: `{"registered":false,"url":null,"registered_at":null}`,
		check: func(t *testing.T, got any) {
			r := got.(*WebhookStatus)
			eq(t, "registered", r.Registered, false)
			if r.URL != nil || r.RegisteredAt != nil {
				t.Error("expected nil url and registered_at")
			}
		},
	},
}

// GetDocument and DownloadReport return raw bytes rather than JSON, so they
// are exercised outside the JSON table.
func TestComplianceBinaryEndpoints(t *testing.T) {
	t.Run("Sessions.GetDocument", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{body: "\xff\xd8jpegbytes", contentType: "image/jpeg"})
		f, err := srv.client().Compliance.Sessions.GetDocument(ctxBG(), "a1b2c3d4", DocumentFrontImage, &GetDocumentParams{NodeID: "feature_t_ocr"})
		if err != nil {
			t.Fatal(err)
		}
		defer f.Body.Close()
		b, _ := io.ReadAll(f.Body)
		eq(t, "bytes", string(b), "\xff\xd8jpegbytes")
		eq(t, "content type", f.ContentType, "image/jpeg")

		r := srv.recorded()[0]
		eq(t, "method", r.Method, "GET")
		eq(t, "path", r.Path, "/api/v1/compliance/sessions/a1b2c3d4/documents/front_image")
		eq(t, "query", r.Query.Get("node_id"), "feature_t_ocr")
		eq(t, "accept", r.Header.Get("Accept"), "*/*")
		eq(t, "api key", r.Header.Get("x-api-key"), testAPIKey)
	})

	t.Run("Sessions.DownloadReport", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{body: "%PDF-1.7", contentType: "application/pdf", header: map[string]string{
			"Content-Disposition": `attachment; filename="verification-report-a1b2c3d4.pdf"`,
		}})
		f, err := srv.client().Compliance.Sessions.DownloadReport(ctxBG(), "a1b2c3d4")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Body.Close()
		b, _ := io.ReadAll(f.Body)
		eq(t, "bytes", string(b), "%PDF-1.7")
		eq(t, "filename", f.Filename, "verification-report-a1b2c3d4.pdf")
		eq(t, "content type", f.ContentType, "application/pdf")

		r := srv.recorded()[0]
		eq(t, "method", r.Method, "GET")
		eq(t, "path", r.Path, "/api/v1/compliance/sessions/a1b2c3d4/report")
	})

	t.Run("binary endpoint JSON error maps to APIError", func(t *testing.T) {
		srv := newMockServer(t, mockResponse{status: 410, body: `{"error":{"code":"report_expired","message":"Report expired","requestId":"req_1"}}`})
		_, err := srv.client().Compliance.Sessions.DownloadReport(ctxBG(), "a1b2c3d4")
		var apiErr *APIError
		if !errorsAs(err, &apiErr) {
			t.Fatalf("expected *APIError, got %v", err)
		}
		eq(t, "status", apiErr.StatusCode, http.StatusGone)
		eq(t, "code", apiErr.Code, "report_expired")
	})
}

func TestScreeningIdempotencyReplay(t *testing.T) {
	srv := newMockServer(t, mockResponse{
		status: 200,
		header: map[string]string{"Idempotent-Replay": "true", "X-Request-ID": "req_replay"},
		body:   `{"check_id":"chk_1","session_id":null,"external_user_id":"u","status":"Approved","result":{"features":[],"results":[],"warnings":[]},"created_at":"x"}`,
	})
	var meta ResponseMetadata
	got, err := srv.client().Compliance.Screening.AMLCheck(ctxBG(), &AMLCheckRequest{ExternalUserID: "u", FullName: "N", IdempotencyKey: "k1"}, WithResponseMetadata(&meta))
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "check", got.CheckID, "chk_1")
	if got.SessionID != nil {
		t.Error("expected nil session_id")
	}
	eq(t, "replay", meta.IdempotentReplay, true)
	eq(t, "status", meta.StatusCode, 200)
	eq(t, "request id", meta.RequestID, "req_replay")
}

func TestScreeningWithoutIdempotencyKeySendsNoHeader(t *testing.T) {
	srv := newMockServer(t, mockResponse{status: 201, body: `{"check_id":"c","address":"a","chain":"ETH","result":{"severity":"LOW","sanctions_hit":false,"risk_factors":[]},"screened_at":"x"}`})
	if _, err := srv.client().Compliance.Screening.CreateWalletScreening(ctxBG(), &WalletScreeningRequest{Address: "a", Chain: "ETH"}); err != nil {
		t.Fatal(err)
	}
	if h := srv.recorded()[0].Header.Get("Idempotency-Key"); h != "" {
		t.Errorf("Idempotency-Key = %q, want none", h)
	}
}

func TestSessionsListAutoPaging(t *testing.T) {
	srv := newMockServer(t,
		mockResponse{body: `{"sessions":[{"session_id":"s1"},{"session_id":"s2"}],"next_cursor":"c2"}`},
		mockResponse{body: `{"sessions":[{"session_id":"s3"}],"next_cursor":null}`},
	)
	it := srv.client().Compliance.Sessions.ListAutoPaging(ctxBG(), &ListSessionsParams{Status: "Approved", Limit: Int64(2)})
	var ids []string
	for it.Next() {
		ids = append(ids, it.Current().SessionID)
	}
	if err := it.Err(); err != nil {
		t.Fatal(err)
	}
	eq(t, "ids", ids, []string{"s1", "s2", "s3"})

	reqs := srv.recorded()
	eq(t, "requests", len(reqs), 2)
	eq(t, "first query", reqs[0].Query, q("status", "Approved", "limit", "2"))
	eq(t, "second query", reqs[1].Query, q("status", "Approved", "limit", "2", "cursor", "c2"))
}

func TestMonitoringListAutoPaging(t *testing.T) {
	srv := newMockServer(t,
		mockResponse{body: `{"subscriptions":[{"external_user_id":"u1"}],"next_cursor":"m2"}`},
		mockResponse{body: `{"subscriptions":[],"next_cursor":"m3"}`},
		mockResponse{body: `{"subscriptions":[{"external_user_id":"u2"}],"next_cursor":null}`},
	)
	it := srv.client().Compliance.Monitoring.ListAutoPaging(ctxBG(), &ListMonitoringParams{Cursor: "m1"})
	var ids []string
	for it.Next() {
		ids = append(ids, it.Current().ExternalUserID)
	}
	if err := it.Err(); err != nil {
		t.Fatal(err)
	}
	eq(t, "ids", ids, []string{"u1", "u2"})
	reqs := srv.recorded()
	eq(t, "requests", len(reqs), 3)
	eq(t, "start cursor", reqs[0].Query.Get("cursor"), "m1")
	eq(t, "empty page followed", reqs[2].Query.Get("cursor"), "m3")
}

func TestAutoPagingStopsOnError(t *testing.T) {
	srv := newMockServer(t,
		mockResponse{body: `{"sessions":[{"session_id":"s1"}],"next_cursor":"c2"}`},
		mockResponse{status: 400, body: `{"error":{"code":"invalid_request","message":"bad cursor","requestId":"r"}}`},
	)
	it := srv.client().Compliance.Sessions.ListAutoPaging(ctxBG(), nil)
	n := 0
	for it.Next() {
		n++
	}
	eq(t, "items before error", n, 1)
	var apiErr *APIError
	if !errorsAs(it.Err(), &apiErr) || apiErr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request APIError, got %v", it.Err())
	}
	if it.Next() {
		t.Error("Next must keep returning false after an error")
	}
}

func TestComplianceClientSideValidation(t *testing.T) {
	srv := newMockServer(t, mockResponse{body: `{}`})
	c := srv.client()
	ctx := ctxBG()
	checks := map[string]error{}
	_, checks["Sessions.Create"] = c.Compliance.Sessions.Create(ctx, nil)
	_, checks["Sessions.UpdateStatus"] = c.Compliance.Sessions.UpdateStatus(ctx, "s", nil)
	_, checks["AMLCheck"] = c.Compliance.Screening.AMLCheck(ctx, nil)
	_, checks["CreateWalletScreening"] = c.Compliance.Screening.CreateWalletScreening(ctx, nil)
	_, checks["Monitoring.Enable"] = c.Compliance.Monitoring.Enable(ctx, nil)
	checks["Monitoring.Disable"] = c.Compliance.Monitoring.Disable(ctx, "", "u")
	for name, err := range checks {
		if err == nil {
			t.Errorf("%s: expected a client-side validation error", name)
		}
	}
	if n := len(srv.recorded()); n != 0 {
		t.Errorf("validation failures must not reach the network; %d requests were sent", n)
	}
}
