package cryptures

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// BlockchainLookupsService looks up transactions by hash, blocks, token
// metadata, and unspent transaction outputs.
type BlockchainLookupsService struct {
	client *Client
}

// ---------------------------------------------------------------------------
// tx.hash
// ---------------------------------------------------------------------------

// Transaction is a single transaction's detail. Its shape depends on the
// chain; use the accessor for the chain you queried:
//
//   - ARB, AVAX, BASE, BNB, CELO, ETH, MATIC, OP: AsEVM
//   - BTC, LTC, DOGE, BCH: AsUTXO
//   - TRON: AsTron
//   - ADA, ALGO, EGLD, FTM, SOL, VET, XLM, XRP: chain-native object, read Raw
//
// Quota cost is not flat: the EVM group costs 10 units per call, every other
// chain 1 unit.
type Transaction struct {
	// Raw is the response body exactly as returned by the API.
	Raw json.RawMessage
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Transaction) UnmarshalJSON(data []byte) error {
	t.Raw = append(t.Raw[:0], data...)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (t Transaction) MarshalJSON() ([]byte, error) { return marshalRaw(t.Raw) }

// EVMTransactionEntry is one participant/asset affected by an EVM
// transaction. A transaction moving native currency and a token yields two
// entries sharing the same Hash.
type EVMTransactionEntry struct {
	Chain            string `json:"chain"`
	Hash             string `json:"hash"`
	Address          string `json:"address"`
	CounterAddress   string `json:"counterAddress,omitempty"`
	BlockNumber      int64  `json:"blockNumber,omitempty"`
	TransactionIndex int64  `json:"transactionIndex,omitempty"`
	// TransactionType is one of "native", "fungible", "nft", "multitoken".
	TransactionType string `json:"transactionType"`
	// TransactionSubtype is "incoming" or "outgoing".
	TransactionSubtype string `json:"transactionSubtype"`
	Amount             string `json:"amount,omitempty"`
	// Timestamp is in milliseconds since the Unix epoch.
	Timestamp int64 `json:"timestamp,omitempty"`
}

// UTXOTransaction is a raw UTXO-chain transaction (BTC, LTC, DOGE, BCH).
type UTXOTransaction struct {
	Hash        string  `json:"hash"`
	BlockNumber int64   `json:"blockNumber,omitempty"`
	Fee         float64 `json:"fee,omitempty"`
	Size        int64   `json:"size,omitempty"`
	VSize       int64   `json:"vsize,omitempty"`
	Weight      int64   `json:"weight,omitempty"`
	Time        int64   `json:"time,omitempty"`
	Version     int64   `json:"version,omitempty"`
	Locktime    int64   `json:"locktime,omitempty"`
	// Inputs and Outputs are chain-native objects, not modelled by the API reference.
	Inputs  []json.RawMessage `json:"inputs"`
	Outputs []json.RawMessage `json:"outputs"`
	Hex     string            `json:"hex,omitempty"`
}

// TronTransaction is TRON's own transaction shape.
type TronTransaction struct {
	TxID        string            `json:"txID"`
	BlockNumber int64             `json:"blockNumber,omitempty"`
	Ret         []json.RawMessage `json:"ret,omitempty"`
	Signature   []string          `json:"signature,omitempty"`
	RawData     json.RawMessage   `json:"rawData"`
}

// AsEVM decodes the transaction as the EVM-group array of entries.
func (t *Transaction) AsEVM() ([]EVMTransactionEntry, error) {
	v, err := decodeRaw[[]EVMTransactionEntry](t.Raw)
	if err != nil {
		return nil, err
	}
	return *v, nil
}

// AsUTXO decodes the transaction as a raw UTXO-chain transaction.
func (t *Transaction) AsUTXO() (*UTXOTransaction, error) { return decodeRaw[UTXOTransaction](t.Raw) }

// AsTron decodes the transaction as TRON's transaction shape.
func (t *Transaction) AsTron() (*TronTransaction, error) { return decodeRaw[TronTransaction](t.Raw) }

// GetTransaction returns a single transaction's full detail by its hash. A
// 404 is often transient for a just-broadcast transaction that is not yet
// indexed.
func (s *BlockchainLookupsService) GetTransaction(ctx context.Context, chain, hash string, opts ...RequestOption) (*Transaction, error) {
	return doJSON[Transaction](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/tx/%s/%s", chain, hash),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// block.get / block.latest
// ---------------------------------------------------------------------------

// Block is a block. Its shape depends on the chain family; use the accessor
// for the chain you queried:
//
//   - ARB, AVAX, BASE, BNB, CELO, ETH, FTM, MATIC, OP: AsEVM
//   - BTC, LTC, DOGE, BCH: AsUTXO
//   - TRON: AsTron
//   - ADA, ALGO, EGLD, SOL, VET, XLM, XRP: chain-native object, read Raw
type Block struct {
	// Raw is the response body exactly as returned by the API.
	Raw json.RawMessage
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Block) UnmarshalJSON(data []byte) error {
	b.Raw = append(b.Raw[:0], data...)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (b Block) MarshalJSON() ([]byte, error) { return marshalRaw(b.Raw) }

// EVMBlock is an EVM-style block: a full header plus fully-expanded
// transactions.
type EVMBlock struct {
	Hash       string  `json:"hash"`
	Number     int64   `json:"number"`
	Height     int64   `json:"height,omitempty"`
	ParentHash string  `json:"parentHash"`
	Timestamp  int64   `json:"timestamp,omitempty"`
	Miner      string  `json:"miner,omitempty"`
	GasLimit   float64 `json:"gasLimit,omitempty"`
	GasUsed    float64 `json:"gasUsed,omitempty"`
	Size       int64   `json:"size,omitempty"`
	Difficulty string  `json:"difficulty,omitempty"`
	// Transactions are chain-native transaction objects.
	Transactions []json.RawMessage `json:"transactions"`
}

// UTXOBlock is a UTXO-chain block header (BTC, LTC, DOGE, BCH).
type UTXOBlock struct {
	Hash          string  `json:"hash"`
	Height        int64   `json:"height"`
	MedianTime    int64   `json:"mediantime,omitempty"`
	Bits          int64   `json:"bits,omitempty"`
	Difficulty    float64 `json:"difficulty,omitempty"`
	Chainwork     string  `json:"chainwork,omitempty"`
	Confirmations int64   `json:"confirmations,omitempty"`
	MerkleRoot    string  `json:"merkleRoot"`
}

// TronBlock is TRON's block shape as returned by GetBlock.
type TronBlock struct {
	BlockNumber      int64  `json:"blockNumber"`
	Hash             string `json:"hash"`
	ParentHash       string `json:"parentHash,omitempty"`
	Timestamp        int64  `json:"timestamp,omitempty"`
	WitnessAddress   string `json:"witnessAddress"`
	WitnessSignature string `json:"witnessSignature,omitempty"`
}

// AsEVM decodes the block as an EVM-style block.
func (b *Block) AsEVM() (*EVMBlock, error) { return decodeRaw[EVMBlock](b.Raw) }

// AsUTXO decodes the block as a UTXO-chain block header.
func (b *Block) AsUTXO() (*UTXOBlock, error) { return decodeRaw[UTXOBlock](b.Raw) }

// AsTron decodes the block as TRON's block shape.
func (b *Block) AsTron() (*TronBlock, error) { return decodeRaw[TronBlock](b.Raw) }

// GetBlock returns a block by its hash or height, on any of the 21 chains.
// "latest" is not accepted; use GetLatestBlock for TRON's current block.
func (s *BlockchainLookupsService) GetBlock(ctx context.Context, chain, hashOrHeight string, opts ...RequestOption) (*Block, error) {
	return doJSON[Block](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/block/%s/%s", chain, hashOrHeight),
		retryable: true,
	}, opts)
}

// TronLatestBlock is TRON's current block, the reference needed to sign a
// TRON transaction locally.
type TronLatestBlock struct {
	// BlockID is the block hash; ref_block_hash is derived from it.
	BlockID     string              `json:"blockID"`
	BlockHeader TronLatestBlockHead `json:"block_header"`
	// Transactions are the block's own confirmed transactions.
	Transactions []json.RawMessage `json:"transactions"`
}

// TronLatestBlockHead is the header of a TronLatestBlock.
type TronLatestBlockHead struct {
	RawData          TronLatestBlockRawData `json:"raw_data"`
	WitnessSignature string                 `json:"witness_signature,omitempty"`
}

// TronLatestBlockRawData is the raw header data of a TronLatestBlock.
type TronLatestBlockRawData struct {
	Timestamp int64 `json:"timestamp"`
	// Number is the block height; ref_block_bytes is derived from its last 2 bytes.
	Number         int64  `json:"number"`
	WitnessAddress string `json:"witness_address,omitempty"`
	Version        int64  `json:"version,omitempty"`
}

// GetLatestBlock returns TRON's current block. chain must be "TRON". Never
// cached.
func (s *BlockchainLookupsService) GetLatestBlock(ctx context.Context, chain string, opts ...RequestOption) (*TronLatestBlock, error) {
	return doJSON[TronLatestBlock](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/block/%s/latest", chain),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// tokens.get
// ---------------------------------------------------------------------------

// GetTokenParams configures GetToken.
type GetTokenParams struct {
	// TokenID looks up one specific NFT within a collection. Leave empty for
	// fungible, native, or collection-level metadata.
	TokenID string
}

// TokenMetadata is metadata for a token, NFT collection, NFT, multitoken,
// or a chain's native currency.
type TokenMetadata struct {
	Symbol   string `json:"symbol,omitempty"`
	Name     string `json:"name,omitempty"`
	Decimals int    `json:"decimals,omitempty"`
	// Supply is cached, not live: it can lag the chain by 24 hours or more.
	Supply string `json:"supply,omitempty"`
	// TokenType is one of "native", "fungible", "nonfungible", "multitoken".
	TokenType string `json:"tokenType,omitempty"`
	Logo      string `json:"logo,omitempty"`
	// MetadataURI is present for a specific NFT only.
	MetadataURI string `json:"metadataURI,omitempty"`
}

// GetToken returns metadata for the token at tokenAddress, or for the
// chain's native currency when tokenAddress is "native". Available for ARB,
// AVAX, BASE, BNB, CELO, ETH, MATIC, OP, SOL. params may be nil.
func (s *BlockchainLookupsService) GetToken(ctx context.Context, chain, tokenAddress string, params *GetTokenParams, opts ...RequestOption) (*TokenMetadata, error) {
	q := newQuery()
	if params != nil {
		q.str("tokenId", params.TokenID)
	}
	return doJSON[TokenMetadata](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/tokens/%s/%s", chain, tokenAddress),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// utxo.list / utxo.batch
// ---------------------------------------------------------------------------

// UTXO is an unspent transaction output.
type UTXO struct {
	Chain         string  `json:"chain,omitempty"`
	Address       string  `json:"address,omitempty"`
	TxHash        string  `json:"txHash"`
	Index         int64   `json:"index"`
	Value         float64 `json:"value"`
	ValueAsString string  `json:"valueAsString,omitempty"`
}

// ListUTXOs returns just enough unspent outputs of address to cover
// totalValue (in the chain's native unit, e.g. BTC, not satoshis). Available
// for BTC, LTC, and DOGE.
func (s *BlockchainLookupsService) ListUTXOs(ctx context.Context, chain, address string, totalValue float64, opts ...RequestOption) ([]UTXO, error) {
	if totalValue <= 0 {
		return nil, errors.New("cryptures: ListUTXOs: totalValue must be positive")
	}
	out, err := doJSON[[]UTXO](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/utxo/%s/%s", chain, address),
		query:     newQuery().float("totalValue", totalValue).values(),
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// UTXOBatchRequest looks up UTXOs for up to 50 addresses.
type UTXOBatchRequest struct {
	// Addresses holds 1-50 addresses.
	Addresses []string `json:"addresses"`
	// TotalValue is the per-address spend target, in the chain's native unit.
	TotalValue float64 `json:"totalValue"`
	// Chain is "bitcoin-mainnet", "litecoin-mainnet", or "doge-mainnet"
	// (not the usual BTC/LTC/DOGE chain code).
	Chain string `json:"chain"`
}

// AddressUTXOs holds the UTXOs selected for one address.
type AddressUTXOs struct {
	Address string `json:"address"`
	UTXOs   []UTXO `json:"utxos"`
	// TransactionPossible reports whether the returned UTXOs sum to at least
	// the requested total value.
	TransactionPossible bool `json:"transactionPossible"`
}

// ListUTXOsBatch returns UTXOs for several addresses in one call, one entry
// per address in the order supplied.
func (s *BlockchainLookupsService) ListUTXOsBatch(ctx context.Context, req *UTXOBatchRequest, opts ...RequestOption) ([]AddressUTXOs, error) {
	if req == nil {
		return nil, errors.New("cryptures: ListUTXOsBatch: nil request")
	}
	out, err := doJSON[[]AddressUTXOs](ctx, s.client, &requestSpec{
		method:    http.MethodPost,
		path:      "/api/v1/blockchain/data/utxo/batch",
		body:      req,
		retryable: true, // read-only lookup
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}
