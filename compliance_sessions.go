package cryptures

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// ComplianceSessionsService manages KYC/KYB verification sessions.
type ComplianceSessionsService struct {
	client *Client
}

// CreateSessionRequest starts a verification session.
type CreateSessionRequest struct {
	// PresetID selects the verification workflow; see ListPresets. A preset
	// whose Kind is "kyb" verifies a company instead of a person.
	PresetID string `json:"preset_id"`
	// ExternalUserID is your own identifier for the end user: at most 200
	// characters, no colon, no control characters.
	ExternalUserID string `json:"external_user_id"`
	// Callback is a public https URL (at most 2048 characters) the end user
	// returns to after verifying.
	Callback string `json:"callback,omitempty"`
	// Language is the hosted page's language, e.g. "en", "es", "pt-BR", "zh-CN".
	Language string `json:"language,omitempty"`
	// Metadata is a free-form JSON object kept with the session (at most 2000
	// characters when serialized).
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// ExpectedDetails are details you already know, to check against the document.
	ExpectedDetails *ExpectedDetails `json:"expected_details,omitempty"`
	// ContactDetails are the end user's contact details.
	ContactDetails *ContactDetails `json:"contact_details,omitempty"`
}

// ExpectedDetails are known details checked against the identity document.
type ExpectedDetails struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	// DateOfBirth is YYYY-MM-DD.
	DateOfBirth string `json:"date_of_birth,omitempty"`
}

// ContactDetails are an end user's contact details.
type ContactDetails struct {
	Email string `json:"email,omitempty"`
}

// CreatedSession is a newly created verification session.
type CreatedSession struct {
	SessionID string `json:"session_id"`
	// URL is the single-use hosted verification page to redirect the end user to.
	URL string `json:"url"`
	// Status is the session's status at creation, typically "Not Started".
	Status string `json:"status"`
}

// Create starts a new verification session for one of your end users and
// returns the hosted verification link. Not retried automatically.
func (s *ComplianceSessionsService) Create(ctx context.Context, req *CreateSessionRequest, opts ...RequestOption) (*CreatedSession, error) {
	if req == nil {
		return nil, errors.New("cryptures: Sessions.Create: nil request")
	}
	return doJSON[CreatedSession](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   "/api/v1/compliance/sessions/create",
		body:   req,
	}, opts)
}

// Session is a verification session and its normalized result.
type Session struct {
	SessionID string `json:"session_id"`
	// Status is one of "Not Started", "In Progress", "Awaiting User",
	// "Resubmitted", "Approved", "Declined", "In Review", "Abandoned",
	// "Expired", "Kyc Expired", "Unknown". The list can grow; handle unknown
	// values gracefully.
	Status string `json:"status"`
	// ExternalUserID is empty if it could not be recovered.
	ExternalUserID string `json:"external_user_id,omitempty"`
	// Kind is "kyc", "kyb", or "aml_standalone".
	Kind string `json:"kind"`
	// Features lists which checks ran. Empty until there is a result.
	Features []string `json:"features"`
	// Results holds one entry per check outcome. Empty until there is a result.
	Results []VerificationResult `json:"results"`
	// Warnings are flattened across every check.
	Warnings []VerificationWarning `json:"warnings"`
	// DocumentURLs maps a document field name to a URL on the API's documents
	// endpoint (see GetDocument). Nil when nothing was captured.
	DocumentURLs map[string]string `json:"document_urls,omitempty"`
}

// Get returns a session's current status and, once available, its result.
func (s *ComplianceSessionsService) Get(ctx context.Context, sessionID string, opts ...RequestOption) (*Session, error) {
	return doJSON[Session](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/compliance/sessions/%s", sessionID),
		retryable: true,
	}, opts)
}

// ListSessionsParams filters and pages List.
type ListSessionsParams struct {
	// Status only returns sessions in this status, e.g. "Approved".
	Status string
	// ExternalUserID only returns sessions for this end user.
	ExternalUserID string
	// CreatedAfter only returns sessions created at or after this ISO-8601
	// date or timestamp.
	CreatedAfter string
	// CreatedBefore only returns sessions created before this ISO-8601 date
	// or timestamp.
	CreatedBefore string
	// Limit is the page size, 1-200. Defaults to 50.
	Limit *int64
	// Cursor is the NextCursor from the previous page.
	Cursor string
}

// SessionList is one page of sessions.
type SessionList struct {
	Sessions []SessionSummary `json:"sessions"`
	// NextCursor is passed as Cursor to fetch the next page. It is empty on
	// the last page.
	NextCursor string `json:"next_cursor"`
}

// SessionSummary is one session in a SessionList.
type SessionSummary struct {
	SessionID      string `json:"session_id"`
	ExternalUserID string `json:"external_user_id"`
	PresetID       string `json:"preset_id"`
	// Status includes "Deleted" for deleted sessions.
	Status string `json:"status"`
	// Kind is "kyc", "kyb", or "aml_standalone".
	Kind      string `json:"kind"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (p *ListSessionsParams) query(cursor string) *queryBuilder {
	q := newQuery()
	if p != nil {
		q.str("status", p.Status).
			str("external_user_id", p.ExternalUserID).
			str("created_after", p.CreatedAfter).
			str("created_before", p.CreatedBefore).
			intPtr("limit", p.Limit)
	}
	return q.str("cursor", cursor)
}

// List returns one page of the project's sessions, newest first. params may
// be nil. Use ListAutoPaging to walk every page.
func (s *ComplianceSessionsService) List(ctx context.Context, params *ListSessionsParams, opts ...RequestOption) (*SessionList, error) {
	cursor := ""
	if params != nil {
		cursor = params.Cursor
	}
	return doJSON[SessionList](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/compliance/sessions",
		query:     params.query(cursor).values(),
		retryable: true,
	}, opts)
}

// ListAutoPaging returns an iterator over every session matching params,
// starting from params.Cursor (if any) and fetching pages as needed.
func (s *ComplianceSessionsService) ListAutoPaging(ctx context.Context, params *ListSessionsParams, opts ...RequestOption) *Iter[SessionSummary] {
	start := ""
	if params != nil {
		start = params.Cursor
	}
	first := true
	return newIter(ctx, func(ctx context.Context, cursor string) ([]SessionSummary, string, error) {
		if first {
			cursor, first = start, false
		}
		page, err := doJSON[SessionList](ctx, s.client, &requestSpec{
			method:    http.MethodGet,
			path:      "/api/v1/compliance/sessions",
			query:     params.query(cursor).values(),
			retryable: true,
		}, opts)
		if err != nil {
			return nil, "", err
		}
		return page.Sessions, page.NextCursor, nil
	})
}

// Preset is a verification workflow.
type Preset struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Kind is "kyc" (verifies a person) or "kyb" (verifies a company).
	Kind string `json:"kind"`
}

// PresetList is the standard set of active presets.
type PresetList struct {
	Presets []Preset `json:"presets"`
}

// ListPresets returns the standard active verification presets; pass a
// preset's ID as CreateSessionRequest.PresetID.
func (s *ComplianceSessionsService) ListPresets(ctx context.Context, opts ...RequestOption) (*PresetList, error) {
	return doJSON[PresetList](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/compliance/presets",
		retryable: true,
	}, opts)
}

// Document fields accepted by GetDocument.
const (
	DocumentPortraitImage         = "portrait_image"
	DocumentFrontImage            = "front_image"
	DocumentBackImage             = "back_image"
	DocumentFrontVideo            = "front_video"
	DocumentBackVideo             = "back_video"
	DocumentFullFrontImage        = "full_front_image"
	DocumentFullBackImage         = "full_back_image"
	DocumentFrontImageCameraFront = "front_image_camera_front"
	DocumentBackImageCameraFront  = "back_image_camera_front"
)

// GetDocumentParams configures GetDocument.
type GetDocumentParams struct {
	// NodeID selects which result item to read when a session has more than
	// one for the same check. Leave empty for the first match.
	NodeID string
}

// GetDocument returns the bytes of one document photo, selfie, or video
// captured during a verification (image/jpeg or video/mp4). field is one of
// the Document* constants. params may be nil. The caller must close the
// returned File's Body.
func (s *ComplianceSessionsService) GetDocument(ctx context.Context, sessionID, field string, params *GetDocumentParams, opts ...RequestOption) (*File, error) {
	q := newQuery()
	if params != nil {
		q.str("node_id", params.NodeID)
	}
	return doFile(ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/compliance/sessions/%s/documents/%s", sessionID, field),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// UpdateSessionStatusRequest records a manual decision.
type UpdateSessionStatusRequest struct {
	// Status is "Approved" or "Declined".
	Status string `json:"status"`
	// Comment is an optional note (at most 500 printable characters).
	Comment string `json:"comment,omitempty"`
}

// SessionStatusUpdate confirms a manual decision.
type SessionStatusUpdate struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

// UpdateStatus manually approves or declines a session, typically one that
// is "In Review". Repeating the same status is safe.
func (s *ComplianceSessionsService) UpdateStatus(ctx context.Context, sessionID string, req *UpdateSessionStatusRequest, opts ...RequestOption) (*SessionStatusUpdate, error) {
	if req == nil {
		return nil, errors.New("cryptures: Sessions.UpdateStatus: nil request")
	}
	return doJSON[SessionStatusUpdate](ctx, s.client, &requestSpec{
		method:    http.MethodPatch,
		path:      pathf("/api/v1/compliance/sessions/%s/status", sessionID),
		body:      req,
		retryable: true,
	}, opts)
}

// DeleteSessionParams configures Delete.
type DeleteSessionParams struct {
	// PrivacyErasure also requests erasure of the end user's personal data.
	// It is honored only on the first successful delete of a session.
	PrivacyErasure *bool
}

// Delete deletes a session. Its stored decision and report PDFs are removed;
// the session stays listed with status "Deleted". Deleting is safe to
// repeat. params may be nil.
func (s *ComplianceSessionsService) Delete(ctx context.Context, sessionID string, params *DeleteSessionParams, opts ...RequestOption) error {
	q := newQuery()
	if params != nil && params.PrivacyErasure != nil {
		q.str("privacy_erasure", strconv.FormatBool(*params.PrivacyErasure))
	}
	return doNoContent(ctx, s.client, &requestSpec{
		method:    http.MethodDelete,
		path:      pathf("/api/v1/compliance/sessions/%s", sessionID),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// Report describes a generated verification report PDF.
type Report struct {
	ReportID    string `json:"report_id"`
	SessionID   string `json:"session_id"`
	GeneratedAt string `json:"generated_at"`
	// ExpiresAt is seven days after GeneratedAt.
	ExpiresAt string `json:"expires_at"`
	SizeBytes int64  `json:"size_bytes"`
	// DownloadURL is fetched with DownloadReport.
	DownloadURL string `json:"download_url"`
}

// CreateReport generates a PDF report for a session from its stored result,
// superseding any previous report. Reports are kept for 7 days.
func (s *ComplianceSessionsService) CreateReport(ctx context.Context, sessionID string, opts ...RequestOption) (*Report, error) {
	return doJSON[Report](ctx, s.client, &requestSpec{
		method:    http.MethodPost,
		path:      pathf("/api/v1/compliance/sessions/%s/report", sessionID),
		retryable: true, // free, and a newer report simply supersedes the old one
	}, opts)
}

// DownloadReport returns the session's most recently generated report PDF.
// A 410 report_expired *APIError means it is past its 7-day window; call
// CreateReport again. The caller must close the returned File's Body.
func (s *ComplianceSessionsService) DownloadReport(ctx context.Context, sessionID string, opts ...RequestOption) (*File, error) {
	return doFile(ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/compliance/sessions/%s/report", sessionID),
		retryable: true,
	}, opts)
}
