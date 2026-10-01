package cryptures

// ComplianceService groups the compliance-domain operations: KYC/KYB
// verification sessions, standalone AML and wallet screening, ongoing
// monitoring, and webhooks.
type ComplianceService struct {
	// Sessions covers verification sessions, presets, captured documents,
	// manual decisions, and report PDFs.
	Sessions *ComplianceSessionsService
	// Screening covers standalone AML screening and blockchain wallet
	// screening.
	Screening *ComplianceScreeningService
	// Monitoring covers ongoing AML monitoring subscriptions.
	Monitoring *ComplianceMonitoringService
	// Webhooks covers compliance-domain webhook registration.
	Webhooks *ComplianceWebhooksService
}

// VerificationResult is one check outcome in a normalized verification
// result. Only the detail fields meaningful for the check (Feature) are set.
type VerificationResult struct {
	// Feature is the check this outcome is for, e.g. "ID_VERIFICATION",
	// "LIVENESS", "FACE_MATCH", "AML", "KYB_REGISTRY".
	Feature string `json:"feature"`
	// Status is this check's own outcome, e.g. "Approved".
	Status string `json:"status"`

	// ID checks.
	DocumentType string `json:"document_type,omitempty"`
	// Name is the holder's first and last name joined into one string.
	Name         string `json:"name,omitempty"`
	DateOfBirth  string `json:"date_of_birth,omitempty"`
	IssuingState string `json:"issuing_state,omitempty"`

	// Liveness/face checks.
	Method string   `json:"method,omitempty"`
	Score  *float64 `json:"score,omitempty"`

	// Screening checks.
	TotalHits  *int64 `json:"total_hits,omitempty"`
	EntityType string `json:"entity_type,omitempty"`

	// Company registry checks (KYB).
	CompanyName        string `json:"company_name,omitempty"`
	RegistrationNumber string `json:"registration_number,omitempty"`
	// CountryCode is ISO 3166-1 alpha-2.
	CountryCode string `json:"country_code,omitempty"`
	// Tier is "lite", "shareholders", or "ubo".
	Tier string `json:"tier,omitempty"`

	// Key-people checks (KYB): how many people were screened.
	PeopleCount *int64 `json:"people_count,omitempty"`
}

// VerificationWarning is a warning raised by a check.
type VerificationWarning struct {
	Feature          string `json:"feature"`
	ShortDescription string `json:"short_description"`
	// LongDescription is empty when the warning has no long form.
	LongDescription string `json:"long_description,omitempty"`
}
