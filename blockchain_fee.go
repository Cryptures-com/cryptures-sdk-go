package cryptures

import (
	"context"
	"errors"
	"net/http"
)

// BlockchainFeeService estimates network fees.
type BlockchainFeeService struct {
	client *Client
}

// FeeEstimate holds recommended fee tiers. For BTC, LTC, and DOGE the tiers
// are in the chain's smallest unit per byte; ETH additionally returns BaseFee
// (in wei).
type FeeEstimate struct {
	Slow   float64 `json:"slow"`
	Medium float64 `json:"medium"`
	Fast   float64 `json:"fast"`
	// BaseFee is set for ETH only.
	BaseFee *float64 `json:"baseFee,omitempty"`
	Block   int64    `json:"block,omitempty"`
	// Time is an RFC 3339 timestamp.
	Time string `json:"time,omitempty"`
}

// Get returns recommended fee tiers (slow/medium/fast) for chain. Only BTC,
// ETH, LTC, and DOGE are supported. Estimates are cached for about 20
// seconds; re-read them close to when you build a transaction.
func (s *BlockchainFeeService) Get(ctx context.Context, chain string, opts ...RequestOption) (*FeeEstimate, error) {
	return doJSON[FeeEstimate](ctx, s.client, &requestSpec{
		method:    http.MethodGet,
		path:      pathf("/api/v1/blockchain/data/fee/%s", chain),
		route:     "/api/v1/blockchain/data/fee/{chain}",
		retryable: true,
	}, opts)
}

// EstimateGasRequest describes the transfer to price.
type EstimateGasRequest struct {
	// From is the sender address.
	From string `json:"from"`
	// To is the recipient address.
	To string `json:"to"`
	// Amount is a decimal string in the chain's native unit.
	Amount string `json:"amount"`
	// Data is optional raw transaction data, for a contract call.
	Data string `json:"data,omitempty"`
	// ContractAddress is set when pricing a token transfer.
	ContractAddress string `json:"contractAddress,omitempty"`
}

// GasEstimate is the estimated gas for a transfer.
type GasEstimate struct {
	// GasPrice is in the chain's smallest unit (e.g. wei), as a decimal string.
	GasPrice string `json:"gasPrice"`
	// GasLimit is the estimated gas units, as a decimal string.
	GasLimit string `json:"gasLimit"`
}

// EstimateGas estimates gas price and limit for a specific transfer. Only
// BNB, AVAX, OP, BASE, CELO, FTM, and MATIC are supported (use Get for ETH).
// This operation is never cached.
func (s *BlockchainFeeService) EstimateGas(ctx context.Context, chain string, req *EstimateGasRequest, opts ...RequestOption) (*GasEstimate, error) {
	if req == nil {
		return nil, errors.New("cryptures: EstimateGas: nil request")
	}
	return doJSON[GasEstimate](ctx, s.client, &requestSpec{
		method:    http.MethodPost,
		path:      pathf("/api/v1/blockchain/data/fee/gas/%s", chain),
		route:     "/api/v1/blockchain/data/fee/gas/{chain}",
		body:      req,
		retryable: true, // read-only estimate
	}, opts)
}
