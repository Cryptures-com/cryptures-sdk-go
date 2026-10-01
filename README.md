# Cryptures Go SDK

Official Go SDK for the Cryptures API — blockchain infrastructure, crypto cards, and compliance in one client.

[![CI](https://github.com/Cryptures-com/cryptures-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/Cryptures-com/cryptures-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Cryptures-com/cryptures-sdk-go.svg)](https://pkg.go.dev/github.com/Cryptures-com/cryptures-sdk-go)

- Typed requests and responses for every API operation
- Context-aware: every call takes a `context.Context`
- Automatic retries with exponential backoff for requests that are safe to repeat
- Idempotency keys for billed screenings
- No dependencies outside the Go standard library

## Requirements

Go 1.21 or later.

## Installation

```sh
go get github.com/Cryptures-com/cryptures-sdk-go
```

Go modules are fetched straight from this public GitHub repository, and versions come from its git tags (for example `v0.1.0`). Unlike npm or PyPI packages, there is no separate registry or publish step: tagging a release here is what makes it installable.

## Authentication

Every request is authenticated with your project's API key, sent as the `x-api-key` header. Keep the key out of source code, for example in an environment variable:

```go
client := cryptures.NewClient(os.Getenv("CRYPTURES_API_KEY"))
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	cryptures "github.com/Cryptures-com/cryptures-sdk-go"
)

func main() {
	client := cryptures.NewClient(os.Getenv("CRYPTURES_API_KEY"))
	ctx := context.Background()

	rate, err := client.Blockchain.Data.GetExchangeRate(ctx, "BTC", &cryptures.ExchangeRateParams{BasePair: "USD"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("1 BTC = %s %s\n", rate.Value, rate.BasePair)
}
```

## Usage

Operations are grouped by domain, mirroring the [API reference](https://docs.cryptures.com/):

| Service | Covers |
| --- | --- |
| `client.Blockchain.Data` | Balances, batch balances, transaction history, portfolios, historical balances, TRC-20 transfers, address screening, exchange rates, Fear & Greed index, market data |
| `client.Blockchain.Operations` | Build/sign/broadcast transactions, broadcast signed transactions, JSON-RPC gateway |
| `client.Blockchain.Wallet` | HD wallet generation, address and private-key derivation |
| `client.Blockchain.Contracts` | Fungible token deploy, mint, burn |
| `client.Blockchain.Fee` | Network fee tiers, EVM gas estimation |
| `client.Blockchain.Lookups` | Transactions by hash, blocks, token metadata, UTXOs |
| `client.Blockchain.NFT` | Collection listings, token owners |
| `client.Blockchain.Storage` | IPFS uploads |
| `client.Card.Cards` | Create, get, list, fund, withdraw, set PIN, block, unblock, terminate, transactions, per-card tags, product catalog |
| `client.Card.Balance` | Project USD balance and ledger |
| `client.Card.Tags` | Card tag management |
| `client.Card.Reports` | Funding/withdrawal summary report |
| `client.Card.Webhooks` | Card webhook registration and status |
| `client.Compliance.Sessions` | KYC/KYB sessions, presets, documents, manual decisions, report PDFs |
| `client.Compliance.Screening` | AML screening, wallet screening |
| `client.Compliance.Monitoring` | Ongoing AML monitoring |
| `client.Compliance.Webhooks` | Compliance webhook registration and status |

### Blockchain

```go
bal, err := client.Blockchain.Data.GetBalance(ctx, "BTC", "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")
if err != nil {
	log.Fatal(err)
}
btc, err := bal.AsUTXO() // BTC, LTC and DOGE share this shape
if err != nil {
	log.Fatal(err)
}
fmt.Println("confirmed:", btc.Balance, "pending in:", btc.IncomingPending)
```

A few blockchain operations return a different JSON shape depending on the chain. Their results (`Balance`, `TransactionHistory`, `Transaction`, `Block`) keep the raw JSON in `Raw` and expose one typed accessor per documented shape: for example `AsSimple`, `AsUTXO`, `AsTron`, `AsCelo`, `AsCardano`, `AsStellar` and `AsXRP` on `Balance`. Call the accessor that matches the chain you queried; the type docs list which chains use which shape.

Sending a transaction takes the request type for the chain's family:

```go
tx, err := client.Blockchain.Operations.Send(ctx, "ETH", &cryptures.SendEVMRequest{
	Currency:       "ETH",
	Amount:         "0.001",
	To:             "0x5041F19dC1659E33848cc0f77cbF7447de562917",
	FromPrivateKey: os.Getenv("SENDER_PRIVATE_KEY"),
})
```

### Card

```go
products, err := client.Card.Cards.ListProducts(ctx)
if err != nil {
	log.Fatal(err)
}

card, err := client.Card.Cards.Create(ctx, &cryptures.CreateCardRequest{
	ProductCode: products.Products[0].ProductCode,
	FirstName:   "Jane",
	LastName:    "Doe",
	Email:       "jane@example.com",
	InitialLoad: cryptures.Float64(20), // leave nil when the product has DisallowInitialLoad
	Tags:        []string{"Marketing"},
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("created card", card.Data.CardID, "ending in", card.Data.LastFour)
```

### Compliance

```go
session, err := client.Compliance.Sessions.Create(ctx, &cryptures.CreateSessionRequest{
	PresetID:       "preset_af363e69-9490-4fa3-a60e-9d3e07e385bc", // from Sessions.ListPresets
	ExternalUserID: "user_42",
	Callback:       "https://example.com/verification/done",
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("redirect the user to", session.URL)
```

## Error handling

Every non-2xx response is returned as a `*cryptures.APIError` with the HTTP status, the machine-readable error code, the message, and the request id. Inspect it with `errors.As`:

```go
_, err := client.Card.Cards.Fund(ctx, "card_a1b2c3d4", 25)

var apiErr *cryptures.APIError
if errors.As(err, &apiErr) {
	switch apiErr.Code {
	case "insufficient_balance":
		// top up the project balance; nothing was charged
	case "forbidden_card":
		// the card does not belong to this project
	default:
		log.Printf("HTTP %d %s: %s (request %s)", apiErr.StatusCode, apiErr.Code, apiErr.Message, apiErr.RequestID)
	}
} else if err != nil {
	// network failure, timeout, or cancelled context
}
```

Card operations forward the card issuer's own error body once Cryptures' checks have passed. Those still arrive as an `*APIError` (with `Code` and `Message` filled from the issuer's body); `apiErr.IsProviderError()` tells you which kind it is, and `apiErr.Body` always holds the raw response.

## Retries and idempotency

Requests that are safe to repeat are retried on network errors and 5xx responses with exponential backoff (250 ms base, with jitter, honouring `Retry-After`), up to 3 retries by default. 4xx responses are never retried.

Requests whose repetition could duplicate a side effect are **never** retried automatically: sending or broadcasting a transaction, deploying/minting/burning tokens, the JSON-RPC gateway, creating, funding or withdrawing from a card, card state changes, creating a verification session, and billed screenings sent without an idempotency key. The API documents that a timed-out broadcast can still confirm on chain, so check before you resend.

AML and wallet screenings accept an idempotency key. With one set, a repeated call returns the original result without screening or charging again, and the SDK retries those calls automatically:

```go
var meta cryptures.ResponseMetadata
check, err := client.Compliance.Screening.AMLCheck(ctx, &cryptures.AMLCheckRequest{
	ExternalUserID: "user_42",
	FullName:       "Jane Doe",
	IdempotencyKey: "aml-user_42-2026-10-01",
}, cryptures.WithResponseMetadata(&meta))
// meta.IdempotentReplay is true when the result is a replay.
```

## Pagination

Cursor-paginated lists can be read one page at a time (`NextCursor` is empty on the last page) or walked with an iterator:

```go
it := client.Compliance.Sessions.ListAutoPaging(ctx, &cryptures.ListSessionsParams{Status: "In Review"})
for it.Next() {
	fmt.Println(it.Current().SessionID)
}
if err := it.Err(); err != nil {
	log.Fatal(err)
}
```

## Configuration

```go
client := cryptures.NewClient(apiKey,
	cryptures.WithBaseURL("https://api.cryptures.com"), // default
	cryptures.WithTimeout(30*time.Second),              // per attempt; default 30s
	cryptures.WithMaxRetries(3),                        // default 3; 0 disables retries
	cryptures.WithHTTPClient(&http.Client{}),           // custom transport, proxy, tracing
)
```

Per-call options:

- `cryptures.WithHeader(key, value)` adds a header to one request.
- `cryptures.WithResponseMetadata(&meta)` captures the status code, `X-Request-ID`, `X-Relay-Cache`, `Idempotent-Replay` and the full headers of the response.

Use the `context.Context` you pass to each method to bound the total time across retries.

## Documentation

- Full API reference: <https://docs.cryptures.com/>
- Go package documentation: <https://pkg.go.dev/github.com/Cryptures-com/cryptures-sdk-go>
- Cryptures: <https://cryptures.com>

## Contributing

Issues and pull requests are welcome. Please run the same checks as CI before opening a pull request:

```sh
gofmt -l .
go vet ./...
go test -race ./...
```

## License

MIT. See [LICENSE](LICENSE).
