package cryptures

import (
	"context"
	"testing"
	"time"
)

func TestCardEndpoints(t *testing.T) { runEndpointCases(t, cardCases) }

var cardCases = []endpointCase{
	{
		name: "Cards.Create",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Create(ctx, &CreateCardRequest{
				ProductCode: "us_493_visa_bin", FirstName: "Jane", LastName: "Doe", Email: "jane@example.com",
				InitialLoad: Float64(20), Tags: []string{"Marketing"},
			})
		},
		method:   "POST",
		path:     "/api/v1/card/cards/create",
		body:     `{"product_code":"us_493_visa_bin","first_name":"Jane","last_name":"Doe","email":"jane@example.com","initial_load":20,"tags":["Marketing"]}`,
		response: `{"status":"success","message":"Card created successfully.","data":{"card_id":"card_a1b2c3d4","status":"active","last_four":"4242","tags":[{"tag_id":"tag_9f1a","name":"Marketing","color":"#4F46E5"}]}}`,
		check: func(t *testing.T, got any) {
			r := got.(*CardResponse)
			eq(t, "status", r.Status, "success")
			eq(t, "card_id", r.Data.CardID, "card_a1b2c3d4")
			eq(t, "tags", r.Data.Tags, []CardTag{{TagID: "tag_9f1a", Name: "Marketing", Color: "#4F46E5"}})
		},
	},
	{
		name: "Cards.Create/no initial load",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Create(ctx, &CreateCardRequest{ProductCode: "us_493_visa_atm", FirstName: "A", LastName: "B", Email: "a@b.co"})
		},
		method:   "POST",
		path:     "/api/v1/card/cards/create",
		body:     `{"product_code":"us_493_visa_atm","first_name":"A","last_name":"B","email":"a@b.co"}`,
		response: `{"status":"success","data":{"card_id":"card_atm","status":"active"}}`,
	},
	{
		name: "Cards.Get",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Get(ctx, "card_a1b2c3d4")
		},
		method:   "GET",
		path:     "/api/v1/card/cards/card_a1b2c3d4",
		response: `{"status":"success","message":"Card fetched successfully.","data":{"card_id":"card_a1b2c3d4","product_code":"us_493_visa_bin","brand":"visa","type":"virtual","currency":"USD","status":"active","name_on_card":"Jane Doe","email":"x@placeholder.invalid","last_four":"1111","expiry_month":"08","expiry_year":"2031","balance":{"amount":10000000,"display_amount":10,"currency":"USD"},"card_number":"4111111111111111","cvv":"123","cardnumber":"4111111111111111","expiredate":"08/31","created_at":"2026-08-29T00:00:00Z","tags":[]}}`,
		check: func(t *testing.T, got any) {
			d := got.(*CardResponse).Data
			eq(t, "brand", d.Brand, "visa")
			eq(t, "pan", d.CardNumber, "4111111111111111")
			eq(t, "pan alt", d.CardNumberAlt, "4111111111111111")
			eq(t, "cvv", d.CVV, "123")
			eq(t, "expiredate", d.ExpireDate, "08/31")
			eq(t, "balance", *d.Balance, CardBalance{Amount: 10000000, DisplayAmount: 10, Currency: "USD"})
		},
	},
	{
		name: "Cards.SetPIN",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.SetPIN(ctx, "card_atm_1a2b3c", "654321")
		},
		method:   "POST",
		path:     "/api/v1/card/cards/card_atm_1a2b3c/pin",
		body:     `{"pin":"654321"}`,
		response: `{"status":"success","message":"Card PIN set successfully."}`,
		check: func(t *testing.T, got any) {
			eq(t, "resp", *got.(*StatusMessage), StatusMessage{Status: "success", Message: "Card PIN set successfully."})
		},
	},
	{
		name: "Cards.Fund",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Fund(ctx, "card_a1b2c3d4", 10)
		},
		method:   "POST",
		path:     "/api/v1/card/cards/card_a1b2c3d4/fund",
		body:     `{"amount":10}`,
		response: `{"status":"success","data":{"card_id":"card_a1b2c3d4","amount":10000000,"display_amount":10}}`,
		check: func(t *testing.T, got any) {
			eq(t, "data", got.(*CardAmountResponse).Data, CardAmountData{CardID: "card_a1b2c3d4", Amount: 10000000, DisplayAmount: 10})
		},
	},
	{
		name: "Cards.Withdraw",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Withdraw(ctx, "card_a1b2c3d4", 2.5)
		},
		method:   "POST",
		path:     "/api/v1/card/cards/card_a1b2c3d4/withdraw",
		body:     `{"amount":2.5}`,
		response: `{"status":"success","message":"Card withdrawal completed successfully.","data":{"card_id":"card_a1b2c3d4","amount":2500000,"display_amount":2.5}}`,
		check: func(t *testing.T, got any) {
			r := got.(*CardAmountResponse)
			eq(t, "amount", r.Data.Amount, int64(2500000))
			eq(t, "message", r.Message, "Card withdrawal completed successfully.")
		},
	},
	{
		name: "Cards.Terminate",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Terminate(ctx, "card_a1b2c3d4")
		},
		method:   "POST",
		path:     "/api/v1/card/cards/card_a1b2c3d4/terminate",
		response: `{"status":"success","message":"Card terminated successfully.","data":{"card_id":"card_a1b2c3d4","status":"terminated"}}`,
		check: func(t *testing.T, got any) {
			eq(t, "data", *got.(*CardStatusResponse).Data, CardStatusData{CardID: "card_a1b2c3d4", Status: "terminated"})
		},
	},
	{
		name: "Cards.Block",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Block(ctx, "card_a1b2c3d4")
		},
		method:   "POST",
		path:     "/api/v1/card/cards/card_a1b2c3d4/block",
		response: `{"status":"success","message":"Card blocked successfully.","data":{"card_id":"card_a1b2c3d4","status":"blocked"}}`,
		check: func(t *testing.T, got any) {
			eq(t, "status", got.(*CardStatusResponse).Data.Status, "blocked")
		},
	},
	{
		name: "Cards.Unblock",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.Unblock(ctx, "card_a1b2c3d4")
		},
		method:   "POST",
		path:     "/api/v1/card/cards/card_a1b2c3d4/unblock",
		response: `{"status":"success","message":"Card unblocked successfully.","data":{"card_id":"card_a1b2c3d4","status":"active"}}`,
		check: func(t *testing.T, got any) {
			eq(t, "status", got.(*CardStatusResponse).Data.Status, "active")
		},
	},
	{
		name: "Cards.ListTransactions",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.ListTransactions(ctx, "card_a1b2c3d4", &CardTransactionsParams{PageNum: "1"})
		},
		method:   "GET",
		path:     "/api/v1/card/cards/card_a1b2c3d4/transactions",
		query:    q("pageNum", "1"),
		response: `{"status":"success","message":"Transactions fetched successfully.","data":{"transactions":[{"id":"260820000000002879","card_id":"card_a1b2c3d4","type":"authorization","status":"completed","amount":460000,"display_amount":0.46,"currency":"USD","merchant_name":"Grab","created_at":"2026-08-20T05:55:00Z"}],"pagination":{"type":"page","page_num":1,"page_size":1,"total":1,"has_more":false}}}`,
		check: func(t *testing.T, got any) {
			d := got.(*CardTransactionsResponse).Data
			eq(t, "merchant", d.Transactions[0].MerchantName, "Grab")
			eq(t, "amount", d.Transactions[0].Amount, int64(460000))
			eq(t, "page", d.Pagination, CardTransactionsPage{Type: "page", PageNum: 1, PageSize: 1, Total: 1, HasMore: false})
		},
	},
	{
		name: "Cards.List",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.List(ctx)
		},
		method:   "GET",
		path:     "/api/v1/card/cards",
		response: `{"data":[{"card_id":"card_a1b2c3d4","product_code":"us_493_visa_bin","email":"jane@example.com","first_name":"Jane","last_name":"Doe","status":"active","last_four":null,"label":null,"created_at":"2026-08-28T00:00:00Z","tags":[]}]}`,
		check: func(t *testing.T, got any) {
			d := got.(*CardList).Data[0]
			eq(t, "email", d.Email, "jane@example.com")
			if d.LastFour != nil || d.Label != nil {
				t.Error("expected nil last_four and label")
			}
			eq(t, "tags", d.Tags, []CardTag{})
		},
	},
	{
		name: "Cards.SetTags",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.SetTags(ctx, "card_a1b2c3d4", []string{"Marketing", "VIP"})
		},
		method:   "PUT",
		path:     "/api/v1/card/cards/card_a1b2c3d4/tags",
		body:     `{"tags":["Marketing","VIP"]}`,
		response: `{"tags":[{"tag_id":"tag_1","name":"Marketing","color":"#4F46E5"},{"tag_id":"tag_2","name":"VIP","color":"#DC2626"}]}`,
		check: func(t *testing.T, got any) {
			eq(t, "count", len(got.(*CardTags).Tags), 2)
		},
	},
	{
		name: "Cards.SetTags/nil clears explicitly",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.SetTags(ctx, "card_x", nil)
		},
		method:   "PUT",
		path:     "/api/v1/card/cards/card_x/tags",
		body:     `{"tags":[]}`,
		response: `{"tags":[]}`,
	},
	{
		name: "Cards.ListProducts",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Cards.ListProducts(ctx)
		},
		method:   "GET",
		path:     "/api/v1/card/products",
		response: `{"products":[{"product_code":"us_493_visa_atm","display_name":"US Visa ATM Card","min_load_usd":null,"max_load_usd":null,"issuance_fee_usd":1,"fund_fee_flat_usd":0,"fund_fee_pct":0,"annual_fee_usd":9,"disallow_initial_load":true},{"product_code":"us_493_visa_bin","display_name":"US Visa Card","min_load_usd":10,"max_load_usd":5000,"issuance_fee_usd":2,"fund_fee_flat_usd":1,"fund_fee_pct":0.02,"annual_fee_usd":null,"disallow_initial_load":false}]}`,
		check: func(t *testing.T, got any) {
			p := got.(*CardProducts).Products
			eq(t, "atm disallow", p[0].DisallowInitialLoad, true)
			if p[0].MinLoadUSD != nil {
				t.Error("expected nil min_load_usd")
			}
			eq(t, "annual", *p[0].AnnualFeeUSD, 9.0)
			eq(t, "max", *p[1].MaxLoadUSD, 5000.0)
			eq(t, "pct", p[1].FundFeePct, 0.02)
			if p[1].AnnualFeeUSD != nil {
				t.Error("expected nil annual_fee_usd")
			}
		},
	},
	{
		name: "Card.Webhooks.Register",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Webhooks.Register(ctx, "https://example.com/webhooks/card")
		},
		method:   "POST",
		path:     "/api/v1/card/webhooks/register",
		body:     `{"url":"https://example.com/webhooks/card"}`,
		response: `{"url":"https://example.com/webhooks/card","secret":"3f1a9c2e"}`,
		check: func(t *testing.T, got any) {
			eq(t, "reg", *got.(*WebhookRegistration), WebhookRegistration{URL: "https://example.com/webhooks/card", Secret: "3f1a9c2e"})
		},
	},
	{
		name: "Card.Webhooks.Status",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Webhooks.Status(ctx)
		},
		method:   "GET",
		path:     "/api/v1/card/webhooks/status",
		response: `{"registered":true,"url":"https://example.com/webhooks/card","registered_at":"2026-09-12T18:04:33Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*WebhookStatus)
			eq(t, "registered", r.Registered, true)
			eq(t, "url", *r.URL, "https://example.com/webhooks/card")
			eq(t, "at", *r.RegisteredAt, "2026-09-12T18:04:33Z")
		},
	},
	{
		name: "Card.Balance.Get",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Balance.Get(ctx)
		},
		method:   "GET",
		path:     "/api/v1/card/balance",
		response: `{"balance_usd":42.5,"updated_at":"2026-08-29T00:00:00Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*ProjectBalance)
			eq(t, "balance", r.BalanceUSD, 42.5)
			eq(t, "updated", *r.UpdatedAt, "2026-08-29T00:00:00Z")
		},
	},
	{
		name: "Card.Balance.ListTransactions",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Balance.ListTransactions(ctx, &BalanceTransactionsParams{Page: Int64(2), Limit: Int64(100)})
		},
		method:   "GET",
		path:     "/api/v1/card/balance/transactions",
		query:    q("page", "2", "limit", "100"),
		response: `{"data":[{"id":"bl_9f1a2b3c","type":"card_fund_debit","amount_usd":11.4,"related_reference":"card_a1b2c3d4","status":"completed","created_at":"2026-08-29T00:00:01Z"},{"id":"bl_2","type":"card_create_debit","amount_usd":5,"related_reference":null,"status":"pending","created_at":"2026-08-29T00:00:00Z"}],"page":2,"limit":100,"total":102}`,
		check: func(t *testing.T, got any) {
			r := got.(*BalanceTransactions)
			eq(t, "total", r.Total, int64(102))
			eq(t, "type", r.Data[0].Type, "card_fund_debit")
			eq(t, "ref", *r.Data[0].RelatedReference, "card_a1b2c3d4")
			if r.Data[1].RelatedReference != nil {
				t.Error("expected nil related_reference")
			}
		},
	},
	{
		name: "Card.Tags.List",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Tags.List(ctx)
		},
		method:   "GET",
		path:     "/api/v1/card/tags",
		response: `{"data":[{"tag_id":"tag_9f1a","name":"Marketing","color":"#4F46E5","created_at":"2026-08-29T00:00:00Z","updated_at":"2026-08-29T00:00:00Z"}]}`,
		check: func(t *testing.T, got any) {
			eq(t, "tag", got.(*TagList).Data[0], Tag{TagID: "tag_9f1a", Name: "Marketing", Color: "#4F46E5", CreatedAt: "2026-08-29T00:00:00Z", UpdatedAt: "2026-08-29T00:00:00Z"})
		},
	},
	{
		name: "Card.Tags.Create",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Tags.Create(ctx, &CreateTagRequest{Name: "Marketing"})
		},
		method:   "POST",
		path:     "/api/v1/card/tags",
		body:     `{"name":"Marketing"}`,
		status:   201,
		response: `{"tag_id":"tag_9f1a","name":"Marketing","color":"#4F46E5","created_at":"2026-08-29T00:00:00Z","updated_at":"2026-08-29T00:00:00Z"}`,
		check: func(t *testing.T, got any) {
			eq(t, "color", got.(*Tag).Color, "#4F46E5")
		},
	},
	{
		name: "Card.Tags.Update",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Tags.Update(ctx, "tag_9f1a", &UpdateTagRequest{Color: String("#DC2626")})
		},
		method:   "PATCH",
		path:     "/api/v1/card/tags/tag_9f1a",
		body:     `{"color":"#DC2626"}`,
		response: `{"tag_id":"tag_9f1a","name":"Marketing","color":"#DC2626","updated_at":"2026-08-29T01:00:00Z"}`,
		check: func(t *testing.T, got any) {
			r := got.(*Tag)
			eq(t, "color", r.Color, "#DC2626")
			eq(t, "updated", r.UpdatedAt, "2026-08-29T01:00:00Z")
		},
	},
	{
		name: "Card.Tags.Delete",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Tags.Delete(ctx, "tag_9f1a")
		},
		method:   "DELETE",
		path:     "/api/v1/card/tags/tag_9f1a",
		response: `{"deleted":true}`,
		check: func(t *testing.T, got any) {
			eq(t, "deleted", got.(*TagDeleted).Deleted, true)
		},
	},
	{
		name: "Card.Reports.Summary",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Reports.Summary(ctx, &ReportSummaryParams{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			})
		},
		method:   "GET",
		path:     "/api/v1/card/reports/summary",
		query:    q("from", "2026-01-01T00:00:00Z", "to", "2026-09-01T00:00:00Z"),
		response: `{"range":{"from":"2026-01-01T00:00:00.000Z","to":"2026-09-01T00:00:00.000Z"},"totals":{"funded_usd":1250,"withdrawn_usd":300,"net_usd":950,"card_count":4},"unattributed":{"funded_usd":0},"by_month":[{"month":"2026-08","funded_usd":200,"withdrawn_usd":0,"net_usd":200}],"by_type":[{"product_code":"visa_standard","funded_usd":1250,"withdrawn_usd":300,"net_usd":950,"card_count":4}],"by_tag":[{"tag_id":"tag_9f1a","name":"Marketing","color":"#4F46E5","funded_usd":400,"withdrawn_usd":0,"net_usd":400,"card_count":2}],"untagged":{"funded_usd":850,"withdrawn_usd":300,"net_usd":550,"card_count":2},"by_card":[{"card_id":"card_a1b2c3d4","first_name":"Jane","last_name":"Doe","last_four":"4242","product_code":"visa_standard","tags":[],"funded_usd":300,"withdrawn_usd":0,"net_usd":300}]}`,
		check: func(t *testing.T, got any) {
			r := got.(*ReportSummary)
			eq(t, "totals", r.Totals, ReportTotals{FundedUSD: 1250, WithdrawnUSD: 300, NetUSD: 950, CardCount: 4})
			eq(t, "month", r.ByMonth[0].Month, "2026-08")
			eq(t, "tag", r.ByTag[0].CardCount, int64(2))
			eq(t, "untagged", r.Untagged.NetUSD, 550.0)
			eq(t, "card last4", *r.ByCard[0].LastFour, "4242")
			eq(t, "range", r.Range.From, "2026-01-01T00:00:00.000Z")
		},
	},
	{
		name: "Card.Reports.Summary/defaults",
		call: func(ctx context.Context, c *Client) (any, error) {
			return c.Card.Reports.Summary(ctx, nil)
		},
		method:   "GET",
		path:     "/api/v1/card/reports/summary",
		response: `{"range":{"from":"a","to":"b"},"totals":{"funded_usd":0,"withdrawn_usd":0,"net_usd":0,"card_count":0},"unattributed":{"funded_usd":0},"by_month":[],"by_type":[],"by_tag":[],"untagged":{"funded_usd":0,"withdrawn_usd":0,"net_usd":0,"card_count":0},"by_card":[]}`,
	},
}

func TestCardClientSideValidation(t *testing.T) {
	srv := newMockServer(t, mockResponse{body: `{}`})
	c := srv.client()
	ctx := ctxBG()
	checks := map[string]error{}
	_, checks["Create"] = c.Card.Cards.Create(ctx, nil)
	_, checks["Fund"] = c.Card.Cards.Fund(ctx, "card", 0)
	_, checks["Withdraw"] = c.Card.Cards.Withdraw(ctx, "card", -1)
	_, checks["Tags.Create"] = c.Card.Tags.Create(ctx, nil)
	_, checks["Tags.Update"] = c.Card.Tags.Update(ctx, "tag", &UpdateTagRequest{})
	for name, err := range checks {
		if err == nil {
			t.Errorf("%s: expected a client-side validation error", name)
		}
	}
	if n := len(srv.recorded()); n != 0 {
		t.Errorf("validation failures must not reach the network; %d requests were sent", n)
	}
}
