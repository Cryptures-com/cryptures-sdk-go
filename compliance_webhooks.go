package cryptures

import "context"

// ComplianceWebhooksService manages the project's compliance-domain webhook.
type ComplianceWebhooksService struct {
	client *Client
}

// Register registers (or replaces) the project's https:// callback URL for
// verification events (status.updated, data.updated). Deliveries are signed
// in the X-Verification-Signature header as t=<ts>,v1=<hmac>, where hmac is
// the lowercase hex HMAC-SHA256 of "<ts>.<raw body>" keyed with the returned
// secret. Only one URL is supported per project.
func (s *ComplianceWebhooksService) Register(ctx context.Context, url string, opts ...RequestOption) (*WebhookRegistration, error) {
	return registerWebhook(ctx, s.client, "/api/v1/compliance/webhooks/register", url, opts)
}

// Status reads back the project's current compliance-domain webhook
// registration.
func (s *ComplianceWebhooksService) Status(ctx context.Context, opts ...RequestOption) (*WebhookStatus, error) {
	return webhookStatus(ctx, s.client, "/api/v1/compliance/webhooks/status", opts)
}
