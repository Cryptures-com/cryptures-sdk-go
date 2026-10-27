package cryptures

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func parseKycForm(t *testing.T, r recordedRequest) *multipart.Form {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "media type", mediaType, "multipart/form-data")
	if params["boundary"] == "" {
		t.Fatal("Content-Type has no multipart boundary")
	}
	form, err := multipart.NewReader(strings.NewReader(string(r.Body)), params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	return form
}

func formFile(t *testing.T, form *multipart.Form, field string) (filename, content string) {
	t.Helper()
	files := form.File[field]
	if len(files) != 1 {
		t.Fatalf("expected one %s part, got %d", field, len(files))
	}
	f, err := files[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return files[0].Filename, string(b)
}

func TestCardSubmitKyc(t *testing.T) {
	srv := newMockServer(t, mockResponse{status: 201, body: `{"status":"success","message":"KYC documents submitted.","data":{"card_id":"card_a1b2c3d4","document_front_uploaded":true,"document_back_uploaded":true,"fee_charged_usd":1,"fee_already_paid":false}}`})
	got, err := srv.client().Card.Cards.SubmitKyc(ctxBG(), "card_a1b2c3d4", &SubmitKycRequest{
		DocumentFront: CardKycDocument{Filename: "front.jpg", Content: strings.NewReader("FRONT-BYTES")},
		DocumentBack:  &CardKycDocument{Filename: "back.pdf", Content: strings.NewReader("BACK-BYTES")},
	})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "status", got.Status, "success")
	eq(t, "message", got.Message, "KYC documents submitted.")
	eq(t, "card_id", got.Data.CardID, "card_a1b2c3d4")
	eq(t, "front uploaded", got.Data.DocumentFrontUploaded, true)
	eq(t, "back uploaded", got.Data.DocumentBackUploaded, true)
	eq(t, "fee charged", got.Data.FeeChargedUSD, 1.0)
	eq(t, "fee already paid", got.Data.FeeAlreadyPaid, false)

	reqs := srv.recorded()
	eq(t, "request count", len(reqs), 1)
	r := reqs[0]
	eq(t, "method", r.Method, "POST")
	eq(t, "path", r.Path, "/api/v1/card/cards/card_a1b2c3d4/kyc")
	eq(t, "api key", r.Header.Get("x-api-key"), testAPIKey)
	form := parseKycForm(t, r)
	name, content := formFile(t, form, "documentFront")
	eq(t, "front filename", name, "front.jpg")
	eq(t, "front content", content, "FRONT-BYTES")
	name, content = formFile(t, form, "documentBack")
	eq(t, "back filename", name, "back.pdf")
	eq(t, "back content", content, "BACK-BYTES")
	eq(t, "file part count", len(form.File), 2)
	eq(t, "value part count", len(form.Value), 0)
}

func TestCardSubmitKycFrontOnlyFree(t *testing.T) {
	srv := newMockServer(t, mockResponse{status: 201, body: `{"status":"success","message":"ok","data":{"card_id":"card_x","document_front_uploaded":true,"document_back_uploaded":false,"fee_charged_usd":0,"fee_already_paid":true}}`})
	got, err := srv.client().Card.Cards.SubmitKyc(ctxBG(), "card_x", &SubmitKycRequest{
		DocumentFront: CardKycDocument{Content: strings.NewReader("F")},
	})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "back uploaded", got.Data.DocumentBackUploaded, false)
	eq(t, "fee charged", got.Data.FeeChargedUSD, 0.0)
	eq(t, "fee already paid", got.Data.FeeAlreadyPaid, true)

	form := parseKycForm(t, srv.recorded()[0])
	name, _ := formFile(t, form, "documentFront")
	eq(t, "default filename", name, "documentFront")
	if _, ok := form.File["documentBack"]; ok {
		t.Fatal("documentBack must be omitted when not provided")
	}
}

func TestCardSubmitKycError(t *testing.T) {
	srv := newMockServer(t, mockResponse{status: 409, body: `{"error":{"code":"kyc_not_supported","message":"This card does not accept KYC documents.","requestId":"req_1"}}`})
	_, err := srv.client().Card.Cards.SubmitKyc(ctxBG(), "card_a1b2c3d4", &SubmitKycRequest{
		DocumentFront: CardKycDocument{Filename: "f.png", Content: strings.NewReader("x")},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	eq(t, "status", apiErr.StatusCode, 409)
	eq(t, "code", apiErr.Code, "kyc_not_supported")
	eq(t, "request id", apiErr.RequestID, "req_1")
}

// SubmitKyc debits the project balance, so a 5xx must never be retried.
func TestCardSubmitKycNeverRetried(t *testing.T) {
	srv := newMockServer(t, mockResponse{status: 504, body: `{"error":{"code":"upstream_timeout","message":"timeout"}}`})
	_, err := srv.client().Card.Cards.SubmitKyc(ctxBG(), "card_a1b2c3d4", &SubmitKycRequest{
		DocumentFront: CardKycDocument{Content: strings.NewReader("x")},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	eq(t, "code", apiErr.Code, "upstream_timeout")
	eq(t, "request count", len(srv.recorded()), 1)
}

func TestCardSubmitKycValidation(t *testing.T) {
	srv := newMockServer(t, mockResponse{status: 201, body: `{}`})
	c := srv.client().Card.Cards
	if _, err := c.SubmitKyc(ctxBG(), "c", nil); err == nil {
		t.Error("expected error for nil request")
	}
	if _, err := c.SubmitKyc(ctxBG(), "c", &SubmitKycRequest{}); err == nil {
		t.Error("expected error for nil front content")
	}
	if _, err := c.SubmitKyc(ctxBG(), "c", &SubmitKycRequest{
		DocumentFront: CardKycDocument{Content: strings.NewReader("x")},
		DocumentBack:  &CardKycDocument{},
	}); err == nil {
		t.Error("expected error for nil back content")
	}
	eq(t, "request count", len(srv.recorded()), 0)
}

func TestCardKycTypes(t *testing.T) {
	var card Card
	raw := `{"card_id":"card_1","status":"active","kyc":{"supported":true,"fee_usd":1,"submitted":true,"submitted_at":"2026-08-29T10:15:00Z"}}`
	if err := json.Unmarshal([]byte(raw), &card); err != nil {
		t.Fatal(err)
	}
	eq(t, "supported", card.Kyc.Supported, true)
	if card.Kyc.FeeUSD == nil {
		t.Fatal("fee_usd should be set")
	}
	eq(t, "fee", *card.Kyc.FeeUSD, 1.0)
	eq(t, "submitted", card.Kyc.Submitted, true)
	if card.Kyc.SubmittedAt == nil {
		t.Fatal("submitted_at should be set")
	}
	eq(t, "submitted_at", *card.Kyc.SubmittedAt, "2026-08-29T10:15:00Z")

	var summary CardSummary
	if err := json.Unmarshal([]byte(`{"card_id":"card_2","kyc":{"supported":false,"fee_usd":null,"submitted":false,"submitted_at":null}}`), &summary); err != nil {
		t.Fatal(err)
	}
	eq(t, "supported", summary.Kyc.Supported, false)
	if summary.Kyc.FeeUSD != nil || summary.Kyc.SubmittedAt != nil {
		t.Fatal("null fee_usd and submitted_at must decode to nil")
	}

	var p CardProduct
	if err := json.Unmarshal([]byte(`{"product_code":"p","kyc_supported":true,"kyc_fee_usd":0}`), &p); err != nil {
		t.Fatal(err)
	}
	eq(t, "kyc_supported", p.KycSupported, true)
	eq(t, "kyc_fee_usd", p.KycFeeUSD, 0.0)
}
