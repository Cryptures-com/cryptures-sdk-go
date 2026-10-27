package cryptures

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// CardKycDocument is one identity document to upload with SubmitKyc.
type CardKycDocument struct {
	// Filename is the file name sent in the multipart part. It is only a
	// label: the API checks the file type from the content, not the name.
	// Defaults to the form field name when empty.
	Filename string
	// Content is the file body, read fully into memory before sending. It
	// must be a PNG, JPG/JPEG or PDF of at most 2 MB; the API validates this
	// and reports violations as an *APIError.
	Content io.Reader
}

// SubmitKycRequest holds the identity documents for SubmitKyc.
type SubmitKycRequest struct {
	// DocumentFront is the front of the identity document. Required.
	DocumentFront CardKycDocument
	// DocumentBack is the back of the identity document (for example the
	// reverse of an ID card). Optional: leave nil to send only the front.
	DocumentBack *CardKycDocument
}

// CardKycSubmission is returned by SubmitKyc, in the API's
// {status, message, data} envelope.
type CardKycSubmission struct {
	Status  string                `json:"status"`
	Message string                `json:"message,omitempty"`
	Data    CardKycSubmissionData `json:"data"`
}

// CardKycSubmissionData is the outcome of a KYC submission.
type CardKycSubmissionData struct {
	CardID string `json:"card_id"`
	// DocumentFrontUploaded is always true on success.
	DocumentFrontUploaded bool `json:"document_front_uploaded"`
	// DocumentBackUploaded reports whether a back document was submitted.
	DocumentBackUploaded bool `json:"document_back_uploaded"`
	// FeeChargedUSD is the USD charged by this call. It is 0 when the fee
	// was already paid for this card (FeeAlreadyPaid), and also when the
	// product's KYC fee (CardProduct.KycFeeUSD) is 0, in which case the
	// upload is free.
	FeeChargedUSD float64 `json:"fee_charged_usd"`
	// FeeAlreadyPaid is true when an earlier submission already paid this
	// card's KYC fee, so this call did not charge it.
	FeeAlreadyPaid bool `json:"fee_already_paid"`
}

// SubmitKyc submits the cardholder's identity documents for an issued card as
// a multipart/form-data request (fields documentFront and documentBack). Only
// cards whose product supports KYC accept documents: check
// Card.Kyc.Supported or CardProduct.KycSupported first.
//
// A KYC fee (CardProduct.KycFeeUSD, set per product) is charged once per card
// from the project's shared USD balance before the documents are submitted.
// The fee can be 0, in which case the upload is free and
// CardKycSubmissionData.FeeChargedUSD is 0. Submitting again for the same
// card is not charged again (FeeAlreadyPaid). If the issuer refuses the
// documents (kyc_rejected) the fee is refunded automatically.
//
// This call is billed and is never retried automatically. If it fails with
// upstream_error (502) or upstream_timeout (504) the outcome is unknown and
// the fee is held for review rather than refunded: check Cards.Get
// (Card.Kyc.Submitted) and Balance.ListTransactions before submitting again.
//
// Failures are returned as an *APIError; Code is one of:
//
//   - invalid_request (400): the body is not valid multipart/form-data.
//   - missing_document_front (400): no documentFront file.
//   - too_many_files (400): a file field other than documentFront or
//     documentBack, or one sent twice.
//   - invalid_file_type (400): a file is not a PNG, JPG/JPEG or PDF.
//   - kyc_rejected (400): the card issuer did not accept the documents; the
//     fee is refunded, so fix the files and submit again.
//   - insufficient_balance (402): the project balance cannot cover the fee;
//     nothing is charged or submitted.
//   - card_not_found (404): the card does not exist or is not in this project.
//   - kyc_not_supported (409): the card's product does not accept KYC.
//   - kyc_email_unknown (409): the card was not issued through the API, so
//     its cardholder email is unknown and it cannot take KYC documents.
//   - kyc_in_progress (409): an earlier submission for this card is still
//     being processed.
//   - file_too_large (413): a file is over 2 MB.
//   - payload_too_large (413): the whole request body is too large.
//   - rate_limited (429): more than 5 submissions for this card in a day, or
//     the per-minute route limit was exceeded. A fee charged for a call
//     refused by the daily limit is not kept.
//   - upstream_error (502) and upstream_timeout (504): see above.
//
// On success a card.kyc_submitted webhook event is sent to the project's
// registered card webhook URL, if any.
func (s *CardCardsService) SubmitKyc(ctx context.Context, cardID string, req *SubmitKycRequest, opts ...RequestOption) (*CardKycSubmission, error) {
	if req == nil {
		return nil, errors.New("cryptures: Cards.SubmitKyc: nil request")
	}
	if req.DocumentFront.Content == nil {
		return nil, errors.New("cryptures: Cards.SubmitKyc: nil DocumentFront content")
	}
	if req.DocumentBack != nil && req.DocumentBack.Content == nil {
		return nil, errors.New("cryptures: Cards.SubmitKyc: nil DocumentBack content")
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := writeKycPart(mw, "documentFront", req.DocumentFront); err != nil {
		return nil, err
	}
	if req.DocumentBack != nil {
		if err := writeKycPart(mw, "documentBack", *req.DocumentBack); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("cryptures: Cards.SubmitKyc: %w", err)
	}

	// Not retryable: the call debits the project balance.
	return doJSON[CardKycSubmission](ctx, s.client, &requestSpec{
		method:      http.MethodPost,
		path:        pathf("/api/v1/card/cards/%s/kyc", cardID),
		route:       "/api/v1/card/cards/{card_id}/kyc",
		rawBody:     buf.Bytes(),
		contentType: mw.FormDataContentType(),
	}, opts)
}

func writeKycPart(mw *multipart.Writer, field string, doc CardKycDocument) error {
	name := doc.Filename
	if name == "" {
		name = field
	}
	part, err := mw.CreateFormFile(field, name)
	if err != nil {
		return fmt.Errorf("cryptures: Cards.SubmitKyc: %w", err)
	}
	if _, err := io.Copy(part, doc.Content); err != nil {
		return fmt.Errorf("cryptures: Cards.SubmitKyc: reading %s: %w", field, err)
	}
	return nil
}
