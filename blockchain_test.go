package cryptures

import (
	"context"
	"encoding/json"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func TestBlockchainDataEndpoints(t *testing.T) { runEndpointCases(t, blockchainDataCases) }

var blockchainDataCases = []endpointCase{
	{
		name: "Data.GetBalance",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetBalance(ctx, "BTC", "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/balance/BTC/1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
		response: `{"balance":"100","incoming":"150","outgoing":"50","incomingPending":"0","outgoingPending":"0"}`,
		check: func(t *testing.T, got any) {
			u, err := got.(*Balance).AsUTXO()
			if err != nil {
				t.Fatal(err)
			}
			eq(t, "balance", u.Balance, "100")
			eq(t, "incoming", u.Incoming, "150")
			eq(t, "outgoingPending", u.OutgoingPending, "0")
		},
	},
	{
		name: "Data.GetBalances",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetBalances(ctx, &BalanceBatchRequest{Chain: "ethereum-mainnet", Addresses: "0xabc,0xdef", Unix: Int64(1758700000)})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/data/balance/batch",
		body:     `{"chain":"ethereum-mainnet","addresses":"0xabc,0xdef","unix":1758700000}`,
		response: `{"result":[{"chain":"ethereum-mainnet","address":"0xabc","balance":"1.5","lastUpdatedBlockNumber":21000000,"type":"native"}],"prevPage":"","nextPage":""}`,
		check: func(t *testing.T, got any) {
			r := got.(*BalanceBatchResponse)
			eq(t, "len", len(r.Result), 1)
			eq(t, "balance", r.Result[0].Balance, "1.5")
			eq(t, "block", r.Result[0].LastUpdatedBlockNumber, int64(21000000))
			eq(t, "type", r.Result[0].Type, "native")
		},
	},
	{
		name: "Data.ListTokenTransfers",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.ListTokenTransfers(ctx, "TRON", "TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf", &TokenTransfersParams{
				Next: "cur1", OnlyConfirmed: Bool(true), OnlyTo: Bool(false), MinTimestamp: Int64(10), ContractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/token-transfers/TRON/TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf",
		query:    q("next", "cur1", "onlyConfirmed", "true", "onlyTo", "false", "minTimestamp", "10", "contractAddress", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"),
		response: `{"transactions":[{"txID":"8ea0","tokenInfo":{"symbol":"USDT","address":"TR7N","decimals":6,"name":"Tether USD"},"from":"TVg","to":"TXY","type":"Transfer","value":"1994194"}],"next":"cur2"}`,
		check: func(t *testing.T, got any) {
			r := got.(*TokenTransfers)
			eq(t, "next", r.Next, "cur2")
			eq(t, "symbol", r.Transactions[0].TokenInfo.Symbol, "USDT")
			eq(t, "decimals", r.Transactions[0].TokenInfo.Decimals, 6)
			eq(t, "value", r.Transactions[0].Value, "1994194")
		},
	},
	{
		name: "Data.GetTransactionHistory",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetTransactionHistory(ctx, "BNB", "0xdef1", &TransactionHistoryParams{PageSize: Int64(50), Offset: Int64(0)})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/history/BNB/0xdef1",
		query:    q("pageSize", "50", "offset", "0"),
		response: `{"result":[{"chain":"bsc-mainnet","hash":"0x5494","address":"0xdef1","counterAddress":"0x0d4a","blockNumber":16819465,"transactionType":"fungible","transactionSubtype":"incoming","amount":"0.99","timestamp":1678715303000}],"prevPage":"","nextPage":""}`,
		check: func(t *testing.T, got any) {
			u, err := got.(*TransactionHistory).AsUnified()
			if err != nil {
				t.Fatal(err)
			}
			eq(t, "subtype", u.Result[0].TransactionSubtype, "incoming")
			eq(t, "counter", u.Result[0].CounterAddress, "0x0d4a")
			eq(t, "ts", u.Result[0].Timestamp, int64(1678715303000))
		},
	},
	{
		name: "Data.GetPortfolio",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetPortfolio(ctx, "ETH", "0xae68", &PortfolioParams{TokenTypes: "native,fungible", ExcludeMetadata: Bool(true), PageSize: Int64(20)})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/portfolio/ETH/0xae68",
		query:    q("tokenTypes", "native,fungible", "excludeMetadata", "true", "pageSize", "20"),
		response: `{"result":[{"chain":"ethereum-mainnet","address":"0xae68","balance":"283.3","denominatedBalance":"283","decimals":18,"tokenAddress":"0x45dd","type":"fungible"}],"prevPage":"","nextPage":"p2"}`,
		check: func(t *testing.T, got any) {
			r := got.(*Portfolio)
			eq(t, "type", r.Result[0].Type, "fungible")
			eq(t, "decimals", r.Result[0].Decimals, 18)
			eq(t, "token", r.Result[0].TokenAddress, "0x45dd")
			eq(t, "next", r.NextPage, "p2")
		},
	},
	{
		name: "Data.GetBalanceHistory",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetBalanceHistory(ctx, "BTC", "1A1z", &BalanceHistoryParams{Time: "2026-01-01T00:00:00Z"})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/balance-history/BTC/1A1z",
		query:    q("time", "2026-01-01T00:00:00Z"),
		response: `{"result":[{"chain":"bitcoin-mainnet","address":"1A1z","balance":"0.5","denominatedBalance":"50000000","decimals":8,"type":"native"}],"prevPage":"","nextPage":""}`,
		check: func(t *testing.T, got any) {
			r := got.(*BalanceHistory)
			eq(t, "balance", r.Result[0].Balance, "0.5")
			eq(t, "denominated", r.Result[0].DenominatedBalance, "50000000")
			eq(t, "decimals", r.Result[0].Decimals, 8)
		},
	},
	{
		name: "Data.CheckAddress",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.CheckAddress(ctx, "0x002bf459dc58584d58886169ea0e80f3ca95ffaf")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/security/0x002bf459dc58584d58886169ea0e80f3ca95ffaf",
		response: `{"status":"invalid","address":"0x002b","source":"CryptoScamDB","description":"Trust trading scam site"}`,
		check: func(t *testing.T, got any) {
			r := got.(*AddressCheck)
			eq(t, "status", r.Status, "invalid")
			eq(t, "source", r.Source, "CryptoScamDB")
		},
	},
	{
		name: "Data.GetExchangeRate",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetExchangeRate(ctx, "BTC", &ExchangeRateParams{BasePair: "EUR"})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/rate/BTC",
		query:    q("basePair", "EUR"),
		response: `{"value":"63000.00","basePair":"EUR","id":"BTC","timestamp":1759481315000}`,
		check: func(t *testing.T, got any) {
			r := got.(*ExchangeRate)
			eq(t, "value", r.Value, "63000.00")
			eq(t, "id", r.ID, "BTC")
			eq(t, "ts", r.Timestamp, int64(1759481315000))
		},
	},
	{
		name: "Data.GetExchangeRateByContract",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetExchangeRateByContract(ctx, &ContractExchangeRateParams{Chain: "ethereum-mainnet", ContractAddress: "0xdac1", BasePair: "USD"})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/rate/contract",
		query:    q("chain", "ethereum-mainnet", "contractAddress", "0xdac1", "basePair", "USD"),
		response: `{"value":"0.85","basePair":"USD","timestamp":1759481315000,"chain":"ethereum-mainnet","address":"0xdac1"}`,
		check: func(t *testing.T, got any) {
			r := got.(*ContractExchangeRate)
			eq(t, "value", r.Value, "0.85")
			eq(t, "address", r.Address, "0xdac1")
		},
	},
	{
		name: "Data.GetExchangeRates",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetExchangeRates(ctx, []ExchangeRateBatchItem{{BatchID: "1", Symbol: "BTC", BasePair: "USD"}, {BatchID: "2", Symbol: "ETH"}})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/data/rate/batch",
		body:     `[{"batchId":"1","symbol":"BTC","basePair":"USD"},{"batchId":"2","symbol":"ETH"}]`,
		response: `[{"batchId":"2","symbol":"ETH","value":"1900","basePair":"EUR","timestamp":1,"source":"x"},{"batchId":"1","symbol":"BTC","value":"63000","basePair":"USD","timestamp":2}]`,
		check: func(t *testing.T, got any) {
			r := got.([]ExchangeRateBatchResult)
			eq(t, "len", len(r), 2)
			eq(t, "batchId", r[0].BatchID, "2")
			eq(t, "source", r[0].Source, "x")
			eq(t, "value", r[1].Value, "63000")
		},
	},
	{
		name: "Data.GetFearGreedIndex",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetFearGreedIndex(ctx, &FearGreedParams{Limit: Int64(2), DateFormat: "world"})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/sentiment/fear-greed",
		query:    q("limit", "2", "date_format", "world"),
		response: `{"name":"Fear and Greed Index","data":[{"value":"63","value_classification":"Greed","timestamp":"1788307200","time_until_update":"29641"}],"metadata":{"error":null}}`,
		check: func(t *testing.T, got any) {
			r := got.(*FearGreedIndex)
			eq(t, "class", r.Data[0].ValueClassification, "Greed")
			eq(t, "until", r.Data[0].TimeUntilUpdate, "29641")
			if r.Metadata.Error != nil {
				t.Errorf("metadata.error = %v, want nil", *r.Metadata.Error)
			}
		},
	},
	{
		name: "Data.GetMarketGlobal",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetMarketGlobal(ctx)
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/global",
		response: `[{"coins_count":14993,"active_markets":28840,"total_mcap":2589177854944.69,"total_volume":120828883081.005,"btc_d":"59.50","eth_d":"11.30","mcap_change":"-1.12","volume_change":"10.84","avg_change_percent":"-0.17","volume_ath":344187126190825200,"mcap_ath":33248067879029.688}]`,
		check: func(t *testing.T, got any) {
			r := got.([]MarketGlobal)
			eq(t, "coins", r[0].CoinsCount, int64(14993))
			eq(t, "btc_d", r[0].BtcD, "59.50")
			eq(t, "mcap", r[0].TotalMcap, 2589177854944.69)
		},
	},
	{
		name: "Data.ListMarketAssets",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.ListMarketAssets(ctx)
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/assets",
		response: `{"data":[{"id":"90","symbol":"BTC","name":"Bitcoin","nameid":"bitcoin","rank":1}]}`,
		check: func(t *testing.T, got any) {
			r := got.(*MarketAssets)
			eq(t, "asset", r.Data[0], MarketAsset{ID: "90", Symbol: "BTC", Name: "Bitcoin", NameID: "bitcoin", Rank: 1})
		},
	},
	{
		name: "Data.ListMarketTickers",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.ListMarketTickers(ctx, &MarketTickersParams{Start: Int64(0), Limit: Int64(50)})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/tickers",
		query:    q("start", "0", "limit", "50"),
		response: `{"data":[{"id":"90","symbol":"BTC","name":"Bitcoin","nameid":"bitcoin","rank":1,"price_usd":"77146.66","percent_change_24h":"-0.98","percent_change_1h":"-0.08","percent_change_7d":"-2.38","price_btc":"1.00","market_cap_usd":"1540684444015.40","volume24":26551212429.49,"volume24a":24063575479.22,"csupply":"19970852.00","tsupply":"19970852","msupply":"21000000"}],"info":{"coins_num":14993,"time":1788363842}}`,
		check: func(t *testing.T, got any) {
			r := got.(*MarketTickers)
			eq(t, "price", r.Data[0].PriceUSD, "77146.66")
			eq(t, "vol", r.Data[0].Volume24, 26551212429.49)
			eq(t, "msupply", r.Data[0].MSupply, "21000000")
			eq(t, "info", r.Info, MarketTickerInfo{CoinsNum: 14993, Time: 1788363842})
		},
	},
	{
		name: "Data.GetMarketTickers",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetMarketTickers(ctx, "90,80")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/tickers/90,80",
		response: `[{"id":"90","symbol":"BTC","name":"Bitcoin","price_usd":"64420.92"},{"id":"80","symbol":"ETH","name":"Ethereum","price_usd":"1912.50"}]`,
		check: func(t *testing.T, got any) {
			r := got.([]MarketTicker)
			eq(t, "len", len(r), 2)
			eq(t, "eth", r[1].PriceUSD, "1912.50")
		},
	},
	{
		name: "Data.GetMarketMovers",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetMarketMovers(ctx, &MarketMoversParams{Sort: "24h"})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/movers",
		query:    q("sort", "24h"),
		response: `{"data":{"winners":[{"id":"184771","symbol":"BPX","name":"Black Phoenix","price_usd":"2.81","percent_change_24h":"22504260.24"}],"losers":[{"id":"44735","symbol":"KTON","name":"Darwinia","price_usd":"0.348423","percent_change_24h":"-72.13"}]}}`,
		check: func(t *testing.T, got any) {
			r := got.(*MarketMovers)
			eq(t, "winner", r.Data.Winners[0].Symbol, "BPX")
			eq(t, "loser", r.Data.Losers[0].PercentChange24h, "-72.13")
		},
	},
	{
		name: "Data.GetCoinInfo",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetCoinInfo(ctx, "90")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/coin/90/info",
		response: `[{"id":"90","symbol":"BTC","name":"Bitcoin","nameid":"bitcoin","website":"https://bitcoin.org/","twitter":"@bitcoin","explorer":"https://blockchain.info/","logo":"https://example.com/b.png","ath":126020.77,"rank":1,"ath_date":"2025-10-06T00:00:00Z","csupply":"19970852","tsupply":"19970852","msupply":"21000000","startdate":null,"platform":null,"first_price":134.3975,"first_price_date":"2013-04-28T08:15:17Z"}]`,
		check: func(t *testing.T, got any) {
			r := got.([]CoinInfo)
			eq(t, "ath", r[0].Ath, 126020.77)
			eq(t, "website", r[0].Website, "https://bitcoin.org/")
			if r[0].StartDate != nil || r[0].Platform != nil {
				t.Error("expected nil startdate/platform")
			}
		},
	},
	{
		name: "Data.GetCoinOHLCV",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetCoinOHLCV(ctx, "90")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/coin/90/ohlcv",
		response: `[[1367136917,135.3,135.98,132.1,134.21,0],[1367223317,134.44,147.49,134,144.54,7]]`,
		check: func(t *testing.T, got any) {
			r := got.([]OHLCVCandle)
			eq(t, "len", len(r), 2)
			eq(t, "time", r[0].Time(), int64(1367136917))
			eq(t, "open", r[0].Open(), 135.3)
			eq(t, "high", r[1].High(), 147.49)
			eq(t, "low", r[1].Low(), 134.0)
			eq(t, "close", r[1].Close(), 144.54)
			eq(t, "volume", r[1].Volume(), 7.0)
		},
	},
	{
		name: "Data.GetCoinMarkets",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetCoinMarkets(ctx, "90")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/coin/90/markets",
		response: `[{"name":"Coinone","base":"BTC","quote":"KRW","price":106110000,"price_usd":150301.67,"volume":115.03,"volume_usd":17288919.76,"time":1788363542}]`,
		check: func(t *testing.T, got any) {
			r := got.([]CoinMarket)
			eq(t, "name", r[0].Name, "Coinone")
			eq(t, "quote", r[0].Quote, "KRW")
			eq(t, "time", r[0].Time, int64(1788363542))
		},
	},
	{
		name: "Data.GetCoinSocial",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetCoinSocial(ctx, "90")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/coin/90/social",
		response: `{"reddit":{"avg_active_users":null,"subscribers":8124337},"twitter":{"followers_count":null,"status_count":12}}`,
		check: func(t *testing.T, got any) {
			r := got.(*CoinSocial)
			if r.Reddit.AvgActiveUsers != nil || r.Twitter.FollowersCount != nil {
				t.Error("expected nil metrics")
			}
			eq(t, "subscribers", *r.Reddit.Subscribers, 8124337.0)
			eq(t, "status", *r.Twitter.StatusCount, 12.0)
		},
	},
	{
		name: "Data.ListExchanges",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.ListExchanges(ctx)
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/exchanges",
		response: `{"5":{"id":"5","name":"Binance","name_id":"binance","volume_usd":7216670230.51,"active_pairs":914,"url":"https://www.binance.com","country":"Japan"}}`,
		check: func(t *testing.T, got any) {
			r := got.(map[string]Exchange)
			eq(t, "name", r["5"].Name, "Binance")
			eq(t, "pairs", r["5"].ActivePairs, int64(914))
		},
	},
	{
		name: "Data.GetExchange",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Data.GetExchange(ctx, "5")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/market/exchanges/5",
		response: `{"0":{"name":"Binance","date_live":null,"url":"https://www.binance.com"},"pairs":[{"base":"BTC","quote":"USDT","volume":1087137239.31,"price":77328,"price_usd":77328,"time":1788363439}]}`,
		check: func(t *testing.T, got any) {
			r := got.(*ExchangeDetail)
			eq(t, "name", r.Info.Name, "Binance")
			if r.Info.DateLive != nil {
				t.Error("expected nil date_live")
			}
			eq(t, "quote", r.Pairs[0].Quote, "USDT")
			eq(t, "price", r.Pairs[0].Price, 77328.0)
		},
	},
}

func TestBlockchainOperationsEndpoints(t *testing.T) {
	runEndpointCases(t, blockchainOperationsCases)
}

var blockchainOperationsCases = []endpointCase{
	{
		name: "Operations.Send/EVM",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Operations.Send(ctx, "ETH", &SendEVMRequest{
				Currency: "ETH", Amount: "0.001", To: "0x5041", Fee: &GasFee{GasLimit: "21000", GasPrice: "50"}, Nonce: Int64(0), FromPrivateKey: "PK",
			})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/transaction/ETH/send",
		body:     `{"currency":"ETH","amount":"0.001","to":"0x5041","fee":{"gasLimit":"21000","gasPrice":"50"},"nonce":0,"fromPrivateKey":"PK"}`,
		response: `{"txId":"0x4a19"}`,
		check: func(t *testing.T, got any) {
			eq(t, "txId", got.(*TxIDResponse).TxID, "0x4a19")
		},
	},
	{
		name: "Operations.Send/UTXO",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Operations.Send(ctx, "BTC", &SendUTXORequest{
				FromUTXO: []UTXOInput{{TxHash: "abc", Index: 1, PrivateKey: "K"}},
				To:       []UTXOOutput{{Address: "bc1q", Value: 0.0001}},
				Fee:      "0.00001", ChangeAddress: "bc1change",
			})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/transaction/BTC/send",
		body:     `{"fromUTXO":[{"txHash":"abc","index":1,"privateKey":"K"}],"to":[{"address":"bc1q","value":0.0001}],"fee":"0.00001","changeAddress":"bc1change"}`,
		response: `{"txId":"btctx"}`,
		check: func(t *testing.T, got any) {
			eq(t, "txId", got.(*TxIDResponse).TxID, "btctx")
		},
	},
	{
		name: "Operations.Send/Account(SOL)",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Operations.Send(ctx, "SOL", &SendAccountRequest{From: "SoLfrom", To: "SoLto", Amount: "1", FromPrivateKey: "PK"})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/transaction/SOL/send",
		body:     `{"from":"SoLfrom","to":"SoLto","amount":"1","fromPrivateKey":"PK"}`,
		response: `{"txId":"soltx"}`,
	},
	{
		name: "Operations.Send/Raw",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Operations.Send(ctx, "XRP", RawSendRequest(`{"fromAccount":"r1","to":"r2","amount":"1","fromSecret":"s"}`))
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/transaction/XRP/send",
		body:     `{"fromAccount":"r1","to":"r2","amount":"1","fromSecret":"s"}`,
		response: `{"txId":"xrptx"}`,
	},
	{
		name: "Operations.Broadcast",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Operations.Broadcast(ctx, "BTC", &BroadcastRequest{TxData: "62BD544D"})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/transaction/BTC/broadcast",
		body:     `{"txData":"62BD544D"}`,
		response: `{"txId":"c83f"}`,
		check: func(t *testing.T, got any) {
			eq(t, "txId", got.(*TxIDResponse).TxID, "c83f")
		},
	},
	{
		name: "Operations.RPC",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Operations.RPC(ctx, "SOL", &RPCRequest{ID: 1, Method: "getSignaturesForAddress", Params: []any{"SoLAddr", map[string]int{"limit": 10}}})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/rpc/SOL",
		body:     `{"jsonrpc":"2.0","id":1,"method":"getSignaturesForAddress","params":["SoLAddr",{"limit":10}]}`,
		response: `{"jsonrpc":"2.0","id":1,"result":[{"signature":"sigExample","slot":123}]}`,
		check: func(t *testing.T, got any) {
			r := got.(*RPCResponse)
			var sigs []struct {
				Signature string `json:"signature"`
				Slot      int64  `json:"slot"`
			}
			if err := r.DecodeResult(&sigs); err != nil {
				t.Fatal(err)
			}
			eq(t, "sig", sigs[0].Signature, "sigExample")
			eq(t, "slot", sigs[0].Slot, int64(123))
			eq(t, "id", string(r.ID), "1")
		},
	},
}

func TestBlockchainWalletEndpoints(t *testing.T) {
	runEndpointCases(t, blockchainWalletCases)
}

var blockchainWalletCases = []endpointCase{
	{
		name: "Wallet.Generate",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Wallet.Generate(ctx, "BTC", nil)
		},
		method:   "GET",
		path:     "/api/v1/blockchain/wallet/BTC",
		response: `{"mnemonic":"abandon about","xpub":"xpub6CUG"}`,
		check: func(t *testing.T, got any) {
			eq(t, "wallet", *got.(*Wallet), Wallet{Mnemonic: "abandon about", Xpub: "xpub6CUG"})
		},
	},
	{
		name: "Wallet.Generate/with mnemonic",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Wallet.Generate(ctx, "SOL", &GenerateWalletParams{Mnemonic: "abandon about"})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/wallet/SOL",
		query:    q("mnemonic", "abandon about"),
		response: `{"mnemonic":"abandon about","address":"SoL1","privateKey":"pk58"}`,
		check: func(t *testing.T, got any) {
			eq(t, "wallet", *got.(*Wallet), Wallet{Mnemonic: "abandon about", Address: "SoL1", PrivateKey: "pk58"})
		},
	},
	{
		name: "Wallet.DeriveAddress",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Wallet.DeriveAddress(ctx, "BTC", "xpub6CUG", 3)
		},
		method:   "GET",
		path:     "/api/v1/blockchain/wallet/BTC/address/xpub6CUG/3",
		response: `{"address":"bc1qderived"}`,
		check: func(t *testing.T, got any) {
			eq(t, "address", got.(*DerivedAddress).Address, "bc1qderived")
		},
	},
	{
		name: "Wallet.DerivePrivateKey",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Wallet.DerivePrivateKey(ctx, "ETH", &DerivePrivateKeyRequest{Mnemonic: "abandon about", Index: 0})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/key/ETH/derive",
		body:     `{"mnemonic":"abandon about","index":0}`,
		response: `{"key":"0xprivatekeytest123"}`,
		check: func(t *testing.T, got any) {
			eq(t, "key", got.(*PrivateKey).Key, "0xprivatekeytest123")
		},
	},
}

func TestBlockchainContractsEndpoints(t *testing.T) {
	runEndpointCases(t, blockchainContractsCases)
}

var blockchainContractsCases = []endpointCase{
	{
		name: "Contracts.DeployToken",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Contracts.DeployToken(ctx, &DeployTokenRequest{Chain: "BSC", Symbol: "TKN", Name: "Token", Supply: "1000", Digits: Int64(18), Address: "0xrecv", FromPrivateKey: "PK"})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/contract/token/deploy",
		body:     `{"chain":"BSC","symbol":"TKN","name":"Token","supply":"1000","digits":18,"address":"0xrecv","fromPrivateKey":"PK"}`,
		response: `{"txId":"deploytx"}`,
		check: func(t *testing.T, got any) {
			eq(t, "txId", got.(*TxIDResponse).TxID, "deploytx")
		},
	},
	{
		name: "Contracts.MintToken",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Contracts.MintToken(ctx, &MintTokenRequest{Chain: "ETH", ContractAddress: "0xc", Amount: "5", To: "0xto", FromPrivateKey: "PK"})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/contract/token/mint",
		body:     `{"chain":"ETH","contractAddress":"0xc","amount":"5","to":"0xto","fromPrivateKey":"PK"}`,
		response: `{"txId":"minttx"}`,
		check: func(t *testing.T, got any) {
			eq(t, "txId", got.(*TxIDResponse).TxID, "minttx")
		},
	},
	{
		name: "Contracts.BurnToken",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Contracts.BurnToken(ctx, &BurnTokenRequest{Chain: "ALGO", ContractAddress: "123", Amount: "5", FromPrivateKey: "PK"})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/operations/contract/token/burn",
		body:     `{"chain":"ALGO","contractAddress":"123","amount":"5","fromPrivateKey":"PK"}`,
		response: `{"txId":"burntx"}`,
		check: func(t *testing.T, got any) {
			eq(t, "txId", got.(*TxIDResponse).TxID, "burntx")
		},
	},
}

func TestBlockchainFeeEndpoints(t *testing.T) { runEndpointCases(t, blockchainFeeCases) }

var blockchainFeeCases = []endpointCase{
	{
		name: "Fee.Get",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Fee.Get(ctx, "ETH")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/fee/ETH",
		response: `{"fast":1.452,"medium":1.193,"slow":1.1,"baseFee":0.9,"block":965924,"time":"2026-09-07T11:55:47.916Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*FeeEstimate)
			eq(t, "fast", r.Fast, 1.452)
			eq(t, "baseFee", *r.BaseFee, 0.9)
			eq(t, "block", r.Block, int64(965924))
		},
	},
	{
		name: "Fee.EstimateGas",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Fee.EstimateGas(ctx, "BNB", &EstimateGasRequest{From: "0xAb58", To: "0xdEaD", Amount: "0.01"})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/data/fee/gas/BNB",
		body:     `{"from":"0xAb58","to":"0xdEaD","amount":"0.01"}`,
		response: `{"gasLimit":"21000","gasPrice":"50000000"}`,
		check: func(t *testing.T, got any) {
			eq(t, "gas", *got.(*GasEstimate), GasEstimate{GasLimit: "21000", GasPrice: "50000000"})
		},
	},
}

func TestBlockchainLookupsEndpoints(t *testing.T) {
	runEndpointCases(t, blockchainLookupsCases)
}

var blockchainLookupsCases = []endpointCase{
	{
		name: "Lookups.GetTransaction",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Lookups.GetTransaction(ctx, "ETH", "0xd49f")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/tx/ETH/0xd49f",
		response: `[{"chain":"ethereum-mainnet","hash":"0xd49f","address":"0x4740","counterAddress":"0x9757","blockNumber":16410533,"transactionIndex":139,"transactionType":"native","transactionSubtype":"outgoing","amount":"-3.9e-17","timestamp":1673765531000}]`,
		check: func(t *testing.T, got any) {
			entries, err := got.(*Transaction).AsEVM()
			if err != nil {
				t.Fatal(err)
			}
			eq(t, "idx", entries[0].TransactionIndex, int64(139))
			eq(t, "subtype", entries[0].TransactionSubtype, "outgoing")
		},
	},
	{
		name: "Lookups.GetBlock",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Lookups.GetBlock(ctx, "BTC", "937061")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/block/BTC/937061",
		response: `{"hash":"0000b29","height":937061,"mediantime":1771321646,"bits":386022526,"difficulty":125864590119494.3,"chainwork":"0001","confirmations":28978,"merkleRoot":"1758"}`,
		check: func(t *testing.T, got any) {
			b, err := got.(*Block).AsUTXO()
			if err != nil {
				t.Fatal(err)
			}
			eq(t, "height", b.Height, int64(937061))
			eq(t, "merkle", b.MerkleRoot, "1758")
			eq(t, "difficulty", b.Difficulty, 125864590119494.3)
		},
	},
	{
		name: "Lookups.GetLatestBlock",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Lookups.GetLatestBlock(ctx, "TRON")
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/block/TRON/latest",
		response: `{"blockID":"0000052131e4","block_header":{"raw_data":{"timestamp":1788856416000,"number":86058853,"witness_address":"417f5e","version":36},"witness_signature":"4d7c"},"transactions":[]}`,
		check: func(t *testing.T, got any) {
			r := got.(*TronLatestBlock)
			eq(t, "blockID", r.BlockID, "0000052131e4")
			eq(t, "number", r.BlockHeader.RawData.Number, int64(86058853))
			eq(t, "witness", r.BlockHeader.WitnessSignature, "4d7c")
		},
	},
	{
		name: "Lookups.GetToken",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Lookups.GetToken(ctx, "ETH", "0xbc4c", &GetTokenParams{TokenID: "1"})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/tokens/ETH/0xbc4c",
		query:    q("tokenId", "1"),
		response: `{"symbol":"BAYC","name":"BoredApe","tokenType":"nonfungible","metadataURI":"ipfs://x"}`,
		check: func(t *testing.T, got any) {
			r := got.(*TokenMetadata)
			eq(t, "type", r.TokenType, "nonfungible")
			eq(t, "uri", r.MetadataURI, "ipfs://x")
		},
	},
	{
		name: "Lookups.ListUTXOs",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Lookups.ListUTXOs(ctx, "BTC", "34xp", 0.001)
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/utxo/BTC/34xp",
		query:    q("totalValue", "0.001"),
		response: `[{"chain":"bitcoin-mainnet","address":"34xp","txHash":"3eb2","index":24,"value":0.00012313,"valueAsString":"0.00012313"}]`,
		check: func(t *testing.T, got any) {
			r := got.([]UTXO)
			eq(t, "utxo", r[0], UTXO{Chain: "bitcoin-mainnet", Address: "34xp", TxHash: "3eb2", Index: 24, Value: 0.00012313, ValueAsString: "0.00012313"})
		},
	},
	{
		name: "Lookups.ListUTXOsBatch",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.Lookups.ListUTXOsBatch(ctx, &UTXOBatchRequest{Addresses: []string{"34xp", "bc1q"}, TotalValue: 0.001, Chain: "bitcoin-mainnet"})
		},
		method:   "POST",
		path:     "/api/v1/blockchain/data/utxo/batch",
		body:     `{"addresses":["34xp","bc1q"],"totalValue":0.001,"chain":"bitcoin-mainnet"}`,
		response: `[{"address":"34xp","utxos":[{"txHash":"3eb2","index":24,"value":0.00012313,"valueAsString":"0.00012313"}],"transactionPossible":true},{"address":"bc1q","utxos":[],"transactionPossible":false}]`,
		check: func(t *testing.T, got any) {
			r := got.([]AddressUTXOs)
			eq(t, "possible0", r[0].TransactionPossible, true)
			eq(t, "idx", r[0].UTXOs[0].Index, int64(24))
			eq(t, "possible1", r[1].TransactionPossible, false)
		},
	},
}

func TestBlockchainNFTEndpoints(t *testing.T) { runEndpointCases(t, blockchainNFTCases) }

var blockchainNFTCases = []endpointCase{
	{
		name: "NFT.ListCollection",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.NFT.ListCollection(ctx, "ETH", "0xbc4c", &CollectionParams{ExcludeMetadata: Bool(false), PageSize: Int64(50), Offset: Int64(0)})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/nft/collection/ETH/0xbc4c",
		query:    q("excludeMetadata", "false", "pageSize", "50", "offset", "0"),
		response: `[{"chain":"ethereum-mainnet","tokenId":"1","tokenAddress":"0xbc4c","tokenType":"nonfungible","metadataURI":"https://ipfs.io/ipfs/Qm","metadata":{"identifier":"1","collection":"boredapeyachtclub","token_standard":"erc721","name":"Bored Ape #1","display_animation_url":null}}]`,
		check: func(t *testing.T, got any) {
			r := got.([]NFT)
			eq(t, "tokenId", r[0].TokenID, "1")
			eq(t, "standard", r[0].Metadata.TokenStandard, "erc721")
			if r[0].Metadata.DisplayAnimationURL != nil {
				t.Error("expected nil display_animation_url")
			}
		},
	},
	{
		name: "NFT.GetOwners",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Blockchain.NFT.GetOwners(ctx, "ETH", "0xbc4c", "1", &OwnersParams{PageSize: Int64(10)})
		},
		method:   "GET",
		path:     "/api/v1/blockchain/data/nft/owner/ETH/0xbc4c/1",
		query:    q("pageSize", "10"),
		response: `["0x1234567890abcdef1234567890abcdef12345678"]`,
		check: func(t *testing.T, got any) {
			eq(t, "owners", got.([]string), []string{"0x1234567890abcdef1234567890abcdef12345678"})
		},
	},
}

// UploadIPFS sends multipart/form-data, so it is tested on its own rather
// than through the JSON-body table.
func TestBlockchainStorageUploadIPFS(t *testing.T) {
	srv := newMockServer(t, mockResponse{status: 201, body: `{"ipfsHash":"bafkreifps7p33l5cl6cbepea3nrvhs46bratq3qglfe6rdc6kusddspwfa"}`})
	got, err := srv.client().Blockchain.Storage.UploadIPFS(ctxBG(), "hello.txt", strings.NewReader("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "hash", got.IPFSHash, "bafkreifps7p33l5cl6cbepea3nrvhs46bratq3qglfe6rdc6kusddspwfa")

	r := srv.recorded()[0]
	eq(t, "method", r.Method, "POST")
	eq(t, "path", r.Path, "/api/v1/blockchain/storage/ipfs")
	eq(t, "api key", r.Header.Get("x-api-key"), testAPIKey)
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "media type", mediaType, "multipart/form-data")
	form, err := multipart.NewReader(strings.NewReader(string(r.Body)), params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("expected one file part, got %d", len(files))
	}
	eq(t, "filename", files[0].Filename, "hello.txt")
	f, _ := files[0].Open()
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	eq(t, "content", string(buf[:n]), "hello world")
}

func TestBalanceShapes(t *testing.T) {
	decode := func(t *testing.T, raw string) *Balance {
		t.Helper()
		var b Balance
		if err := json.Unmarshal([]byte(raw), &b); err != nil {
			t.Fatal(err)
		}
		return &b
	}

	t.Run("simple", func(t *testing.T) {
		s, err := decode(t, `{"balance":"0.1234"}`).AsSimple()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "balance", s.Balance, "0.1234")
	})
	t.Run("celo", func(t *testing.T) {
		s, err := decode(t, `{"celo":"1","cUsd":"2","cEur":"3"}`).AsCelo()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "celo", *s, CeloBalance{Celo: "1", CUSD: "2", CEUR: "3"})
	})
	t.Run("cardano", func(t *testing.T) {
		s, err := decode(t, `[{"currency":{"symbol":"ADA","decimals":6},"value":"10"}]`).AsCardano()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "entry", s[0], CardanoAssetBalance{Currency: CardanoCurrency{Symbol: "ADA", Decimals: 6}, Value: "10"})
	})
	t.Run("stellar", func(t *testing.T) {
		s, err := decode(t, `{"account_id":"G1","sequence":"5","balances":[{"asset_type":"native","balance":"9"},{"asset_type":"credit_alphanum4","balance":"1","limit":"100","asset_code":"USDC","asset_issuer":"GI"}]}`).AsStellar()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "account", s.AccountID, "G1")
		eq(t, "code", s.Balances[1].AssetCode, "USDC")
	})
	t.Run("xrp", func(t *testing.T) {
		s, err := decode(t, `{"balance":"1000000","assets":[{"balance":"5","currency":"USD"}]}`).AsXRP()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "asset", s.Assets[0], XRPAssetBalance{Balance: "5", Currency: "USD"})
	})
	t.Run("tron", func(t *testing.T) {
		s, err := decode(t, `{"address":"T1","balance":1500000,"trc10":[],"trc20":[{"TR7":"10"}],"bandwidth":{"free":1}}`).AsTron()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "balance", s.Balance, 1500000.0)
		eq(t, "trc20", string(s.TRC20[0]), `{"TR7":"10"}`)
	})
	t.Run("wrong accessor errors", func(t *testing.T) {
		if _, err := decode(t, `[{"currency":{"symbol":"ADA"},"value":"1"}]`).AsSimple(); err == nil {
			t.Error("expected an error decoding an array as an object")
		}
	})
	t.Run("marshal round-trip", func(t *testing.T) {
		b := decode(t, `{"balance":"1"}`)
		out, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "json", string(out), `{"balance":"1"}`)
	})
}

func TestTransactionHistoryShapes(t *testing.T) {
	t.Run("tron", func(t *testing.T) {
		h := &TransactionHistory{Raw: json.RawMessage(`{"transactions":[{"txID":"a"}],"next":"n1"}`)}
		r, err := h.AsTron()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "next", r.Next, "n1")
		eq(t, "len", len(r.Transactions), 1)
	})
	t.Run("chain native", func(t *testing.T) {
		h := &TransactionHistory{Raw: json.RawMessage(`[{"hash":"a"},{"hash":"b"}]`)}
		r, err := h.AsChainNative()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "len", len(r), 2)
	})
}

func TestTransactionAndBlockShapes(t *testing.T) {
	t.Run("utxo tx", func(t *testing.T) {
		tx := &Transaction{Raw: json.RawMessage(`{"hash":"h","inputs":[{}],"outputs":[{},{}],"fee":120}`)}
		r, err := tx.AsUTXO()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "outputs", len(r.Outputs), 2)
		eq(t, "fee", r.Fee, 120.0)
	})
	t.Run("tron tx", func(t *testing.T) {
		tx := &Transaction{Raw: json.RawMessage(`{"txID":"t","rawData":{"x":1},"signature":["s"]}`)}
		r, err := tx.AsTron()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "txID", r.TxID, "t")
		eq(t, "sig", r.Signature, []string{"s"})
	})
	t.Run("evm block", func(t *testing.T) {
		b := &Block{Raw: json.RawMessage(`{"hash":"h","number":5,"parentHash":"p","transactions":[{"hash":"t1"}]}`)}
		r, err := b.AsEVM()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "number", r.Number, int64(5))
		eq(t, "txs", len(r.Transactions), 1)
	})
	t.Run("tron block", func(t *testing.T) {
		b := &Block{Raw: json.RawMessage(`{"blockNumber":9,"hash":"h","witnessAddress":"w"}`)}
		r, err := b.AsTron()
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "witness", r.WitnessAddress, "w")
	})
}

func TestRPCErrorInBody(t *testing.T) {
	srv := newMockServer(t, mockResponse{body: `{"jsonrpc":"2.0","id":"a","error":{"code":-32601,"message":"Method not found"}}`})
	resp, err := srv.client().Blockchain.Operations.RPC(ctxBG(), "ETH", &RPCRequest{JSONRPC: "2.0", ID: "a", Method: "nope"})
	if err != nil {
		t.Fatal(err)
	}
	var out any
	err = resp.DecodeResult(&out)
	var rpcErr *RPCError
	if err == nil || !errorsAs(err, &rpcErr) {
		t.Fatalf("expected *RPCError, got %v", err)
	}
	eq(t, "code", rpcErr.Code, int64(-32601))
	eq(t, "id", string(resp.ID), `"a"`)
}

func TestBlockchainClientSideValidation(t *testing.T) {
	srv := newMockServer(t, mockResponse{body: `{}`})
	c := srv.client()
	ctx := ctxBG()
	defer func() {
		if n := len(srv.recorded()); n != 0 {
			t.Errorf("validation failures must not reach the network; %d requests were sent", n)
		}
	}()
	checks := map[string]error{}
	_, checks["GetPortfolio"] = c.Blockchain.Data.GetPortfolio(ctx, "ETH", "0x1", nil)
	_, checks["GetExchangeRateByContract"] = c.Blockchain.Data.GetExchangeRateByContract(ctx, &ContractExchangeRateParams{Chain: "ethereum-mainnet"})
	_, checks["GetBalances"] = c.Blockchain.Data.GetBalances(ctx, nil)
	_, checks["Send"] = c.Blockchain.Operations.Send(ctx, "ETH", nil)
	_, checks["Broadcast"] = c.Blockchain.Operations.Broadcast(ctx, "ETH", nil)
	_, checks["RPC"] = c.Blockchain.Operations.RPC(ctx, "ETH", nil)
	_, checks["DeriveAddress"] = c.Blockchain.Wallet.DeriveAddress(ctx, "BTC", "xpub", -1)
	_, checks["DerivePrivateKey"] = c.Blockchain.Wallet.DerivePrivateKey(ctx, "BTC", nil)
	_, checks["DeployToken"] = c.Blockchain.Contracts.DeployToken(ctx, nil)
	_, checks["MintToken"] = c.Blockchain.Contracts.MintToken(ctx, nil)
	_, checks["BurnToken"] = c.Blockchain.Contracts.BurnToken(ctx, nil)
	_, checks["EstimateGas"] = c.Blockchain.Fee.EstimateGas(ctx, "BNB", nil)
	_, checks["ListUTXOs"] = c.Blockchain.Lookups.ListUTXOs(ctx, "BTC", "a", 0)
	_, checks["ListUTXOsBatch"] = c.Blockchain.Lookups.ListUTXOsBatch(ctx, nil)
	_, checks["UploadIPFS"] = c.Blockchain.Storage.UploadIPFS(ctx, "f", nil)
	_, checks["RawSendRequest"] = c.Blockchain.Operations.Send(ctx, "ETH", RawSendRequest(`{not json`))
	for name, err := range checks {
		if err == nil {
			t.Errorf("%s: expected a client-side validation error", name)
		}
	}
}
