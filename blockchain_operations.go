package cryptures

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// BlockchainOperationsService builds, signs, and broadcasts transactions, and
// forwards raw JSON-RPC requests to chain nodes.
//
// Send and Broadcast are never retried automatically by this SDK, whatever
// the client's retry setting: the API gives up waiting on a slow chain after
// about five seconds and answers 500, but a transaction that was already
// broadcast can still confirm afterwards. Retrying blindly risks spending
// twice. Confirm the transaction is absent (Blockchain.Lookups.GetTransaction
// or Blockchain.Data.GetTransactionHistory) before resending.
type BlockchainOperationsService struct {
	client *Client
}

// ---------------------------------------------------------------------------
// tx.send
// ---------------------------------------------------------------------------

// SendTransactionRequest is the request body for Send. Its shape depends on
// the chain; use the type for the chain's group:
//
//   - ETH, MATIC, BNB, AVAX, ARB, OP, BASE, CELO, FTM: *SendEVMRequest
//   - BTC, LTC, DOGE, BCH: *SendUTXORequest
//   - ADA: *SendCardanoRequest
//   - TRON, SOL, ALGO, VET: *SendAccountRequest
//   - XRP: *SendXRPRequest
//   - XLM: *SendXLMRequest
//   - EGLD: *SendEGLDRequest
//
// RawSendRequest sends a caller-built JSON body unchanged, for any field the
// typed requests do not cover.
type SendTransactionRequest interface {
	isSendTransactionRequest()
}

// GasFee sets explicit gas parameters.
type GasFee struct {
	GasLimit string `json:"gasLimit"`
	// GasPrice is in Gwei for EVM chains.
	GasPrice string `json:"gasPrice"`
}

// SendEVMRequest sends a native-currency transfer on an EVM-style chain
// (ETH, MATIC, BNB, AVAX, ARB, OP, BASE, CELO, FTM).
type SendEVMRequest struct {
	// Currency is the chain's native currency code, e.g. "ETH", "MATIC", "CELO".
	Currency string `json:"currency"`
	// Amount is in the native currency (not the smallest unit).
	Amount string `json:"amount"`
	// To is the recipient address.
	To string `json:"to"`
	// Fee is optional; gas is estimated automatically when nil.
	Fee *GasFee `json:"fee,omitempty"`
	// Nonce is optional; it is looked up automatically when nil.
	Nonce *int64 `json:"nonce,omitempty"`
	// FromPrivateKey is the sender's private key (required).
	FromPrivateKey string `json:"fromPrivateKey"`
	// Data is optional hex-encoded contract call data.
	Data string `json:"data,omitempty"`
}

// SendUTXORequest sends a transaction on a UTXO chain (BTC, LTC, DOGE, BCH).
type SendUTXORequest struct {
	FromUTXO []UTXOInput  `json:"fromUTXO"`
	To       []UTXOOutput `json:"to"`
	// Fee is optional; it is deducted from outputs when empty.
	Fee string `json:"fee,omitempty"`
	// ChangeAddress optionally receives remaining funds.
	ChangeAddress string `json:"changeAddress,omitempty"`
}

// UTXOInput is a UTXO to spend, with the private key that signs it.
type UTXOInput struct {
	TxHash     string `json:"txHash"`
	Index      int64  `json:"index"`
	PrivateKey string `json:"privateKey"`
}

// UTXOOutput is a transaction output.
type UTXOOutput struct {
	Address string  `json:"address"`
	Value   float64 `json:"value"`
}

// SendCardanoRequest sends an ADA transaction. Set exactly one of
// FromAddress (automatic coin selection) or FromUTXO (explicit UTXOs).
type SendCardanoRequest struct {
	FromAddress []CardanoAddressInput `json:"fromAddress,omitempty"`
	FromUTXO    []CardanoUTXOInput    `json:"fromUTXO,omitempty"`
	To          []UTXOOutput          `json:"to"`
}

// CardanoAddressInput selects funds by address.
type CardanoAddressInput struct {
	Address string `json:"address"`
}

// CardanoUTXOInput selects funds by explicit UTXO.
type CardanoUTXOInput struct {
	TxHash string `json:"txHash"`
	Index  int64  `json:"index"`
}

// SendAccountRequest sends a native transfer on a simple account-based chain
// (TRON, SOL, ALGO, VET).
type SendAccountRequest struct {
	// From is the sender address. Required for SOL; not used by the other chains.
	From           string `json:"from,omitempty"`
	To             string `json:"to"`
	Amount         string `json:"amount"`
	FromPrivateKey string `json:"fromPrivateKey"`
}

// SendXRPRequest sends XRP, or an issued currency when IssuerAccount and
// Token are set.
type SendXRPRequest struct {
	FromAccount    string `json:"fromAccount"`
	To             string `json:"to"`
	Amount         string `json:"amount"`
	FromSecret     string `json:"fromSecret"`
	Fee            string `json:"fee,omitempty"`
	SourceTag      *int64 `json:"sourceTag,omitempty"`
	DestinationTag *int64 `json:"destinationTag,omitempty"`
	// IssuerAccount and Token select an issued (non-native) currency.
	IssuerAccount string `json:"issuerAccount,omitempty"`
	Token         string `json:"token,omitempty"`
}

// SendXLMRequest sends XLM, or an issued asset when Token and IssuerAccount
// are set.
type SendXLMRequest struct {
	FromAccount string `json:"fromAccount"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	FromSecret  string `json:"fromSecret"`
	// Initialize creates the destination account if it does not exist yet.
	Initialize *bool  `json:"initialize,omitempty"`
	Message    string `json:"message,omitempty"`
	// Token and IssuerAccount select an issued (non-native) asset.
	Token         string `json:"token,omitempty"`
	IssuerAccount string `json:"issuerAccount,omitempty"`
}

// SendEGLDRequest sends EGLD (MultiversX). Unlike the EVM chains, the sender
// address is required alongside the private key.
type SendEGLDRequest struct {
	FromPrivateKey string  `json:"fromPrivateKey"`
	From           string  `json:"from"`
	To             string  `json:"to"`
	Amount         string  `json:"amount"`
	Fee            *GasFee `json:"fee,omitempty"`
	Data           string  `json:"data,omitempty"`
}

// RawSendRequest is a caller-built JSON request body, sent unchanged. Use it
// only for chain-specific fields the typed requests do not model.
type RawSendRequest json.RawMessage

// MarshalJSON implements json.Marshaler.
func (r RawSendRequest) MarshalJSON() ([]byte, error) {
	if !json.Valid(r) {
		return nil, errors.New("cryptures: RawSendRequest is not valid JSON")
	}
	return r, nil
}

func (*SendEVMRequest) isSendTransactionRequest()     {}
func (*SendUTXORequest) isSendTransactionRequest()    {}
func (*SendCardanoRequest) isSendTransactionRequest() {}
func (*SendAccountRequest) isSendTransactionRequest() {}
func (*SendXRPRequest) isSendTransactionRequest()     {}
func (*SendXLMRequest) isSendTransactionRequest()     {}
func (*SendEGLDRequest) isSendTransactionRequest()    {}
func (RawSendRequest) isSendTransactionRequest()      {}

// Send builds a transaction from raw ingredients, signs it with the supplied
// private key or secret, and broadcasts it on chain. It is never retried
// automatically; see BlockchainOperationsService.
func (s *BlockchainOperationsService) Send(ctx context.Context, chain string, req SendTransactionRequest, opts ...RequestOption) (*TxIDResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: Send: nil request")
	}
	return doJSON[TxIDResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   pathf("/api/v1/blockchain/operations/transaction/%s/send", chain),
		body:   req,
	}, opts)
}

// ---------------------------------------------------------------------------
// tx.broadcast
// ---------------------------------------------------------------------------

// BroadcastRequest carries an already-signed raw transaction.
type BroadcastRequest struct {
	// TxData is the raw signed transaction (hex or the chain's native
	// serialization), 1-500,000 characters.
	TxData string `json:"txData"`
}

// Broadcast broadcasts an already-built and signed raw transaction. It is
// never retried automatically; see BlockchainOperationsService.
func (s *BlockchainOperationsService) Broadcast(ctx context.Context, chain string, req *BroadcastRequest, opts ...RequestOption) (*TxIDResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: Broadcast: nil request")
	}
	return doJSON[TxIDResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   pathf("/api/v1/blockchain/operations/transaction/%s/broadcast", chain),
		body:   req,
	}, opts)
}

// ---------------------------------------------------------------------------
// rpc.gateway
// ---------------------------------------------------------------------------

// RPCRequest is a JSON-RPC 2.0 request forwarded to the chain's node as-is.
type RPCRequest struct {
	// JSONRPC defaults to "2.0" when empty.
	JSONRPC string `json:"jsonrpc"`
	// ID is the request id (an integer or a string), echoed back on the response.
	ID any `json:"id,omitempty"`
	// Method is the node's RPC method name; the available set varies by chain.
	Method string `json:"method"`
	// Params are the method parameters, typically a slice. Their shape
	// depends on Method.
	Params any `json:"params,omitempty"`
}

// RPCResponse is a JSON-RPC 2.0 response, returned as-is from the node. An
// RPC-level failure (bad method, bad params, reverted eth_call) arrives with
// HTTP 200 and a non-nil Error, so always check Error.
type RPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	// ID is the request id echoed back (an integer or a string).
	ID json.RawMessage `json:"id,omitempty"`
	// Result is present on success; its shape depends on the method.
	Result json.RawMessage `json:"result,omitempty"`
	// Error is present on an RPC-level error instead of Result.
	Error *RPCError `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	return fmt.Sprintf("cryptures: JSON-RPC error %d: %s", e.Code, e.Message)
}

// DecodeResult unmarshals Result into v. It returns the RPC error instead if
// the node reported one.
func (r *RPCResponse) DecodeResult(v any) error {
	if r.Error != nil {
		return r.Error
	}
	if len(r.Result) == 0 {
		return errors.New("cryptures: JSON-RPC response has no result")
	}
	return json.Unmarshal(r.Result, v)
}

// RPC forwards a JSON-RPC 2.0 request to the given chain's node and returns
// the node's response. For a state-changing contract call, sign locally and
// use eth_sendRawTransaction; eth_sendTransaction cannot work against these
// shared nodes. RPC calls are not retried automatically, since the method may
// have side effects.
func (s *BlockchainOperationsService) RPC(ctx context.Context, chain string, req *RPCRequest, opts ...RequestOption) (*RPCResponse, error) {
	if req == nil {
		return nil, errors.New("cryptures: RPC: nil request")
	}
	body := *req
	if body.JSONRPC == "" {
		body.JSONRPC = "2.0"
	}
	return doJSON[RPCResponse](ctx, s.client, &requestSpec{
		method: http.MethodPost,
		path:   pathf("/api/v1/blockchain/operations/rpc/%s", chain),
		body:   &body,
	}, opts)
}
