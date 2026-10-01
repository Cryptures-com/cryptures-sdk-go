package cryptures

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// BlockchainDataService covers chain data (balances, history, portfolios,
// address screening), exchange rates, and market data.
//
// Several operations return a different response shape depending on the
// chain queried. Those responses are modelled as a type that keeps the raw
// JSON and exposes one typed accessor per documented shape (for example
// Balance.AsUTXO or TransactionHistory.AsUnified); call the accessor that
// matches the chain you queried.
type BlockchainDataService struct {
	client *Client
}

// ---------------------------------------------------------------------------
// balance.check
// ---------------------------------------------------------------------------

// Balance is the native-token balance of an address. Its shape depends on
// the chain; use the accessor for the chain you queried:
//
//   - ETH, SOL, BNB, MATIC, AVAX, ALGO, ARB, OP, BASE, FTM, VET, EGLD: AsSimple
//   - BTC, LTC, DOGE: AsUTXO
//   - TRON: AsTron
//   - CELO: AsCelo
//   - ADA: AsCardano
//   - XLM: AsStellar
//   - XRP: AsXRP
//
// Bitcoin Cash (BCH) has no balance-check coverage and returns a 404.
type Balance struct {
	// Raw is the response body exactly as returned by the API.
	Raw json.RawMessage
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Balance) UnmarshalJSON(data []byte) error {
	b.Raw = append(b.Raw[:0], data...)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (b Balance) MarshalJSON() ([]byte, error) { return marshalRaw(b.Raw) }

// SimpleBalance is the balance shape for ETH, SOL, BNB, MATIC, AVAX, ALGO,
// ARB, OP, BASE, FTM, VET, and EGLD.
type SimpleBalance struct {
	// Balance is the native balance, as a decimal string in the chain's own unit.
	Balance string `json:"balance"`
}

// UTXOBalance is the balance shape for BTC, LTC, and DOGE. All amounts are in
// satoshis (or the chain's equivalent smallest unit).
type UTXOBalance struct {
	// Balance is the confirmed balance (confirmed incoming minus confirmed outgoing).
	Balance string `json:"balance"`
	// Incoming is the incoming sum, including confirmed and pending mempool transactions.
	Incoming string `json:"incoming"`
	// Outgoing is the outgoing sum, including confirmed and pending mempool transactions.
	Outgoing string `json:"outgoing"`
	// IncomingPending is the pending incoming sum.
	IncomingPending string `json:"incomingPending"`
	// OutgoingPending is the pending outgoing sum.
	OutgoingPending string `json:"outgoingPending"`
}

// CeloBalance is the balance shape for CELO: three named currencies and no
// generic balance field.
type CeloBalance struct {
	Celo string `json:"celo"`
	CUSD string `json:"cUsd"`
	CEUR string `json:"cEur"`
}

// CardanoAssetBalance is one entry of the ADA multi-asset balance array.
type CardanoAssetBalance struct {
	Currency CardanoCurrency `json:"currency"`
	// Value is the held amount of the asset, as a decimal string.
	Value string `json:"value"`
}

// CardanoCurrency identifies a Cardano asset.
type CardanoCurrency struct {
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals,omitempty"`
}

// StellarBalance is the balance shape for XLM: a Stellar account object.
type StellarBalance struct {
	AccountID string `json:"account_id,omitempty"`
	Sequence  string `json:"sequence,omitempty"`
	// Balances carries one entry per held asset.
	Balances []StellarAssetBalance `json:"balances"`
}

// StellarAssetBalance is one held asset on a Stellar account.
type StellarAssetBalance struct {
	// AssetType is e.g. "native" for XLM itself, "credit_alphanum4" for an issued asset.
	AssetType string `json:"asset_type"`
	Balance   string `json:"balance"`
	// Limit is the trustline limit. Empty for the native asset.
	Limit string `json:"limit,omitempty"`
	// AssetCode is empty for the native asset.
	AssetCode string `json:"asset_code,omitempty"`
	// AssetIssuer is empty for the native asset.
	AssetIssuer string `json:"asset_issuer,omitempty"`
}

// XRPBalance is the balance shape for XRP.
type XRPBalance struct {
	// Balance is the native XRP balance, in drops (1 XRP = 1,000,000 drops).
	Balance string `json:"balance"`
	// Assets lists issued (non-native) currencies held by the account.
	Assets []XRPAssetBalance `json:"assets"`
}

// XRPAssetBalance is one issued currency held by an XRP account.
type XRPAssetBalance struct {
	Balance  string `json:"balance"`
	Currency string `json:"currency"`
}

// TronBalance is the balance shape for TRON: a full account object.
type TronBalance struct {
	Address string `json:"address"`
	// Balance is the native TRX balance, in SUN (1 TRX = 1,000,000 SUN).
	Balance float64 `json:"balance"`
	// TRC10 and TRC20 hold the account's token entries, whose fields are not
	// modelled by the API reference.
	TRC10 []json.RawMessage `json:"trc10,omitempty"`
	TRC20 []json.RawMessage `json:"trc20,omitempty"`
	// Bandwidth is the account's bandwidth object, not modelled by the API reference.
	Bandwidth json.RawMessage `json:"bandwidth,omitempty"`
}

// AsSimple decodes the balance as the simple { balance } shape.
func (b *Balance) AsSimple() (*SimpleBalance, error) { return decodeRaw[SimpleBalance](b.Raw) }

// AsUTXO decodes the balance as the BTC/LTC/DOGE shape.
func (b *Balance) AsUTXO() (*UTXOBalance, error) { return decodeRaw[UTXOBalance](b.Raw) }

// AsCelo decodes the balance as the CELO shape.
func (b *Balance) AsCelo() (*CeloBalance, error) { return decodeRaw[CeloBalance](b.Raw) }

// AsCardano decodes the balance as the ADA multi-asset array.
func (b *Balance) AsCardano() ([]CardanoAssetBalance, error) {
	v, err := decodeRaw[[]CardanoAssetBalance](b.Raw)
	if err != nil {
		return nil, err
	}
	return *v, nil
}

// AsStellar decodes the balance as the XLM account shape.
func (b *Balance) AsStellar() (*StellarBalance, error) { return decodeRaw[StellarBalance](b.Raw) }

// AsXRP decodes the balance as the XRP shape.
func (b *Balance) AsXRP() (*XRPBalance, error) { return decodeRaw[XRPBalance](b.Raw) }

// AsTron decodes the balance as the TRON account shape.
func (b *Balance) AsTron() (*TronBalance, error) { return decodeRaw[TronBalance](b.Raw) }

// GetBalance returns the native-token balance of address on chain (a chain
// code such as "BTC" or "ETH"). See Balance for the per-chain shapes.
func (s *BlockchainDataService) GetBalance(ctx context.Context, chain, address string, opts ...RequestOption) (*Balance, error) {
	return doJSON[Balance](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/balance/%s/%s", chain, address),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// balance.batch
// ---------------------------------------------------------------------------

// BalanceBatchRequest asks for native balances of up to 10 addresses on the
// same chain, as of one point in time. Exactly one of BlockNumber, Time, or
// Unix must be set; omitting all three is a validation error (it does not
// fall back to the current balance).
type BalanceBatchRequest struct {
	// Chain is a network-id style identifier, not the usual chain code: one
	// of "bitcoin-mainnet", "ethereum-mainnet", "bsc-mainnet",
	// "polygon-mainnet", "avax-mainnet", "arb-one-mainnet",
	// "optimism-mainnet", "base-mainnet", "celo-mainnet".
	Chain string `json:"chain"`
	// Addresses is a comma-separated list of up to 10 addresses.
	Addresses string `json:"addresses"`
	// BlockNumber looks up balances as of this block.
	BlockNumber *int64 `json:"blockNumber,omitempty"`
	// Time is an ISO-8601-ish timestamp to look up balances as of.
	Time string `json:"time,omitempty"`
	// Unix is a Unix timestamp to look up balances as of.
	Unix *int64 `json:"unix,omitempty"`
}

// BalanceBatchResponse holds one balance per requested address.
type BalanceBatchResponse struct {
	Result   []BatchBalance `json:"result"`
	PrevPage string         `json:"prevPage,omitempty"`
	NextPage string         `json:"nextPage,omitempty"`
}

// BatchBalance is one address's balance in a BalanceBatchResponse.
type BatchBalance struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
	// Balance is a plain decimal string in the chain's native unit.
	Balance                string `json:"balance"`
	LastUpdatedBlockNumber int64  `json:"lastUpdatedBlockNumber"`
	// Type is e.g. "native".
	Type string `json:"type"`
}

// GetBalances returns native balances for up to 10 addresses on one chain in
// a single call. Coverage: BTC, ETH, BNB, MATIC, AVAX, ARB, OP, BASE, CELO.
// This operation is never cached.
func (s *BlockchainDataService) GetBalances(ctx context.Context, req *BalanceBatchRequest, opts ...RequestOption) (*BalanceBatchResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: GetBalances: nil request")
	}
	return doJSON[BalanceBatchResponse](ctx, s.client, &requestSpec{
		method:    http.MethodPost,
		path:      "/api/v1/blockchain/data/balance/batch",
		body:      req,
		retryable: true, // read-only lookup
	}, opts)
}

// ---------------------------------------------------------------------------
// token.transfers
// ---------------------------------------------------------------------------

// TokenTransfersParams filters TRC-20 transfer history. Every field is
// optional and passed through unmodified.
type TokenTransfersParams struct {
	// Next is a pagination cursor from a previous response's Next field.
	Next string
	// OnlyConfirmed restricts results to confirmed transfers.
	OnlyConfirmed *bool
	// OnlyUnconfirmed restricts results to unconfirmed transfers.
	OnlyUnconfirmed *bool
	// OnlyTo restricts results to transfers where the address is the recipient.
	OnlyTo *bool
	// OnlyFrom restricts results to transfers where the address is the sender.
	OnlyFrom *bool
	// OrderBy is the sort order for results.
	OrderBy string
	// MinTimestamp includes only transfers at or after this timestamp.
	MinTimestamp *int64
	// MaxTimestamp includes only transfers at or before this timestamp.
	MaxTimestamp *int64
	// ContractAddress restricts results to one TRC-20 token contract.
	ContractAddress string
}

// TokenTransfers is a page of TRC-20 transfer history.
type TokenTransfers struct {
	Transactions []TokenTransfer `json:"transactions"`
	// Next is the cursor for the following page, when more results exist.
	Next string `json:"next,omitempty"`
}

// TokenTransfer is one TRC-20 transfer (or approval) record.
type TokenTransfer struct {
	TxID      string    `json:"txID"`
	TokenInfo TokenInfo `json:"tokenInfo"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	// Type is e.g. "Transfer" or "Approval".
	Type  string `json:"type"`
	Value string `json:"value"`
}

// TokenInfo describes the token involved in a TokenTransfer.
type TokenInfo struct {
	Symbol string `json:"symbol"`
	// Address is the TRC-20 token contract address.
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
	Name     string `json:"name"`
}

// ListTokenTransfers returns TRC-20 token transfer history for a TRON
// address. chain must be "TRON". params may be nil.
func (s *BlockchainDataService) ListTokenTransfers(ctx context.Context, chain, address string, params *TokenTransfersParams, opts ...RequestOption) (*TokenTransfers, error) {
	q := newQuery()
	if params != nil {
		q.str("next", params.Next).
			boolPtr("onlyConfirmed", params.OnlyConfirmed).
			boolPtr("onlyUnconfirmed", params.OnlyUnconfirmed).
			boolPtr("onlyTo", params.OnlyTo).
			boolPtr("onlyFrom", params.OnlyFrom).
			str("orderBy", params.OrderBy).
			intPtr("minTimestamp", params.MinTimestamp).
			intPtr("maxTimestamp", params.MaxTimestamp).
			str("contractAddress", params.ContractAddress)
	}
	return doJSON[TokenTransfers](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/token-transfers/%s/%s", chain, address),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// tx.history
// ---------------------------------------------------------------------------

// TransactionHistoryParams controls paging for GetTransactionHistory.
type TransactionHistoryParams struct {
	// PageSize is 1-50. On chain-native-shape chains the API sends 50 when
	// omitted. Ignored on EGLD.
	PageSize *int64
	// Offset is the pagination offset. Ignored on EGLD.
	Offset *int64
}

// TransactionHistory is an address's transaction history. There is no single
// shape across chains; use the accessor for the chain you queried:
//
//   - BNB, AVAX, ARB, OP, BASE, CELO: AsUnified
//   - TRON: AsTron
//   - BTC, LTC, DOGE, BCH, ETH, MATIC, XRP, XLM, EGLD: AsChainNative
//
// SOL, ADA, ALGO, VET, and FTM have no transaction-history coverage (404).
type TransactionHistory struct {
	// Raw is the response body exactly as returned by the API.
	Raw json.RawMessage
}

// UnmarshalJSON implements json.Unmarshaler.
func (h *TransactionHistory) UnmarshalJSON(data []byte) error {
	h.Raw = append(h.Raw[:0], data...)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (h TransactionHistory) MarshalJSON() ([]byte, error) { return marshalRaw(h.Raw) }

// UnifiedTransactionHistory is the normalized history shape returned for
// BNB, AVAX, ARB, OP, BASE, and CELO.
type UnifiedTransactionHistory struct {
	Result   []UnifiedTransfer `json:"result"`
	PrevPage string            `json:"prevPage,omitempty"`
	NextPage string            `json:"nextPage,omitempty"`
}

// UnifiedTransfer is one normalized transfer record.
type UnifiedTransfer struct {
	Chain          string `json:"chain,omitempty"`
	Hash           string `json:"hash,omitempty"`
	Address        string `json:"address,omitempty"`
	CounterAddress string `json:"counterAddress,omitempty"`
	TokenAddress   string `json:"tokenAddress,omitempty"`
	TokenID        string `json:"tokenId,omitempty"`
	BlockNumber    int64  `json:"blockNumber,omitempty"`
	// TransactionType is one of "fungible", "nft", "multitoken", "native".
	TransactionType string `json:"transactionType,omitempty"`
	// TransactionSubtype is one of "incoming", "outgoing", "zero-transfer".
	TransactionSubtype string `json:"transactionSubtype,omitempty"`
	Amount             string `json:"amount,omitempty"`
	// Timestamp is in milliseconds since the Unix epoch.
	Timestamp int64 `json:"timestamp,omitempty"`
}

// TronTransactionHistory is TRON's own history shape.
type TronTransactionHistory struct {
	// Transactions are TRON transaction objects, not modelled by the API reference.
	Transactions []json.RawMessage `json:"transactions"`
	// Next is the cursor for the following page, when more results exist.
	Next string `json:"next,omitempty"`
}

// AsUnified decodes the history as the normalized BNB/AVAX/ARB/OP/BASE/CELO shape.
func (h *TransactionHistory) AsUnified() (*UnifiedTransactionHistory, error) {
	return decodeRaw[UnifiedTransactionHistory](h.Raw)
}

// AsTron decodes the history as TRON's { transactions, next } shape.
func (h *TransactionHistory) AsTron() (*TronTransactionHistory, error) {
	return decodeRaw[TronTransactionHistory](h.Raw)
}

// AsChainNative decodes the history as a plain array of chain-native
// transaction objects (BTC, LTC, DOGE, BCH, ETH, MATIC, XRP, XLM, EGLD),
// whose fields differ per chain and are returned undecoded.
func (h *TransactionHistory) AsChainNative() ([]json.RawMessage, error) {
	v, err := decodeRaw[[]json.RawMessage](h.Raw)
	if err != nil {
		return nil, err
	}
	return *v, nil
}

// GetTransactionHistory returns the transaction history of address on chain.
// params may be nil. See TransactionHistory for the per-chain shapes.
func (s *BlockchainDataService) GetTransactionHistory(ctx context.Context, chain, address string, params *TransactionHistoryParams, opts ...RequestOption) (*TransactionHistory, error) {
	q := newQuery()
	if params != nil {
		q.intPtr("pageSize", params.PageSize).intPtr("offset", params.Offset)
	}
	return doJSON[TransactionHistory](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/history/%s/%s", chain, address),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// portfolio.get
// ---------------------------------------------------------------------------

// PortfolioParams configures GetPortfolio.
type PortfolioParams struct {
	// TokenTypes is a required comma-separated filter: one or more of
	// "native", "fungible", "nft", "multitoken".
	TokenTypes string
	// ExcludeMetadata excludes NFT/multitoken metadata. Defaults to false.
	ExcludeMetadata *bool
	// PageSize is 1-50. Defaults to 50.
	PageSize *int64
	// Offset is the pagination offset.
	Offset *int64
}

// Portfolio is a page of an address's holdings.
type Portfolio struct {
	Result   []PortfolioItem `json:"result"`
	PrevPage string          `json:"prevPage,omitempty"`
	NextPage string          `json:"nextPage,omitempty"`
}

// PortfolioItem is one holding, discriminated by Type.
type PortfolioItem struct {
	Chain string `json:"chain,omitempty"`
	// Type is one of "native", "fungible", "nft", "multitoken".
	Type               string `json:"type,omitempty"`
	Address            string `json:"address,omitempty"`
	Balance            string `json:"balance,omitempty"`
	DenominatedBalance string `json:"denominatedBalance,omitempty"`
	Decimals           int    `json:"decimals,omitempty"`
	TokenAddress       string `json:"tokenAddress,omitempty"`
	TokenID            string `json:"tokenId,omitempty"`
	MetadataURI        string `json:"metadataURI,omitempty"`
	// Metadata is the NFT/multitoken metadata object, not modelled by the API reference.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// GetPortfolio returns native, fungible-token, and NFT/multitoken balances
// for address. Available for ETH, SOL, BNB, MATIC, AVAX, ARB, OP, BASE, CELO.
// params.TokenTypes is required.
func (s *BlockchainDataService) GetPortfolio(ctx context.Context, chain, address string, params *PortfolioParams, opts ...RequestOption) (*Portfolio, error) {
	if params == nil || params.TokenTypes == "" {
		return nil, errors.New("cryptures: GetPortfolio: params.TokenTypes is required")
	}
	q := newQuery().
		str("tokenTypes", params.TokenTypes).
		boolPtr("excludeMetadata", params.ExcludeMetadata).
		intPtr("pageSize", params.PageSize).
		intPtr("offset", params.Offset)
	return doJSON[Portfolio](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/portfolio/%s/%s", chain, address),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// balance-history.get
// ---------------------------------------------------------------------------

// BalanceHistoryParams selects the point in time for GetBalanceHistory. Set
// at most one field; with none set, the current balance is returned.
type BalanceHistoryParams struct {
	// Time is an ISO-8601-ish timestamp, e.g. "2022-12-24T00:20".
	Time string
	// BlockNumber is a block number.
	BlockNumber *int64
	// Unix is a Unix timestamp.
	Unix *int64
}

// BalanceHistory is an address's balance as of a point in time.
type BalanceHistory struct {
	Result   []HistoricalBalance `json:"result"`
	PrevPage string              `json:"prevPage,omitempty"`
	NextPage string              `json:"nextPage,omitempty"`
}

// HistoricalBalance is one balance entry in a BalanceHistory.
type HistoricalBalance struct {
	Chain              string `json:"chain,omitempty"`
	Address            string `json:"address,omitempty"`
	Balance            string `json:"balance,omitempty"`
	DenominatedBalance string `json:"denominatedBalance,omitempty"`
	Decimals           int    `json:"decimals,omitempty"`
	// Type is one of "native", "fungible", "nft", "multitoken".
	Type string `json:"type,omitempty"`
}

// GetBalanceHistory returns address's native balance as of a past point in
// time. Available for BTC, ETH, BNB, MATIC, AVAX, ARB, OP, BASE, CELO.
// params may be nil.
func (s *BlockchainDataService) GetBalanceHistory(ctx context.Context, chain, address string, params *BalanceHistoryParams, opts ...RequestOption) (*BalanceHistory, error) {
	q := newQuery()
	if params != nil {
		q.str("time", params.Time).intPtr("blockNumber", params.BlockNumber).intPtr("unix", params.Unix)
	}
	return doJSON[BalanceHistory](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/balance-history/%s/%s", chain, address),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// security.address-check
// ---------------------------------------------------------------------------

// AddressCheck is the result of screening an address against a
// malicious-address intelligence feed.
type AddressCheck struct {
	// Status is "valid" for a clean or unflagged address, "invalid" for a flagged one.
	Status string `json:"status"`
	// Address is present only when Status is "invalid".
	Address string `json:"address,omitempty"`
	// Source is the intelligence source that flagged the address. Present only when Status is "invalid".
	Source string `json:"source,omitempty"`
	// Description briefly explains the flagged risk. Present only when Status is "invalid".
	Description string `json:"description,omitempty"`
}

// CheckAddress screens a single BTC, ETH, LTC, SOL, or TRON address for
// malicious activity. A 404 *APIError means the feed has no record for the
// address: treat it as "no screening result", not as clean or flagged.
func (s *BlockchainDataService) CheckAddress(ctx context.Context, address string, opts ...RequestOption) (*AddressCheck, error) {
	return doJSON[AddressCheck](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/security/%s", address),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// exchange.rate / exchange.rate.contract / exchange.rate.batch
// ---------------------------------------------------------------------------

// ExchangeRateParams configures GetExchangeRate.
type ExchangeRateParams struct {
	// BasePair is the currency to price against. Defaults to "USD".
	BasePair string
}

// ExchangeRate is the price of one symbol against a base pair.
type ExchangeRate struct {
	// Value is the rate, as a decimal string.
	Value    string `json:"value"`
	BasePair string `json:"basePair"`
	// ID is the symbol that was priced.
	ID string `json:"id"`
	// Timestamp is when the rate was determined, in milliseconds since the Unix epoch.
	Timestamp int64 `json:"timestamp"`
}

// GetExchangeRate returns the exchange rate of symbol (crypto or fiat).
// params may be nil. A well-formed pair with no rate on record returns a 403
// *APIError whose Code is a rate-not-found code (e.g. "rate.not.found"); this
// is not an authentication failure.
func (s *BlockchainDataService) GetExchangeRate(ctx context.Context, symbol string, params *ExchangeRateParams, opts ...RequestOption) (*ExchangeRate, error) {
	q := newQuery()
	if params != nil {
		q.str("basePair", params.BasePair)
	}
	return doJSON[ExchangeRate](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/rate/%s", symbol),
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ContractExchangeRateParams configures GetExchangeRateByContract.
type ContractExchangeRateParams struct {
	// Chain is required, in network-id format, e.g. "ethereum-mainnet".
	Chain string
	// ContractAddress is the required token contract address.
	ContractAddress string
	// BasePair defaults to "EUR" (not USD) when empty.
	BasePair string
}

// ContractExchangeRate is the price of one unit of a token identified by its
// contract address.
type ContractExchangeRate struct {
	Value     string `json:"value"`
	BasePair  string `json:"basePair"`
	Timestamp int64  `json:"timestamp"`
	Chain     string `json:"chain"`
	Address   string `json:"address"`
}

// GetExchangeRateByContract returns the exchange rate of a token identified
// by its contract address. Note that BasePair defaults to EUR, not USD.
func (s *BlockchainDataService) GetExchangeRateByContract(ctx context.Context, params *ContractExchangeRateParams, opts ...RequestOption) (*ContractExchangeRate, error) {
	if params == nil || params.Chain == "" || params.ContractAddress == "" {
		return nil, errors.New("cryptures: GetExchangeRateByContract: params.Chain and params.ContractAddress are required")
	}
	q := newQuery().
		str("chain", params.Chain).
		str("contractAddress", params.ContractAddress).
		str("basePair", params.BasePair)
	return doJSON[ContractExchangeRate](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/blockchain/data/rate/contract",
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ExchangeRateBatchItem is one entry of a batch rate request.
type ExchangeRateBatchItem struct {
	// BatchID is your own id, echoed back to correlate the response entry.
	BatchID string `json:"batchId"`
	Symbol  string `json:"symbol"`
	// BasePair defaults to EUR when empty.
	BasePair string `json:"basePair,omitempty"`
}

// ExchangeRateBatchResult is one entry of a batch rate response.
type ExchangeRateBatchResult struct {
	BatchID   string `json:"batchId"`
	Symbol    string `json:"symbol"`
	Value     string `json:"value"`
	BasePair  string `json:"basePair"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source,omitempty"`
}

// GetExchangeRates returns exchange rates for several symbols in one call.
// The response order is not guaranteed to match the request order: correlate
// entries by BatchID.
func (s *BlockchainDataService) GetExchangeRates(ctx context.Context, items []ExchangeRateBatchItem, opts ...RequestOption) ([]ExchangeRateBatchResult, error) {
	if items == nil {
		items = []ExchangeRateBatchItem{}
	}
	out, err := doJSON[[]ExchangeRateBatchResult](ctx, s.client, &requestSpec{
		method:    http.MethodPost,
		path:      "/api/v1/blockchain/data/rate/batch",
		body:      items,
		retryable: true, // read-only lookup
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ---------------------------------------------------------------------------
// sentiment.fear-greed
// ---------------------------------------------------------------------------

// FearGreedParams configures GetFearGreedIndex.
type FearGreedParams struct {
	// Limit is the number of days of history to return. Defaults to 1 (today only).
	Limit *int64
	// DateFormat sets the Timestamp format, e.g. "world" for DD-MM-YYYY.
	// Defaults to a Unix timestamp.
	DateFormat string
}

// FearGreedIndex is the crypto Fear & Greed Index.
type FearGreedIndex struct {
	Name     string            `json:"name"`
	Data     []FearGreedValue  `json:"data"`
	Metadata FearGreedMetadata `json:"metadata"`
}

// FearGreedValue is one day's index value.
type FearGreedValue struct {
	// Value is the index value, 0-100.
	Value string `json:"value"`
	// ValueClassification is e.g. "Extreme Fear", "Fear", "Neutral", "Greed", "Extreme Greed".
	ValueClassification string `json:"value_classification"`
	Timestamp           string `json:"timestamp"`
	// TimeUntilUpdate is seconds until the next update. Present only on the most recent entry.
	TimeUntilUpdate string `json:"time_until_update,omitempty"`
}

// FearGreedMetadata carries the data source's own status.
type FearGreedMetadata struct {
	// Error is nil when the request succeeded.
	Error *string `json:"error"`
}

// GetFearGreedIndex returns the crypto Fear & Greed Index. params may be nil.
func (s *BlockchainDataService) GetFearGreedIndex(ctx context.Context, params *FearGreedParams, opts ...RequestOption) (*FearGreedIndex, error) {
	q := newQuery()
	if params != nil {
		q.intPtr("limit", params.Limit).str("date_format", params.DateFormat)
	}
	return doJSON[FearGreedIndex](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/blockchain/data/sentiment/fear-greed",
		query:     q.values(),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// market.*
// ---------------------------------------------------------------------------

// MarketGlobal holds aggregate market statistics.
type MarketGlobal struct {
	CoinsCount    int64   `json:"coins_count"`
	ActiveMarkets int64   `json:"active_markets"`
	TotalMcap     float64 `json:"total_mcap"`
	TotalVolume   float64 `json:"total_volume"`
	// BtcD is Bitcoin dominance, as a percentage string.
	BtcD string `json:"btc_d"`
	// EthD is Ethereum dominance, as a percentage string.
	EthD             string  `json:"eth_d"`
	McapChange       string  `json:"mcap_change"`
	VolumeChange     string  `json:"volume_change"`
	AvgChangePercent string  `json:"avg_change_percent"`
	VolumeAth        float64 `json:"volume_ath"`
	McapAth          float64 `json:"mcap_ath"`
}

// GetMarketGlobal returns aggregate market statistics. The API wraps the
// result in a one-element array, which is returned as-is.
func (s *BlockchainDataService) GetMarketGlobal(ctx context.Context, opts ...RequestOption) ([]MarketGlobal, error) {
	out, err := doJSON[[]MarketGlobal](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/blockchain/data/market/global",
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// MarketAssets lists every tracked coin.
type MarketAssets struct {
	Data []MarketAsset `json:"data"`
}

// MarketAsset is a tracked coin, without price data.
type MarketAsset struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	NameID string `json:"nameid"`
	Rank   int64  `json:"rank"`
}

// ListMarketAssets returns a lightweight list of every tracked coin (id,
// symbol, name, rank; no price data).
func (s *BlockchainDataService) ListMarketAssets(ctx context.Context, opts ...RequestOption) (*MarketAssets, error) {
	return doJSON[MarketAssets](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/blockchain/data/market/assets",
		retryable: true,
	}, opts)
}

// MarketTickersParams pages through ListMarketTickers.
type MarketTickersParams struct {
	// Start is the pagination start offset. Defaults to 0.
	Start *int64
	// Limit is the number of results, up to 100. Defaults to 100.
	Limit *int64
}

// MarketTickers is a page of coins with price/market data.
type MarketTickers struct {
	Data []MarketTicker   `json:"data"`
	Info MarketTickerInfo `json:"info"`
}

// MarketTickerInfo describes a MarketTickers page.
type MarketTickerInfo struct {
	CoinsNum int64 `json:"coins_num"`
	Time     int64 `json:"time"`
}

// MarketTicker is a coin with price/market data. GetMarketTickers documents
// only ID, Symbol, Name, PriceUSD, PercentChange24h, and MarketCapUSD.
type MarketTicker struct {
	ID               string  `json:"id"`
	Symbol           string  `json:"symbol"`
	Name             string  `json:"name"`
	NameID           string  `json:"nameid,omitempty"`
	Rank             int64   `json:"rank,omitempty"`
	PriceUSD         string  `json:"price_usd,omitempty"`
	PercentChange24h string  `json:"percent_change_24h,omitempty"`
	PercentChange1h  string  `json:"percent_change_1h,omitempty"`
	PercentChange7d  string  `json:"percent_change_7d,omitempty"`
	PriceBTC         string  `json:"price_btc,omitempty"`
	MarketCapUSD     string  `json:"market_cap_usd,omitempty"`
	Volume24         float64 `json:"volume24,omitempty"`
	Volume24a        float64 `json:"volume24a,omitempty"`
	CSupply          string  `json:"csupply,omitempty"`
	TSupply          string  `json:"tsupply,omitempty"`
	MSupply          string  `json:"msupply,omitempty"`
}

// ListMarketTickers returns a page of all tracked coins with
// price/market-cap/volume data. params may be nil.
func (s *BlockchainDataService) ListMarketTickers(ctx context.Context, params *MarketTickersParams, opts ...RequestOption) (*MarketTickers, error) {
	q := newQuery()
	if params != nil {
		q.intPtr("start", params.Start).intPtr("limit", params.Limit)
	}
	return doJSON[MarketTickers](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/blockchain/data/market/tickers",
		query:     q.values(),
		retryable: true,
	}, opts)
}

// GetMarketTickers returns price/market data for one coin id, or a
// comma-separated list of ids such as "90,80". Unknown ids yield an empty
// slice rather than an error.
func (s *BlockchainDataService) GetMarketTickers(ctx context.Context, ids string, opts ...RequestOption) ([]MarketTicker, error) {
	out, err := doJSON[[]MarketTicker](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/market/tickers/%s", ids),
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// MarketMoversParams configures GetMarketMovers.
type MarketMoversParams struct {
	// Sort is passed through unvalidated; results are ranked by 24-hour
	// change regardless.
	Sort string
}

// MarketMovers holds the top gaining and losing coins.
type MarketMovers struct {
	Data MarketMoversData `json:"data"`
}

// MarketMoversData holds the winners and losers lists.
type MarketMoversData struct {
	Winners []MarketMover `json:"winners"`
	Losers  []MarketMover `json:"losers"`
}

// MarketMover is one entry of MarketMoversData. The API reference documents
// these fields by example only.
type MarketMover struct {
	ID               string `json:"id"`
	Symbol           string `json:"symbol"`
	Name             string `json:"name"`
	PriceUSD         string `json:"price_usd"`
	PercentChange24h string `json:"percent_change_24h"`
}

// GetMarketMovers returns the top gaining and losing coins by 24-hour change.
// params may be nil.
func (s *BlockchainDataService) GetMarketMovers(ctx context.Context, params *MarketMoversParams, opts ...RequestOption) (*MarketMovers, error) {
	q := newQuery()
	if params != nil {
		q.str("sort", params.Sort)
	}
	return doJSON[MarketMovers](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/blockchain/data/market/movers",
		query:     q.values(),
		retryable: true,
	}, opts)
}

// CoinInfo is metadata for one coin.
type CoinInfo struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	NameID   string `json:"nameid"`
	Website  string `json:"website"`
	Twitter  string `json:"twitter"`
	Explorer string `json:"explorer"`
	Logo     string `json:"logo"`
	// Ath is the all-time high price, in USD.
	Ath            float64 `json:"ath"`
	Rank           int64   `json:"rank"`
	AthDate        string  `json:"ath_date"`
	CSupply        string  `json:"csupply"`
	TSupply        string  `json:"tsupply"`
	MSupply        string  `json:"msupply"`
	StartDate      *string `json:"startdate"`
	Platform       *string `json:"platform"`
	FirstPrice     float64 `json:"first_price"`
	FirstPriceDate string  `json:"first_price_date"`
}

// GetCoinInfo returns metadata for one coin: logo, website, all-time high,
// supply, launch date. The API wraps the result in a one-element array,
// which is returned as-is.
func (s *BlockchainDataService) GetCoinInfo(ctx context.Context, id string, opts ...RequestOption) ([]CoinInfo, error) {
	out, err := doJSON[[]CoinInfo](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/market/coin/%s/info", id),
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// OHLCVCandle is a daily candle: [unixTimestamp, open, high, low, close, volume].
type OHLCVCandle [6]float64

// Time returns the candle's Unix timestamp, in seconds.
func (c OHLCVCandle) Time() int64 { return int64(c[0]) }

// Open returns the opening price.
func (c OHLCVCandle) Open() float64 { return c[1] }

// High returns the high price.
func (c OHLCVCandle) High() float64 { return c[2] }

// Low returns the low price.
func (c OHLCVCandle) Low() float64 { return c[3] }

// Close returns the closing price.
func (c OHLCVCandle) Close() float64 { return c[4] }

// Volume returns the traded volume.
func (c OHLCVCandle) Volume() float64 { return c[5] }

// GetCoinOHLCV returns roughly 365 days of daily OHLCV candles for a coin.
func (s *BlockchainDataService) GetCoinOHLCV(ctx context.Context, id string, opts ...RequestOption) ([]OHLCVCandle, error) {
	out, err := doJSON[[]OHLCVCandle](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/market/coin/%s/ohlcv", id),
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// CoinMarket is one exchange market trading a coin.
type CoinMarket struct {
	// Name is the exchange name.
	Name      string  `json:"name"`
	Base      string  `json:"base"`
	Quote     string  `json:"quote"`
	Price     float64 `json:"price"`
	PriceUSD  float64 `json:"price_usd"`
	Volume    float64 `json:"volume"`
	VolumeUSD float64 `json:"volume_usd"`
	Time      int64   `json:"time"`
}

// GetCoinMarkets returns the top exchange markets currently trading a coin.
func (s *BlockchainDataService) GetCoinMarkets(ctx context.Context, id string, opts ...RequestOption) ([]CoinMarket, error) {
	out, err := doJSON[[]CoinMarket](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/market/coin/%s/markets", id),
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// CoinSocial holds Reddit and Twitter stats for a coin. Individual metrics
// are frequently nil when no data is available.
type CoinSocial struct {
	Reddit  RedditStats  `json:"reddit"`
	Twitter TwitterStats `json:"twitter"`
}

// RedditStats are a coin's Reddit stats.
type RedditStats struct {
	AvgActiveUsers *float64 `json:"avg_active_users"`
	Subscribers    *float64 `json:"subscribers"`
}

// TwitterStats are a coin's Twitter stats.
type TwitterStats struct {
	FollowersCount *float64 `json:"followers_count"`
	StatusCount    *float64 `json:"status_count"`
}

// GetCoinSocial returns Reddit/Twitter follower and activity stats for a coin.
func (s *BlockchainDataService) GetCoinSocial(ctx context.Context, id string, opts ...RequestOption) (*CoinSocial, error) {
	return doJSON[CoinSocial](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/market/coin/%s/social", id),
		retryable: true,
	}, opts)
}

// Exchange is a tracked exchange.
type Exchange struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	NameID      string  `json:"name_id"`
	VolumeUSD   float64 `json:"volume_usd"`
	ActivePairs int64   `json:"active_pairs"`
	URL         string  `json:"url"`
	Country     string  `json:"country"`
}

// ListExchanges returns every tracked exchange, keyed by exchange id.
func (s *BlockchainDataService) ListExchanges(ctx context.Context, opts ...RequestOption) (map[string]Exchange, error) {
	out, err := doJSON[map[string]Exchange](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      "/api/v1/blockchain/data/market/exchanges",
		retryable: true,
	}, opts)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ExchangeDetail is one exchange's metadata and top trading pairs.
type ExchangeDetail struct {
	// Info is the exchange metadata, which the API keys as "0".
	Info  ExchangeInfo   `json:"0"`
	Pairs []ExchangePair `json:"pairs"`
}

// ExchangeInfo is an exchange's metadata.
type ExchangeInfo struct {
	Name     string  `json:"name"`
	DateLive *string `json:"date_live"`
	URL      string  `json:"url"`
}

// ExchangePair is one trading pair on an exchange.
type ExchangePair struct {
	Base     string  `json:"base"`
	Quote    string  `json:"quote"`
	Volume   float64 `json:"volume"`
	Price    float64 `json:"price"`
	PriceUSD float64 `json:"price_usd"`
	Time     int64   `json:"time"`
}

// GetExchange returns one exchange's metadata and its current top trading pairs.
func (s *BlockchainDataService) GetExchange(ctx context.Context, id string, opts ...RequestOption) (*ExchangeDetail, error) {
	return doJSON[ExchangeDetail](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/market/exchanges/%s", id),
		retryable: true,
	}, opts)
}

// ---------------------------------------------------------------------------
// shared helpers for raw-JSON union types
// ---------------------------------------------------------------------------

func decodeRaw[T any](raw json.RawMessage) (*T, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("cryptures: empty response body")
	}
	out := new(T)
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

func marshalRaw(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return raw, nil
}
