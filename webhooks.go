package cryptures

import (
	"context"
	"net/http"
)

// WebhookRegistration is returned when a webhook URL is registered.
type WebhookRegistration struct {
	// URL is the registered URL, normalized.
	URL string `json:"url"`
	// Secret is a 64-character hex secret that signs every future delivery to
	// URL. It is returned only once: store it.
	Secret string `json:"secret"`
}

// WebhookStatus is the current webhook registration state. The signing
// secret is never included; re-register to rotate it.
type WebhookStatus struct {
	// Registered reports whether a webhook URL is registered.
	Registered bool `json:"registered"`
	// URL is the registered URL, or nil if none is registered.
	URL *string `json:"url"`
	// RegisteredAt is the RFC 3339 time of the latest (re-)registration, or
	// nil if none is registered.
	RegisteredAt *string `json:"registered_at"`
}

type registerWebhookRequest struct {
	URL string `json:"url"`
}

func registerWebhook(ctx context.Context, c *Client, path, url string, opts []RequestOption) (*WebhookRegistration, error) {
	return doJSON[WebhookRegistration](ctx, c, &requestSpec{
		method: http.MethodPost,
		path:   path,
		body:   registerWebhookRequest{URL: url},
		// Re-registering simply overwrites the URL and secret, and the
		// secret in the final response is the one in effect.
		retryable: true,
	}, opts)
}

func webhookStatus(ctx context.Context, c *Client, path string, opts []RequestOption) (*WebhookStatus, error) {
	return doJSON[WebhookStatus](ctx, c, &requestSpec{
		method:    http.MethodGet,
		path:      path,
		retryable: true,
	}, opts)
}

// CardWebhooksService manages the project's card-domain webhook.
type CardWebhooksService struct {
	client *Client
}

// Register registers (or replaces) the project's https:// callback URL for
// card-domain events (card.created, card.funded, card.withdrawn, and
// processor-originated card events). Deliveries are signed with the returned
// secret in the X-Card-Signature header. Only one URL is supported per
// project.
func (s *CardWebhooksService) Register(ctx context.Context, url string, opts ...RequestOption) (*WebhookRegistration, error) {
	return registerWebhook(ctx, s.client, "/api/v1/card/webhooks/register", url, opts)
}

// Status reads back the project's current card-domain webhook registration.
func (s *CardWebhooksService) Status(ctx context.Context, opts ...RequestOption) (*WebhookStatus, error) {
	return webhookStatus(ctx, s.client, "/api/v1/card/webhooks/status", opts)
}
