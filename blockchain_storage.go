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

// BlockchainStorageService uploads files to IPFS.
type BlockchainStorageService struct {
	client *Client
}

// IPFSUpload is the result of an IPFS upload.
type IPFSUpload struct {
	// IPFSHash is the uploaded file's IPFS content hash (CID).
	IPFSHash string `json:"ipfsHash"`
}

// UploadIPFS uploads the contents of file to IPFS as a multipart/form-data
// "file" field named filename, and returns its content hash (CID). The file
// is read fully into memory before sending.
func (s *BlockchainStorageService) UploadIPFS(ctx context.Context, filename string, file io.Reader, opts ...RequestOption) (*IPFSUpload, error) {
	if file == nil {
		return nil, errors.New("cryptures: UploadIPFS: nil file")
	}
	if filename == "" {
		filename = "file"
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("cryptures: UploadIPFS: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("cryptures: UploadIPFS: reading file: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("cryptures: UploadIPFS: %w", err)
	}
	return doJSON[IPFSUpload](ctx, s.client, &requestSpec{
		method:      http.MethodPost,
		path:        "/api/v1/blockchain/storage/ipfs",
		rawBody:     buf.Bytes(),
		contentType: mw.FormDataContentType(),
	}, opts)
}
