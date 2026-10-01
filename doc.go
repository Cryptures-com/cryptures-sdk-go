// Package cryptures is the official Go client for the Cryptures API:
// blockchain infrastructure, crypto cards, and compliance in one client.
//
// Create a client with your project API key and call operations through the
// domain-grouped services:
//
//	client := cryptures.NewClient(os.Getenv("CRYPTURES_API_KEY"))
//
//	bal, err := client.Blockchain.Data.GetBalance(ctx, "ETH", "0x...")
//	card, err := client.Card.Cards.Create(ctx, &cryptures.CreateCardRequest{...})
//	sess, err := client.Compliance.Sessions.Create(ctx, &cryptures.CreateSessionRequest{...})
//
// # Authentication
//
// Every request carries the API key in the x-api-key header.
//
// # Errors
//
// Any non-2xx response is returned as an *APIError carrying the HTTP status,
// the machine-readable error code, the message, and the request id. Use
// errors.As to inspect it. Transport failures and response-decoding failures
// are returned as ordinary wrapped errors.
//
// # Retries
//
// Requests that are safe to repeat (reads, idempotent writes, and POSTs sent
// with an idempotency key) are retried on network errors and 5xx responses
// with exponential backoff and jitter, up to WithMaxRetries times (default
// 3). 4xx responses are never retried. Requests whose repetition could
// duplicate a side effect, such as broadcasting a transaction, creating or
// funding a card, or running a billed screening without an idempotency key,
// are never retried automatically.
//
// # Response shapes that vary by chain
//
// A few blockchain operations return a different JSON shape depending on the
// chain. Those results (Balance, TransactionHistory, Transaction, Block) keep
// the raw JSON and expose one typed accessor per documented shape, such as
// Balance.AsUTXO or Block.AsEVM.
//
// Full API reference: https://docs.cryptures.com/
package cryptures
