package cryptures_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	cryptures "github.com/Cryptures-com/cryptures-sdk-go"
)

func Example() {
	client := cryptures.NewClient(os.Getenv("CRYPTURES_API_KEY"))
	ctx := context.Background()

	bal, err := client.Blockchain.Data.GetBalance(ctx, "ETH", "0xae680ed83baf08a8028118bd19859f8a0e744cc6")
	if err != nil {
		log.Fatal(err)
	}
	eth, err := bal.AsSimple()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("ETH balance:", eth.Balance)
}

func ExampleNewClient() {
	client := cryptures.NewClient(
		os.Getenv("CRYPTURES_API_KEY"),
		cryptures.WithTimeout(15*time.Second),
		cryptures.WithMaxRetries(5),
	)
	_ = client
}

func ExampleAPIError() {
	client := cryptures.NewClient(os.Getenv("CRYPTURES_API_KEY"))

	_, err := client.Card.Cards.Fund(context.Background(), "card_a1b2c3d4", 25)
	var apiErr *cryptures.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "insufficient_balance":
			log.Printf("top up the project balance first (request %s)", apiErr.RequestID)
		case "forbidden_card":
			log.Printf("card does not belong to this project")
		default:
			log.Printf("API error %d %s: %s", apiErr.StatusCode, apiErr.Code, apiErr.Message)
		}
	} else if err != nil {
		log.Printf("transport error: %v", err)
	}
}

func ExampleIter() {
	client := cryptures.NewClient(os.Getenv("CRYPTURES_API_KEY"))

	it := client.Compliance.Sessions.ListAutoPaging(context.Background(), &cryptures.ListSessionsParams{
		Status: "In Review",
	})
	for it.Next() {
		s := it.Current()
		fmt.Println(s.SessionID, s.ExternalUserID)
	}
	if err := it.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleComplianceScreeningService_CreateWalletScreening() {
	client := cryptures.NewClient(os.Getenv("CRYPTURES_API_KEY"))

	var meta cryptures.ResponseMetadata
	screening, err := client.Compliance.Screening.CreateWalletScreening(context.Background(),
		&cryptures.WalletScreeningRequest{
			Address:        "0x0000000000000000000000000000000000000000",
			Chain:          "ETH",
			IdempotencyKey: "screen-order-1234", // makes retries safe and free
		},
		cryptures.WithResponseMetadata(&meta),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(screening.Result.Severity, "replayed:", meta.IdempotentReplay)
}
