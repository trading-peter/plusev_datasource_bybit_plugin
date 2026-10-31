package main

import (
	"strconv"
	"strings"
	"time"

	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	"github.com/trading-peter/plusev_datasource_bybit_plugin/bybit"
)


// handleGetOrders returns open orders, querying both linear and spot categories.
func (p *BybitPlugin) handleGetOrders(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	orders := make([]acct.Order, 0)

	// Fetch linear (perps) orders
	linearResp, err := p.client.GetRealtimeOrders("linear")
	if err != nil {
		log.ErrorWithData("Failed to fetch linear orders", map[string]any{"error": err})
	} else {
		for _, o := range linearResp.Result.List {
			orders = append(orders, normalizeBybitOrder(o, "linear"))
		}
	}

	// Fetch spot orders
	spotResp, err := p.client.GetRealtimeOrders("spot")
	if err != nil {
		log.ErrorWithData("Failed to fetch spot orders", map[string]any{"error": err})
	} else {
		for _, o := range spotResp.Result.List {
			orders = append(orders, normalizeBybitOrder(o, "spot"))
		}
	}

	fetchedAt := time.Now()

	resp := acct.OrdersResponse{
		FetchedAt: fetchedAt,
		Scopes:    map[acct.ScopeType]acct.OrderScope{},
		Extra:     map[string]any{},
	}

	// Group orders by category scope
	linearOrders := make([]acct.Order, 0)
	spotOrders := make([]acct.Order, 0)

	for _, o := range orders {
		if o.Components["category"] == "spot" {
			spotOrders = append(spotOrders, o)
		} else {
			linearOrders = append(linearOrders, o)
		}
	}

	if len(linearOrders) > 0 {
		resp.Scopes[acct.ScopeFutures] = acct.OrderScope{Orders: linearOrders}
	}

	if len(spotOrders) > 0 {
		resp.Scopes[acct.ScopeSpot] = acct.OrderScope{Orders: spotOrders}
	}

	return plugin.SuccessResponse(resp, time.Second*3)
}


func normalizeBybitOrder(o bybit.Order, category string) acct.Order {
	order := acct.Order{
		OrderID:           o.OrderID,
		ClientOrderID:     o.OrderLinkId,
		Symbol:            o.Symbol,
		Side:              normalizeBybitOrderSide(o.Side),
		Type:              normalizeBybitOrderType(o.OrderType, o.StopOrderType),
		Status:            normalizeBybitOrderStatus(o.OrderStatus),
		Price:             o.Price,
		Quantity:          o.Qty,
		ExecutedQuantity:  o.CumExecQty,
		RemainingQuantity: o.LeavesQty,
		ReduceOnly:        o.ReduceOnly,
		Components: map[string]string{
			"category":      category,
			"time_in_force": o.TimeInForce,
			"trigger_price": o.TriggerPrice,
			"take_profit":   o.TakeProfit,
			"stop_loss":     o.StopLoss,
		},
		Extra: map[string]any{},
	}

	if o.AvgPrice != "" && o.AvgPrice != "0" {
		order.AveragePrice = o.AvgPrice
	}

	if t := parseBybitTimestamp(o.CreatedTime); !t.IsZero() {
		order.CreatedAt = t
	}

	if t := parseBybitTimestamp(o.UpdatedTime); !t.IsZero() {
		order.UpdatedAt = t
	}

	if o.TimeInForce == "PostOnly" {
		order.PostOnly = true
	}

	return order
}


func normalizeBybitOrderSide(side string) acct.OrderSide {
	s := strings.ToUpper(strings.TrimSpace(side))
	switch s {
	case "BUY":
		return acct.OrderSideBuy
	case "SELL":
		return acct.OrderSideSell
	}
	return acct.OrderSideUnknown
}


func normalizeBybitOrderType(orderType, stopOrderType string) acct.OrderType {
	ot := strings.ToUpper(strings.TrimSpace(orderType))
	st := strings.ToUpper(strings.TrimSpace(stopOrderType))

	// If there's a stop order type, map it first
	switch st {
	case "STOP", "STOP_LOSS", "STP":
		if ot == "LIMIT" {
			return acct.OrderTypeStopLimit
		}
		return acct.OrderTypeStop
	case "TAKE_PROFIT", "TP":
		if ot == "LIMIT" {
			return acct.OrderTypeTakeProfitLimit
		}
		return acct.OrderTypeTakeProfit
	case "TRAILING_STOP", "TPSL":
		return acct.OrderTypeTrailingStop
	}

	switch ot {
	case "LIMIT":
		return acct.OrderTypeLimit
	case "MARKET":
		return acct.OrderTypeMarket
	}
	return acct.OrderTypeUnknown
}


func normalizeBybitOrderStatus(status string) acct.OrderStatus {
	s := strings.ToUpper(strings.TrimSpace(safeReplace(status)))
	switch s {
	case "NEW", "UNTRIGGERED", "TRIGGERED":
		return acct.OrderStatusNew
	case "PARTIALLYFILLED", "PARTIALLY_FILLED":
		return acct.OrderStatusPartiallyFilled
	case "FILLED":
		return acct.OrderStatusFilled
	case "CANCELLED", "CANCELED":
		return acct.OrderStatusCanceled
	case "REJECTED":
		return acct.OrderStatusRejected
	case "EXPIRED", "DEACTIVATED":
		return acct.OrderStatusExpired
	case "":
		return acct.OrderStatusUnknown
	}
	return acct.OrderStatus(s)
}


func safeReplace(s string) string {
	return strings.NewReplacer(" ", "_", "-", "_").Replace(s)
}


func parseBybitTimestamp(ms string) time.Time {
	if ms == "" {
		return time.Time{}
	}

	n, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.UnixMilli(n)
}
