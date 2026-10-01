package cryptures

import (
	"context"
	"errors"
	"net/http"
)

// BlockchainContractsService deploys fungible token contracts and mints or
// burns tokens on them. Each call signs with the supplied private key and
// broadcasts a real transaction, so none is retried automatically: a 500 does
// not mean the transaction was not broadcast. Check the sender's recent
// transactions before resending.
//
// The Chain field in these request bodies uses its own codes, which differ
// from the chain codes used elsewhere in the API: ETH, BSC (= BNB), MATIC,
// AVAX, ETH_BASE (= BASE), ETH_OP (= OP), FTM, ETH_ARB (= ARB), ALGO, CELO,
// SOL. Mint does not accept ALGO or SOL; burn does not accept SOL.
type BlockchainContractsService struct {
	client *Client
}

// DeployTokenRequest deploys a fungible token contract.
type DeployTokenRequest struct {
	// Chain is one of ETH, BSC, MATIC, AVAX, ETH_BASE, ETH_OP, FTM, ETH_ARB,
	// ALGO, CELO, SOL.
	Chain    string `json:"chain"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	TotalCap string `json:"totalCap,omitempty"`
	Supply   string `json:"supply"`
	Digits   *int64 `json:"digits,omitempty"`
	// Address receives the initial supply.
	Address        string `json:"address"`
	FromPrivateKey string `json:"fromPrivateKey"`
}

// DeployToken deploys a new fungible token contract and broadcasts the
// deployment transaction. Retrieve the deployed contract address afterwards
// with Blockchain.Operations.RPC (eth_getTransactionReceipt on EVM chains).
func (s *BlockchainContractsService) DeployToken(ctx context.Context, req *DeployTokenRequest, opts ...RequestOption) (*TxIDResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: DeployToken: nil request")
	}
	return doJSON[TxIDResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   "/api/v1/blockchain/operations/contract/token/deploy",
		body:   req,
	}, opts)
}

// MintTokenRequest mints additional tokens on a deployed contract.
type MintTokenRequest struct {
	// Chain is one of ETH, BSC, MATIC, AVAX, ETH_BASE, ETH_OP, FTM, ETH_ARB, CELO.
	Chain           string `json:"chain"`
	ContractAddress string `json:"contractAddress"`
	Amount          string `json:"amount"`
	To              string `json:"to"`
	FromPrivateKey  string `json:"fromPrivateKey"`
}

// MintToken mints additional tokens on an already-deployed contract.
func (s *BlockchainContractsService) MintToken(ctx context.Context, req *MintTokenRequest, opts ...RequestOption) (*TxIDResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: MintToken: nil request")
	}
	return doJSON[TxIDResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   "/api/v1/blockchain/operations/contract/token/mint",
		body:   req,
	}, opts)
}

// BurnTokenRequest burns tokens on a deployed contract.
type BurnTokenRequest struct {
	// Chain is one of ETH, BSC, MATIC, AVAX, ETH_BASE, ETH_OP, FTM, ETH_ARB,
	// ALGO, CELO.
	Chain           string `json:"chain"`
	ContractAddress string `json:"contractAddress"`
	Amount          string `json:"amount"`
	FromPrivateKey  string `json:"fromPrivateKey"`
}

// BurnToken burns tokens on an already-deployed contract.
func (s *BlockchainContractsService) BurnToken(ctx context.Context, req *BurnTokenRequest, opts ...RequestOption) (*TxIDResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: BurnToken: nil request")
	}
	return doJSON[TxIDResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   "/api/v1/blockchain/operations/contract/token/burn",
		body:   req,
	}, opts)
}
