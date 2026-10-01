package cryptures

import (
	"context"
	"io"
	"mime"
)

// File is a binary response body, such as a verification document image or
// a report PDF. The caller must close Body.
type File struct {
	// ContentType is the response Content-Type, e.g. "image/jpeg",
	// "video/mp4", or "application/pdf".
	ContentType string
	// Filename is taken from the Content-Disposition header, when present.
	Filename string
	// ContentLength is the body length, or -1 when unknown.
	ContentLength int64
	// Body streams the file contents. Close it when done.
	Body io.ReadCloser
}

// doFile performs a request whose successful response is a binary body.
func doFile(ctx context.Context, c *Client, spec *requestSpec, opts []RequestOption) (*File, error) {
	spec.accept = "*/*"
	resp, err := c.send(ctx, spec, opts)
	if err != nil {
		return nil, err
	}
	f := &File{
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
		Body:          resp.Body,
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			f.Filename = params["filename"]
		}
	}
	return f, nil
}
