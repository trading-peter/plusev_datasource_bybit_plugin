package bybit

import (
	"encoding/json"
	"fmt"

	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
)


// GetRealtimeOrders queries open orders for a given category.
// Bybit's /v5/order/realtime returns unfilled or partially filled orders by default.
func (c *Client) GetRealtimeOrders(category string) (OrderRealtimeResponse, error) {
	endpoint := c.baseURL + "/v5/order/realtime?category=" + category

	req := &rt.Request{
		Method: "GET",
		URL:    endpoint,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp OrderRealtimeResponse
	_, err := c.requester.Send(req, &resp)
	if err != nil {
		return OrderRealtimeResponse{}, fmt.Errorf("failed to fetch realtime orders: %w", err)
	}

	if !resp.OK() {
		return resp, fmt.Errorf("bybit order/realtime returned retCode=%d retMsg=%s", resp.RetCode, resp.RetMsg)
	}

	return resp, nil
}


// CreateOrder places a single order via POST /v5/order/create.
func (c *Client) CreateOrder(order CreateOrderRequest) (CreateOrderResponse, error) {
	body, err := json.Marshal(order)
	if err != nil {
		return CreateOrderResponse{}, fmt.Errorf("failed to marshal create order: %w", err)
	}

	req := &rt.Request{
		Method: "POST",
		URL:    c.baseURL + "/v5/order/create",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}

	c.addAuthHeaders(req, string(body))

	var resp CreateOrderResponse
	_, sendErr := c.requester.Send(req, &resp)
	if sendErr != nil {
		return CreateOrderResponse{}, fmt.Errorf("failed to create order: %w", sendErr)
	}

	if !resp.OK() {
		return resp, fmt.Errorf("bybit order/create returned retCode=%d retMsg=%s", resp.RetCode, resp.RetMsg)
	}

	return resp, nil
}
