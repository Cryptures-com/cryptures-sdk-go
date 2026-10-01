package cryptures

import (
	"context"
	"errors"
	"net/http"
)

// CardTagsService manages the project's card tags. To set the tags on one
// card, use Card.Cards.SetTags. An unknown tag id and another project's tag
// id both return a 403 forbidden_tag *APIError.
type CardTagsService struct {
	client *Client
}

// Tag is one of the project's card tags.
type Tag struct {
	TagID string `json:"tag_id"`
	Name  string `json:"name"`
	// Color is a 6-digit hex color with a leading "#", e.g. "#4F46E5".
	Color     string `json:"color"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// TagList is every tag the project has created, newest first.
type TagList struct {
	Data []Tag `json:"data"`
}

// List returns every tag the project has created, newest first.
func (s *CardTagsService) List(ctx context.Context, opts ...RequestOption) (*TagList, error) {
	return doJSON[TagList](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/card/tags",
		retryable: true,
	}, opts)
}

// CreateTagRequest creates a tag.
type CreateTagRequest struct {
	// Name is 1-50 characters, unique per project (case-insensitive).
	Name string `json:"name"`
	// Color is optional: "#" followed by exactly 6 hex digits. It is
	// auto-assigned from a palette when empty.
	Color string `json:"color,omitempty"`
}

// Create creates a new tag. A duplicate name returns a 409 tag_name_taken
// *APIError.
func (s *CardTagsService) Create(ctx context.Context, req *CreateTagRequest, opts ...RequestOption) (*Tag, error) {
	if req == nil {
		return nil, errors.New("cryptures: Tags.Create: nil request")
	}
	return doJSON[Tag](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   "/api/v1/card/tags",
		body:   req,
	}, opts)
}

// UpdateTagRequest renames and/or recolors a tag. Set at least one field.
type UpdateTagRequest struct {
	// Name is 1-50 characters, unique per project (case-insensitive).
	Name *string `json:"name,omitempty"`
	// Color is "#" followed by exactly 6 hex digits.
	Color *string `json:"color,omitempty"`
}

// Update renames or recolors a tag the project owns. The returned Tag has no
// CreatedAt.
func (s *CardTagsService) Update(ctx context.Context, tagID string, req *UpdateTagRequest, opts ...RequestOption) (*Tag, error) {
	if req == nil || (req.Name == nil && req.Color == nil) {
		return nil, errors.New("cryptures: Tags.Update: at least one of Name or Color is required")
	}
	return doJSON[Tag](ctx, s.client, &requestSpec{
		method:    http.MethodPatch,
		path:      pathf("/api/v1/card/tags/%s", tagID),
		body:      req,
		retryable: true, // setting the same values again is idempotent
	}, opts)
}

// TagDeleted confirms a tag deletion.
type TagDeleted struct {
	Deleted bool `json:"deleted"`
}

// Delete deletes a tag and removes it from every card it was attached to.
// The cards themselves are not affected.
func (s *CardTagsService) Delete(ctx context.Context, tagID string, opts ...RequestOption) (*TagDeleted, error) {
	return doJSON[TagDeleted](ctx, s.client, &requestSpec{
		method:    http.MethodDelete,
		path:      pathf("/api/v1/card/tags/%s", tagID),
		retryable: true,
	}, opts)
}
