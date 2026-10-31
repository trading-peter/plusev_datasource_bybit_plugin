package main

import (
	"strings"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
	"github.com/trading-peter/plusev_datasource_bybit_plugin/bybit"
)


// handleCreateOrders places orders one at a time (Bybit has no batch endpoint).
// TP/SL can be attached directly to orders via takeProfit/stopLoss params.
func (p *BybitPlugin) handleCreateOrders(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.CreateOrdersParamsFromMap(params)
	if err := parsed.Validate(); err != nil {
		return plugin.ErrorResponse(err)
	}

	results := make([]acct.CreateOrderResult, len(parsed.Orders))

	for i := range results {
		results[i] = acct.CreateOrderResult{Index: i}
	}

	for i, order := range parsed.Orders {
		req, err := buildBybitCreateOrderRequest(order)
		if err != nil {
			results[i].Success = false
			results[i].Error = err.Error()
			continue
		}

		resp, err := p.client.CreateOrder(req)
		if err != nil {
			results[i].Success = false
			results[i].Error = err.Error()
			continue
		}

		results[i].Success = true
		results[i].ClientOrderID = order.ClientOrderID

		createdOrder := &acct.Order{
			OrderID:       resp.Result.OrderID,
			ClientOrderID: resp.Result.OrderLinkId,
			Symbol:        order.Market.Symbol,
			Side:          order.Side,
			Type:          order.Type,
			Quantity:      order.Quantity,
			Price:         order.Price,
		}

		results[i].Order = createdOrder
	}

	resp := acct.CreateOrdersResponse{
		SubmittedAt: time.Now(),
		Results:     results,
	}

	return plugin.SuccessResponse(resp)
}


// buildBybitCreateOrderRequest converts a normalized CreateOrderParams to a Bybit CreateOrderRequest.
func buildBybitCreateOrderRequest(order ex.CreateOrderParams) (bybit.CreateOrderRequest, error) {
	category := bybit.AssetTypeToCategory(order.Market.AssetType)
	symbol := strings.TrimSpace(order.Market.Symbol)
	if symbol == "" {
		return bybit.CreateOrderRequest{}, errString("market.symbol is required")
	}

	side := normalizeSideToBybit(order.Side)
	qty := order.Quantity
	price := strings.TrimSpace(order.Price)

	// Determine order type and Bybit-specific fields
	var orderType string
	var timeInForce string
	var triggerPrice string
	var triggerDirection int
	var takeProfit string
	var stopLoss string
	var marketUnit string

	switch order.Type {
	case acct.OrderTypeLimit:
		orderType = "Limit"

	case acct.OrderTypeMarket:
		orderType = "Market"

	case acct.OrderTypeStop, acct.OrderTypeStopMarket:
		orderType = "Market"
		triggerPrice = firstNonEmpty(order.TriggerPrice, order.StopPrice)
		if order.Side == acct.OrderSideSell {
			triggerDirection = 2 // price falls
		} else {
			triggerDirection = 1 // price rises
		}

	case acct.OrderTypeStopLimit:
		orderType = "Limit"
		triggerPrice = firstNonEmpty(order.TriggerPrice, order.StopPrice)
		if price == "" {
			return bybit.CreateOrderRequest{}, errString("price is required for stop_limit orders")
		}
		if order.Side == acct.OrderSideSell {
			triggerDirection = 2
		} else {
			triggerDirection = 1
		}

	case acct.OrderTypeTakeProfit, acct.OrderTypeTakeProfitMarket:
		orderType = "Market"
		triggerPrice = firstNonEmpty(order.TriggerPrice, order.TakeProfitPrice)
		if order.Side == acct.OrderSideSell {
			triggerDirection = 1 // price rises to TP
		} else {
			triggerDirection = 2 // price falls to TP
		}

	case acct.OrderTypeTakeProfitLimit:
		orderType = "Limit"
		triggerPrice = firstNonEmpty(order.TriggerPrice, order.TakeProfitPrice)
		if price == "" {
			return bybit.CreateOrderRequest{}, errString("price is required for take_profit_limit orders")
		}
		if order.Side == acct.OrderSideSell {
			triggerDirection = 1
		} else {
			triggerDirection = 2
		}

	case acct.OrderTypeTrailingStop:
		orderType = "Market"
		triggerPrice = firstNonEmpty(order.ActivationPrice, order.TriggerPrice)

	default:
		return bybit.CreateOrderRequest{}, errString("unsupported order type: " + string(order.Type))
	}

	// Time-in-force
	switch {
	case order.PostOnly:
		timeInForce = "PostOnly"
	case order.TimeInForce == acct.TimeInForceIOC:
		timeInForce = "IOC"
	case order.TimeInForce == acct.TimeInForceFOK:
		timeInForce = "FOK"
	case order.TimeInForce == acct.TimeInForceGTX:
		timeInForce = "PostOnly"
	default:
		if orderType == "Limit" {
			timeInForce = "GTC"
		} else {
			timeInForce = "IOC"
		}
	}

	// Attached TP/SL on the order itself
	if tp := strings.TrimSpace(order.TakeProfitPrice); tp != "" {
		takeProfit = tp
	}

	if sl := strings.TrimSpace(order.StopPrice); sl != "" {
		stopLoss = sl
	}

	// Spot market buy: use quote quantity if QuoteQuantity is set
	if category == "spot" && orderType == "Market" && order.QuoteQuantity != "" {
		qty = order.QuoteQuantity
		marketUnit = "quoteCoin"
	}

	// If no quantity, try QuoteQuantity for non-spot
	if qty == "" && order.QuoteQuantity != "" && category != "spot" {
		qty = order.QuoteQuantity
	}

	req := bybit.CreateOrderRequest{
		Category:         category,
		Symbol:           symbol,
		Side:             side,
		OrderType:        orderType,
		Qty:              qty,
		Price:            price,
		TimeInForce:      timeInForce,
		ReduceOnly:       order.ReduceOnly,
		TakeProfit:       takeProfit,
		StopLoss:         stopLoss,
		PositionIdx:      0,
		TriggerPrice:     triggerPrice,
		TriggerDirection: triggerDirection,
		MarketUnit:       marketUnit,
	}

	if order.ClientOrderID != "" {
		req.OrderLinkId = order.ClientOrderID
	}

	return req, nil
}


func normalizeSideToBybit(side acct.OrderSide) string {
	switch side {
	case acct.OrderSideBuy:
		return "Buy"
	case acct.OrderSideSell:
		return "Sell"
	}
	return "Buy"
}


func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}


type errString string

func (e errString) Error() string { return string(e) }
