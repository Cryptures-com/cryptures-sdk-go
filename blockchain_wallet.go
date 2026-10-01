package cryptures

import (
	"context"
	"errors"
	"net/http"
	"strconv"
)

// BlockchainWalletService generates HD wallets and derives addresses and
// private keys. Responses from Generate and DerivePrivateKey contain real,
// usable secret material in plaintext: store it securely, it cannot be
// retrieved again from the API.
type BlockchainWalletService struct {
	client *Client
}

// GenerateWalletParams configures Generate.
type GenerateWalletParams struct {
	// Mnemonic, when set, makes the call deterministic: instead of a fresh
	// random wallet, the wallet for this existing mnemonic is returned
	// (recovering its xpub or equivalent).
	//
	// WARNING: the mnemonic is sent as a URL query parameter, which may be
	// recorded in access logs and by intermediaries. Prefer deriving locally
	// when you already hold the mnemonic.
	Mnemonic string
}

// Wallet is a generated wallet. Which fields are set depends on the chain:
//
//   - HD chains (BTC, ETH, BNB, MATIC, AVAX, TRON, LTC, DOGE, ARB, OP, BASE,
//     CELO, FTM, BCH, ADA, VET): Mnemonic and Xpub
//   - XRP, XLM, ALGO: Address and Secret
//   - SOL: Mnemonic, Address, and PrivateKey
//   - EGLD: Mnemonic only
type Wallet struct {
	// Mnemonic is the BIP-39 mnemonic phrase, in plaintext.
	Mnemonic string `json:"mnemonic,omitempty"`
	// Xpub is the extended public key, for DeriveAddress.
	Xpub string `json:"xpub,omitempty"`
	// Address is the generated account address (XRP, XLM, ALGO, SOL).
	Address string `json:"address,omitempty"`
	// Secret is the account's secret seed/key, in plaintext (XRP, XLM, ALGO).
	Secret string `json:"secret,omitempty"`
	// PrivateKey is the raw base58 private key, in plaintext (SOL).
	PrivateKey string `json:"privateKey,omitempty"`
}

// Generate creates a brand-new wallet for chain and returns its secret
// material in plaintext. params may be nil.
func (s *BlockchainWalletService) Generate(ctx context.Context, chain string, params *GenerateWalletParams, opts ...RequestOption) (*Wallet, error) {
	q := newQuery()
	if params != nil {
		q.str("mnemonic", params.Mnemonic)
	}
	return doJSON[Wallet](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/wallet/%s", chain),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// DerivedAddress is an address derived from an xpub.
type DerivedAddress struct {
	Address string `json:"address"`
}

// DeriveAddress derives the address at HD index from an extended public key.
// Available for BTC, ETH, BNB, MATIC, AVAX, TRON, LTC, DOGE, ARB, OP, BASE,
// CELO, FTM, BCH, ADA, VET, and EGLD.
//
// WARNING (EGLD only): EGLD has no xpub, so for that chain the xpub argument
// must be the wallet's mnemonic, which is then placed in the URL path and may
// be logged. Treat any EGLD mnemonic sent this way as compromised and prefer
// deriving EGLD addresses locally.
func (s *BlockchainWalletService) DeriveAddress(ctx context.Context, chain, xpub string, index int64, opts ...RequestOption) (*DerivedAddress, error) {
	if index < 0 {
		return nil, errors.New("cryptures: DeriveAddress: index must be >= 0")
	}
	return doJSON[DerivedAddress](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/wallet/%s/address/%s/%s", chain, xpub, strconv.FormatInt(index, 10)),
		retryable: true,
	}, opts)
}

// DerivePrivateKeyRequest is the input to DerivePrivateKey.
type DerivePrivateKeyRequest struct {
	// Mnemonic is the BIP-39 mnemonic phrase (as returned by Generate).
	Mnemonic string `json:"mnemonic"`
	// Index is the HD derivation index (>= 0).
	Index int64 `json:"index"`
}

// PrivateKey is a derived private key.
type PrivateKey struct {
	// Key is the derived private key, in plaintext.
	Key string `json:"key"`
}

// DerivePrivateKey derives the private key for a mnemonic at an HD index.
// The mnemonic travels in the request body, never the URL. Same chain
// coverage as DeriveAddress.
func (s *BlockchainWalletService) DerivePrivateKey(ctx context.Context, chain string, req *DerivePrivateKeyRequest, opts ...RequestOption) (*PrivateKey, error) {
	if req == nil {
		return nil, errors.New("cryptures: DerivePrivateKey: nil request")
	}
	return doJSON[PrivateKey](ctx, s.client, &requestSpec{
		method:    http.MethodPost,
		path:      pathf("/api/v1/blockchain/key/%s/derive", chain),
		body:      req,
		retryable: true, // deterministic, no side effects
	}, opts)
}
