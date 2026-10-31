package bybit

import (
	"encoding/json"
	"fmt"
	"strings"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)


// CancelOrder cancels a single order by orderId or orderLinkId.
func (c *Client) CancelOrder(req CancelOrderRequest) (CancelOrderResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CancelOrderResponse{}, fmt.Errorf("failed to marshal cancel order: %w", err)
	}

	httpReq := &rt.Request{
		Method: "POST",
		URL:    c.baseURL + "/v5/order/cancel",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}

	c.addAuthHeaders(httpReq, string(body))

	var resp CancelOrderResponse
	_, sendErr := c.requester.Send(httpReq, &resp)
	if sendErr != nil {
		return CancelOrderResponse{}, fmt.Errorf("failed to cancel order: %w", sendErr)
	}

	if !resp.OK() {
		return resp, fmt.Errorf("bybit order/cancel returned retCode=%d retMsg=%s", resp.RetCode, resp.RetMsg)
	}

	return resp, nil
}


// CancelAllOrders cancels all orders for a category.
// If symbol is provided, cancels only that symbol's orders.
func (c *Client) CancelAllOrders(category, symbol string) (CancelAllResponse, error) {
	payload := map[string]string{
		"category": category,
	}

	if strings.TrimSpace(symbol) != "" {
		payload["symbol"] = symbol
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return CancelAllResponse{}, fmt.Errorf("failed to marshal cancel-all: %w", err)
	}

	req := &rt.Request{
		Method: "POST",
		URL:    c.baseURL + "/v5/order/cancel-all",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}

	c.addAuthHeaders(req, string(body))

	var resp CancelAllResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CancelAllResponse{}, fmt.Errorf("failed to cancel all orders: %w", sendErr)
	}

	if !resp.OK() {
		return resp, fmt.Errorf("bybit order/cancel-all returned retCode=%d retMsg=%s", resp.RetCode, resp.RetMsg)
	}

	return resp, nil
}
