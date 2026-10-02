package cryptures

import (
	"context"
	"net/http"
)

// BlockchainNFTService lists NFT collections and token owners. Available for
// ETH, SOL, BNB, MATIC, AVAX, ARB, OP, BASE, and CELO. To list the NFTs one
// wallet holds, use Blockchain.Data.GetPortfolio with TokenTypes "nft".
type BlockchainNFTService struct {
	client *Client
}

// CollectionParams pages through ListCollection.
type CollectionParams struct {
	// ExcludeMetadata excludes per-token metadata. Defaults to false.
	ExcludeMetadata *bool
	// PageSize is the number of items per page.
	PageSize *int64
	// Offset is the pagination offset.
	Offset *int64
}

// NFT is one token in a collection.
type NFT struct {
	Chain        string `json:"chain,omitempty"`
	TokenID      string `json:"tokenId,omitempty"`
	TokenAddress string `json:"tokenAddress,omitempty"`
	TokenType    string `json:"tokenType,omitempty"`
	MetadataURI  string `json:"metadataURI,omitempty"`
	// Metadata is nil when excluded.
	Metadata *NFTMetadata `json:"metadata,omitempty"`
}

// NFTMetadata is an NFT's metadata.
type NFTMetadata struct {
	Identifier          string  `json:"identifier,omitempty"`
	Collection          string  `json:"collection,omitempty"`
	Contract            string  `json:"contract,omitempty"`
	TokenStandard       string  `json:"token_standard,omitempty"`
	Name                string  `json:"name,omitempty"`
	Description         string  `json:"description,omitempty"`
	ImageURL            string  `json:"image_url,omitempty"`
	DisplayImageURL     string  `json:"display_image_url,omitempty"`
	DisplayAnimationURL *string `json:"display_animation_url,omitempty"`
	MetadataURL         string  `json:"metadata_url,omitempty"`
}

// ListCollection returns the NFTs belonging to the collection at
// collectionAddress. params may be nil.
func (s *BlockchainNFTService) ListCollection(ctx context.Context, chain, collectionAddress string, params *CollectionParams, opts ...RequestOption) ([]NFT, error) {
	q := newQuery()
	if params != nil {
		q.boolPtr("excludeMetadata", params.ExcludeMetadata).
			intPtr("pageSize", params.PageSize).
			intPtr("offset", params.Offset)
	}
	out, err := doJSON[[]NFT](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/nft/collection/%s/%s", chain, collectionAddress),
		route:     "/api/v1/blockchain/data/nft/collection/{chain}/{collectionAddress}",
		query:     q.values(),
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// OwnersParams pages through GetOwners.
type OwnersParams struct {
	// PageSize is the number of items per page.
	PageSize *int64
	// Offset is the pagination offset.
	Offset *int64
}

// GetOwners returns the current owner address(es) of one token: a single
// address for an ERC-721 token, possibly several for an ERC-1155 token.
// params may be nil.
func (s *BlockchainNFTService) GetOwners(ctx context.Context, chain, tokenAddress, tokenID string, params *OwnersParams, opts ...RequestOption) ([]string, error) {
	q := newQuery()
	if params != nil {
		q.intPtr("pageSize", params.PageSize).intPtr("offset", params.Offset)
	}
	out, err := doJSON[[]string](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/nft/owner/%s/%s/%s", chain, tokenAddress, tokenID),
		route:     "/api/v1/blockchain/data/nft/owner/{chain}/{tokenAddress}/{tokenId}",
		query:     q.values(),
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}
