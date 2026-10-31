package main

import (
	"strings"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
	"github.com/trading-peter/plusev_datasource_bybit_plugin/bybit"
)


// handleCancelOrders cancels orders by ID or cancels all.
// If IDs is empty, cancels all orders for the market's category.
// If IDs are provided, cancels each by orderId or orderLinkId.
func (p *BybitPlugin) handleCancelOrders(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	parsed := ex.CancelOrdersParamsFromMap(params)
	if err := parsed.Validate(); err != nil {
		return plugin.ErrorResponse(err)
	}

	category := bybit.AssetTypeToCategory(parsed.Market.AssetType)
	symbol := strings.TrimSpace(parsed.Market.Symbol)

	results := make([]acct.CancelOrderResult, 0)

	if len(parsed.IDs) == 0 {
		// Cancel all orders
		_, err := p.client.CancelAllOrders(category, symbol)

		res := acct.CancelOrderResult{Index: 0}

		if err != nil {
			res.Success = false
			res.Error = err.Error()
		} else {
			res.Success = true
		}

		results = append(results, res)

		resp := acct.CancelOrdersResponse{
			SubmittedAt: time.Now(),
			Results:     results,
		}

		return plugin.SuccessResponse(resp)
	}

	for i, rawID := range parsed.IDs {
		id := strings.TrimSpace(rawID)
		res := acct.CancelOrderResult{Index: i}

		if id == "" {
			res.Success = false
			res.Error = "id is required"
			results = append(results, res)
			continue
		}

		cancelReq := bybit.CancelOrderRequest{
			Category: category,
			Symbol:   symbol,
		}

		// Determine if this is an orderId or orderLinkId.
		// Bybit orderIds are numeric strings; orderLinkIds are user-customized.
		// We try orderId first, then fall back to orderLinkId.
		if isNumeric(id) {
			cancelReq.OrderID = id
			res.OrderID = id
		} else {
			cancelReq.OrderLinkId = id
			res.ClientOrderID = id
		}

		_, err := p.client.CancelOrder(cancelReq)
		if err != nil {
			// If orderId failed, try as orderLinkId
			if isNumeric(id) {
				cancelReq.OrderID = ""
				cancelReq.OrderLinkId = id
				_, err2 := p.client.CancelOrder(cancelReq)
				if err2 == nil {
					res.Success = true
					res.ClientOrderID = id
					results = append(results, res)
					continue
				}
			}

			res.Success = false
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		res.Success = true
		results = append(results, res)
	}

	resp := acct.CancelOrdersResponse{
		SubmittedAt: time.Now(),
		Results:     results,
	}

	return plugin.SuccessResponse(resp)
}


func isNumeric(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}
