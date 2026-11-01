package main

import (
	"strings"
	"testing"

	ex "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange/account"
	tt "github.com/plusev-terminal/go-plugin-common/v2/trading"
	"github.com/trading-peter/plusev_datasource_bybit_plugin/bybit"
)

func perpMarket(symbol string) tt.Market {
	return tt.Market{Symbol: symbol, AssetType: tt.AssetTypePerpetual}
}

func TestBuildBybitCreateOrderRequestLimit(t *testing.T) {
	req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:   perpMarket("BTCUSDT"),
		Side:     acct.OrderSideBuy,
		Type:     acct.OrderTypeLimit,
		Quantity: "1",
		Price:    "25000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Category != "linear" || req.OrderType != "Limit" || req.TimeInForce != "GTC" {
		t.Fatalf("bad plain limit mapping: %+v", req)
	}
	if req.TriggerPrice != "" || req.TriggerDirection != 0 || req.TriggerBy != "" {
		t.Fatalf("plain limit must not be conditional: %+v", req)
	}
}

func TestBuildBybitCreateOrderRequestStopAliasIsTriggerNotLeg(t *testing.T) {
	// StopPrice on a stop order is the trigger alias; it must not ALSO be
	// attached as a stop-loss leg.
	req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:    perpMarket("BTCUSDT"),
		Side:      acct.OrderSideSell,
		Type:      acct.OrderTypeStop,
		Quantity:  "1",
		StopPrice: "24000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.TriggerPrice != "24000" {
		t.Fatalf("stop alias must become the trigger, got %q", req.TriggerPrice)
	}
	if req.StopLoss != "" {
		t.Fatalf("trigger alias must not be re-attached as stopLoss, got %q", req.StopLoss)
	}
	if req.TriggerDirection != 2 {
		t.Fatalf("sell stop triggers when price falls, want direction 2, got %d", req.TriggerDirection)
	}
}

func TestBuildBybitCreateOrderRequestTakeProfitAliasIsTriggerNotLeg(t *testing.T) {
	req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:          perpMarket("BTCUSDT"),
		Side:            acct.OrderSideSell,
		Type:            acct.OrderTypeTakeProfit,
		Quantity:        "1",
		TakeProfitPrice: "30000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.TriggerPrice != "30000" {
		t.Fatalf("take profit alias must become the trigger, got %q", req.TriggerPrice)
	}
	if req.TakeProfit != "" {
		t.Fatalf("trigger alias must not be re-attached as takeProfit, got %q", req.TakeProfit)
	}
	if req.TriggerDirection != 1 {
		t.Fatalf("sell TP triggers when price rises, want direction 1, got %d", req.TriggerDirection)
	}
}

func TestBuildBybitCreateOrderRequestAttachedTPSLOnPlainOrder(t *testing.T) {
	req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:          perpMarket("BTCUSDT"),
		Side:            acct.OrderSideBuy,
		Type:            acct.OrderTypeLimit,
		Quantity:        "1",
		Price:           "25000",
		TakeProfitPrice: "30000",
		StopPrice:       "24000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.TakeProfit != "30000" || req.StopLoss != "24000" {
		t.Fatalf("attached TP/SL legs must survive on a plain order: %+v", req)
	}
	if req.TriggerPrice != "" {
		t.Fatalf("plain limit with attached legs must stay non-conditional, got trigger %q", req.TriggerPrice)
	}
}

func TestBuildBybitCreateOrderRequestTriggerPriceWinsOverAlias(t *testing.T) {
	req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:       perpMarket("BTCUSDT"),
		Side:         acct.OrderSideBuy,
		Type:         acct.OrderTypeStopLimit,
		Quantity:     "1",
		Price:        "23900",
		TriggerPrice: "24000",
		StopPrice:    "23800",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.TriggerPrice != "24000" {
		t.Fatalf("triggerPrice must win over the alias, got %q", req.TriggerPrice)
	}
	if req.StopLoss != "" {
		t.Fatalf("stop-type alias must never become an attached leg, got %q", req.StopLoss)
	}
}

func TestBuildBybitCreateOrderRequestTriggerSource(t *testing.T) {
	req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:        perpMarket("BTCUSDT"),
		Side:          acct.OrderSideSell,
		Type:          acct.OrderTypeStop,
		Quantity:      "1",
		TriggerPrice:  "24000",
		TriggerSource: acct.TriggerSourceMark,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.TriggerBy != "MarkPrice" {
		t.Fatalf("triggerSource mark must map to triggerBy MarkPrice, got %q", req.TriggerBy)
	}

	req, err = buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:       perpMarket("BTCUSDT"),
		Side:         acct.OrderSideSell,
		Type:         acct.OrderTypeStop,
		Quantity:     "1",
		TriggerPrice: "24000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.TriggerBy != "" {
		t.Fatalf("empty triggerSource must omit triggerBy (Bybit LastPrice default), got %q", req.TriggerBy)
	}
}

func TestBuildBybitCreateOrderRequestTimeInForce(t *testing.T) {
	cases := []struct {
		name string
		tif  acct.TimeInForce
		want string
	}{
		{"ioc", acct.TimeInForceIOC, "IOC"},
		{"fok", acct.TimeInForceFOK, "FOK"},
		{"post_only", acct.TimeInForcePostOnly, "PostOnly"},
		{"gtx", acct.TimeInForceGTX, "PostOnly"},
	}
	for _, tc := range cases {
		req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
			Market:      perpMarket("BTCUSDT"),
			Side:        acct.OrderSideBuy,
			Type:        acct.OrderTypeLimit,
			Quantity:    "1",
			Price:       "25000",
			TimeInForce: tc.tif,
		})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if req.TimeInForce != tc.want {
			t.Fatalf("%s: want timeInForce %q, got %q", tc.name, tc.want, req.TimeInForce)
		}
	}
}

func TestBuildBybitCreateOrderRequestRejectsUnsupported(t *testing.T) {
	gtd := ex.CreateOrderParams{
		Market:      perpMarket("BTCUSDT"),
		Side:        acct.OrderSideBuy,
		Type:        acct.OrderTypeLimit,
		Quantity:    "1",
		Price:       "25000",
		TimeInForce: acct.TimeInForceGTD,
		ExpireTime:  "1735689600000",
	}
	if _, err := buildBybitCreateOrderRequest(gtd); err == nil || !strings.Contains(err.Error(), "gtd") {
		t.Fatalf("GTD must be rejected (no Bybit v5 expireTime), got %v", err)
	}

	trailing := ex.CreateOrderParams{
		Market:          perpMarket("BTCUSDT"),
		Side:            acct.OrderSideSell,
		Type:            acct.OrderTypeTrailingStop,
		Quantity:        "1",
		TrailingPercent: "1.5",
	}
	if _, err := buildBybitCreateOrderRequest(trailing); err == nil {
		t.Fatal("trailing_stop must be rejected: /v5/order/create has no trailing-distance parameter")
	}
}

func TestBuildBybitCreateOrderRequestSpotQuoteQuantity(t *testing.T) {
	req, err := buildBybitCreateOrderRequest(ex.CreateOrderParams{
		Market:        tt.Market{Symbol: "BTCUSDT", AssetType: tt.AssetTypeSpot},
		Side:          acct.OrderSideBuy,
		Type:          acct.OrderTypeMarket,
		QuoteQuantity: "200",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Category != "spot" || req.MarketUnit != "quoteCoin" || req.Qty != "200" {
		t.Fatalf("spot market buy by quote quantity: %+v", req)
	}
}

// TestGetOrderCapabilitiesDeclaresWhatTheBuilderSends pins the capability
// declaration to the builder's real behavior: every order it accepts passes
// the host-side check, and the deliberately unsupported features fail it.
func TestGetOrderCapabilitiesDeclaresWhatTheBuilderSends(t *testing.T) {
	p := &BybitPlugin{}
	caps := p.GetOrderCapabilities()

	accepted := []ex.CreateOrderParams{
		{Market: perpMarket("BTCUSDT"), Side: acct.OrderSideBuy, Type: acct.OrderTypeLimit, Quantity: "1", Price: "25000"},
		{Market: tt.Market{Symbol: "BTCUSDT", AssetType: tt.AssetTypeSpot}, Side: acct.OrderSideBuy, Type: acct.OrderTypeMarket, QuoteQuantity: "200"},
		{Market: perpMarket("BTCUSDT"), Side: acct.OrderSideSell, Type: acct.OrderTypeStop, Quantity: "1", TriggerPrice: "24000", TriggerSource: acct.TriggerSourceIndex, ReduceOnly: true},
		{Market: perpMarket("BTCUSDT"), Side: acct.OrderSideBuy, Type: acct.OrderTypeLimit, Quantity: "1", Price: "25000", TimeInForce: acct.TimeInForcePostOnly, ClientOrderID: "cl-1"},
		{Market: perpMarket("BTCUSDT"), Side: acct.OrderSideBuy, Type: acct.OrderTypeTakeProfitLimit, Quantity: "1", Price: "29900", TriggerPrice: "30000"},
	}
	for i, o := range accepted {
		if err := caps.Check(o); err != nil {
			t.Fatalf("accepted[%d] (%s) must pass the capability check: %v", i, o.Type, err)
		}
		// Whatever the host lets through, the builder must actually build.
		if _, err := buildBybitCreateOrderRequest(o); err != nil {
			t.Fatalf("accepted[%d] (%s) must build: %v", i, o.Type, err)
		}
	}

	rejected := []ex.CreateOrderParams{
		{Market: tt.Market{Symbol: "BTCUSDT", AssetType: tt.AssetTypeOption}, Side: acct.OrderSideBuy, Type: acct.OrderTypeLimit, Quantity: "1", Price: "1"},
		{Market: perpMarket("BTCUSDT"), Side: acct.OrderSideSell, Type: acct.OrderTypeTrailingStop, Quantity: "1", TrailingPercent: "1.5"},
		{Market: perpMarket("BTCUSDT"), Side: acct.OrderSideBuy, Type: acct.OrderTypeLimit, Quantity: "1", Price: "1", TimeInForce: acct.TimeInForceGTD, ExpireTime: "1735689600000"},
		{Market: perpMarket("BTCUSDT"), Side: acct.OrderSideBuy, Type: acct.OrderTypeLimit, Quantity: "1", Price: "1", ClosePosition: true},
	}
	for i, o := range rejected {
		if err := caps.Check(o); err == nil {
			t.Fatalf("rejected[%d] (%s %s) must fail the capability check", i, o.Type, o.TimeInForce)
		}
	}
}

func TestNormalizeBybitOrderAbsentPrices(t *testing.T) {
	got := normalizeBybitOrder(bybit.Order{
		OrderID:      "1",
		Symbol:       "BTCUSDT",
		OrderType:    "Market",
		Price:        "0",
		TriggerPrice: "0.00000000",
		AvgPrice:     "0",
		Qty:          "1",
	}, "linear")

	if got.Price != "" {
		t.Fatalf("absent price must normalize to empty, got %q", got.Price)
	}
	if got.TriggerPrice != "" {
		t.Fatalf("absent trigger price must normalize to empty, got %q", got.TriggerPrice)
	}
	if got.AveragePrice != "" {
		t.Fatalf("absent average price must normalize to empty, got %q", got.AveragePrice)
	}
}

func TestNormalizeBybitOrderTriggerSource(t *testing.T) {
	got := normalizeBybitOrder(bybit.Order{OrderID: "1", Symbol: "BTCUSDT", TriggerBy: "MarkPrice"}, "linear")
	if got.TriggerSource != acct.TriggerSourceMark {
		t.Fatalf("triggerBy MarkPrice must map to mark, got %q", got.TriggerSource)
	}

	got = normalizeBybitOrder(bybit.Order{OrderID: "1", Symbol: "BTCUSDT"}, "linear")
	if got.TriggerSource != "" {
		t.Fatalf("absent triggerBy must stay empty (venue default), got %q", got.TriggerSource)
	}
}
